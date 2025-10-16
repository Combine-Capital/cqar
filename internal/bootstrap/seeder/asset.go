package seeder

import (
	"context"
	"fmt"
	"strings"

	"github.com/Combine-Capital/cqar/internal/bootstrap"
	"github.com/Combine-Capital/cqar/internal/bootstrap/client"
	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AssetSeeder handles seeding assets into CQAR from external sources
type AssetSeeder struct {
	coinGeckoClient *client.CoinGeckoClient
	cqarClient      *client.CQARClient
	dryRun          bool
	limit           int
}

// NewAssetSeeder creates a new AssetSeeder instance
func NewAssetSeeder(
	coinGeckoClient *client.CoinGeckoClient,
	cqarClient *client.CQARClient,
	dryRun bool,
	limit int,
) *AssetSeeder {
	return &AssetSeeder{
		coinGeckoClient: coinGeckoClient,
		cqarClient:      cqarClient,
		dryRun:          dryRun,
		limit:           limit,
	}
}

// SeedAssets seeds assets from CoinGecko Top 100
func (s *AssetSeeder) SeedAssets(ctx context.Context) (*bootstrap.SeedResult, error) {
	result := &bootstrap.SeedResult{}

	log.Info().Msg("Fetching CoinGecko Top 100 assets")

	// Fetch Top 100 assets from CoinGecko
	assets, err := s.coinGeckoClient.GetTop100Assets(ctx)
	if err != nil {
		return result, fmt.Errorf("fetch top 100 assets: %w", err)
	}

	// Apply limit if specified
	if s.limit > 0 && s.limit < len(assets) {
		assets = assets[:s.limit]
		log.Info().Int("limit", s.limit).Msg("Limiting asset processing")
	}

	log.Info().Int("count", len(assets)).Msg("Starting asset seeding")

	// Process each asset
	for i, assetSummary := range assets {
		log.Info().
			Int("current", i+1).
			Int("total", len(assets)).
			Str("symbol", assetSummary.Symbol).
			Str("name", assetSummary.Name).
			Msg("Processing asset")

		// Fetch detailed asset information
		details, err := s.coinGeckoClient.GetAssetByID(ctx, assetSummary.ID)
		if err != nil {
			log.Error().
				Err(err).
				Str("symbol", assetSummary.Symbol).
				Str("coingecko_id", assetSummary.ID).
				Msg("Failed to fetch asset details")
			result.AddFailure(assetSummary.Symbol, "failed to fetch details", err)
			continue
		}

		// Convert to AssetData
		assetData, err := s.convertToAssetData(details)
		if err != nil {
			log.Warn().
				Err(err).
				Str("symbol", assetSummary.Symbol).
				Msg("Asset data validation failed")
			result.AddSkipped(assetSummary.Symbol, err.Error())
			continue
		}

		// Seed the asset
		if err := s.seedAsset(ctx, assetData, result); err != nil {
			log.Error().
				Err(err).
				Str("symbol", assetData.Symbol).
				Msg("Failed to seed asset")
			// Continue processing other assets
		}
	}

	log.Info().
		Int("total", result.TotalProcessed).
		Int("succeeded", result.Succeeded).
		Int("failed", result.Failed).
		Int("skipped", result.Skipped).
		Msg("Asset seeding completed")

	return result, nil
}

// convertToAssetData converts CoinGecko AssetDetails to validated AssetData
func (s *AssetSeeder) convertToAssetData(details *client.AssetDetails) (*bootstrap.AssetData, error) {
	// CRITICAL: Validate required fields - never hallucinate data
	if details.Symbol == "" {
		return nil, fmt.Errorf("missing required field: symbol")
	}
	if details.Name == "" {
		return nil, fmt.Errorf("missing required field: name")
	}

	// Normalize symbol to uppercase
	symbol := strings.ToUpper(strings.TrimSpace(details.Symbol))
	name := strings.TrimSpace(details.Name)

	// Determine asset type based on symbol and name patterns
	assetType := determineAssetType(symbol, name)

	// Extract description (English only)
	description := ""
	if desc, ok := details.Description["en"]; ok {
		description = cleanDescription(desc)
	}

	// Extract logo URL
	logoURL := ""
	if details.Image.Large != "" {
		logoURL = details.Image.Large
	} else if details.Image.Small != "" {
		logoURL = details.Image.Small
	}

	// Extract homepage
	homepage := ""
	if len(details.Links.Homepage) > 0 && details.Links.Homepage[0] != "" {
		homepage = details.Links.Homepage[0]
	}

	// Determine category
	category := determineCategory(assetType, symbol, name)

	return &bootstrap.AssetData{
		Symbol:      symbol,
		Name:        name,
		Type:        assetType,
		Category:    category,
		Description: description,
		LogoURL:     logoURL,
		Homepage:    homepage,
		CoinGeckoID: details.ID,
	}, nil
}

// seedAsset creates a single asset in CQAR
func (s *AssetSeeder) seedAsset(ctx context.Context, assetData *bootstrap.AssetData, result *bootstrap.SeedResult) error {
	// Check if asset already exists by searching for symbol
	existing, err := s.cqarClient.SearchAssets(ctx, assetData.Symbol)
	if err == nil && len(existing) > 0 {
		// Check if any exact match exists
		for _, asset := range existing {
			if asset.Symbol != nil && strings.EqualFold(*asset.Symbol, assetData.Symbol) {
				log.Info().
					Str("symbol", assetData.Symbol).
					Str("asset_id", *asset.AssetId).
					Msg("Asset already exists, skipping")
				result.AddSkipped(assetData.Symbol, "already exists")
				return nil
			}
		}
	}

	if s.dryRun {
		log.Info().
			Str("symbol", assetData.Symbol).
			Str("name", assetData.Name).
			Str("type", assetData.Type.String()).
			Msg("[DRY RUN] Would create asset")
		result.AddSuccess()
		return nil
	}

	// Build CreateAssetRequest
	// Note: AssetId is generated by the manager, not provided in request
	req := &servicesv1.CreateAssetRequest{
		Symbol:    &assetData.Symbol,
		Name:      &assetData.Name,
		AssetType: &assetData.Type,
	}

	// Add optional fields if present
	if assetData.Category != "" {
		req.Category = &assetData.Category
	}
	if assetData.Description != "" {
		req.Description = &assetData.Description
	}
	if assetData.LogoURL != "" {
		req.LogoUrl = &assetData.LogoURL
	}
	if assetData.Homepage != "" {
		req.WebsiteUrl = &assetData.Homepage
	}

	// Create the asset
	resp, err := s.cqarClient.CreateAsset(ctx, req)
	if err != nil {
		// Check if it's a duplicate error (already exists)
		st, ok := status.FromError(err)
		if ok && st.Code() == codes.AlreadyExists {
			log.Info().
				Str("symbol", assetData.Symbol).
				Msg("Asset already exists (concurrent creation)")
			result.AddSkipped(assetData.Symbol, "already exists")
			return nil
		}

		result.AddFailure(assetData.Symbol, "failed to create asset", err)
		return fmt.Errorf("create asset: %w", err)
	}

	assetID := ""
	if resp.AssetId != nil {
		assetID = *resp.AssetId
	}

	log.Info().
		Str("asset_id", assetID).
		Str("symbol", assetData.Symbol).
		Str("name", assetData.Name).
		Str("type", assetData.Type.String()).
		Msg("Asset created successfully")

	result.AddSuccess()
	return nil
}

// determineAssetType determines the asset type based on symbol and name
func determineAssetType(symbol, name string) assetsv1.AssetType {
	symbolUpper := strings.ToUpper(symbol)
	nameLower := strings.ToLower(name)

	// Stablecoins - map to ERC20 (stablecoin is stored in category)
	stablecoins := []string{"USDT", "USDC", "DAI", "BUSD", "TUSD", "USDP", "GUSD", "FRAX", "LUSD"}
	for _, stable := range stablecoins {
		if symbolUpper == stable {
			return assetsv1.AssetType_ASSET_TYPE_ERC20
		}
	}

	// Check for stablecoin keywords in name
	if strings.Contains(nameLower, "stablecoin") ||
		strings.Contains(nameLower, "usd") && (strings.Contains(nameLower, "coin") || strings.Contains(nameLower, "dollar")) {
		return assetsv1.AssetType_ASSET_TYPE_ERC20
	}

	// Native cryptocurrencies (Layer 1s)
	nativeAssets := []string{"BTC", "ETH", "BNB", "SOL", "ADA", "AVAX", "MATIC", "DOT", "ATOM", "XRP", "LTC", "BCH", "XLM", "ALGO", "NEAR", "FTM", "ONE"}
	for _, native := range nativeAssets {
		if symbolUpper == native {
			return assetsv1.AssetType_ASSET_TYPE_NATIVE
		}
	}

	// Wrapped assets
	if strings.HasPrefix(symbolUpper, "W") && len(symbolUpper) > 1 {
		// WETH, WBTC, etc.
		return assetsv1.AssetType_ASSET_TYPE_ERC20
	}

	// Governance tokens
	if strings.Contains(nameLower, "governance") || strings.Contains(nameLower, "dao") {
		return assetsv1.AssetType_ASSET_TYPE_ERC20
	}

	// Default to ERC20-compatible token
	// This covers most DeFi tokens, even if they're on other EVM chains
	return assetsv1.AssetType_ASSET_TYPE_ERC20
}

// determineCategory determines the asset category
func determineCategory(assetType assetsv1.AssetType, symbol, name string) string {
	nameLower := strings.ToLower(name)
	symbolUpper := strings.ToUpper(symbol)

	// Stablecoins - check first
	stablecoins := []string{"USDT", "USDC", "DAI", "BUSD", "TUSD", "USDP", "GUSD", "FRAX", "LUSD"}
	for _, stable := range stablecoins {
		if symbolUpper == stable {
			if strings.Contains(nameLower, "algorithmic") {
				return "algorithmic-stablecoin"
			}
			return "fiat-backed-stablecoin"
		}
	}

	// Native assets
	if assetType == assetsv1.AssetType_ASSET_TYPE_NATIVE {
		return "layer-1"
	}

	// DeFi categories
	if strings.Contains(nameLower, "defi") || strings.Contains(nameLower, "decentralized finance") {
		return "defi"
	}

	// Exchange tokens
	exchangeKeywords := []string{"exchange", "dex", "swap"}
	for _, keyword := range exchangeKeywords {
		if strings.Contains(nameLower, keyword) {
			return "exchange"
		}
	}

	// Gaming/Metaverse
	if strings.Contains(nameLower, "gaming") || strings.Contains(nameLower, "game") ||
		strings.Contains(nameLower, "metaverse") || strings.Contains(nameLower, "nft") {
		return "gaming-metaverse"
	}

	// Meme coins
	memeKeywords := []string{"meme", "dog", "cat", "pepe", "shib"}
	for _, keyword := range memeKeywords {
		if strings.Contains(nameLower, keyword) {
			return "meme"
		}
	}

	// Default
	return "cryptocurrency"
}

// cleanDescription cleans and truncates description text
func cleanDescription(desc string) string {
	// Remove HTML tags (basic cleanup)
	desc = strings.ReplaceAll(desc, "<p>", "")
	desc = strings.ReplaceAll(desc, "</p>", "\n")
	desc = strings.ReplaceAll(desc, "<br>", "\n")
	desc = strings.ReplaceAll(desc, "<br/>", "\n")
	desc = strings.ReplaceAll(desc, "<a href", "")
	desc = strings.TrimSpace(desc)

	// Truncate to first 500 characters for database efficiency
	if len(desc) > 500 {
		desc = desc[:500] + "..."
	}

	return desc
}
