package manager

import (
	"context"
	"fmt"

	"github.com/Combine-Capital/cqar/internal/repository"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// VenueManager handles business logic for venue operations, venue asset mapping,
// and venue symbol resolution
type VenueManager struct {
	repo          repository.Repository
	assetManager  *AssetManager
	symbolManager *SymbolManager
}

// NewVenueManager creates a new VenueManager instance
func NewVenueManager(repo repository.Repository, assetManager *AssetManager, symbolManager *SymbolManager) *VenueManager {
	return &VenueManager{
		repo:          repo,
		assetManager:  assetManager,
		symbolManager: symbolManager,
	}
}

// CreateVenue creates a new venue with validation
func (m *VenueManager) CreateVenue(ctx context.Context, venue *venuesv1.Venue) error {
	// Validate required fields
	if err := ValidateRequiredVenueFields(venue); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate chain_id exists if provided (for DEX venues)
	if venue.ChainId != nil && *venue.ChainId != "" {
		chain, err := m.repo.GetChain(ctx, *venue.ChainId)
		if err != nil || chain == nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("chain_id does not exist: %s", *venue.ChainId))
		}
	}

	// Create the venue in the repository
	if err := m.repo.CreateVenue(ctx, venue); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create venue: %v", err))
	}

	return nil
}

// GetVenue retrieves a venue by ID
func (m *VenueManager) GetVenue(ctx context.Context, venueID string) (*venuesv1.Venue, error) {
	if venueID == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}

	venue, err := m.repo.GetVenue(ctx, venueID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("venue not found: %s", venueID))
	}

	return venue, nil
}

// ListVenues retrieves venues with optional filtering
func (m *VenueManager) ListVenues(ctx context.Context, filter *repository.VenueFilter) ([]*venuesv1.Venue, error) {
	venues, err := m.repo.ListVenues(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list venues: %v", err))
	}

	return venues, nil
}

// CreateVenueAsset creates a new venue asset mapping with validation
func (m *VenueManager) CreateVenueAsset(ctx context.Context, venueAsset *venuesv1.VenueAsset) error {
	// Validate required fields
	if err := ValidateRequiredVenueAssetFields(venueAsset); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate venue_id exists
	if _, err := m.GetVenue(ctx, *venueAsset.VenueId); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("venue_id does not exist: %s", *venueAsset.VenueId))
	}

	// Validate asset_id exists
	if _, err := m.assetManager.GetAsset(ctx, *venueAsset.AssetId); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("asset_id does not exist: %s", *venueAsset.AssetId))
	}

	// Validate fees
	if err := ValidateFees(venueAsset.WithdrawalFee); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid withdrawal_fee: %v", err))
	}

	if err := ValidateFees(venueAsset.DepositFee); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid deposit_fee: %v", err))
	}

	// Create the venue asset in the repository
	if err := m.repo.CreateVenueAsset(ctx, venueAsset); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create venue asset: %v", err))
	}

	return nil
}

// GetVenueAsset retrieves a venue asset by venue_id and asset_id
func (m *VenueManager) GetVenueAsset(ctx context.Context, venueID, assetID string) (*venuesv1.VenueAsset, error) {
	if venueID == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}

	if assetID == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	venueAsset, err := m.repo.GetVenueAsset(ctx, venueID, assetID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("venue asset not found: venue=%s asset=%s", venueID, assetID))
	}

	return venueAsset, nil
}

// ListVenueAssets retrieves venue assets with optional filtering
func (m *VenueManager) ListVenueAssets(ctx context.Context, filter *repository.VenueAssetFilter) ([]*venuesv1.VenueAsset, error) {
	venueAssets, err := m.repo.ListVenueAssets(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list venue assets: %v", err))
	}

	return venueAssets, nil
}

// CreateVenueSymbol creates a new venue symbol mapping with validation
func (m *VenueManager) CreateVenueSymbol(ctx context.Context, venueSymbol *venuesv1.VenueSymbol) error {
	// Validate required fields
	if err := ValidateRequiredVenueSymbolFields(venueSymbol); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate venue_id exists
	if _, err := m.GetVenue(ctx, *venueSymbol.VenueId); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("venue_id does not exist: %s", *venueSymbol.VenueId))
	}

	// Validate symbol_id exists
	if _, err := m.symbolManager.GetSymbol(ctx, *venueSymbol.SymbolId); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("symbol_id does not exist: %s", *venueSymbol.SymbolId))
	}

	// Validate fees are in valid range (0-100%)
	if err := ValidateFees(venueSymbol.MakerFee); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid maker_fee: %v", err))
	}

	if err := ValidateFees(venueSymbol.TakerFee); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid taker_fee: %v", err))
	}

	// Create the venue symbol in the repository
	if err := m.repo.CreateVenueSymbol(ctx, venueSymbol); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create venue symbol: %v", err))
	}

	return nil
}

