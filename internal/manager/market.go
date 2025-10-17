package manager

import (
	"context"
	"fmt"

	"github.com/Combine-Capital/cqar/internal/repository"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MarketManager handles business logic for market operations with validation
type MarketManager struct {
	repo              repository.Repository
	instrumentManager *InstrumentManager
	venueManager      *VenueManager
	assetManager      *AssetManager
	eventPublisher    *EventPublisher
}

// NewMarketManager creates a new MarketManager instance
func NewMarketManager(
	repo repository.Repository,
	instrumentManager *InstrumentManager,
	venueManager *VenueManager,
	assetManager *AssetManager,
	eventPublisher *EventPublisher,
) *MarketManager {
	return &MarketManager{
		repo:              repo,
		instrumentManager: instrumentManager,
		venueManager:      venueManager,
		assetManager:      assetManager,
		eventPublisher:    eventPublisher,
	}
}

// CreateMarket creates a new market with comprehensive validation
func (m *MarketManager) CreateMarket(ctx context.Context, market *marketsv1.Market) error {
	// Validate required fields
	if err := ValidateRequiredMarketFields(market); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate instrument_id exists
	if market.InstrumentId != nil && *market.InstrumentId != "" {
		if _, err := m.instrumentManager.GetInstrument(ctx, *market.InstrumentId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("instrument_id does not exist: %s", *market.InstrumentId))
		}
	}

	// Validate venue_id exists
	if market.VenueId != nil && *market.VenueId != "" {
		if _, err := m.venueManager.GetVenue(ctx, *market.VenueId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("venue_id does not exist: %s", *market.VenueId))
		}
	}

	// Validate settlement_asset_id exists if provided
	if market.SettlementAssetId != nil && *market.SettlementAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *market.SettlementAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("settlement_asset_id does not exist: %s", *market.SettlementAssetId))
		}
	}

	// Validate price_currency_asset_id exists if provided
	if market.PriceCurrencyAssetId != nil && *market.PriceCurrencyAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *market.PriceCurrencyAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("price_currency_asset_id does not exist: %s", *market.PriceCurrencyAssetId))
		}
	}

	// Validate market specifications
	if err := ValidateMarketSpecs(market); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the market in the repository
	if err := m.repo.CreateMarket(ctx, market); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create market: %v", err))
	}

	// Publish MarketListed event asynchronously
	if m.eventPublisher != nil {
		m.eventPublisher.PublishMarketListed(ctx, market)
	}

	return nil
}

// GetMarket retrieves a market by ID
func (m *MarketManager) GetMarket(ctx context.Context, marketID string) (*marketsv1.Market, error) {
	if marketID == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}

	market, err := m.repo.GetMarket(ctx, marketID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("market not found: %s", marketID))
	}

	return market, nil
}

// ResolveMarket resolves a market by venue_id and venue_symbol
// This is the critical method for mapping venue-specific symbols to market_id
func (m *MarketManager) ResolveMarket(ctx context.Context, venueID, venueSymbol string) (*marketsv1.Market, error) {
	if venueID == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}
	if venueSymbol == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_symbol is required")
	}

	market, err := m.repo.ResolveMarket(ctx, venueID, venueSymbol)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("market not found for venue %s symbol %s", venueID, venueSymbol))
	}

	return market, nil
}

// ListMarketsByInstrument retrieves all markets for a given instrument
func (m *MarketManager) ListMarketsByInstrument(ctx context.Context, instrumentID string) ([]*marketsv1.Market, error) {
	if instrumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "instrument_id is required")
	}

	markets, err := m.repo.ListMarketsByInstrument(ctx, instrumentID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list markets: %v", err))
	}

	return markets, nil
}

// ListMarketsByVenue retrieves all markets for a given venue
func (m *MarketManager) ListMarketsByVenue(ctx context.Context, venueID string) ([]*marketsv1.Market, error) {
	if venueID == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}

	markets, err := m.repo.ListMarketsByVenue(ctx, venueID)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list markets: %v", err))
	}

	return markets, nil
}
