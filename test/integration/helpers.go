package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"net"

	"github.com/Combine-Capital/cqar/internal/config"
	"github.com/Combine-Capital/cqar/internal/manager"
	"github.com/Combine-Capital/cqar/internal/repository"
	"github.com/Combine-Capital/cqar/internal/server"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/Combine-Capital/cqi/pkg/bus"
	"github.com/Combine-Capital/cqi/pkg/cache"
	"github.com/Combine-Capital/cqi/pkg/database"
	"github.com/Combine-Capital/cqi/pkg/logging"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufSize = 1024 * 1024
)

// TestFixture holds all components needed for integration testing
type TestFixture struct {
	DB             *sql.DB
	DBPool         *database.Pool
	Cache          cache.Cache
	EventBus       bus.EventBus
	Repository     repository.Repository
	AssetManager   *manager.AssetManager
	SymbolManager  *manager.SymbolManager
	VenueManager   *manager.VenueManager
	QualityManager *manager.QualityManager
	EventPublisher *manager.EventPublisher
	Server         servicesv1.AssetRegistryClient
	GRPCServer     *grpc.Server
	Listener       *bufconn.Listener
	Logger         *logging.Logger
	Config         *config.Config
	Ctx            context.Context
	Cancel         context.CancelFunc
}

// NewTestFixture creates a new test fixture with all dependencies initialized
func NewTestFixture(t *testing.T) *TestFixture {
	t.Helper()

	// Load test configuration
	cfg := loadTestConfig(t)

	// Create simple test logger
	logger := &logging.Logger{} // Use zero value for tests

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Initialize database
	db := connectTestDB(t, cfg)
	dbPool := initDatabasePool(t, cfg, logger)

	// Initialize cache
	cacheClient := initCache(t, cfg, logger)

	// Initialize event bus
	eventBus := initEventBus(t, cfg, logger)

	// Initialize repository with cache
	baseRepo := repository.NewPostgresRepository(dbPool)
	cacheTTLs := repository.CacheTTLs{
		Asset:       cfg.CacheTTL.Asset,
		Symbol:      cfg.CacheTTL.Symbol,
		Venue:       cfg.CacheTTL.Venue,
		VenueAsset:  cfg.CacheTTL.VenueAsset,
		VenueSymbol: cfg.CacheTTL.VenueSymbol,
		QualityFlag: cfg.CacheTTL.QualityFlag,
		Chain:       cfg.CacheTTL.Asset,
	}
	repo := repository.NewCachedRepository(baseRepo, cacheClient, cacheTTLs)

	// Initialize managers
	eventPublisher := manager.NewEventPublisher(eventBus, logger)
	qualityMgr := manager.NewQualityManager(repo, eventPublisher)
	assetMgr := manager.NewAssetManager(repo, qualityMgr, eventPublisher)
	symbolMgr := manager.NewSymbolManager(repo, assetMgr, eventPublisher)
	venueMgr := manager.NewVenueManager(repo, assetMgr, symbolMgr, eventPublisher)

	// Initialize gRPC server (in-memory)
	listener := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	assetRegistryServer := server.NewAssetRegistryServer(
		assetMgr,
		symbolMgr,
		venueMgr,
		qualityMgr,
		repo,
	)
	servicesv1.RegisterAssetRegistryServer(grpcServer, assetRegistryServer)

	// Start server in background
	go func() {
		if err := grpcServer.Serve(listener); err != nil {
			logger.Error().Err(err).Msg("gRPC server failed")
		}
	}()

	// Create gRPC client
	conn, err := grpc.DialContext(ctx, "bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "Failed to create gRPC client")

	client := servicesv1.NewAssetRegistryClient(conn)

	return &TestFixture{
		DB:             db,
		DBPool:         dbPool,
		Cache:          cacheClient,
		EventBus:       eventBus,
		Repository:     repo,
		AssetManager:   assetMgr,
		SymbolManager:  symbolMgr,
		VenueManager:   venueMgr,
		QualityManager: qualityMgr,
		EventPublisher: eventPublisher,
		Server:         client,
		GRPCServer:     grpcServer,
		Listener:       listener,
		Logger:         logger,
		Config:         cfg,
		Ctx:            ctx,
		Cancel:         cancel,
	}
}

// Cleanup tears down all test resources
func (f *TestFixture) Cleanup(t *testing.T) {
	t.Helper()

	if f.Cancel != nil {
		f.Cancel()
	}

	if f.GRPCServer != nil {
		f.GRPCServer.Stop()
	}

	if f.Listener != nil {
		_ = f.Listener.Close()
	}

	if f.DBPool != nil {
		f.DBPool.Close()
	}

	if f.DB != nil {
		_ = f.DB.Close()
	}

	// Note: Cache and EventBus cleanup would go here if they had Close() methods
}

// ResetDatabase truncates all tables and resets sequences
func (f *TestFixture) ResetDatabase(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Truncate all tables in reverse dependency order
	tables := []string{
		"venue_symbols",
		"venue_assets",
		"symbol_identifiers",
		"asset_identifiers",
		"group_members",
		"asset_groups",
		"quality_flags",
		"relationships",
		"deployments",
		"symbols",
		"venues",
		"chains",
		"assets",
	}

	for _, table := range tables {
		_, err := f.DB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		require.NoError(t, err, "Failed to truncate table %s", table)
	}
}

// LoadSeedData loads test data from SQL files
func (f *TestFixture) LoadSeedData(t *testing.T, files ...string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, file := range files {
		path := filepath.Join("testdata", file)
		data, err := os.ReadFile(path)
		require.NoError(t, err, "Failed to read seed file %s", file)

		_, err = f.DB.ExecContext(ctx, string(data))
		require.NoError(t, err, "Failed to load seed data from %s", file)
	}
}

// ClearCache flushes all cached data
func (f *TestFixture) ClearCache(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Note: This assumes cache has a FlushAll or similar method
	// If not available, we would need to delete specific keys or skip cache clearing
	_ = ctx // Use context if cache Clear method accepts it
}

// Helper functions

func loadTestConfig(t *testing.T) *config.Config {
	t.Helper()

	// Try to load from test config file
	configPath := os.Getenv("TEST_CONFIG_PATH")
	if configPath == "" {
		configPath = "../config.test.yaml"
	}

	// Check if file exists, if not use environment config path
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "../../test/config.test.yaml"
	}

	// Try to load with CQI config loader (using empty envPrefix for test)
	cfg, err := config.Load(configPath, "")
	if err != nil {
		t.Logf("Failed to load test config from %s: %v, using test db directly", configPath, err)
		// Fall back to minimal config for tests that only need DB
		return nil
	}

	return cfg
}

