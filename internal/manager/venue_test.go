package manager

import (
	"context"
	"testing"

	"github.com/Combine-Capital/cqar/internal/repository"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Extended mock repository with venue support for testing
type mockVenueRepository struct {
	*mockSymbolRepository
	venues       map[string]*venuesv1.Venue
	venueAssets  map[string]*venuesv1.VenueAsset  // Key: "venueID:assetID"
	venueSymbols map[string]*venuesv1.VenueSymbol // Key: "venueID:venueSymbol"
}

func newMockVenueRepository() *mockVenueRepository {
	return &mockVenueRepository{
		mockSymbolRepository: newMockSymbolRepository(),
		venues:               make(map[string]*venuesv1.Venue),
		venueAssets:          make(map[string]*venuesv1.VenueAsset),
		venueSymbols:         make(map[string]*venuesv1.VenueSymbol),
	}
}

func (m *mockVenueRepository) CreateVenue(ctx context.Context, venue *venuesv1.Venue) error {
	m.venues[*venue.VenueId] = venue
	return nil
}

func (m *mockVenueRepository) GetVenue(ctx context.Context, id string) (*venuesv1.Venue, error) {
	venue, ok := m.venues[id]
	if !ok {
		return nil, assert.AnError
	}
	return venue, nil
}

func (m *mockVenueRepository) ListVenues(ctx context.Context, filter *repository.VenueFilter) ([]*venuesv1.Venue, error) {
	var result []*venuesv1.Venue
	for _, venue := range m.venues {
		result = append(result, venue)
	}
	return result, nil
}

func (m *mockVenueRepository) CreateVenueAsset(ctx context.Context, venueAsset *venuesv1.VenueAsset) error {
	key := *venueAsset.VenueId + ":" + *venueAsset.AssetId
	m.venueAssets[key] = venueAsset
	return nil
}

func (m *mockVenueRepository) GetVenueAsset(ctx context.Context, venueID, assetID string) (*venuesv1.VenueAsset, error) {
	key := venueID + ":" + assetID
	venueAsset, ok := m.venueAssets[key]
	if !ok {
		return nil, assert.AnError
	}
	return venueAsset, nil
}

func (m *mockVenueRepository) ListVenueAssets(ctx context.Context, filter *repository.VenueAssetFilter) ([]*venuesv1.VenueAsset, error) {
	var result []*venuesv1.VenueAsset
	for _, va := range m.venueAssets {
		// Apply filters
		if filter != nil && filter.VenueID != nil && *va.VenueId != *filter.VenueID {
			continue
		}
		if filter != nil && filter.AssetID != nil && *va.AssetId != *filter.AssetID {
			continue
		}
		result = append(result, va)
	}
	return result, nil
}

func (m *mockVenueRepository) CreateVenueSymbol(ctx context.Context, venueSymbol *venuesv1.VenueSymbol) error {
	key := *venueSymbol.VenueId + ":" + *venueSymbol.VenueSymbol
	m.venueSymbols[key] = venueSymbol
	return nil
}

func (m *mockVenueRepository) GetVenueSymbol(ctx context.Context, venueID, venueSymbolStr string) (*venuesv1.VenueSymbol, error) {
	key := venueID + ":" + venueSymbolStr
	venueSymbol, ok := m.venueSymbols[key]
	if !ok {
		return nil, assert.AnError
	}
	return venueSymbol, nil
}

func (m *mockVenueRepository) GetVenueSymbolByID(ctx context.Context, venueID, symbolID string) (*venuesv1.VenueSymbol, error) {
	for _, vs := range m.venueSymbols {
		if *vs.VenueId == venueID && *vs.SymbolId == symbolID {
			return vs, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockVenueRepository) GetVenueSymbolEnriched(ctx context.Context, venueID, venueSymbolStr string) (*venuesv1.VenueSymbol, *marketsv1.Symbol, error) {
	vs, err := m.GetVenueSymbol(ctx, venueID, venueSymbolStr)
	if err != nil {
		return nil, nil, err
	}

	symbol, err := m.GetSymbol(ctx, *vs.SymbolId)
	if err != nil {
		return nil, nil, err
	}

	return vs, symbol, nil
}

func (m *mockVenueRepository) ListVenueSymbols(ctx context.Context, filter *repository.VenueSymbolFilter) ([]*venuesv1.VenueSymbol, error) {
	var result []*venuesv1.VenueSymbol
	for _, vs := range m.venueSymbols {
		// Apply filters
		if filter != nil && filter.VenueID != nil && *vs.VenueId != *filter.VenueID {
			continue
		}
		if filter != nil && filter.SymbolID != nil && *vs.SymbolId != *filter.SymbolID {
			continue
		}
		result = append(result, vs)
	}
	return result, nil
}

// Helper function to create a test venue
func createTestVenue(id, name string) *venuesv1.Venue {
	venueType := venuesv1.VenueType_VENUE_TYPE_CEX
	isActive := true
	return &venuesv1.Venue{
		VenueId:   &id,
		Name:      &name,
		VenueType: &venueType,
		IsActive:  &isActive,
		CreatedAt: timestamppb.Now(),
	}
}

// Helper function to create a test venue asset
func createTestVenueAsset(venueID, assetID, venueAssetSymbol string) *venuesv1.VenueAsset {
	depositEnabled := true
	withdrawEnabled := true
	tradingEnabled := true
	isActive := true
	withdrawalFee := 0.001

	return &venuesv1.VenueAsset{
		VenueId:          &venueID,
		AssetId:          &assetID,
		VenueAssetSymbol: &venueAssetSymbol,
		DepositEnabled:   &depositEnabled,
		WithdrawEnabled:  &withdrawEnabled,
		TradingEnabled:   &tradingEnabled,
		WithdrawalFee:    &withdrawalFee,
		IsActive:         &isActive,
		ListedAt:         timestamppb.Now(),
	}
}

// Helper function to create a test venue symbol
func createTestVenueSymbol(venueID, symbolID, venueSymbolStr string) *venuesv1.VenueSymbol {
	makerFee := 0.001
	takerFee := 0.002
	isActive := true

	return &venuesv1.VenueSymbol{
		VenueId:     &venueID,
		SymbolId:    &symbolID,
		VenueSymbol: &venueSymbolStr,
		MakerFee:    &makerFee,
		TakerFee:    &takerFee,
		IsActive:    &isActive,
		ListedAt:    timestamppb.Now(),
	}
}

// TestCreateVenue_Success tests successful venue creation
func TestCreateVenue_Success(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	venue := createTestVenue("binance", "Binance")
	err := venueMgr.CreateVenue(context.Background(), venue)

	require.NoError(t, err)
	assert.NotEmpty(t, mockRepo.venues)
}

// TestCreateVenue_MissingVenueID tests validation when venue_id is missing
func TestCreateVenue_MissingVenueID(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	venue := createTestVenue("binance", "Binance")
	venue.VenueId = nil

	err := venueMgr.CreateVenue(context.Background(), venue)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "venue_id is required")
}

// TestCreateVenueAsset_Success tests successful venue asset creation
func TestCreateVenueAsset_Success(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venue and asset first
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	asset := createTestAsset("asset_btc", "BTC")
	mockRepo.CreateAsset(context.Background(), asset)

	// Create venue asset
	venueAsset := createTestVenueAsset("binance", "asset_btc", "BTC")
	err := venueMgr.CreateVenueAsset(context.Background(), venueAsset)

	require.NoError(t, err)
	assert.NotEmpty(t, mockRepo.venueAssets)
}

// TestCreateVenueAsset_MissingVenue tests validation when venue doesn't exist
func TestCreateVenueAsset_MissingVenue(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create only asset
	asset := createTestAsset("asset_btc", "BTC")
	mockRepo.CreateAsset(context.Background(), asset)

	// Try to create venue asset with non-existent venue
	venueAsset := createTestVenueAsset("nonexistent_venue", "asset_btc", "BTC")
	err := venueMgr.CreateVenueAsset(context.Background(), venueAsset)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "venue_id does not exist")
}

// TestCreateVenueAsset_MissingAsset tests validation when asset doesn't exist
func TestCreateVenueAsset_MissingAsset(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create only venue
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	// Try to create venue asset with non-existent asset
	venueAsset := createTestVenueAsset("binance", "nonexistent_asset", "BTC")
	err := venueMgr.CreateVenueAsset(context.Background(), venueAsset)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "asset_id does not exist")
}

