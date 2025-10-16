package seeder

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FileBasedSeeder seeds data from static JSON files
type FileBasedSeeder struct {
	assetClient servicesv1.AssetRegistryClient
	dataDir     string
}

// NewFileBasedSeeder creates a new file-based seeder
func NewFileBasedSeeder(
	assetClient servicesv1.AssetRegistryClient,
	dataDir string,
) *FileBasedSeeder {
	return &FileBasedSeeder{
		assetClient: assetClient,
		dataDir:     dataDir,
	}
}

// Chain represents a blockchain from the JSON file
type Chain struct {
	ChainID           string   `json:"chain_id"`
	Name              string   `json:"name"`
	ChainType         string   `json:"chain_type"`
	NativeAssetSymbol string   `json:"native_asset_symbol"`
	RPCUrls           []string `json:"rpc_urls"`
	BlockExplorerURL  string   `json:"block_explorer_url"`
}

// Asset represents an asset from the JSON file
type Asset struct {
	ID          string                 `json:"id"`
	Symbol      string                 `json:"symbol"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Category    string                 `json:"category"`
	Description string                 `json:"description"`
	LogoURL     string                 `json:"logo_url"`
	WebsiteURL  string                 `json:"website_url"`
	CoinGeckoID string                 `json:"coingecko_id"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Deployment represents an asset deployment from the JSON file
type Deployment struct {
	AssetSymbol     string `json:"asset_symbol"`
	ChainID         string `json:"chain_id"`
	ContractAddress string `json:"contract_address"`
	Decimals        int32  `json:"decimals"`
	IsNative        bool   `json:"is_native"`
}

// SeedAll seeds chains, assets, and deployments from JSON files
func (s *FileBasedSeeder) SeedAll(ctx context.Context) error {
	log.Info().Str("data_dir", s.dataDir).Msg("Starting file-based bootstrap seeding")

	// Step 1: Seed Chains
	if err := s.SeedChains(ctx); err != nil {
		return fmt.Errorf("failed to seed chains: %w", err)
	}

	// Step 2: Seed Assets
	if err := s.SeedAssets(ctx); err != nil {
		return fmt.Errorf("failed to seed assets: %w", err)
	}

	// Step 3: Seed Deployments
	if err := s.SeedDeployments(ctx); err != nil {
		return fmt.Errorf("failed to seed deployments: %w", err)
	}

	log.Info().Msg("File-based bootstrap seeding completed successfully")
	return nil
}

// SeedChains seeds chains from chains.json
func (s *FileBasedSeeder) SeedChains(ctx context.Context) error {
	chainsFile := filepath.Join(s.dataDir, "chains.json")
	log.Info().Str("file", chainsFile).Msg("Seeding chains from file")

	data, err := os.ReadFile(chainsFile)
	if err != nil {
		return fmt.Errorf("failed to read chains file: %w", err)
	}

	var chains []Chain
	if err := json.Unmarshal(data, &chains); err != nil {
		return fmt.Errorf("failed to unmarshal chains: %w", err)
	}

	log.Info().Int("count", len(chains)).Msg("Loaded chains from file")

	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, chain := range chains {
		req := &servicesv1.CreateChainRequest{
			ChainType:        &chain.ChainID, // ChainType is actually the chain ID
			Name:             &chain.Name,
			BlockExplorerUrl: &chain.BlockExplorerURL,
		}

		_, err := s.assetClient.CreateChain(ctx, req)
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
				log.Debug().Str("chain_id", chain.ChainID).Msg("Chain already exists, skipping")
				skipCount++
				continue
			}
			log.Error().
				Str("chain_id", chain.ChainID).
				Err(err).
				Msg("Failed to create chain")
			errorCount++
			continue
		}

		log.Info().
			Str("chain_id", chain.ChainID).
			Str("name", chain.Name).
			Msg("Created chain")
		successCount++
	}

	log.Info().
		Int("success", successCount).
		Int("skipped", skipCount).
		Int("errors", errorCount).
		Msg("Chain seeding complete")

	return nil
}

