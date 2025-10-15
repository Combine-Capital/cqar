package manager

import (
	"context"
	"testing"

	"github.com/Combine-Capital/cqar/internal/repository"
	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mockRepository implements repository.Repository for testing
type mockRepository struct {
	assets        map[string]*assetsv1.Asset
	deployments   map[string]*assetsv1.AssetDeployment
	chains        map[string]*assetsv1.Chain
	relationships map[string]*assetsv1.AssetRelationship
	groups        map[string]*assetsv1.AssetGroup
	flags         map[string]*assetsv1.AssetQualityFlag
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		assets:        make(map[string]*assetsv1.Asset),
		deployments:   make(map[string]*assetsv1.AssetDeployment),
		chains:        make(map[string]*assetsv1.Chain),
		relationships: make(map[string]*assetsv1.AssetRelationship),
		groups:        make(map[string]*assetsv1.AssetGroup),
		flags:         make(map[string]*assetsv1.AssetQualityFlag),
	}
}

func (m *mockRepository) CreateAsset(ctx context.Context, asset *assetsv1.Asset) error {
	m.assets[*asset.AssetId] = asset
	return nil
}

func (m *mockRepository) GetAsset(ctx context.Context, id string) (*assetsv1.Asset, error) {
	asset, ok := m.assets[id]
	if !ok {
		return nil, assert.AnError
	}
	return asset, nil
}

func (m *mockRepository) UpdateAsset(ctx context.Context, asset *assetsv1.Asset) error {
	if _, ok := m.assets[*asset.AssetId]; !ok {
		return assert.AnError
	}
	m.assets[*asset.AssetId] = asset
	return nil
}

func (m *mockRepository) DeleteAsset(ctx context.Context, id string) error {
	if _, ok := m.assets[id]; !ok {
		return assert.AnError
	}
	delete(m.assets, id)
	return nil
}

func (m *mockRepository) ListAssets(ctx context.Context, filter *repository.AssetFilter) ([]*assetsv1.Asset, error) {
	var result []*assetsv1.Asset
	for _, asset := range m.assets {
		result = append(result, asset)
	}
	return result, nil
}

func (m *mockRepository) SearchAssets(ctx context.Context, query string, filter *repository.AssetFilter) ([]*assetsv1.Asset, error) {
	return m.ListAssets(ctx, filter)
}

func (m *mockRepository) CreateAssetDeployment(ctx context.Context, deployment *assetsv1.AssetDeployment) error {
	m.deployments[*deployment.DeploymentId] = deployment
	return nil
}

func (m *mockRepository) GetAssetDeployment(ctx context.Context, id string) (*assetsv1.AssetDeployment, error) {
	deployment, ok := m.deployments[id]
	if !ok {
		return nil, assert.AnError
	}
	return deployment, nil
}

func (m *mockRepository) ListAssetDeployments(ctx context.Context, filter *repository.DeploymentFilter) ([]*assetsv1.AssetDeployment, error) {
	var result []*assetsv1.AssetDeployment
	for _, deployment := range m.deployments {
		result = append(result, deployment)
	}
	return result, nil
}

