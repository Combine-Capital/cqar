package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Combine-Capital/cqar/internal/bootstrap"
	"github.com/Combine-Capital/cqar/internal/bootstrap/seeder"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

var (
	configPath = flag.String("config", "config.bootstrap.yaml", "Path to bootstrap configuration file")
	dataDir    = flag.String("data-dir", "bootstrap_data", "Directory containing JSON data files")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
)

// authInterceptor adds authorization header to all gRPC requests
func authInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Add authorization header with dev API key (matches config.yaml auth.api_keys)
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer dev_key_cqmd_12345")
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func main() {
	flag.Parse()

	// Setup logging
	setupLogging(*verbose)

	log.Info().
		Str("config", *configPath).
		Str("data_dir", *dataDir).
		Msg("Starting CQAR file-based bootstrap")

	// Load configuration
	cfg, err := bootstrap.LoadConfig(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// Setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
	}()

	// Connect to CQAR gRPC service
	log.Info().Str("endpoint", cfg.CQAR.GRPCEndpoint).Msg("Connecting to CQAR service")
	conn, err := grpc.Dial(
		cfg.CQAR.GRPCEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(authInterceptor()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to CQAR")
	}
	defer conn.Close()

	// Create AssetRegistry client
	assetClient := servicesv1.NewAssetRegistryClient(conn)

	// Verify connectivity by listing chains
	log.Info().Msg("Verifying CQAR service connectivity...")
	_, err = assetClient.ListChains(ctx, &servicesv1.ListChainsRequest{})
	if err != nil {
		log.Fatal().
			Str("endpoint", cfg.CQAR.GRPCEndpoint).
			Err(err).
			Msg("Failed to connect to CQAR service - is it running?")
	}
	log.Info().Msg("CQAR service is reachable")

	// Get absolute path to data directory
	absDataDir, err := filepath.Abs(*dataDir)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to resolve data directory path")
	}

	// Create file-based seeder
	fileSeeder := seeder.NewFileBasedSeeder(assetClient, absDataDir)

	// Run seeding
	if err := fileSeeder.SeedAll(ctx); err != nil {
		log.Fatal().Err(err).Msg("Seeding failed")
	}

	log.Info().Msg("Bootstrap completed successfully")
}

// setupLogging configures structured logging with zerolog
func setupLogging(verbose bool) {
	// Configure pretty logging for console output
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()

	// Set log level
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
