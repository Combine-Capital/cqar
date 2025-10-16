package manager

import (
	"context"
	"testing"
	"time"

	"github.com/Combine-Capital/cqar/internal/repository"
	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Extended mock repository with symbol support for testing
type mockSymbolRepository struct {
	*mockRepository
	symbols map[string]*marketsv1.Symbol
}

func newMockSymbolRepository() *mockSymbolRepository {
	return &mockSymbolRepository{
		mockRepository: newMockRepository(),
		symbols:        make(map[string]*marketsv1.Symbol),
	}
}

func (m *mockSymbolRepository) CreateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error {
	m.symbols[*symbol.SymbolId] = symbol
	return nil
}

func (m *mockSymbolRepository) GetSymbol(ctx context.Context, id string) (*marketsv1.Symbol, error) {
	symbol, ok := m.symbols[id]
	if !ok {
		return nil, assert.AnError
	}
	return symbol, nil
}

func (m *mockSymbolRepository) UpdateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error {
	if _, ok := m.symbols[*symbol.SymbolId]; !ok {
		return assert.AnError
	}
	m.symbols[*symbol.SymbolId] = symbol
	return nil
}

func (m *mockSymbolRepository) DeleteSymbol(ctx context.Context, id string) error {
	if _, ok := m.symbols[id]; !ok {
		return assert.AnError
	}
	delete(m.symbols, id)
	return nil
}

func (m *mockSymbolRepository) ListSymbols(ctx context.Context, filter *repository.SymbolFilter) ([]*marketsv1.Symbol, error) {
	var result []*marketsv1.Symbol
	for _, symbol := range m.symbols {
		// Apply filters
		if filter != nil && filter.SymbolType != nil && symbol.GetSymbolType().String() != *filter.SymbolType {
			continue
		}
		result = append(result, symbol)
	}
	return result, nil
}

func (m *mockSymbolRepository) SearchSymbols(ctx context.Context, query string, filter *repository.SymbolFilter) ([]*marketsv1.Symbol, error) {
	return m.ListSymbols(ctx, filter)
}

// Helper function to create a test symbol
func createTestSymbol(baseAssetID, quoteAssetID string) *marketsv1.Symbol {
	symbolID := "sym_test_123"
	symbolType := marketsv1.SymbolType_SYMBOL_TYPE_SPOT
	tickSize := 0.01
	lotSize := 0.00001
	minOrderSize := 0.0001
	maxOrderSize := 10000.0

	return &marketsv1.Symbol{
		SymbolId:     &symbolID,
		BaseAssetId:  &baseAssetID,
		QuoteAssetId: &quoteAssetID,
		SymbolType:   &symbolType,
		TickSize:     &tickSize,
		LotSize:      &lotSize,
		MinOrderSize: &minOrderSize,
		MaxOrderSize: &maxOrderSize,
		CreatedAt:    timestamppb.Now(),
		UpdatedAt:    timestamppb.Now(),
	}
}

// Helper function to create test assets
func createTestAsset(id, symbol string) *assetsv1.Asset {
	assetType := assetsv1.AssetType_ASSET_TYPE_NATIVE
	return &assetsv1.Asset{
		AssetId:   &id,
		Symbol:    &symbol,
		Name:      &symbol,
		AssetType: &assetType,
		CreatedAt: timestamppb.Now(),
		UpdatedAt: timestamppb.Now(),
	}
}

// TestCreateSymbol_Success tests successful symbol creation
func TestCreateSymbol_Success(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets first
	btcID := "asset_btc"
	usdtID := "asset_usdt"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdtID, "USDT"))

	// Create symbol
	symbol := createTestSymbol(btcID, usdtID)
	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.NoError(t, err)
	assert.NotEmpty(t, mockRepo.symbols)
}

// TestCreateSymbol_MissingBaseAsset tests validation when base_asset_id doesn't exist
func TestCreateSymbol_MissingBaseAsset(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create only quote asset
	usdtID := "asset_usdt"
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdtID, "USDT"))

	// Try to create symbol with non-existent base asset
	symbol := createTestSymbol("asset_nonexistent", usdtID)
	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "base_asset_id does not exist")
}

// TestCreateSymbol_MissingQuoteAsset tests validation when quote_asset_id doesn't exist
func TestCreateSymbol_MissingQuoteAsset(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create only base asset
	btcID := "asset_btc"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))

	// Try to create symbol with non-existent quote asset
	symbol := createTestSymbol(btcID, "asset_nonexistent")
	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "quote_asset_id does not exist")
}