func (m *mockRepository) GetAssetDeploymentByChain(ctx context.Context, assetID, chainID string) (*assetsv1.AssetDeployment, error) {
	for _, deployment := range m.deployments {
		if *deployment.AssetId == assetID && *deployment.ChainId == chainID {
			return deployment, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockRepository) CreateAssetRelationship(ctx context.Context, relationship *assetsv1.AssetRelationship) error {
	m.relationships[*relationship.RelationshipId] = relationship
	return nil
}

func (m *mockRepository) GetAssetRelationship(ctx context.Context, id string) (*assetsv1.AssetRelationship, error) {
	relationship, ok := m.relationships[id]
	if !ok {
		return nil, assert.AnError
	}
	return relationship, nil
}

func (m *mockRepository) ListAssetRelationships(ctx context.Context, filter *repository.RelationshipFilter) ([]*assetsv1.AssetRelationship, error) {
	var result []*assetsv1.AssetRelationship
	for _, relationship := range m.relationships {
		// Apply filter
		if filter != nil && filter.FromAssetID != nil && *relationship.FromAssetId != *filter.FromAssetID {
			continue
		}
		if filter != nil && filter.RelationshipType != nil && relationship.GetRelationshipType().String() != *filter.RelationshipType {
			continue
		}
		result = append(result, relationship)
	}
	return result, nil
}

func (m *mockRepository) CreateAssetGroup(ctx context.Context, group *assetsv1.AssetGroup) error {
	m.groups[*group.GroupId] = group
	return nil
}

func (m *mockRepository) GetAssetGroup(ctx context.Context, id string) (*assetsv1.AssetGroup, error) {
	group, ok := m.groups[id]
	if !ok {
		return nil, assert.AnError
	}
	return group, nil
}

func (m *mockRepository) GetAssetGroupByName(ctx context.Context, name string) (*assetsv1.AssetGroup, error) {
	for _, group := range m.groups {
		if *group.Name == name {
			return group, nil
		}
	}
	return nil, assert.AnError
}

func (m *mockRepository) AddAssetToGroup(ctx context.Context, groupID, assetID string, weight float64) error {
	return nil
}

func (m *mockRepository) RemoveAssetFromGroup(ctx context.Context, groupID, assetID string) error {
	return nil
}

func (m *mockRepository) ListAssetGroups(ctx context.Context, filter *repository.AssetGroupFilter) ([]*assetsv1.AssetGroup, error) {
	var result []*assetsv1.AssetGroup
	for _, group := range m.groups {
		result = append(result, group)
	}
	return result, nil
}

func (m *mockRepository) GetChain(ctx context.Context, id string) (*assetsv1.Chain, error) {
	chain, ok := m.chains[id]
	if !ok {
		return nil, assert.AnError
	}
	return chain, nil
}

func (m *mockRepository) CreateChain(ctx context.Context, chain *assetsv1.Chain) error {
	m.chains[*chain.ChainId] = chain
	return nil
}

func (m *mockRepository) ListChains(ctx context.Context, filter *repository.ChainFilter) ([]*assetsv1.Chain, error) {
	var result []*assetsv1.Chain
	for _, chain := range m.chains {
		result = append(result, chain)
	}
	return result, nil
}

func (m *mockRepository) RaiseQualityFlag(ctx context.Context, flag *assetsv1.AssetQualityFlag) error {
	m.flags[*flag.FlagId] = flag
	return nil
}

func (m *mockRepository) ResolveQualityFlag(ctx context.Context, id string, resolvedBy string, resolutionNotes string) error {
	flag, ok := m.flags[id]
	if !ok {
		return assert.AnError
	}
	flag.ResolvedAt = timestamppb.Now()
	rb := resolvedBy
	flag.ResolvedBy = &rb
	rn := resolutionNotes
	flag.ResolutionNotes = &rn
	return nil
}

func (m *mockRepository) GetQualityFlag(ctx context.Context, id string) (*assetsv1.AssetQualityFlag, error) {
	flag, ok := m.flags[id]
	if !ok {
		return nil, assert.AnError
	}
	return flag, nil
}

func (m *mockRepository) ListQualityFlags(ctx context.Context, filter *repository.QualityFlagFilter) ([]*assetsv1.AssetQualityFlag, error) {
	var result []*assetsv1.AssetQualityFlag
	for _, flag := range m.flags {
		// Apply filters
		if filter != nil && filter.AssetID != nil && *flag.AssetId != *filter.AssetID {
			continue
		}
		if filter != nil && filter.Severity != nil && flag.GetSeverity().String() != *filter.Severity {
			continue
		}
		if filter != nil && filter.ActiveOnly && flag.ResolvedAt != nil {
			continue
		}
		result = append(result, flag)
	}
	return result, nil
}

// Stub implementations for interface compliance
func (m *mockRepository) CreateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error {
	return nil
}

func (m *mockRepository) GetSymbol(ctx context.Context, id string) (*marketsv1.Symbol, error) {
	return nil, nil
}

func (m *mockRepository) UpdateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error {
	return nil
}

func (m *mockRepository) DeleteSymbol(ctx context.Context, id string) error {
	return nil
}

func (m *mockRepository) ListSymbols(ctx context.Context, filter *repository.SymbolFilter) ([]*marketsv1.Symbol, error) {
	return nil, nil
}

func (m *mockRepository) SearchSymbols(ctx context.Context, query string, filter *repository.SymbolFilter) ([]*marketsv1.Symbol, error) {
	return nil, nil
}

func (m *mockRepository) CreateSymbolIdentifier(ctx context.Context, identifier *marketsv1.SymbolIdentifier) error {
	return nil
}

func (m *mockRepository) GetSymbolIdentifier(ctx context.Context, id string) (*marketsv1.SymbolIdentifier, error) {
	return nil, nil
}

func (m *mockRepository) ListSymbolIdentifiers(ctx context.Context, filter *repository.SymbolIdentifierFilter) ([]*marketsv1.SymbolIdentifier, error) {
	return nil, nil
}

func (m *mockRepository) CreateVenue(ctx context.Context, venue *venuesv1.Venue) error {
	return nil
}

func (m *mockRepository) GetVenue(ctx context.Context, id string) (*venuesv1.Venue, error) {
	return nil, nil
}

func (m *mockRepository) ListVenues(ctx context.Context, filter *repository.VenueFilter) ([]*venuesv1.Venue, error) {
	return nil, nil
}

func (m *mockRepository) CreateVenueAsset(ctx context.Context, venueAsset *venuesv1.VenueAsset) error {
	return nil
}

func (m *mockRepository) GetVenueAsset(ctx context.Context, venueID, assetID string) (*venuesv1.VenueAsset, error) {
	return nil, nil
}

func (m *mockRepository) ListVenueAssets(ctx context.Context, filter *repository.VenueAssetFilter) ([]*venuesv1.VenueAsset, error) {
	return nil, nil
}

func (m *mockRepository) CreateVenueSymbol(ctx context.Context, venueSymbol *venuesv1.VenueSymbol) error {
	return nil
}

func (m *mockRepository) GetVenueSymbol(ctx context.Context, venueID, venueSymbol string) (*venuesv1.VenueSymbol, error) {
	return nil, nil
}

func (m *mockRepository) GetVenueSymbolByID(ctx context.Context, venueID, symbolID string) (*venuesv1.VenueSymbol, error) {
	return nil, nil
}

func (m *mockRepository) ListVenueSymbols(ctx context.Context, filter *repository.VenueSymbolFilter) ([]*venuesv1.VenueSymbol, error) {
	return nil, nil
}

func (m *mockRepository) GetVenueSymbolEnriched(ctx context.Context, venueID, venueSymbol string) (*venuesv1.VenueSymbol, *marketsv1.Symbol, error) {
	return nil, nil, nil
}

func (m *mockRepository) CreateAssetIdentifier(ctx context.Context, identifier *assetsv1.AssetIdentifier) error {
	return nil
}

func (m *mockRepository) GetAssetIdentifier(ctx context.Context, id string) (*assetsv1.AssetIdentifier, error) {
	return nil, nil
}

func (m *mockRepository) ListAssetIdentifiers(ctx context.Context, filter *repository.AssetIdentifierFilter) ([]*assetsv1.AssetIdentifier, error) {
	return nil, nil
}

func (m *mockRepository) WithTransaction(ctx context.Context, fn func(repo repository.Repository) error) error {
	return fn(m)
}

func (m *mockRepository) Ping(ctx context.Context) error {
	return nil
}

// Test helper functions
func strPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

// Tests

func TestAssetManager_CreateAsset_Validation(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	qualityMgr := NewQualityManager(repo)
	mgr := NewAssetManager(repo, qualityMgr)

	tests := []struct {
		name        string
		asset       *assetsv1.Asset
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "valid asset",
			asset: &assetsv1.Asset{
				AssetId:   strPtr("btc"),
				Symbol:    strPtr("BTC"),
				Name:      strPtr("Bitcoin"),
				AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_NATIVE),
			},
			wantErr: false,
		},
		{
			name: "missing symbol",
			asset: &assetsv1.Asset{
				AssetId:   strPtr("test"),
				Name:      strPtr("Test Asset"),
				AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_ERC20),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing name",
			asset: &assetsv1.Asset{
				AssetId:   strPtr("test"),
				Symbol:    strPtr("TEST"),
				AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_ERC20),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing asset type",
			asset: &assetsv1.Asset{
				AssetId: strPtr("test"),
				Symbol:  strPtr("TEST"),
				Name:    strPtr("Test Asset"),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "empty symbol",
			asset: &assetsv1.Asset{
				AssetId:   strPtr("test"),
				Symbol:    strPtr("  "),
				Name:      strPtr("Test Asset"),
				AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_ERC20),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.CreateAsset(ctx, tt.asset)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAssetManager_CreateAssetDeployment_Validation(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	qualityMgr := NewQualityManager(repo)
	mgr := NewAssetManager(repo, qualityMgr)

	// Setup: Create asset and chain
	asset := &assetsv1.Asset{
		AssetId:   strPtr("usdc"),
		Symbol:    strPtr("USDC"),
		Name:      strPtr("USD Coin"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_ERC20),
	}
	repo.assets["usdc"] = asset

	evmChainType := "EVM"
	chain := &assetsv1.Chain{
		ChainId:   strPtr("ethereum"),
		ChainName: strPtr("Ethereum"),
		ChainType: &evmChainType,
	}
	repo.chains["ethereum"] = chain

	tests := []struct {
		name        string
		deployment  *assetsv1.AssetDeployment
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "valid EVM deployment",
			deployment: &assetsv1.AssetDeployment{
				DeploymentId: strPtr("ethereum:0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				AssetId:      strPtr("usdc"),
				ChainId:      strPtr("ethereum"),
				Address:      strPtr("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				Decimals:     int32Ptr(6),
			},
			wantErr: false,
		},
		{
			name: "invalid address format",
			deployment: &assetsv1.AssetDeployment{
				DeploymentId: strPtr("ethereum:invalid"),
				AssetId:      strPtr("usdc"),
				ChainId:      strPtr("ethereum"),
				Address:      strPtr("invalid"),
				Decimals:     int32Ptr(6),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "invalid decimals - too high",
			deployment: &assetsv1.AssetDeployment{
				DeploymentId: strPtr("ethereum:0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				AssetId:      strPtr("usdc"),
				ChainId:      strPtr("ethereum"),
				Address:      strPtr("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				Decimals:     int32Ptr(25),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "invalid decimals - negative",
			deployment: &assetsv1.AssetDeployment{
				DeploymentId: strPtr("ethereum:0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				AssetId:      strPtr("usdc"),
				ChainId:      strPtr("ethereum"),
				Address:      strPtr("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				Decimals:     int32Ptr(-1),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing asset_id",
			deployment: &assetsv1.AssetDeployment{
				DeploymentId: strPtr("ethereum:0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				ChainId:      strPtr("ethereum"),
				Address:      strPtr("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				Decimals:     int32Ptr(6),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "nonexistent asset",
			deployment: &assetsv1.AssetDeployment{
				DeploymentId: strPtr("ethereum:0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				AssetId:      strPtr("nonexistent"),
				ChainId:      strPtr("ethereum"),
				Address:      strPtr("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"),
				Decimals:     int32Ptr(6),
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.CreateAssetDeployment(ctx, tt.deployment)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAssetManager_CreateAssetRelationship_CycleDetection(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	qualityMgr := NewQualityManager(repo)
	mgr := NewAssetManager(repo, qualityMgr)

	// Setup: Create assets
	eth := &assetsv1.Asset{
		AssetId:   strPtr("eth"),
		Symbol:    strPtr("ETH"),
		Name:      strPtr("Ethereum"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_NATIVE),
	}
	weth := &assetsv1.Asset{
		AssetId:   strPtr("weth"),
		Symbol:    strPtr("WETH"),
		Name:      strPtr("Wrapped Ether"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_WRAPPED),
	}
	steth := &assetsv1.Asset{
		AssetId:   strPtr("steth"),
		Symbol:    strPtr("stETH"),
		Name:      strPtr("Staked Ether"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_RECEIPT_TOKEN),
	}

	repo.assets["eth"] = eth
	repo.assets["weth"] = weth
	repo.assets["steth"] = steth

	relType := assetsv1.RelationshipType_RELATIONSHIP_TYPE_WRAPS

	// Test 1: Create valid relationship WETH -> ETH
	rel1 := &assetsv1.AssetRelationship{
		RelationshipId:   strPtr("rel1"),
		FromAssetId:      strPtr("weth"),
		ToAssetId:        strPtr("eth"),
		RelationshipType: &relType,
	}
	err := mgr.CreateAssetRelationship(ctx, rel1)
	require.NoError(t, err)

	// Test 2: Create another relationship stETH -> ETH (should succeed, no cycle)
	rel2 := &assetsv1.AssetRelationship{
		RelationshipId:   strPtr("rel2"),
		FromAssetId:      strPtr("steth"),
		ToAssetId:        strPtr("eth"),
		RelationshipType: &relType,
	}
	err = mgr.CreateAssetRelationship(ctx, rel2)
	require.NoError(t, err)

	// Test 3: Try to create cycle ETH -> WETH (should fail, creates cycle)
	rel3 := &assetsv1.AssetRelationship{
		RelationshipId:   strPtr("rel3"),
		FromAssetId:      strPtr("eth"),
		ToAssetId:        strPtr("weth"),
		RelationshipType: &relType,
	}
	err = mgr.CreateAssetRelationship(ctx, rel3)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "cycle")
}

func TestAssetManager_CreateAssetRelationship_SelfReferential(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	qualityMgr := NewQualityManager(repo)
	mgr := NewAssetManager(repo, qualityMgr)

	// Setup: Create asset
	eth := &assetsv1.Asset{
		AssetId:   strPtr("eth"),
		Symbol:    strPtr("ETH"),
		Name:      strPtr("Ethereum"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_NATIVE),
	}
	repo.assets["eth"] = eth

	relType := assetsv1.RelationshipType_RELATIONSHIP_TYPE_WRAPS

	// Try to create self-referential relationship
	rel := &assetsv1.AssetRelationship{
		RelationshipId:   strPtr("rel1"),
		FromAssetId:      strPtr("eth"),
		ToAssetId:        strPtr("eth"),
		RelationshipType: &relType,
	}

	err := mgr.CreateAssetRelationship(ctx, rel)
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
	assert.Contains(t, st.Message(), "self-referential")
}

func TestAssetManager_CreateAssetGroup_MemberValidation(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	qualityMgr := NewQualityManager(repo)
	mgr := NewAssetManager(repo, qualityMgr)

	// Setup: Create assets
	eth := &assetsv1.Asset{
		AssetId:   strPtr("eth"),
		Symbol:    strPtr("ETH"),
		Name:      strPtr("Ethereum"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_NATIVE),
	}
	weth := &assetsv1.Asset{
		AssetId:   strPtr("weth"),
		Symbol:    strPtr("WETH"),
		Name:      strPtr("Wrapped Ether"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_WRAPPED),
	}

	repo.assets["eth"] = eth
	repo.assets["weth"] = weth

	tests := []struct {
		name        string
		group       *assetsv1.AssetGroup
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "valid group with existing members",
			group: &assetsv1.AssetGroup{
				GroupId: strPtr("eth_group"),
				Name:    strPtr("All ETH Variants"),
				Members: []*assetsv1.AssetGroupMember{
					{AssetId: strPtr("eth"), Weight: float64Ptr(1.0)},
					{AssetId: strPtr("weth"), Weight: float64Ptr(1.0)},
				},
			},
			wantErr: false,
		},
		{
			name: "missing group name",
			group: &assetsv1.AssetGroup{
				GroupId: strPtr("eth_group"),
				Members: []*assetsv1.AssetGroupMember{
					{AssetId: strPtr("eth"), Weight: float64Ptr(1.0)},
				},
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "nonexistent member asset",
			group: &assetsv1.AssetGroup{
				GroupId: strPtr("eth_group"),
				Name:    strPtr("All ETH Variants"),
				Members: []*assetsv1.AssetGroupMember{
					{AssetId: strPtr("nonexistent"), Weight: float64Ptr(1.0)},
				},
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.CreateAssetGroup(ctx, tt.group)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func assetTypePtr(t assetsv1.AssetType) *assetsv1.AssetType {
	return &t
}
