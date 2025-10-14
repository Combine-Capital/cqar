package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/Combine-Capital/cqar/internal/config"
	"github.com/Combine-Capital/cqi/pkg/service"
)

var (
	version   = "0.1.0"
	buildTime = "unknown"
	gitCommit = "unknown"

	configPath = flag.String("config", "config.yaml", "path to configuration file")
	showHelp   = flag.Bool("help", false, "show help message")
	showVer    = flag.Bool("version", false, "show version information")
)

func main() {
	flag.Parse()

	// Show help
	if *showHelp {
		printHelp()
		os.Exit(0)
	}

	// Show version
	if *showVer {
		printVersion()
		os.Exit(0)
	}

	// Load configuration using CQI
	cfg := config.MustLoad(*configPath, "CQAR")

	// Create context
	ctx := context.Background()

	// Initialize observability via CQI Bootstrap
	bootstrap, err := service.NewBootstrap(ctx, &cfg.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize bootstrap: %v\n", err)
		os.Exit(1)
	}
	defer bootstrap.Cleanup(ctx)

	bootstrap.Logger.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Str("git_commit", gitCommit).
		Str("environment", cfg.Service.Env).
		Msg("Starting CQAR service")

	// TODO: Initialize service components (database, cache, event bus, etc.)
	// This will be implemented in later commits as dependencies are added

	bootstrap.Logger.Info().
		Int("grpc_port", cfg.Server.GRPCPort).
		Int("http_port", cfg.Server.HTTPPort).
		Msg("CQAR service initialized")

	// TODO: Create and start HTTP/gRPC services
	// This will be implemented in Commit 9

	// Wait for shutdown signal using CQI's WaitForShutdown
	bootstrap.Logger.Info().Msg("Service ready, waiting for shutdown signal")
	service.WaitForShutdown(ctx /* services will be added here */)

	bootstrap.Logger.Info().Msg("CQAR service stopped")
}

func printHelp() {
	fmt.Fprintf(os.Stdout, "CQAR - Crypto Quant Asset Registry\n\n")
	fmt.Fprintf(os.Stdout, "A gRPC microservice providing canonical reference data for crypto assets,\n")
	fmt.Fprintf(os.Stdout, "symbols, venues, and their relationships.\n\n")
	fmt.Fprintf(os.Stdout, "USAGE:\n")
	fmt.Fprintf(os.Stdout, "    cqar [OPTIONS]\n\n")
	fmt.Fprintf(os.Stdout, "OPTIONS:\n")
	fmt.Fprintf(os.Stdout, "    -config <path>     Path to configuration file (default: config.yaml)\n")
	fmt.Fprintf(os.Stdout, "    -help              Show this help message\n")
	fmt.Fprintf(os.Stdout, "    -version           Show version information\n\n")
	fmt.Fprintf(os.Stdout, "ENVIRONMENT VARIABLES:\n")
	fmt.Fprintf(os.Stdout, "    CQAR_DATABASE_HOST      Database hostname\n")
	fmt.Fprintf(os.Stdout, "    CQAR_DATABASE_PASSWORD  Database password\n")
	fmt.Fprintf(os.Stdout, "    CQAR_CACHE_HOST         Redis cache host\n")
	fmt.Fprintf(os.Stdout, "    CQAR_CACHE_PASSWORD     Redis password\n")
	fmt.Fprintf(os.Stdout, "    CQAR_EVENT_BUS_URL      NATS JetStream URL\n\n")
	fmt.Fprintf(os.Stdout, "For more information, see: https://github.com/Combine-Capital/cqar\n")
}

func printVersion() {
	fmt.Fprintf(os.Stdout, "CQAR %s\n", version)
	fmt.Fprintf(os.Stdout, "Build Time: %s\n", buildTime)
	fmt.Fprintf(os.Stdout, "Git Commit: %s\n", gitCommit)
}
