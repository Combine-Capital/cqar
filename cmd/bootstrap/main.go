package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Combine-Capital/cqar/internal/bootstrap"
	"github.com/Combine-Capital/cqar/internal/bootstrap/client"
	"github.com/Combine-Capital/cqar/internal/bootstrap/seeder"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	configPath = flag.String("config", "config.bootstrap.yaml", "Path to bootstrap configuration file")
	dryRun     = flag.Bool("dry-run", false, "Perform a dry run without creating entities")
	limit      = flag.Int("limit", 0, "Limit the number of assets to process (0 = no limit)")
	verbose    = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Setup logging
	setupLogging(*verbose)

	log.Info().
		Str("config", *configPath).
		Bool("dry_run", *dryRun).
		Int("limit", *limit).
		Msg("Starting CQAR bootstrap tool")

	// Load configuration
	cfg, err := bootstrap.LoadConfig(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatal().Err(err).Msg("Invalid configuration")
	}

	// Warn about rate limits if no CoinGecko API key
	if cfg.CoinGecko.APIKey == "" {
		log.Warn().
			Msg("CoinGecko API key not provided - using free tier (10-50 requests/minute)")
		log.Warn().
			Msg("For higher rate limits, set COINGECKO_API_KEY environment variable or add to config")
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

	// Create API clients
	clients, err := createClients(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create API clients")
	}
	defer clients.Close()

	log.Info().
		Str("coinbase_url", cfg.Coinbase.BaseURL).
		Str("coingecko_url", cfg.CoinGecko.BaseURL).
		Str("cqar_endpoint", cfg.CQAR.GRPCEndpoint).
		Msg("API clients initialized successfully")

	// Run seeding process
	if err := runSeeding(ctx, clients, cfg); err != nil {
		log.Fatal().Err(err).Msg("Seeding failed")
	}

	log.Info().Msg("Bootstrap completed successfully")
}

// runSeeding executes the seeding workflow: chains -> assets
func runSeeding(ctx context.Context, clients *Clients, cfg *bootstrap.Config) error {
	// Step 1: Seed chains
	log.Info().Msg("=== Step 1: Seeding Chains ===")
	chainSeeder := seeder.NewChainSeeder(clients.CQAR, *dryRun)
	chainResult, err := chainSeeder.SeedChains(ctx)
	if err != nil {
		return fmt.Errorf("seed chains: %w", err)
	}

	logSeedResult("Chains", chainResult)

	// Step 2: Seed assets
	log.Info().Msg("=== Step 2: Seeding Assets ===")
	assetSeeder := seeder.NewAssetSeeder(clients.CoinGecko, clients.CQAR, *dryRun, *limit)
	assetResult, err := assetSeeder.SeedAssets(ctx)
	if err != nil {
		return fmt.Errorf("seed assets: %w", err)
	}

	logSeedResult("Assets", assetResult)

	// Step 3: Seed deployments using cached asset data
	log.Info().Msg("=== Step 3: Seeding Deployments ===")
	deploymentSeeder := seeder.NewDeploymentSeeder(
		clients.CoinGecko,
		clients.CQAR,
		*dryRun,
		*limit,
		assetSeeder.GetAssetDetailsCache(), // Pass cached data to avoid duplicate API calls
	)
	deploymentResult, err := deploymentSeeder.SeedDeployments(ctx)
	if err != nil {
		return fmt.Errorf("seed deployments: %w", err)
	}

	logSeedResult("Deployments", deploymentResult)

	// Step 4: Summary
	log.Info().Msg("=== Bootstrap Summary ===")
	log.Info().
		Int("chains_processed", chainResult.TotalProcessed).
		Int("chains_succeeded", chainResult.Succeeded).
		Int("chains_failed", chainResult.Failed).
		Int("chains_skipped", chainResult.Skipped).
		Int("assets_processed", assetResult.TotalProcessed).
		Int("assets_succeeded", assetResult.Succeeded).
		Int("assets_failed", assetResult.Failed).
		Int("assets_skipped", assetResult.Skipped).
		Int("deployments_processed", deploymentResult.TotalProcessed).
		Int("deployments_succeeded", deploymentResult.Succeeded).
		Int("deployments_failed", deploymentResult.Failed).
		Int("deployments_skipped", deploymentResult.Skipped).
		Msg("Bootstrap complete")

	return nil
}

