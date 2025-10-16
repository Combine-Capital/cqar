package seeder

import (
	"context"
	"fmt"

	"github.com/Combine-Capital/cqar/internal/bootstrap"
	"github.com/Combine-Capital/cqar/internal/bootstrap/client"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ChainSeeder handles seeding chains into CQAR
type ChainSeeder struct {
	cqarClient *client.CQARClient
	dryRun     bool
}

// NewChainSeeder creates a new ChainSeeder instance
func NewChainSeeder(cqarClient *client.CQARClient, dryRun bool) *ChainSeeder {
	return &ChainSeeder{
		cqarClient: cqarClient,
		dryRun:     dryRun,
	}
}

// SeedChains seeds all configured chains into CQAR
func (s *ChainSeeder) SeedChains(ctx context.Context) (*bootstrap.SeedResult, error) {
	result := &bootstrap.SeedResult{}

	chains := getDefaultChains()

	log.Info().Int("count", len(chains)).Msg("Starting chain seeding")

	for i, chainData := range chains {
		log.Info().
			Int("current", i+1).
			Int("total", len(chains)).
			Str("chain", chainData.Name).
			Msg("Processing chain")

		if err := s.seedChain(ctx, chainData, result); err != nil {
			log.Error().
				Err(err).
				Str("chain", chainData.Name).
				Msg("Failed to seed chain")
			// Continue processing other chains
		}
	}

	log.Info().
		Int("total", result.TotalProcessed).
		Int("succeeded", result.Succeeded).
		Int("failed", result.Failed).
		Int("skipped", result.Skipped).
		Msg("Chain seeding completed")

	return result, nil
}

// seedChain seeds a single chain
func (s *ChainSeeder) seedChain(ctx context.Context, chainData bootstrap.ChainData, result *bootstrap.SeedResult) error {
	// Check if chain already exists
	existing, err := s.cqarClient.GetChain(ctx, chainData.ChainID)
	if err == nil && existing != nil {
		log.Info().
			Str("chain", chainData.Name).
			Str("chain_id", chainData.ChainID).
			Msg("Chain already exists, skipping")
		result.AddSkipped(chainData.Name, "already exists")
		return nil
	}

	// If error is not "not found", it's a real error
	if err != nil {
		st, ok := status.FromError(err)
		if !ok || st.Code() != codes.NotFound {
			result.AddFailure(chainData.Name, "error checking existing chain", err)
			return err
		}
	}

	if s.dryRun {
		log.Info().
			Str("chain", chainData.Name).
			Str("type", chainData.Type).
			Msg("[DRY RUN] Would create chain")
		result.AddSuccess()
		return nil
	}

	// Create chain request
	// Note: CreateChainRequest uses Name (not ChainName) and ChainType
	req := &servicesv1.CreateChainRequest{
		ChainType:        &chainData.ChainID, // ChainType serves as the chain identifier
		Name:             &chainData.Name,
		BlockExplorerUrl: &chainData.ExplorerURL,
	}

	// Note: RPCURLs and NativeAssetId are not in CreateChainRequest
	// They are handled at the database/repository level, not via gRPC

	// Create the chain
	_, err = s.cqarClient.CreateChain(ctx, req)
	if err != nil {
		result.AddFailure(chainData.Name, "failed to create chain", err)
		return fmt.Errorf("create chain: %w", err)
	}

	log.Info().
		Str("chain_id", chainData.ChainID).
		Str("name", chainData.Name).
		Str("type", chainData.Type).
		Msg("Chain created successfully")

	result.AddSuccess()
	return nil
}

// getDefaultChains returns the predefined list of chains to seed
func getDefaultChains() []bootstrap.ChainData {
	return []bootstrap.ChainData{
		{
			ChainID: "ethereum",
			Name:    "Ethereum",
			Type:    "EVM",
			RPCURLs: []string{
				"https://eth.llamarpc.com",
				"https://rpc.ankr.com/eth",
			},
			ExplorerURL: "https://etherscan.io",
		},
		{
			ChainID: "polygon_pos", // Changed from polygon-pos to match DB constraint
			Name:    "Polygon",
			Type:    "EVM",
			RPCURLs: []string{
				"https://polygon-rpc.com",
				"https://rpc.ankr.com/polygon",
			},
			ExplorerURL: "https://polygonscan.com",
		},
		{
			ChainID: "binance_smart_chain", // Changed from binance-smart-chain
			Name:    "BSC",
			Type:    "EVM",
			RPCURLs: []string{
				"https://bsc-dataseed.binance.org",
				"https://rpc.ankr.com/bsc",
			},
			ExplorerURL: "https://bscscan.com",
		},
		{
			ChainID: "solana",
			Name:    "Solana",
			Type:    "NON_EVM",
			RPCURLs: []string{
				"https://api.mainnet-beta.solana.com",
			},
			ExplorerURL: "https://solscan.io",
		},
		{
			ChainID: "bitcoin",
			Name:    "Bitcoin",
			Type:    "UTXO",
			RPCURLs: []string{
				"https://blockstream.info/api",
			},
			ExplorerURL: "https://blockstream.info",
		},
		{
			ChainID: "arbitrum_one", // Changed from arbitrum-one
			Name:    "Arbitrum",
			Type:    "EVM",
			RPCURLs: []string{
				"https://arb1.arbitrum.io/rpc",
				"https://rpc.ankr.com/arbitrum",
			},
			ExplorerURL: "https://arbiscan.io",
		},
		{
			ChainID: "optimistic_ethereum", // Changed from optimistic-ethereum
			Name:    "Optimism",
			Type:    "EVM",
			RPCURLs: []string{
				"https://mainnet.optimism.io",
				"https://rpc.ankr.com/optimism",
			},
			ExplorerURL: "https://optimistic.etherscan.io",
		},
	}
}