// TestCreateSymbol_InvalidTickSize tests validation when tick_size <= 0
func TestCreateSymbol_InvalidTickSize(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdtID := "asset_usdt"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdtID, "USDT"))

	// Create symbol with invalid tick_size
	symbol := createTestSymbol(btcID, usdtID)
	invalidTickSize := 0.0
	symbol.TickSize = &invalidTickSize

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "tick_size must be greater than 0")
}

// TestCreateSymbol_InvalidLotSize tests validation when lot_size <= 0
func TestCreateSymbol_InvalidLotSize(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdtID := "asset_usdt"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdtID, "USDT"))

	// Create symbol with invalid lot_size
	symbol := createTestSymbol(btcID, usdtID)
	invalidLotSize := -0.01
	symbol.LotSize = &invalidLotSize

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "lot_size must be greater than 0")
}

// TestCreateSymbol_MinOrderSizeGreaterThanMax tests validation when min_order_size >= max_order_size
func TestCreateSymbol_MinOrderSizeGreaterThanMax(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdtID := "asset_usdt"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdtID, "USDT"))

	// Create symbol with invalid order sizes
	symbol := createTestSymbol(btcID, usdtID)
	minOrderSize := 1000.0
	maxOrderSize := 100.0
	symbol.MinOrderSize = &minOrderSize
	symbol.MaxOrderSize = &maxOrderSize

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "min_order_size")
	assert.Contains(t, st.Message(), "must be less than max_order_size")
}

// TestCreateSymbol_OptionWithoutStrikePrice tests validation when option type lacks strike_price
func TestCreateSymbol_OptionWithoutStrikePrice(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdID := "asset_usd"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdID, "USD"))

	// Create option symbol without strike_price
	symbol := createTestSymbol(btcID, usdID)
	optionType := marketsv1.SymbolType_SYMBOL_TYPE_OPTION
	symbol.SymbolType = &optionType
	// Missing strike_price

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "strike_price is required")
}

// TestCreateSymbol_OptionWithInvalidStrikePrice tests validation when strike_price <= 0
func TestCreateSymbol_OptionWithInvalidStrikePrice(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdID := "asset_usd"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdID, "USD"))

	// Create option symbol with invalid strike_price
	symbol := createTestSymbol(btcID, usdID)
	optionType := marketsv1.SymbolType_SYMBOL_TYPE_OPTION
	symbol.SymbolType = &optionType
	invalidStrike := 0.0
	symbol.StrikePrice = &invalidStrike
	optType := marketsv1.OptionType_OPTION_TYPE_CALL
	symbol.OptionType = &optType
	symbol.Expiry = timestamppb.New(time.Now().Add(24 * time.Hour))

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "strike_price must be greater than 0")
}

// TestCreateSymbol_OptionWithoutExpiry tests validation when option type lacks expiry
func TestCreateSymbol_OptionWithoutExpiry(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdID := "asset_usd"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdID, "USD"))

	// Create option symbol without expiry
	symbol := createTestSymbol(btcID, usdID)
	optionType := marketsv1.SymbolType_SYMBOL_TYPE_OPTION
	symbol.SymbolType = &optionType
	strikePrice := 50000.0
	symbol.StrikePrice = &strikePrice
	// Missing expiry

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "expiry is required")
}

// TestCreateSymbol_OptionWithPastExpiry tests validation when expiry is in the past
func TestCreateSymbol_OptionWithPastExpiry(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdID := "asset_usd"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdID, "USD"))

	// Create option symbol with past expiry
	symbol := createTestSymbol(btcID, usdID)
	optionType := marketsv1.SymbolType_SYMBOL_TYPE_OPTION
	symbol.SymbolType = &optionType
	strikePrice := 50000.0
	symbol.StrikePrice = &strikePrice
	symbol.Expiry = timestamppb.New(time.Now().Add(-24 * time.Hour)) // Yesterday
	optType := marketsv1.OptionType_OPTION_TYPE_CALL
	symbol.OptionType = &optType

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "expiry must be in the future")
}

