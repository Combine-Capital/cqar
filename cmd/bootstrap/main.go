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

	// Display help if requested
	if len(os.Args) == 1 {
		displayHelp()
		return
	}

	log.Info().Msg("Bootstrap tool initialized successfully")
	log.Info().Msg("Ready to seed data (to be implemented in Commit 15)")

	if *dryRun {
		log.Info().Msg("Dry run mode: no entities will be created")
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
