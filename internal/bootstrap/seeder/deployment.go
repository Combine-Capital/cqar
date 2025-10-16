package seeder

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Combine-Capital/cqar/internal/bootstrap"
	"github.com/Combine-Capital/cqar/internal/bootstrap/client"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Contract address validation patterns
var (
	// EVM address: 0x followed by 40 hex characters
	evmAddressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
	// Solana address: base58, typically 32-44 characters
	solanaAddressRegex = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)
)

// DeploymentSeeder handles seeding asset deployments into CQAR
type DeploymentSeeder struct {
	coinGeckoClient   *client.CoinGeckoClient
	cqarClient        *client.CQARClient
	dryRun            bool
	limit             int
	assetDetailsCache map[string]*client.AssetDetails // Cached from asset seeding
}

// NewDeploymentSeeder creates a new DeploymentSeeder instance
func NewDeploymentSeeder(
	coinGeckoClient *client.CoinGeckoClient,
	cqarClient *client.CQARClient,
	dryRun bool,
	limit int,
	assetDetailsCache map[string]*client.AssetDetails,
) *DeploymentSeeder {
	return &DeploymentSeeder{
		coinGeckoClient:   coinGeckoClient,
		cqarClient:        cqarClient,
		dryRun:            dryRun,
		limit:             limit,
		assetDetailsCache: assetDetailsCache,
	}
}

// SeedDeployments seeds asset deployments from cached CoinGecko data
func (s *DeploymentSeeder) SeedDeployments(ctx context.Context) (*bootstrap.SeedResult, error) {
	result := &bootstrap.SeedResult{}

	log.Info().Msg("Starting deployment seeding from cached asset data")

	// Check if we have cached data
	if len(s.assetDetailsCache) == 0 {
		log.Warn().Msg("No cached asset details available - run asset seeding first")
		return result, nil
	}

	log.Info().
		Int("cached_assets", len(s.assetDetailsCache)).
		Msg("Using cached CoinGecko data - NO additional API calls needed!")

	// Convert cache to slice for processing
	type assetCacheEntry struct {
		id      string
		details *client.AssetDetails
	}
	cachedAssets := make([]assetCacheEntry, 0, len(s.assetDetailsCache))
	for id, details := range s.assetDetailsCache {
		cachedAssets = append(cachedAssets, assetCacheEntry{id: id, details: details})
	}

	log.Info().Int("count", len(cachedAssets)).Msg("Processing deployments for cached assets")

	// Process each asset's deployments
	for i, entry := range cachedAssets {
		details := entry.details
		symbol := details.Symbol

		log.Info().
			Int("current", i+1).
			Int("total", len(cachedAssets)).
			Str("symbol", symbol).
			Msg("Processing asset deployments")

		// Find the asset in CQAR by symbol (use uppercase for consistency)
		symbolUpper := strings.ToUpper(symbol)
		cqarAssets, err := s.cqarClient.SearchAssets(ctx, symbolUpper)
		if err != nil {
			log.Error().
				Err(err).
				Str("symbol", symbolUpper).
				Msg("Failed to search for asset in CQAR")
			result.AddFailure(symbolUpper, "asset not found in CQAR", err)
			continue
		}

		if len(cqarAssets) == 0 {
			log.Warn().
				Str("symbol", symbolUpper).
				Msg("Asset not found in CQAR, skipping deployments")
			result.AddSkipped(symbolUpper, "asset not found in CQAR")
			continue
		}

		// Use the first matching asset (should be exact symbol match)
		cqarAsset := cqarAssets[0]
		assetID := ""
		if cqarAsset.AssetId != nil {
			assetID = *cqarAsset.AssetId
		}

		// Extract platform deployments from cached data (no API call needed!)
		platforms := client.GetPlatformInfo(details)
		if len(platforms) == 0 {
			log.Debug().
				Str("symbol", symbolUpper).
				Msg("No platform deployments found, skipping")
			result.AddSkipped(symbolUpper, "no deployments")
			continue
		}

		// Process each platform deployment
		for _, platform := range platforms {
			if err := s.seedDeployment(ctx, assetID, symbolUpper, platform, result); err != nil {
				log.Error().
					Err(err).
					Str("symbol", symbolUpper).
					Str("chain", platform.ChainID).
					Msg("Failed to seed deployment")
				// Continue with other deployments
			}
		}
	}

	log.Info().
		Int("total", result.TotalProcessed).
		Int("succeeded", result.Succeeded).
		Int("failed", result.Failed).
		Int("skipped", result.Skipped).
		Msg("Deployment seeding completed")

	return result, nil
}

