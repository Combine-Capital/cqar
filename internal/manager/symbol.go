package manager

import (
	"context"
	"fmt"
	"time"

	"github.com/Combine-Capital/cqar/internal/repository"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SymbolManager handles business logic for symbol operations with validation
// and market specification checking
type SymbolManager struct {
	repo           repository.Repository
	assetManager   *AssetManager
	eventPublisher *EventPublisher
}

// NewSymbolManager creates a new SymbolManager instance
func NewSymbolManager(repo repository.Repository, assetManager *AssetManager, eventPublisher *EventPublisher) *SymbolManager {
	return &SymbolManager{
		repo:           repo,
		assetManager:   assetManager,
		eventPublisher: eventPublisher,
	}
}

// CreateSymbol creates a new symbol with comprehensive validation
func (m *SymbolManager) CreateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error {
	// Validate required fields
	if err := ValidateRequiredSymbolFields(symbol); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate base_asset_id exists
	if symbol.BaseAssetId != nil && *symbol.BaseAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *symbol.BaseAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("base_asset_id does not exist: %s", *symbol.BaseAssetId))
		}
	}

	// Validate quote_asset_id exists
	if symbol.QuoteAssetId != nil && *symbol.QuoteAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *symbol.QuoteAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("quote_asset_id does not exist: %s", *symbol.QuoteAssetId))
		}
	}

	// Validate settlement_asset_id exists if provided
	if symbol.SettlementAssetId != nil && *symbol.SettlementAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *symbol.SettlementAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("settlement_asset_id does not exist: %s", *symbol.SettlementAssetId))
		}
	}

	// Validate market specifications
	if err := ValidateMarketSpecs(symbol); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate option-specific fields if symbol type is OPTION
	if symbol.SymbolType != nil && *symbol.SymbolType == marketsv1.SymbolType_SYMBOL_TYPE_OPTION {
		if err := ValidateOptionFields(symbol); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
	}

	// Create the symbol in the repository
	if err := m.repo.CreateSymbol(ctx, symbol); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create symbol: %v", err))
	}

	// Publish SymbolCreated event asynchronously
	if m.eventPublisher != nil {
		m.eventPublisher.PublishSymbolCreated(ctx, symbol)
	}

	return nil
}

// GetSymbol retrieves a symbol by ID
func (m *SymbolManager) GetSymbol(ctx context.Context, symbolID string) (*marketsv1.Symbol, error) {
	if symbolID == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol_id is required")
	}

	symbol, err := m.repo.GetSymbol(ctx, symbolID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("symbol not found: %s", symbolID))
	}

	return symbol, nil
}

// UpdateSymbol updates an existing symbol with validation
func (m *SymbolManager) UpdateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error {
	// Validate required fields
	if err := ValidateRequiredSymbolFields(symbol); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if symbol.SymbolId == nil || *symbol.SymbolId == "" {
		return status.Error(codes.InvalidArgument, "symbol_id is required for update")
	}

	// Verify symbol exists
	existing, err := m.repo.GetSymbol(ctx, *symbol.SymbolId)
	if err != nil || existing == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("symbol not found: %s", *symbol.SymbolId))
	}

	// Validate base_asset_id exists if changed
	if symbol.BaseAssetId != nil && *symbol.BaseAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *symbol.BaseAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("base_asset_id does not exist: %s", *symbol.BaseAssetId))
		}
	}

	// Validate quote_asset_id exists if changed
	if symbol.QuoteAssetId != nil && *symbol.QuoteAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *symbol.QuoteAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("quote_asset_id does not exist: %s", *symbol.QuoteAssetId))
		}
	}

	// Validate market specifications
	if err := ValidateMarketSpecs(symbol); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate option-specific fields if symbol type is OPTION
	if symbol.SymbolType != nil && *symbol.SymbolType == marketsv1.SymbolType_SYMBOL_TYPE_OPTION {
		if err := ValidateOptionFields(symbol); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
	}

	// Update the symbol
	if err := m.repo.UpdateSymbol(ctx, symbol); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to update symbol: %v", err))
	}

	return nil
}

// DeleteSymbol deletes a symbol by ID
func (m *SymbolManager) DeleteSymbol(ctx context.Context, symbolID string) error {
	if symbolID == "" {
		return status.Error(codes.InvalidArgument, "symbol_id is required")
	}

	// Verify symbol exists
	existing, err := m.repo.GetSymbol(ctx, symbolID)
	if err != nil || existing == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("symbol not found: %s", symbolID))
	}

	// Delete the symbol
	if err := m.repo.DeleteSymbol(ctx, symbolID); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to delete symbol: %v", err))
	}

	return nil
}