// logSeedResult logs detailed seeding results
func logSeedResult(entityType string, result *bootstrap.SeedResult) {
	log.Info().
		Str("entity_type", entityType).
		Int("total", result.TotalProcessed).
		Int("succeeded", result.Succeeded).
		Int("failed", result.Failed).
		Int("skipped", result.Skipped).
		Msg("Seeding results")

	// Log failures
	if len(result.Errors) > 0 {
		log.Warn().
			Int("count", len(result.Errors)).
			Msg("Failed entities")
		for i, err := range result.Errors {
			if i >= 10 {
				log.Warn().
					Int("remaining", len(result.Errors)-10).
					Msg("... and more failures (showing first 10)")
				break
			}
			log.Error().
				Str("entity", err.Entity).
				Str("reason", err.Reason).
				Err(err.Error).
				Msg("Seed failure")
		}
	}

	// Log skipped entities
	if len(result.SkippedReasons) > 0 {
		log.Info().
			Int("count", len(result.SkippedReasons)).
			Msg("Skipped entities")
		for i, skip := range result.SkippedReasons {
			if i >= 10 {
				log.Info().
					Int("remaining", len(result.SkippedReasons)-10).
					Msg("... and more skipped (showing first 10)")
				break
			}
			log.Debug().
				Str("entity", skip.Entity).
				Str("reason", skip.Reason).
				Msg("Skipped")
		}
	}
}

// setupLogging configures structured logging with zerolog
func setupLogging(verbose bool) {
	// Configure pretty logging for console output
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	})

	// Set log level
	if verbose {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// createClients initializes all external API clients
func createClients(ctx context.Context, cfg *bootstrap.Config) (*Clients, error) {
	// Create Coinbase client
	coinbaseClient := client.NewCoinbaseClient(cfg.Coinbase.BaseURL, cfg.Coinbase.APIKey)

	// Create CoinGecko client with rate limiting
	coinGeckoClient := client.NewCoinGeckoClient(
		cfg.CoinGecko.BaseURL,
		cfg.CoinGecko.APIKey,
		cfg.CoinGecko.RateLimitPerSecond,
	)

	// Create CQAR gRPC client
	cqarClient, err := client.NewCQARClient(ctx, cfg.CQAR.GRPCEndpoint)
	if err != nil {
		return nil, fmt.Errorf("create CQAR client: %w", err)
	}

	// Verify CQAR service is running by listing chains
	log.Info().Msg("Verifying CQAR service connectivity...")
	_, err = cqarClient.ListChains(ctx)
	if err != nil {
		return nil, fmt.Errorf("CQAR service check failed - ensure CQAR is running on %s: %w", cfg.CQAR.GRPCEndpoint, err)
	}
	log.Info().Msg("CQAR service is reachable and responding")

	return &Clients{
		Coinbase:  coinbaseClient,
		CoinGecko: coinGeckoClient,
		CQAR:      cqarClient,
	}, nil
}

// Clients holds all external API client instances
type Clients struct {
	Coinbase  *client.CoinbaseClient
	CoinGecko *client.CoinGeckoClient
	CQAR      *client.CQARClient
}

// Close releases all client resources
func (c *Clients) Close() {
	if c.CQAR != nil {
		if err := c.CQAR.Close(); err != nil {
			log.Error().Err(err).Msg("Failed to close CQAR client")
		}
	}
}

// displayHelp shows usage information
func displayHelp() {
	fmt.Println(`
CQAR Bootstrap Tool - Data Seeding Utility

USAGE:
    bootstrap [OPTIONS]

OPTIONS:
    --config string    Path to bootstrap configuration file (default: config.bootstrap.yaml)
    --dry-run         Perform a dry run without creating entities (default: false)
    --limit int       Limit the number of assets to process, 0 = no limit (default: 0)
    --verbose         Enable verbose logging (default: false)

EXAMPLES:
    # Run with default configuration
    ./bin/bootstrap --config config.bootstrap.yaml

    # Dry run to validate data without creating entities
    ./bin/bootstrap --dry-run

    # Process only first 10 assets for testing
    ./bin/bootstrap --limit 10

    # Verbose logging for debugging
    ./bin/bootstrap --verbose --limit 5

DESCRIPTION:
    The bootstrap tool seeds the CQAR database with initial production data from
    authoritative sources (Coinbase Top 100, CoinGecko API). It creates chains,
    assets, and asset deployments via CQAR gRPC service.

DATA SOURCES:
    - Coinbase: Top 100 asset list (symbol, name, rank)
    - CoinGecko: Asset details, contract addresses, chain deployments, metadata

REQUIREMENTS:
    - CQAR service must be running and accessible
    - CoinGecko API key (optional - free tier works without key, but has lower rate limits)
    - Coinbase API key (optional)
    - Network connectivity to API endpoints

NOTE:
    This tool validates data before insertion and never hallucinates information.
    Assets with missing or unverified data will be skipped with detailed logging.
    
    Free tier CoinGecko (no API key): 10-50 requests/minute
    With API key: Higher rate limits available
`)
}