// SeedAssets seeds assets from assets.json
func (s *FileBasedSeeder) SeedAssets(ctx context.Context) error {
	assetsFile := filepath.Join(s.dataDir, "assets.json")
	log.Info().Str("file", assetsFile).Msg("Seeding assets from file")

	data, err := os.ReadFile(assetsFile)
	if err != nil {
		return fmt.Errorf("failed to read assets file: %w", err)
	}

	var assets []Asset
	if err := json.Unmarshal(data, &assets); err != nil {
		return fmt.Errorf("failed to unmarshal assets: %w", err)
	}

	log.Info().Int("count", len(assets)).Msg("Loaded assets from file")

	successCount := 0
	skipCount := 0
	errorCount := 0

	for _, asset := range assets {
		// Convert asset type string to proto enum
		assetType := parseAssetType(asset.Type)

		req := &servicesv1.CreateAssetRequest{
			Symbol:    &asset.Symbol,
			Name:      &asset.Name,
			AssetType: &assetType,
		}

		// Add optional fields
		if asset.Category != "" {
			req.Category = &asset.Category
		}
		if asset.Description != "" {
			req.Description = &asset.Description
		}
		if asset.LogoURL != "" {
			req.LogoUrl = &asset.LogoURL
		}

		_, err := s.assetClient.CreateAsset(ctx, req)
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
				log.Debug().Str("asset_id", asset.ID).Msg("Asset already exists, skipping")
				skipCount++
				continue
			}
			log.Error().
				Str("asset_id", asset.ID).
				Err(err).
				Msg("Failed to create asset")
			errorCount++
			continue
		}

		log.Info().
			Str("asset_id", asset.ID).
			Str("symbol", asset.Symbol).
			Str("name", asset.Name).
			Msg("Created asset")
		successCount++
	}

	log.Info().
		Int("success", successCount).
		Int("skipped", skipCount).
		Int("errors", errorCount).
		Msg("Asset seeding complete")

	return nil
}

// SeedDeployments seeds deployments from deployments.json
func (s *FileBasedSeeder) SeedDeployments(ctx context.Context) error {
	deploymentsFile := filepath.Join(s.dataDir, "deployments.json")
	log.Info().Str("file", deploymentsFile).Msg("Seeding deployments from file")

	data, err := os.ReadFile(deploymentsFile)
	if err != nil {
		return fmt.Errorf("failed to read deployments file: %w", err)
	}

	var deployments []Deployment
	if err := json.Unmarshal(data, &deployments); err != nil {
		return fmt.Errorf("failed to unmarshal deployments: %w", err)
	}

	log.Info().Int("count", len(deployments)).Msg("Loaded deployments from file")

	successCount := 0
	skipCount := 0
	errorCount := 0

	// First, list all assets to build a symbol->ID map
	assetMap := make(map[string]string) // symbol -> asset_id
	listResp, err := s.assetClient.ListAssets(ctx, &servicesv1.ListAssetsRequest{})
	if err != nil {
		return fmt.Errorf("failed to list assets: %w", err)
	}

	for _, asset := range listResp.Assets {
		if asset.Symbol != nil && asset.AssetId != nil {
			assetMap[*asset.Symbol] = *asset.AssetId
		}
	}

	log.Info().Int("asset_count", len(assetMap)).Msg("Built asset symbol->ID map")

	for _, deployment := range deployments {
		// Look up the asset ID by symbol
		assetID, found := assetMap[deployment.AssetSymbol]
		if !found {
			log.Error().
				Str("symbol", deployment.AssetSymbol).
				Str("chain_id", deployment.ChainID).
				Msg("Asset not found by symbol")
			errorCount++
			continue
		}

		// Convert 0x0000...0000 to "native" for native tokens
		contractAddress := deployment.ContractAddress
		if deployment.IsNative && contractAddress == "0x0000000000000000000000000000000000000000" {
			contractAddress = "native"
		}

		req := &servicesv1.CreateAssetDeploymentRequest{
			AssetId:         &assetID,
			ChainId:         &deployment.ChainID,
			ContractAddress: &contractAddress,
			Decimals:        &deployment.Decimals,
			IsNative:        &deployment.IsNative,
		}

		_, err := s.assetClient.CreateAssetDeployment(ctx, req)
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
				log.Debug().
					Str("symbol", deployment.AssetSymbol).
					Str("asset_id", assetID).
					Str("chain_id", deployment.ChainID).
					Msg("Deployment already exists, skipping")
				skipCount++
				continue
			}
			log.Error().
				Str("symbol", deployment.AssetSymbol).
				Str("asset_id", assetID).
				Str("chain_id", deployment.ChainID).
				Str("contract_address", deployment.ContractAddress).
				Err(err).
				Msg("Failed to create deployment")
			errorCount++
			continue
		}

		log.Info().
			Str("symbol", deployment.AssetSymbol).
			Str("asset_id", assetID).
			Str("chain_id", deployment.ChainID).
			Str("contract_address", deployment.ContractAddress).
			Msg("Created deployment")
		successCount++
	}

	log.Info().
		Int("success", successCount).
		Int("skipped", skipCount).
		Int("errors", errorCount).
		Msg("Deployment seeding complete")

	return nil
}

// parseAssetType converts string to AssetType enum
func parseAssetType(s string) assetsv1.AssetType {
	switch s {
	case "ASSET_TYPE_NATIVE":
		return assetsv1.AssetType_ASSET_TYPE_NATIVE
	case "ASSET_TYPE_ERC20", "ASSET_TYPE_BEP20", "ASSET_TYPE_SPL":
		return assetsv1.AssetType_ASSET_TYPE_ERC20 // All token standards map to ERC20
	default:
		return assetsv1.AssetType_ASSET_TYPE_UNSPECIFIED
	}
}