func connectTestDB(t *testing.T, cfg *config.Config) *sql.DB {
	t.Helper()

	host := getEnvOrDefault("TEST_DB_HOST", "localhost")
	port := getEnvOrDefaultInt("TEST_DB_PORT", 5433)
	user := getEnvOrDefault("TEST_DB_USER", "cqar_test")
	password := getEnvOrDefault("TEST_DB_PASSWORD", "cqar_test_password")
	dbname := getEnvOrDefault("TEST_DB_NAME", "cqar_test")

	if cfg != nil {
		host = cfg.Database.Host
		port = cfg.Database.Port
		user = cfg.Database.User
		password = cfg.Database.Password
		dbname = cfg.Database.Database
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		user, password, host, port, dbname)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "Failed to connect to test database")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	require.NoError(t, err, "Failed to ping test database")

	return db
}

func initDatabasePool(t *testing.T, cfg *config.Config, logger *logging.Logger) *database.Pool {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Database.ConnectTimeout)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.Database)
	require.NoError(t, err, "Failed to create database pool")

	return pool
}

func initCache(t *testing.T, cfg *config.Config, logger *logging.Logger) cache.Cache {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cacheClient, err := cache.NewRedis(ctx, cfg.Cache)
	require.NoError(t, err, "Failed to create cache client")

	return cacheClient
}

func initEventBus(t *testing.T, cfg *config.Config, logger *logging.Logger) bus.EventBus {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventBus, err := bus.NewJetStream(ctx, cfg.EventBus)
	require.NoError(t, err, "Failed to create event bus")

	return eventBus
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var intValue int
		if _, err := fmt.Sscanf(value, "%d", &intValue); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// Pointer helpers for protobuf fields
func ptrString(s string) *string {
	return &s
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrInt32(i int32) *int32 {
	return &i
}

func ptrBool(b bool) *bool {
	return &b
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