// ListSymbols retrieves symbols with optional filtering
func (m *SymbolManager) ListSymbols(ctx context.Context, filter *repository.SymbolFilter) ([]*marketsv1.Symbol, error) {
	symbols, err := m.repo.ListSymbols(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list symbols: %v", err))
	}

	return symbols, nil
}

// SearchSymbols searches for symbols by query string
func (m *SymbolManager) SearchSymbols(ctx context.Context, query string, filter *repository.SymbolFilter) ([]*marketsv1.Symbol, error) {
	symbols, err := m.repo.SearchSymbols(ctx, query, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to search symbols: %v", err))
	}

	return symbols, nil
}

// ValidateRequiredSymbolFields validates that all required fields for a symbol are present
func ValidateRequiredSymbolFields(symbol *marketsv1.Symbol) error {
	if symbol == nil {
		return fmt.Errorf("symbol cannot be nil")
	}

	if symbol.BaseAssetId == nil || *symbol.BaseAssetId == "" {
		return fmt.Errorf("base_asset_id is required")
	}

	if symbol.QuoteAssetId == nil || *symbol.QuoteAssetId == "" {
		return fmt.Errorf("quote_asset_id is required")
	}

	if symbol.SymbolType == nil || *symbol.SymbolType == marketsv1.SymbolType_SYMBOL_TYPE_UNSPECIFIED {
		return fmt.Errorf("symbol_type is required")
	}

	return nil
}

// ValidateMarketSpecs validates market specifications are valid
func ValidateMarketSpecs(symbol *marketsv1.Symbol) error {
	if symbol == nil {
		return fmt.Errorf("symbol cannot be nil")
	}

	// Validate tick_size > 0
	if symbol.TickSize != nil && *symbol.TickSize <= 0 {
		return fmt.Errorf("tick_size must be greater than 0, got %f", *symbol.TickSize)
	}

	// Validate lot_size > 0
	if symbol.LotSize != nil && *symbol.LotSize <= 0 {
		return fmt.Errorf("lot_size must be greater than 0, got %f", *symbol.LotSize)
	}

	// Validate min_order_size < max_order_size if both are set
	if symbol.MinOrderSize != nil && symbol.MaxOrderSize != nil {
		if *symbol.MinOrderSize >= *symbol.MaxOrderSize {
			return fmt.Errorf("min_order_size (%f) must be less than max_order_size (%f)", *symbol.MinOrderSize, *symbol.MaxOrderSize)
		}
	}

	// Ensure min_order_size is positive if set
	if symbol.MinOrderSize != nil && *symbol.MinOrderSize <= 0 {
		return fmt.Errorf("min_order_size must be greater than 0, got %f", *symbol.MinOrderSize)
	}

	// Ensure max_order_size is positive if set
	if symbol.MaxOrderSize != nil && *symbol.MaxOrderSize <= 0 {
		return fmt.Errorf("max_order_size must be greater than 0, got %f", *symbol.MaxOrderSize)
	}

	return nil
}

// ValidateOptionFields validates option-specific fields for OPTION symbol type
func ValidateOptionFields(symbol *marketsv1.Symbol) error {
	if symbol == nil {
		return fmt.Errorf("symbol cannot be nil")
	}

	// Require strike_price for options
	if symbol.StrikePrice == nil {
		return fmt.Errorf("strike_price is required for OPTION symbol type")
	}

	// Validate strike_price > 0
	if *symbol.StrikePrice <= 0 {
		return fmt.Errorf("strike_price must be greater than 0, got %f", *symbol.StrikePrice)
	}

	// Require expiry for options
	if symbol.Expiry == nil {
		return fmt.Errorf("expiry is required for OPTION symbol type")
	}

	// Validate expiry is in the future
	expiryTime := symbol.Expiry.AsTime()
	if expiryTime.Before(time.Now()) {
		return fmt.Errorf("expiry must be in the future, got %s", expiryTime.Format(time.RFC3339))
	}

	// Require option_type (CALL or PUT)
	if symbol.OptionType == nil {
		return fmt.Errorf("option_type is required for OPTION symbol type")
	}

	// Validate option_type is valid (CALL or PUT, not UNSPECIFIED)
	if *symbol.OptionType == marketsv1.OptionType_OPTION_TYPE_UNSPECIFIED {
		return fmt.Errorf("option_type must be CALL or PUT, not UNSPECIFIED")
	}

	return nil
}