// GetVenueSymbol retrieves a venue symbol by venue_id and venue_symbol string, enriched with canonical symbol data
// This is the primary cqmd use case: resolve "BTCUSDT" on "binance" to canonical symbol with market specs
func (m *VenueManager) GetVenueSymbol(ctx context.Context, venueID, venueSymbolStr string) (*venuesv1.VenueSymbol, *marketsv1.Symbol, error) {
	if venueID == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}

	if venueSymbolStr == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "venue_symbol is required")
	}

	// Use the repository's enriched method which joins with symbols table
	venueSymbol, symbol, err := m.repo.GetVenueSymbolEnriched(ctx, venueID, venueSymbolStr)
	if err != nil {
		return nil, nil, status.Error(codes.NotFound, fmt.Sprintf("venue symbol not found: venue=%s, symbol=%s", venueID, venueSymbolStr))
	}

	return venueSymbol, symbol, nil
}

// GetVenueSymbolByID retrieves a venue symbol by venue_id and symbol_id
func (m *VenueManager) GetVenueSymbolByID(ctx context.Context, venueID, symbolID string) (*venuesv1.VenueSymbol, error) {
	if venueID == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}

	if symbolID == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol_id is required")
	}

	venueSymbol, err := m.repo.GetVenueSymbolByID(ctx, venueID, symbolID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("venue symbol not found: venue=%s, symbolID=%s", venueID, symbolID))
	}

	return venueSymbol, nil
}

// ListVenueSymbols retrieves venue symbols with optional filtering
func (m *VenueManager) ListVenueSymbols(ctx context.Context, filter *repository.VenueSymbolFilter) ([]*venuesv1.VenueSymbol, error) {
	venueSymbols, err := m.repo.ListVenueSymbols(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list venue symbols: %v", err))
	}

	return venueSymbols, nil
}

// ValidateRequiredVenueFields validates that all required fields for a venue are present
func ValidateRequiredVenueFields(venue *venuesv1.Venue) error {
	if venue == nil {
		return fmt.Errorf("venue cannot be nil")
	}

	if venue.VenueId == nil || *venue.VenueId == "" {
		return fmt.Errorf("venue_id is required")
	}

	if venue.Name == nil || *venue.Name == "" {
		return fmt.Errorf("name is required")
	}

	if venue.VenueType == nil || *venue.VenueType == venuesv1.VenueType_VENUE_TYPE_UNSPECIFIED {
		return fmt.Errorf("venue_type is required")
	}

	return nil
}

// ValidateRequiredVenueAssetFields validates that all required fields for a venue asset are present
func ValidateRequiredVenueAssetFields(venueAsset *venuesv1.VenueAsset) error {
	if venueAsset == nil {
		return fmt.Errorf("venue asset cannot be nil")
	}

	if venueAsset.VenueId == nil || *venueAsset.VenueId == "" {
		return fmt.Errorf("venue_id is required")
	}

	if venueAsset.AssetId == nil || *venueAsset.AssetId == "" {
		return fmt.Errorf("asset_id is required")
	}

	return nil
}

// ValidateRequiredVenueSymbolFields validates that all required fields for a venue symbol are present
func ValidateRequiredVenueSymbolFields(venueSymbol *venuesv1.VenueSymbol) error {
	if venueSymbol == nil {
		return fmt.Errorf("venue symbol cannot be nil")
	}

	if venueSymbol.VenueId == nil || *venueSymbol.VenueId == "" {
		return fmt.Errorf("venue_id is required")
	}

	if venueSymbol.SymbolId == nil || *venueSymbol.SymbolId == "" {
		return fmt.Errorf("symbol_id is required")
	}

	if venueSymbol.VenueSymbol == nil || *venueSymbol.VenueSymbol == "" {
		return fmt.Errorf("venue_symbol is required")
	}

	return nil
}

// ValidateFees validates that fees are in valid range (0-100% = 0.0-1.0 or 0-100 depending on scale)
// Assuming fees are stored as percentages (0-100) or decimal (0.0-1.0)
// Based on common practice, we'll assume decimal format (0.0-1.0 = 0%-100%)
func ValidateFees(fee *float64) error {
	if fee == nil {
		return nil // Optional field
	}

	// Support both percentage (0-100) and decimal (0.0-1.0) formats
	// Check if fee is in decimal format (0.0-1.0) or percentage format (0-100)
	if *fee < 0 {
		return fmt.Errorf("fee cannot be negative, got %f", *fee)
	}

	// Allow up to 150% to support some edge cases (very high fees)
	if *fee > 150.0 {
		return fmt.Errorf("fee cannot exceed 150%%, got %f", *fee)
	}

	return nil
}
