package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Combine-Capital/cqar/internal/config"
	"github.com/Combine-Capital/cqar/internal/manager"
	"github.com/Combine-Capital/cqar/internal/repository"
	"github.com/Combine-Capital/cqar/internal/server"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/Combine-Capital/cqi/pkg/auth"
	"github.com/Combine-Capital/cqi/pkg/bus"
	"github.com/Combine-Capital/cqi/pkg/database"
	"github.com/Combine-Capital/cqi/pkg/logging"
	"github.com/Combine-Capital/cqi/pkg/metrics"
	cqiservice "github.com/Combine-Capital/cqi/pkg/service"
	"github.com/Combine-Capital/cqi/pkg/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Service implements the CQI service interface for CQAR.
// It manages the lifecycle of all service components: database, repository, managers,
// gRPC server, and HTTP health server.
type Service struct {
	cfg         *config.Config
	logger      *logging.Logger
	dbPool      *database.Pool
	eventBus    bus.EventBus
	grpcService *cqiservice.GRPCService
	httpService *cqiservice.HTTPService
}

// New creates a new CQAR service instance with the given configuration and logger.
func New(cfg *config.Config, logger *logging.Logger) *Service {
	return &Service{
		cfg:    cfg,
		logger: logger,
	}
}

// Start initializes all service components and starts the gRPC and HTTP servers.
// Initialization order:
// 1. Database pool
// 2. Event bus
// 3. Repository layer
// 4. Event publisher
// 5. Business logic managers
// 6. gRPC server with AssetRegistry implementation
// 7. HTTP health server
func (s *Service) Start(ctx context.Context) error {
	s.logger.Info().Msg("Initializing CQAR service components")

	// Initialize database pool
	if err := s.initDatabase(ctx); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize event bus
	if err := s.initEventBus(ctx); err != nil {
		return fmt.Errorf("failed to initialize event bus: %w", err)
	}

	// Initialize repository
	repo := repository.NewPostgresRepository(s.dbPool)

	// Initialize event publisher
	eventPublisher := manager.NewEventPublisher(s.eventBus, s.logger)

	// Initialize managers with event publisher
	qualityMgr := manager.NewQualityManager(repo, eventPublisher)
	assetMgr := manager.NewAssetManager(repo, qualityMgr, eventPublisher)
	symbolMgr := manager.NewSymbolManager(repo, assetMgr, eventPublisher)
	venueMgr := manager.NewVenueManager(repo, assetMgr, symbolMgr, eventPublisher)

	// Create gRPC server with AssetRegistry implementation
	assetRegistryServer := server.NewAssetRegistryServer(
		assetMgr,
		symbolMgr,
		venueMgr,
		qualityMgr,
		repo,
	)

	// Build interceptor chain: auth → logging → metrics → tracing
	// Note: Interceptors are applied in reverse order (last interceptor wraps first)
	unaryInterceptors := []grpc.UnaryServerInterceptor{
		tracing.GRPCUnaryServerInterceptor(s.cfg.Service.Name),  // Tracing (innermost - wraps handler)
		metrics.UnaryServerInterceptor(s.cfg.Metrics.Namespace), // Metrics
		logging.UnaryServerInterceptor(s.logger),                // Logging
		auth.APIKeyUnaryInterceptor(s.cfg.Auth.APIKeys),         // Auth (outermost - first to execute)
	}

	streamInterceptors := []grpc.StreamServerInterceptor{
		tracing.GRPCStreamServerInterceptor(s.cfg.Service.Name),  // Tracing (innermost - wraps handler)
		metrics.StreamServerInterceptor(s.cfg.Metrics.Namespace), // Metrics
		logging.StreamServerInterceptor(s.logger),                // Logging
		auth.APIKeyStreamInterceptor(s.cfg.Auth.APIKeys),         // Auth (outermost - first to execute)
	}

	// Create gRPC server with interceptor chain
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(unaryInterceptors...),
		grpc.ChainStreamInterceptor(streamInterceptors...),
	)
	servicesv1.RegisterAssetRegistryServer(grpcServer, assetRegistryServer)
	grpc_health_v1.RegisterHealthServer(grpcServer, NewHealthServer(s.dbPool))
	reflection.Register(grpcServer) // Enable reflection for grpcurl

	// Create gRPC service wrapper
	grpcAddr := fmt.Sprintf(":%d", s.cfg.Server.GRPCPort)
	s.grpcService = cqiservice.NewGRPCServiceWithServer(
		"cqar-grpc",
		grpcAddr,
		grpcServer,
		cqiservice.WithGRPCShutdownTimeout(s.cfg.Server.ShutdownTimeout),
	)

	// Create HTTP health server
	httpAddr := fmt.Sprintf(":%d", s.cfg.Server.HTTPPort)
	httpHandler := s.createHealthHandler(repo)
	s.httpService = cqiservice.NewHTTPService(
		"cqar-http",
		httpAddr,
		httpHandler,
		cqiservice.WithReadTimeout(s.cfg.Server.ReadTimeout),
		cqiservice.WithWriteTimeout(s.cfg.Server.WriteTimeout),
		cqiservice.WithShutdownTimeout(s.cfg.Server.ShutdownTimeout),
	)

	// Start gRPC server
	s.logger.Info().
		Int("port", s.cfg.Server.GRPCPort).
		Msg("Starting gRPC server")
	if err := s.grpcService.Start(ctx); err != nil {
		return fmt.Errorf("failed to start gRPC server: %w", err)
	}

	// Start HTTP server
	s.logger.Info().
		Int("port", s.cfg.Server.HTTPPort).
		Msg("Starting HTTP health server")
	if err := s.httpService.Start(ctx); err != nil {
		// Stop gRPC server if HTTP server fails to start
		_ = s.grpcService.Stop(context.Background())
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}

	s.logger.Info().
		Int("grpc_port", s.cfg.Server.GRPCPort).
		Int("http_port", s.cfg.Server.HTTPPort).
		Msg("CQAR service started successfully")

	return nil
}