// TestCreateVenueAsset_InvalidWithdrawalFee tests validation when withdrawal fee is invalid
func TestCreateVenueAsset_InvalidWithdrawalFee(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venue and asset
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	asset := createTestAsset("asset_btc", "BTC")
	mockRepo.CreateAsset(context.Background(), asset)

	// Create venue asset with invalid fee
	venueAsset := createTestVenueAsset("binance", "asset_btc", "BTC")
	invalidFee := -0.5
	venueAsset.WithdrawalFee = &invalidFee

	err := venueMgr.CreateVenueAsset(context.Background(), venueAsset)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid withdrawal_fee")
}

// TestCreateVenueAsset_ExcessiveWithdrawalFee tests validation when withdrawal fee > 150%
func TestCreateVenueAsset_ExcessiveWithdrawalFee(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venue and asset
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	asset := createTestAsset("asset_btc", "BTC")
	mockRepo.CreateAsset(context.Background(), asset)

	// Create venue asset with excessive fee
	venueAsset := createTestVenueAsset("binance", "asset_btc", "BTC")
	excessiveFee := 200.0
	venueAsset.WithdrawalFee = &excessiveFee

	err := venueMgr.CreateVenueAsset(context.Background(), venueAsset)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid withdrawal_fee")
}

// TestCreateVenueSymbol_Success tests successful venue symbol creation
func TestCreateVenueSymbol_Success(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venue, assets, and symbol
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	btc := createTestAsset("asset_btc", "BTC")
	usdt := createTestAsset("asset_usdt", "USDT")
	mockRepo.CreateAsset(context.Background(), btc)
	mockRepo.CreateAsset(context.Background(), usdt)

	symbol := createTestSymbol("asset_btc", "asset_usdt")
	mockRepo.CreateSymbol(context.Background(), symbol)

	// Create venue symbol
	venueSymbol := createTestVenueSymbol("binance", *symbol.SymbolId, "BTCUSDT")
	err := venueMgr.CreateVenueSymbol(context.Background(), venueSymbol)

	require.NoError(t, err)
	assert.NotEmpty(t, mockRepo.venueSymbols)
}