// TestCreateSymbol_OptionWithoutOptionType tests validation when option type lacks option_type field
func TestCreateSymbol_OptionWithoutOptionType(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdID := "asset_usd"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdID, "USD"))

	// Create option symbol without option_type
	symbol := createTestSymbol(btcID, usdID)
	symbolType := marketsv1.SymbolType_SYMBOL_TYPE_OPTION
	symbol.SymbolType = &symbolType
	strikePrice := 50000.0
	symbol.StrikePrice = &strikePrice
	symbol.Expiry = timestamppb.New(time.Now().Add(24 * time.Hour))
	// Missing option_type

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "option_type is required")
}

// TestCreateSymbol_OptionSuccess tests successful option symbol creation
func TestCreateSymbol_OptionSuccess(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdID := "asset_usd"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdID, "USD"))

	// Create valid option symbol
	symbol := createTestSymbol(btcID, usdID)
	symbolType := marketsv1.SymbolType_SYMBOL_TYPE_OPTION
	symbol.SymbolType = &symbolType
	strikePrice := 50000.0
	symbol.StrikePrice = &strikePrice
	symbol.Expiry = timestamppb.New(time.Now().Add(30 * 24 * time.Hour)) // 30 days in future
	optType := marketsv1.OptionType_OPTION_TYPE_CALL
	symbol.OptionType = &optType

	err := symbolMgr.CreateSymbol(context.Background(), symbol)

	require.NoError(t, err)
	assert.NotEmpty(t, mockRepo.symbols)
}

// TestListSymbols_FilterByType tests listing symbols filtered by type
func TestListSymbols_FilterByType(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets
	btcID := "asset_btc"
	usdtID := "asset_usdt"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdtID, "USDT"))

	// Create SPOT symbol
	spotSymbol := createTestSymbol(btcID, usdtID)
	spotID := "sym_spot_123"
	spotSymbol.SymbolId = &spotID
	mockRepo.CreateSymbol(context.Background(), spotSymbol)

	// Create PERPETUAL symbol
	perpSymbol := createTestSymbol(btcID, usdtID)
	perpID := "sym_perp_123"
	perpType := marketsv1.SymbolType_SYMBOL_TYPE_PERPETUAL
	perpSymbol.SymbolId = &perpID
	perpSymbol.SymbolType = &perpType
	mockRepo.CreateSymbol(context.Background(), perpSymbol)

	// Filter by SPOT
	spotTypeStr := marketsv1.SymbolType_SYMBOL_TYPE_SPOT.String()
	filter := &repository.SymbolFilter{
		SymbolType: &spotTypeStr,
	}
	symbols, err := symbolMgr.ListSymbols(context.Background(), filter)

	require.NoError(t, err)
	assert.Len(t, symbols, 1)
	assert.Equal(t, spotID, *symbols[0].SymbolId)
}

// TestGetSymbol_NotFound tests getting a non-existent symbol
func TestGetSymbol_NotFound(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	symbol, err := symbolMgr.GetSymbol(context.Background(), "nonexistent")

	require.Error(t, err)
	assert.Nil(t, symbol)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

// TestUpdateSymbol_Success tests successful symbol update
func TestUpdateSymbol_Success(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets and symbol
	btcID := "asset_btc"
	usdtID := "asset_usdt"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdtID, "USDT"))

	symbol := createTestSymbol(btcID, usdtID)
	mockRepo.CreateSymbol(context.Background(), symbol)

	// Update the symbol
	newTickSize := 0.1
	symbol.TickSize = &newTickSize
	err := symbolMgr.UpdateSymbol(context.Background(), symbol)

	require.NoError(t, err)
	updatedSymbol, _ := mockRepo.GetSymbol(context.Background(), *symbol.SymbolId)
	assert.Equal(t, newTickSize, *updatedSymbol.TickSize)
}

// TestDeleteSymbol_Success tests successful symbol deletion
func TestDeleteSymbol_Success(t *testing.T) {
	mockRepo := newMockSymbolRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)

	// Create test assets and symbol
	btcID := "asset_btc"
	usdtID := "asset_usdt"
	mockRepo.CreateAsset(context.Background(), createTestAsset(btcID, "BTC"))
	mockRepo.CreateAsset(context.Background(), createTestAsset(usdtID, "USDT"))

	symbol := createTestSymbol(btcID, usdtID)
	mockRepo.CreateSymbol(context.Background(), symbol)

	// Delete the symbol
	err := symbolMgr.DeleteSymbol(context.Background(), *symbol.SymbolId)

	require.NoError(t, err)
	_, err = mockRepo.GetSymbol(context.Background(), *symbol.SymbolId)
	assert.Error(t, err)
}