// seedDeployment seeds a single asset deployment
func (s *DeploymentSeeder) seedDeployment(
	ctx context.Context,
	assetID, symbol string,
	platform client.PlatformInfo,
	result *bootstrap.SeedResult,
) error {
	// Map CoinGecko platform ID to CQAR chain ID
	chainID, chainType, err := s.mapPlatformToChain(platform.ChainID)
	if err != nil {
		log.Warn().
			Str("symbol", symbol).
			Str("platform", platform.ChainID).
			Err(err).
			Msg("Unknown platform, skipping")
		result.AddSkipped(fmt.Sprintf("%s-%s", symbol, platform.ChainID), "unknown platform")
		return nil
	}

	// Validate contract address format
	if err := validateContractAddress(platform.ContractAddress, chainType); err != nil {
		log.Warn().
			Str("symbol", symbol).
			Str("chain", chainID).
			Str("contract_address", platform.ContractAddress).
			Err(err).
			Msg("Invalid contract address, skipping")
		result.AddSkipped(fmt.Sprintf("%s-%s", symbol, chainID), fmt.Sprintf("invalid contract address: %s", err.Error()))
		return nil
	}

	// Validate decimals
	decimals := platform.Decimals
	if decimals < 0 || decimals > 18 {
		log.Warn().
			Str("symbol", symbol).
			Str("chain", chainID).
			Int32("decimals", decimals).
			Msg("Invalid decimals, defaulting to 18")
		decimals = 18
	}

	// Check if deployment already exists
	existingDeployments, err := s.cqarClient.ListAssetDeployments(ctx, assetID)
	if err != nil {
		// Log but continue - may be a genuine "not found" scenario
		log.Debug().
			Err(err).
			Str("asset_id", assetID).
			Msg("Error listing existing deployments")
	} else {
		// Check if this specific deployment exists
		for _, existing := range existingDeployments {
			if existing.ChainId != nil && *existing.ChainId == chainID {
				log.Debug().
					Str("symbol", symbol).
					Str("chain", chainID).
					Msg("Deployment already exists, skipping")
				result.AddSkipped(fmt.Sprintf("%s-%s", symbol, chainID), "already exists")
				return nil
			}
		}
	}

	if s.dryRun {
		log.Info().
			Str("symbol", symbol).
			Str("chain", chainID).
			Str("contract_address", platform.ContractAddress).
			Int32("decimals", decimals).
			Msg("[DRY RUN] Would create deployment")
		result.AddSuccess()
		return nil
	}

	// Create deployment request
	// Note: IsNative is not set here as all CoinGecko platform deployments are token contracts
	isNative := false
	req := &servicesv1.CreateAssetDeploymentRequest{
		AssetId:         &assetID,
		ChainId:         &chainID,
		ContractAddress: &platform.ContractAddress,
		Decimals:        &decimals,
		IsNative:        &isNative,
	}

	// Create the deployment
	_, err = s.cqarClient.CreateAssetDeployment(ctx, req)
	if err != nil {
		// Check if it's a duplicate/already exists error
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Info().
				Str("symbol", symbol).
				Str("chain", chainID).
				Msg("Deployment already exists (concurrent creation)")
			result.AddSkipped(fmt.Sprintf("%s-%s", symbol, chainID), "already exists")
			return nil
		}

		// Check for foreign key violations (asset or chain doesn't exist)
		if ok && (st.Code() == codes.InvalidArgument || st.Code() == codes.NotFound) {
			log.Warn().
				Str("symbol", symbol).
				Str("chain", chainID).
				Err(err).
				Msg("Asset or chain not found, skipping")
			result.AddSkipped(fmt.Sprintf("%s-%s", symbol, chainID), "asset or chain not found")
			return nil
		}

		result.AddFailure(fmt.Sprintf("%s-%s", symbol, chainID), "failed to create deployment", err)
		return fmt.Errorf("create deployment: %w", err)
	}

	log.Info().
		Str("symbol", symbol).
		Str("chain", chainID).
		Str("contract_address", platform.ContractAddress).
		Int32("decimals", decimals).
		Msg("Deployment created successfully")

	result.AddSuccess()
	return nil
}

// mapPlatformToChain maps CoinGecko platform IDs to CQAR chain IDs
func (s *DeploymentSeeder) mapPlatformToChain(coinGeckoPlatform string) (chainID, chainType string, err error) {
	// Normalize platform ID
	platform := strings.ToLower(coinGeckoPlatform)

	switch platform {
	case "ethereum":
		return "ethereum", "EVM", nil
	case "polygon-pos":
		return "polygon_pos", "EVM", nil
	case "binance-smart-chain":
		return "binance_smart_chain", "EVM", nil
	case "solana":
		return "solana", "SOLANA", nil
	case "bitcoin":
		return "bitcoin", "UTXO", nil
	case "arbitrum-one":
		return "arbitrum_one", "EVM", nil
	case "optimistic-ethereum":
		return "optimistic_ethereum", "EVM", nil
	case "avalanche":
		return "avalanche", "EVM", nil
	case "base":
		return "base", "EVM", nil
	default:
		return "", "", fmt.Errorf("unknown platform: %s", coinGeckoPlatform)
	}
}

// validateContractAddress validates contract address format based on chain type
func validateContractAddress(contractAddress, chainType string) error {
	if contractAddress == "" {
		return fmt.Errorf("contract address is empty")
	}

	switch chainType {
	case "EVM":
		if !evmAddressRegex.MatchString(contractAddress) {
			return fmt.Errorf("invalid EVM address format (expected 0x followed by 40 hex characters)")
		}
	case "SOLANA":
		if !solanaAddressRegex.MatchString(contractAddress) {
			return fmt.Errorf("invalid Solana address format (expected base58 address, 32-44 characters)")
		}
	case "UTXO":
		// Bitcoin native asset has no contract address
		// This case shouldn't occur in practice since Bitcoin won't have a contract address in CoinGecko platforms
		return fmt.Errorf("UTXO chains do not support contract addresses")
	default:
		// For unknown chain types, basic validation
		if strings.TrimSpace(contractAddress) == "" {
			return fmt.Errorf("contract address cannot be empty")
		}
	}

	return nil
}