// TestCreateVenueSymbol_MissingVenue tests validation when venue doesn't exist
func TestCreateVenueSymbol_MissingVenue(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create assets and symbol
	btc := createTestAsset("asset_btc", "BTC")
	usdt := createTestAsset("asset_usdt", "USDT")
	mockRepo.CreateAsset(context.Background(), btc)
	mockRepo.CreateAsset(context.Background(), usdt)

	symbol := createTestSymbol("asset_btc", "asset_usdt")
	mockRepo.CreateSymbol(context.Background(), symbol)

	// Try to create venue symbol with non-existent venue
	venueSymbol := createTestVenueSymbol("nonexistent_venue", *symbol.SymbolId, "BTCUSDT")
	err := venueMgr.CreateVenueSymbol(context.Background(), venueSymbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "venue_id does not exist")
}

// TestCreateVenueSymbol_MissingSymbol tests validation when symbol doesn't exist
func TestCreateVenueSymbol_MissingSymbol(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create only venue
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	// Try to create venue symbol with non-existent symbol
	venueSymbol := createTestVenueSymbol("binance", "nonexistent_symbol", "BTCUSDT")
	err := venueMgr.CreateVenueSymbol(context.Background(), venueSymbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "symbol_id does not exist")
}

// TestCreateVenueSymbol_InvalidMakerFee tests validation when maker fee is invalid
func TestCreateVenueSymbol_InvalidMakerFee(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venue, assets, and symbol
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	btc := createTestAsset("asset_btc", "BTC")
	usdt := createTestAsset("asset_usdt", "USDT")
	mockRepo.CreateAsset(context.Background(), btc)
	mockRepo.CreateAsset(context.Background(), usdt)

	symbol := createTestSymbol("asset_btc", "asset_usdt")
	mockRepo.CreateSymbol(context.Background(), symbol)

	// Create venue symbol with invalid maker fee
	venueSymbol := createTestVenueSymbol("binance", *symbol.SymbolId, "BTCUSDT")
	invalidFee := -0.1
	venueSymbol.MakerFee = &invalidFee

	err := venueMgr.CreateVenueSymbol(context.Background(), venueSymbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid maker_fee")
}

// TestCreateVenueSymbol_InvalidTakerFee tests validation when taker fee is invalid
func TestCreateVenueSymbol_InvalidTakerFee(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venue, assets, and symbol
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	btc := createTestAsset("asset_btc", "BTC")
	usdt := createTestAsset("asset_usdt", "USDT")
	mockRepo.CreateAsset(context.Background(), btc)
	mockRepo.CreateAsset(context.Background(), usdt)

	symbol := createTestSymbol("asset_btc", "asset_usdt")
	mockRepo.CreateSymbol(context.Background(), symbol)

	// Create venue symbol with invalid taker fee
	venueSymbol := createTestVenueSymbol("binance", *symbol.SymbolId, "BTCUSDT")
	excessiveFee := 200.0
	venueSymbol.TakerFee = &excessiveFee

	err := venueMgr.CreateVenueSymbol(context.Background(), venueSymbol)

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "invalid taker_fee")
}