// Stop gracefully shuts down all service components.
// Components are stopped in reverse order of initialization:
// 1. HTTP server (stop accepting health checks)
// 2. gRPC server (drain in-flight requests)
// 3. Event bus (flush pending events)
// 4. Database pool (close connections)
func (s *Service) Stop(ctx context.Context) error {
	s.logger.Info().Msg("Shutting down CQAR service")

	// Stop HTTP server
	if s.httpService != nil {
		if err := s.httpService.Stop(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Failed to stop HTTP server")
		}
	}

	// Stop gRPC server
	if s.grpcService != nil {
		if err := s.grpcService.Stop(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Failed to stop gRPC server")
		}
	}

	// Close event bus
	if s.eventBus != nil {
		if err := s.eventBus.Close(); err != nil {
			s.logger.Error().Err(err).Msg("Failed to close event bus")
		} else {
			s.logger.Info().Msg("Event bus closed")
		}
	}

	// Close database pool
	if s.dbPool != nil {
		s.dbPool.Close()
		s.logger.Info().Msg("Database pool closed")
	}

	s.logger.Info().Msg("CQAR service stopped successfully")
	return nil
}

// Name returns the service name for identification.
func (s *Service) Name() string {
	return s.cfg.Service.Name
}

// Health performs a health check on the service.
// It checks database connectivity and returns an error if unhealthy.
func (s *Service) Health() error {
	if s.dbPool == nil {
		return fmt.Errorf("database pool not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.Database.ConnectTimeout)
	defer cancel()
	return s.dbPool.HealthCheck(ctx)
}

// initDatabase initializes the PostgreSQL database connection pool.
func (s *Service) initDatabase(ctx context.Context) error {
	s.logger.Info().
		Str("host", s.cfg.Database.Host).
		Int("port", s.cfg.Database.Port).
		Str("database", s.cfg.Database.Database).
		Msg("Connecting to database")

	pool, err := database.NewPool(ctx, s.cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to create database pool: %w", err)
	}

	// Verify connectivity
	if err := pool.HealthCheck(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("database health check failed: %w", err)
	}

	s.dbPool = pool
	s.logger.Info().Msg("Database connection established")
	return nil
}

// initEventBus initializes the NATS JetStream event bus.
func (s *Service) initEventBus(ctx context.Context) error {
	s.logger.Info().
		Strs("servers", s.cfg.EventBus.Servers).
		Str("stream_name", s.cfg.EventBus.StreamName).
		Msg("Connecting to event bus")

	eventBus, err := bus.NewJetStream(ctx, s.cfg.EventBus)
	if err != nil {
		return fmt.Errorf("failed to create event bus: %w", err)
	}

	s.eventBus = eventBus
	s.logger.Info().Msg("Event bus connection established")
	return nil
}

// createHealthHandler creates an HTTP handler for health check endpoints.
func (s *Service) createHealthHandler(repo repository.Repository) http.Handler {
	mux := http.NewServeMux()

	// Liveness endpoint - always returns 200 if the process is running
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Readiness endpoint - returns 200 if service is ready to accept traffic
	// Checks database connectivity
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), s.cfg.Database.ConnectTimeout)
		defer cancel()

		// Check database health
		if err := repo.Ping(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Readiness check failed: database unhealthy")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(fmt.Sprintf(`{"status":"unhealthy","component":"database","error":"%s"}`, err.Error())))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready","components":{"database":"ok"}}`))
	})

	// Health endpoint - comprehensive health check (alias for /health/ready)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), s.cfg.Database.ConnectTimeout)
		defer cancel()

		if err := repo.Ping(ctx); err != nil {
			s.logger.Error().Err(err).Msg("Health check failed: database unhealthy")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(fmt.Sprintf(`{"status":"unhealthy","component":"database","error":"%s"}`, err.Error())))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","components":{"database":"ok"}}`))
	})

	return mux
}

// HealthServer implements gRPC health checking protocol.
type HealthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	dbPool *database.Pool
}

// NewHealthServer creates a new gRPC health server.
func NewHealthServer(dbPool *database.Pool) *HealthServer {
	return &HealthServer{
		dbPool: dbPool,
	}
}

// Check performs a health check.
func (h *HealthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	// Check database connectivity
	if err := h.dbPool.HealthCheck(ctx); err != nil {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch performs a streaming health check (not implemented).
func (h *HealthServer) Watch(req *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	// For MVP, we don't implement streaming health checks
	return nil
}