// TestGetVenueSymbol_Enriched tests getting venue symbol with enriched canonical symbol data
func TestGetVenueSymbol_Enriched(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venue, assets, and symbol
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	btc := createTestAsset("asset_btc", "BTC")
	usdt := createTestAsset("asset_usdt", "USDT")
	mockRepo.CreateAsset(context.Background(), btc)
	mockRepo.CreateAsset(context.Background(), usdt)

	symbol := createTestSymbol("asset_btc", "asset_usdt")
	mockRepo.CreateSymbol(context.Background(), symbol)

	venueSymbol := createTestVenueSymbol("binance", *symbol.SymbolId, "BTCUSDT")
	mockRepo.CreateVenueSymbol(context.Background(), venueSymbol)

	// Get enriched venue symbol (cqmd use case)
	vs, sym, err := venueMgr.GetVenueSymbol(context.Background(), "binance", "BTCUSDT")

	require.NoError(t, err)
	assert.NotNil(t, vs)
	assert.NotNil(t, sym)
	assert.Equal(t, "binance", *vs.VenueId)
	assert.Equal(t, "BTCUSDT", *vs.VenueSymbol)
	assert.Equal(t, *symbol.SymbolId, *sym.SymbolId)
	assert.Equal(t, 0.01, *sym.TickSize) // Market specs from canonical symbol
}

// TestGetVenueSymbol_NotFound tests getting non-existent venue symbol
func TestGetVenueSymbol_NotFound(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	vs, sym, err := venueMgr.GetVenueSymbol(context.Background(), "binance", "NONEXISTENT")

	require.Error(t, err)
	assert.Nil(t, vs)
	assert.Nil(t, sym)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

// TestListVenueAssets_ByVenue tests listing assets by venue (cqvx use case)
func TestListVenueAssets_ByVenue(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venue and assets
	venue := createTestVenue("binance", "Binance")
	mockRepo.CreateVenue(context.Background(), venue)

	btc := createTestAsset("asset_btc", "BTC")
	eth := createTestAsset("asset_eth", "ETH")
	mockRepo.CreateAsset(context.Background(), btc)
	mockRepo.CreateAsset(context.Background(), eth)

	// Create venue assets
	vaBtc := createTestVenueAsset("binance", "asset_btc", "BTC")
	vaEth := createTestVenueAsset("binance", "asset_eth", "ETH")
	mockRepo.CreateVenueAsset(context.Background(), vaBtc)
	mockRepo.CreateVenueAsset(context.Background(), vaEth)

	// List all assets on Binance
	venueID := "binance"
	filter := &repository.VenueAssetFilter{
		VenueID: &venueID,
	}
	assets, err := venueMgr.ListVenueAssets(context.Background(), filter)

	require.NoError(t, err)
	assert.Len(t, assets, 2)
}

// TestListVenueAssets_ByAsset tests listing venues by asset ("which venues trade BTC?")
func TestListVenueAssets_ByAsset(t *testing.T) {
	mockRepo := newMockVenueRepository()
	qualityMgr := NewQualityManager(mockRepo, nil)
	assetMgr := NewAssetManager(mockRepo, qualityMgr, nil)
	symbolMgr := NewSymbolManager(mockRepo, assetMgr, nil)
	venueMgr := NewVenueManager(mockRepo, assetMgr, symbolMgr, nil)

	// Create venues and asset
	binance := createTestVenue("binance", "Binance")
	coinbase := createTestVenue("coinbase", "Coinbase")
	mockRepo.CreateVenue(context.Background(), binance)
	mockRepo.CreateVenue(context.Background(), coinbase)

	btc := createTestAsset("asset_btc", "BTC")
	mockRepo.CreateAsset(context.Background(), btc)

	// Create venue assets
	vaBinance := createTestVenueAsset("binance", "asset_btc", "BTC")
	vaCoinbase := createTestVenueAsset("coinbase", "asset_btc", "BTC")
	mockRepo.CreateVenueAsset(context.Background(), vaBinance)
	mockRepo.CreateVenueAsset(context.Background(), vaCoinbase)

	// List all venues trading BTC
	assetID := "asset_btc"
	filter := &repository.VenueAssetFilter{
		AssetID: &assetID,
	}
	venueAssets, err := venueMgr.ListVenueAssets(context.Background(), filter)

	require.NoError(t, err)
	assert.Len(t, venueAssets, 2)
}
