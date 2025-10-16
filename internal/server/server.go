package server

import (
	"context"
	"strconv"

	"github.com/Combine-Capital/cqar/internal/manager"
	"github.com/Combine-Capital/cqar/internal/repository"
	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AssetRegistryServer implements the CQC AssetRegistry gRPC service interface.
// It embeds UnimplementedAssetRegistryServer for forward compatibility and holds
// manager dependencies for business logic delegation.
type AssetRegistryServer struct {
	servicesv1.UnimplementedAssetRegistryServer
	assetManager   *manager.AssetManager
	symbolManager  *manager.SymbolManager
	venueManager   *manager.VenueManager
	qualityManager *manager.QualityManager
	repo           repository.Repository
}

// NewAssetRegistryServer creates a new AssetRegistryServer with the given dependencies.
func NewAssetRegistryServer(
	assetManager *manager.AssetManager,
	symbolManager *manager.SymbolManager,
	venueManager *manager.VenueManager,
	qualityManager *manager.QualityManager,
	repo repository.Repository,
) *AssetRegistryServer {
	return &AssetRegistryServer{
		assetManager:   assetManager,
		symbolManager:  symbolManager,
		venueManager:   venueManager,
		qualityManager: qualityManager,
		repo:           repo,
	}
}

// Helper functions for pointer field handling

// derefString safely dereferences a *string, returning empty string if nil
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefInt32 safely dereferences a *int32, returning 0 if nil
func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

// ptrBool creates a pointer to a bool value
func ptrBool(b bool) *bool {
	return &b
}

// ptrString creates a pointer to a string value
func ptrString(s string) *string {
	return &s
}

// Core Asset Methods (Commit 9a)

// CreateAsset creates a new canonical asset in the registry.
// Request fields are validated and mapped to Asset domain object before manager call.
func (s *AssetRegistryServer) CreateAsset(ctx context.Context, req *servicesv1.CreateAssetRequest) (*servicesv1.CreateAssetResponse, error) {
	// Validate required fields at gRPC layer
	if req.Symbol == nil || *req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}
	if req.Name == nil || *req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.AssetType == nil {
		return nil, status.Error(codes.InvalidArgument, "asset_type is required")
	}

	// Construct Asset domain object from request fields
	asset := &assetsv1.Asset{
		Symbol:      req.Symbol,
		Name:        req.Name,
		AssetType:   req.AssetType,
		Category:    req.Category,
		Description: req.Description,
		LogoUrl:     req.LogoUrl,
		WebsiteUrl:  req.WebsiteUrl,
	}

	// Manager validates and creates asset (modifies asset in-place with generated ID)
	if err := s.assetManager.CreateAsset(ctx, asset); err != nil {
		return nil, err // Manager already wrapped error with status.Error
	}

	return &servicesv1.CreateAssetResponse{Asset: asset}, nil
}

// GetAsset retrieves an asset by ID.
func (s *AssetRegistryServer) GetAsset(ctx context.Context, req *servicesv1.GetAssetRequest) (*servicesv1.GetAssetResponse, error) {
	// Validate required fields
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Manager handles retrieval and error wrapping
	asset, err := s.assetManager.GetAsset(ctx, *req.AssetId)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.GetAssetResponse{Asset: asset}, nil
}

// UpdateAsset updates an existing asset.
// All fields are optional except asset_id - nil fields are not updated.
func (s *AssetRegistryServer) UpdateAsset(ctx context.Context, req *servicesv1.UpdateAssetRequest) (*servicesv1.UpdateAssetResponse, error) {
	// Validate required fields
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Construct asset from request fields
	// AssetId is required, other fields are optional for partial updates
	asset := &assetsv1.Asset{
		AssetId:     req.AssetId,
		Symbol:      req.Symbol,
		Name:        req.Name,
		AssetType:   req.AssetType,
		Category:    req.Category,
		Description: req.Description,
		LogoUrl:     req.LogoUrl,
		WebsiteUrl:  req.WebsiteUrl,
	}

	// Manager validates and updates
	if err := s.assetManager.UpdateAsset(ctx, asset); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.UpdateAssetResponse{Asset: asset}, nil
}

// DeleteAsset soft-deletes an asset by ID.
func (s *AssetRegistryServer) DeleteAsset(ctx context.Context, req *servicesv1.DeleteAssetRequest) (*servicesv1.DeleteAssetResponse, error) {
	// Validate required fields
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Manager handles deletion and error wrapping
	if err := s.assetManager.DeleteAsset(ctx, *req.AssetId); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.DeleteAssetResponse{Success: ptrBool(true)}, nil
}

// ListAssets lists assets with optional filtering and pagination.
// Supports filtering by asset_type and category.
func (s *AssetRegistryServer) ListAssets(ctx context.Context, req *servicesv1.ListAssetsRequest) (*servicesv1.ListAssetsResponse, error) {
	// Build filter from request parameters
	filter := &repository.AssetFilter{}

	// Handle pagination
	if req.PageSize != nil && *req.PageSize > 0 {
		filter.Limit = int(*req.PageSize)
	} else {
		filter.Limit = 50 // default page size
	}

	// Parse page token as offset
	if req.PageToken != nil && *req.PageToken != "" {
		if offset, err := strconv.Atoi(*req.PageToken); err == nil {
			filter.Offset = offset
		}
	}

	// Apply optional filters
	if req.AssetType != nil {
		// Convert AssetType enum to string for filter
		typeStr := req.AssetType.String()
		filter.Type = &typeStr
	}
	if req.Category != nil && *req.Category != "" {
		filter.Category = req.Category
	}

	// Retrieve assets from manager
	assets, err := s.assetManager.ListAssets(ctx, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	// Calculate next page token if there might be more results
	var nextPageToken *string
	if len(assets) == filter.Limit {
		// More results likely exist
		nextOffset := filter.Offset + filter.Limit
		token := strconv.Itoa(nextOffset)
		nextPageToken = &token
	}

	return &servicesv1.ListAssetsResponse{
		Assets:        assets,
		NextPageToken: nextPageToken,
	}, nil
}

// SearchAssets searches assets by query string with optional filtering.
// Query is matched against asset symbol, name, and description.
func (s *AssetRegistryServer) SearchAssets(ctx context.Context, req *servicesv1.SearchAssetsRequest) (*servicesv1.SearchAssetsResponse, error) {
	// Validate required fields
	if req.Query == nil || *req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	// Build filter - SearchAssetsRequest may not have pagination fields
	// Use default limit
	filter := &repository.AssetFilter{
		Limit: 50, // default limit for search
	}

	// Manager handles search with validation and error wrapping
	assets, err := s.assetManager.SearchAssets(ctx, *req.Query, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.SearchAssetsResponse{Assets: assets}, nil
}

// Asset Deployment Methods (Commit 9b)

// CreateAssetDeployment creates a new asset deployment on a specific chain.
// Handles conversion from request fields to AssetDeployment domain object.
func (s *AssetRegistryServer) CreateAssetDeployment(ctx context.Context, req *servicesv1.CreateAssetDeploymentRequest) (*servicesv1.CreateAssetDeploymentResponse, error) {
	// Validate required fields
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}
	if req.ChainId == nil || *req.ChainId == "" {
		return nil, status.Error(codes.InvalidArgument, "chain_id is required")
	}

	// Determine address based on is_native flag and contract_address
	var address *string
	if req.IsNative != nil && *req.IsNative {
		// Native token uses "native" as address
		nativeAddr := "native"
		address = &nativeAddr
	} else if req.ContractAddress != nil && *req.ContractAddress != "" {
		address = req.ContractAddress
	} else {
		return nil, status.Error(codes.InvalidArgument, "contract_address is required for non-native deployments")
	}

	// Construct AssetDeployment domain object
	deployment := &assetsv1.AssetDeployment{
		AssetId:  req.AssetId,
		ChainId:  req.ChainId,
		Address:  address,
		Decimals: req.Decimals,
	}

	// Manager validates and creates deployment
	if err := s.assetManager.CreateAssetDeployment(ctx, deployment); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.CreateAssetDeploymentResponse{Deployment: deployment}, nil
}

// GetAssetDeployment retrieves a deployment by its ID.
func (s *AssetRegistryServer) GetAssetDeployment(ctx context.Context, req *servicesv1.GetAssetDeploymentRequest) (*servicesv1.GetAssetDeploymentResponse, error) {
	// Validate required fields
	if req.DeploymentId == nil || *req.DeploymentId == "" {
		return nil, status.Error(codes.InvalidArgument, "deployment_id is required")
	}

	// Manager handles retrieval and error wrapping
	deployment, err := s.assetManager.GetAssetDeployment(ctx, *req.DeploymentId)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.GetAssetDeploymentResponse{Deployment: deployment}, nil
}

// ListAssetDeployments lists deployments with optional filtering by asset_id.
func (s *AssetRegistryServer) ListAssetDeployments(ctx context.Context, req *servicesv1.ListAssetDeploymentsRequest) (*servicesv1.ListAssetDeploymentsResponse, error) {
	// Build DeploymentFilter from request
	filter := &repository.DeploymentFilter{}

	// Filter by asset_id if provided
	if req.AssetId != nil && *req.AssetId != "" {
		filter.AssetID = req.AssetId
	}

	// Set default limit
	filter.Limit = 100 // reasonable default for deployments

	// Manager handles listing and error wrapping
	deployments, err := s.assetManager.ListAssetDeployments(ctx, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.ListAssetDeploymentsResponse{Deployments: deployments}, nil
}

// Asset Relationship Methods (Commit 9b)

// CreateAssetRelationship creates a relationship between two assets.
// Supports various relationship types like WRAPS, DERIVES_FROM, etc.
func (s *AssetRegistryServer) CreateAssetRelationship(ctx context.Context, req *servicesv1.CreateAssetRelationshipRequest) (*servicesv1.CreateAssetRelationshipResponse, error) {
	// Validate required fields
	if req.SourceAssetId == nil || *req.SourceAssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "source_asset_id is required")
	}
	if req.TargetAssetId == nil || *req.TargetAssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "target_asset_id is required")
	}
	if req.RelationshipType == nil {
		return nil, status.Error(codes.InvalidArgument, "relationship_type is required")
	}

	// Construct AssetRelationship domain object
	// Note: Request uses SourceAssetId/TargetAssetId, domain uses FromAssetId/ToAssetId
	relationship := &assetsv1.AssetRelationship{
		FromAssetId:      req.SourceAssetId,
		ToAssetId:        req.TargetAssetId,
		RelationshipType: req.RelationshipType,
		Description:      req.Description,
	}

	// Manager validates and creates relationship (includes cycle detection)
	if err := s.assetManager.CreateAssetRelationship(ctx, relationship); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.CreateAssetRelationshipResponse{Relationship: relationship}, nil
}

// ListAssetRelationships lists relationships with optional filtering by asset_id.
func (s *AssetRegistryServer) ListAssetRelationships(ctx context.Context, req *servicesv1.ListAssetRelationshipsRequest) (*servicesv1.ListAssetRelationshipsResponse, error) {
	// Build RelationshipFilter from request
	filter := &repository.RelationshipFilter{}

	// Filter by asset_id (can match either FromAssetID or ToAssetID)
	// For now, filter by FromAssetID - caller can specify which direction they want
	if req.AssetId != nil && *req.AssetId != "" {
		filter.FromAssetID = req.AssetId
	}

	// Set default limit
	filter.Limit = 100 // reasonable default for relationships

	// Manager handles listing and error wrapping
	relationships, err := s.assetManager.ListAssetRelationships(ctx, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.ListAssetRelationshipsResponse{Relationships: relationships}, nil
}

// Asset Group Methods (Commit 9b)

// CreateAssetGroup creates a new asset group.
// Groups are used to organize assets by category, index, or portfolio.
func (s *AssetRegistryServer) CreateAssetGroup(ctx context.Context, req *servicesv1.CreateAssetGroupRequest) (*servicesv1.CreateAssetGroupResponse, error) {
	// Validate required fields
	if req.Name == nil || *req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	// Construct AssetGroup domain object
	// Note: GroupType is in request but not in domain object (may be stored in metadata)
	group := &assetsv1.AssetGroup{
		Name:        req.Name,
		Description: req.Description,
	}

	// Manager validates and creates group
	if err := s.assetManager.CreateAssetGroup(ctx, group); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.CreateAssetGroupResponse{Group: group}, nil
}

// GetAssetGroup retrieves an asset group by ID.
func (s *AssetRegistryServer) GetAssetGroup(ctx context.Context, req *servicesv1.GetAssetGroupRequest) (*servicesv1.GetAssetGroupResponse, error) {
	// Validate required fields
	if req.GroupId == nil || *req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}

	// Manager handles retrieval and error wrapping
	group, err := s.assetManager.GetAssetGroup(ctx, *req.GroupId)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.GetAssetGroupResponse{Group: group}, nil
}

// AddAssetToGroup adds an asset to an existing group with optional weight.
func (s *AssetRegistryServer) AddAssetToGroup(ctx context.Context, req *servicesv1.AddAssetToGroupRequest) (*servicesv1.AddAssetToGroupResponse, error) {
	// Validate required fields
	if req.GroupId == nil || *req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Extract weight (default to 1.0 if not provided)
	weight := 1.0
	if req.Weight != nil {
		weight = *req.Weight
	}

	// Manager validates and adds asset to group
	if err := s.assetManager.AddAssetToGroup(ctx, *req.GroupId, *req.AssetId, weight); err != nil {
		return nil, err // Manager already wrapped error
	}

	// Construct the member response
	// Note: Manager doesn't return the member, so we construct it from input
	member := &assetsv1.AssetGroupMember{
		GroupId: req.GroupId,
		AssetId: req.AssetId,
		Weight:  req.Weight,
	}

	return &servicesv1.AddAssetToGroupResponse{Member: member}, nil
}

// RemoveAssetFromGroup removes an asset from a group.
func (s *AssetRegistryServer) RemoveAssetFromGroup(ctx context.Context, req *servicesv1.RemoveAssetFromGroupRequest) (*servicesv1.RemoveAssetFromGroupResponse, error) {
	// Validate required fields
	if req.GroupId == nil || *req.GroupId == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Manager validates and removes asset from group
	if err := s.assetManager.RemoveAssetFromGroup(ctx, *req.GroupId, *req.AssetId); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.RemoveAssetFromGroupResponse{Success: ptrBool(true)}, nil
}

// Quality Flag Methods (Commit 9c)

// RaiseQualityFlag creates a quality flag for an asset.
// Flags indicate data quality issues, security concerns, or other problems.
func (s *AssetRegistryServer) RaiseQualityFlag(ctx context.Context, req *servicesv1.RaiseQualityFlagRequest) (*servicesv1.RaiseQualityFlagResponse, error) {
	// Validate required fields
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}
	if req.FlagType == nil {
		return nil, status.Error(codes.InvalidArgument, "flag_type is required")
	}
	if req.Severity == nil {
		return nil, status.Error(codes.InvalidArgument, "severity is required")
	}

	// Construct AssetQualityFlag domain object
	// Note: Request has Description and RaisedBy fields
	// Domain object uses Reason for description, Source for raised_by context
	flag := &assetsv1.AssetQualityFlag{
		AssetId:  req.AssetId,
		FlagType: req.FlagType,
		Severity: req.Severity,
		Reason:   req.Description, // Map description to reason
		Source:   req.RaisedBy,    // Map raised_by to source
	}

	// Manager validates and raises flag
	if err := s.qualityManager.RaiseQualityFlag(ctx, flag); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.RaiseQualityFlagResponse{Flag: flag}, nil
}

// ResolveQualityFlag marks a quality flag as resolved.
// This indicates the issue has been addressed or confirmed as false positive.
func (s *AssetRegistryServer) ResolveQualityFlag(ctx context.Context, req *servicesv1.ResolveQualityFlagRequest) (*servicesv1.ResolveQualityFlagResponse, error) {
	// Validate required fields
	if req.FlagId == nil || *req.FlagId == "" {
		return nil, status.Error(codes.InvalidArgument, "flag_id is required")
	}

	// Extract optional fields
	resolvedBy := derefString(req.ResolvedBy)
	resolutionNotes := derefString(req.ResolutionNotes)

	// Manager validates and resolves flag
	if err := s.qualityManager.ResolveQualityFlag(ctx, *req.FlagId, resolvedBy, resolutionNotes); err != nil {
		return nil, err // Manager already wrapped error
	}

	// Retrieve the resolved flag to return
	flag, err := s.repo.GetQualityFlag(ctx, *req.FlagId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to retrieve resolved flag")
	}

	return &servicesv1.ResolveQualityFlagResponse{Flag: flag}, nil
}

// ListQualityFlags lists all quality flags with optional filtering.
// Supports filtering by asset_id and minimum severity level.
func (s *AssetRegistryServer) ListQualityFlags(ctx context.Context, req *servicesv1.ListQualityFlagsRequest) (*servicesv1.ListQualityFlagsResponse, error) {
	// Build filter from request
	filter := &repository.QualityFlagFilter{}

	// Filter by asset_id if provided
	if req.AssetId != nil && *req.AssetId != "" {
		filter.AssetID = req.AssetId
	}

	// Filter by minimum severity if provided
	if req.MinSeverity != nil {
		severityStr := req.MinSeverity.String()
		filter.Severity = &severityStr
	}

	// Filter by resolved status - default is active only (not include_resolved)
	if req.IncludeResolved == nil || !*req.IncludeResolved {
		filter.ActiveOnly = true
	}

	// Set default limit
	filter.Limit = 100 // reasonable default for flags

	// Manager handles listing and error wrapping
	flags, err := s.qualityManager.ListQualityFlags(ctx, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.ListQualityFlagsResponse{Flags: flags}, nil
}

// Chain Methods (Commit 9c)

// CreateChain registers a new blockchain network.
// Chains are required for tracking asset deployments and venue protocols.
func (s *AssetRegistryServer) CreateChain(ctx context.Context, req *servicesv1.CreateChainRequest) (*servicesv1.CreateChainResponse, error) {
	// Validate required fields
	if req.Name == nil || *req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.ChainType == nil || *req.ChainType == "" {
		return nil, status.Error(codes.InvalidArgument, "chain_type is required")
	}

	// Construct Chain domain object
	// Note: Request has ChainId as int64 (network_id) and separate ChainType string
	// Domain uses ChainId as string identifier and NetworkId as int64
	// We'll use ChainType as ChainId and req.ChainId as NetworkId
	chain := &assetsv1.Chain{
		ChainId:     req.ChainType, // Use chain_type as the identifier
		ChainName:   req.Name,
		ChainType:   req.ChainType,
		NetworkId:   req.ChainId, // Network ID (e.g., 1 for Ethereum)
		ExplorerUrl: req.BlockExplorerUrl,
	}

	// Repository validates and creates chain
	if err := s.repo.CreateChain(ctx, chain); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create chain: %v", err)
	}

	return &servicesv1.CreateChainResponse{Chain: chain}, nil
}

// GetChain retrieves a specific chain by ID.
func (s *AssetRegistryServer) GetChain(ctx context.Context, req *servicesv1.GetChainRequest) (*servicesv1.GetChainResponse, error) {
	// Validate required fields
	if req.ChainId == nil || *req.ChainId == "" {
		return nil, status.Error(codes.InvalidArgument, "chain_id is required")
	}

	// Repository handles retrieval
	chain, err := s.repo.GetChain(ctx, *req.ChainId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "chain not found: %v", err)
	}

	return &servicesv1.GetChainResponse{Chain: chain}, nil
}

// ListChains lists all registered blockchain networks with optional filtering.
func (s *AssetRegistryServer) ListChains(ctx context.Context, req *servicesv1.ListChainsRequest) (*servicesv1.ListChainsResponse, error) {
	// Build filter from request
	filter := &repository.ChainFilter{
		Limit: 100, // default limit
	}

	// Apply optional filters
	if req.ChainType != nil && *req.ChainType != "" {
		filter.ChainType = req.ChainType
	}

	// Repository handles listing
	chains, err := s.repo.ListChains(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list chains: %v", err)
	}

	return &servicesv1.ListChainsResponse{Chains: chains}, nil
}

// Symbol Methods (Commit 9c)

// CreateSymbol creates a new trading symbol/market.
// Symbols represent trading pairs (e.g., BTC/USDT) with market specifications.
func (s *AssetRegistryServer) CreateSymbol(ctx context.Context, req *servicesv1.CreateSymbolRequest) (*servicesv1.CreateSymbolResponse, error) {
	// Validate required fields
	if req.Symbol == nil || *req.Symbol == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol is required")
	}
	if req.BaseAssetId == nil || *req.BaseAssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "base_asset_id is required")
	}
	if req.QuoteAssetId == nil || *req.QuoteAssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "quote_asset_id is required")
	}

	// Construct Symbol domain object
	symbol := &marketsv1.Symbol{
		Symbol:            req.Symbol,
		SymbolType:        req.SymbolType,
		BaseAssetId:       req.BaseAssetId,
		QuoteAssetId:      req.QuoteAssetId,
		SettlementAssetId: req.SettlementAssetId,
		TickSize:          req.TickSize,
		LotSize:           req.LotSize,
	}

	// Manager validates and creates symbol
	if err := s.symbolManager.CreateSymbol(ctx, symbol); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.CreateSymbolResponse{Symbol: symbol}, nil
}

// GetSymbol retrieves a specific symbol by ID.
func (s *AssetRegistryServer) GetSymbol(ctx context.Context, req *servicesv1.GetSymbolRequest) (*servicesv1.GetSymbolResponse, error) {
	// Validate required fields
	if req.SymbolId == nil || *req.SymbolId == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol_id is required")
	}

	// Manager handles retrieval and error wrapping
	symbol, err := s.symbolManager.GetSymbol(ctx, *req.SymbolId)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.GetSymbolResponse{Symbol: symbol}, nil
}

// UpdateSymbol updates an existing symbol's metadata.
func (s *AssetRegistryServer) UpdateSymbol(ctx context.Context, req *servicesv1.UpdateSymbolRequest) (*servicesv1.UpdateSymbolResponse, error) {
	// Validate required fields
	if req.SymbolId == nil || *req.SymbolId == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol_id is required")
	}

	// Construct symbol from request fields
	// SymbolId is required, other fields are optional for partial updates
	symbol := &marketsv1.Symbol{
		SymbolId:   req.SymbolId,
		Symbol:     req.Symbol,
		SymbolType: req.SymbolType,
		TickSize:   req.TickSize,
		LotSize:    req.LotSize,
		IsActive:   req.IsActive,
	}

	// Manager validates and updates
	if err := s.symbolManager.UpdateSymbol(ctx, symbol); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.UpdateSymbolResponse{Symbol: symbol}, nil
}

// DeleteSymbol soft-deletes a symbol by ID.
func (s *AssetRegistryServer) DeleteSymbol(ctx context.Context, req *servicesv1.DeleteSymbolRequest) (*servicesv1.DeleteSymbolResponse, error) {
	// Validate required fields
	if req.SymbolId == nil || *req.SymbolId == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol_id is required")
	}

	// Manager handles deletion and error wrapping
	if err := s.symbolManager.DeleteSymbol(ctx, *req.SymbolId); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.DeleteSymbolResponse{Success: ptrBool(true)}, nil
}

// ListSymbols retrieves a paginated list of symbols with optional filtering.
func (s *AssetRegistryServer) ListSymbols(ctx context.Context, req *servicesv1.ListSymbolsRequest) (*servicesv1.ListSymbolsResponse, error) {
	// Build filter from request parameters
	filter := &repository.SymbolFilter{}

	// Handle pagination
	if req.PageSize != nil && *req.PageSize > 0 {
		filter.Limit = int(*req.PageSize)
	} else {
		filter.Limit = 50 // default page size
	}

	// Parse page token as offset
	if req.PageToken != nil && *req.PageToken != "" {
		if offset, err := strconv.Atoi(*req.PageToken); err == nil {
			filter.Offset = offset
		}
	}

	// Apply optional filters
	if req.SymbolType != nil {
		typeStr := req.SymbolType.String()
		filter.SymbolType = &typeStr
	}
	if req.BaseAssetId != nil && *req.BaseAssetId != "" {
		filter.BaseAssetID = req.BaseAssetId
	}
	if req.QuoteAssetId != nil && *req.QuoteAssetId != "" {
		filter.QuoteAssetID = req.QuoteAssetId
	}

	// Retrieve symbols from manager
	symbols, err := s.symbolManager.ListSymbols(ctx, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	// Calculate next page token if there might be more results
	var nextPageToken *string
	if len(symbols) == filter.Limit {
		// More results likely exist
		nextOffset := filter.Offset + filter.Limit
		token := strconv.Itoa(nextOffset)
		nextPageToken = &token
	}

	return &servicesv1.ListSymbolsResponse{
		Symbols:       symbols,
		NextPageToken: nextPageToken,
	}, nil
}

// SearchSymbols searches for symbols by query string.
func (s *AssetRegistryServer) SearchSymbols(ctx context.Context, req *servicesv1.SearchSymbolsRequest) (*servicesv1.SearchSymbolsResponse, error) {
	// Validate required fields
	if req.Query == nil || *req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}

	// Build filter - use default limit
	filter := &repository.SymbolFilter{
		Limit: 50, // default limit for search
	}

	// Manager handles search with validation and error wrapping
	symbols, err := s.symbolManager.SearchSymbols(ctx, *req.Query, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.SearchSymbolsResponse{Symbols: symbols}, nil
}

// CreateSymbolIdentifier adds an external identifier mapping for a symbol.
// Currently unimplemented - placeholder for future integration.
func (s *AssetRegistryServer) CreateSymbolIdentifier(ctx context.Context, req *servicesv1.CreateSymbolIdentifierRequest) (*servicesv1.CreateSymbolIdentifierResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateSymbolIdentifier not yet implemented")
}

// GetSymbolIdentifier retrieves a symbol identifier mapping.
// Currently unimplemented - placeholder for future integration.
func (s *AssetRegistryServer) GetSymbolIdentifier(ctx context.Context, req *servicesv1.GetSymbolIdentifierRequest) (*servicesv1.GetSymbolIdentifierResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetSymbolIdentifier not yet implemented")
}

// ListSymbolIdentifiers lists all identifier mappings for a symbol.
// Currently unimplemented - placeholder for future integration.
func (s *AssetRegistryServer) ListSymbolIdentifiers(ctx context.Context, req *servicesv1.ListSymbolIdentifiersRequest) (*servicesv1.ListSymbolIdentifiersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListSymbolIdentifiers not yet implemented")
}

// Asset Identifier Methods (Commit 9c - Stubs)

// CreateAssetIdentifier adds an external identifier mapping for an asset.
// Currently unimplemented - placeholder for future integration.
func (s *AssetRegistryServer) CreateAssetIdentifier(ctx context.Context, req *servicesv1.CreateAssetIdentifierRequest) (*servicesv1.CreateAssetIdentifierResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateAssetIdentifier not yet implemented")
}

// GetAssetIdentifier retrieves an identifier mapping.
// Currently unimplemented - placeholder for future integration.
func (s *AssetRegistryServer) GetAssetIdentifier(ctx context.Context, req *servicesv1.GetAssetIdentifierRequest) (*servicesv1.GetAssetIdentifierResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetAssetIdentifier not yet implemented")
}

// ListAssetIdentifiers lists all identifier mappings for an asset.
// Currently unimplemented - placeholder for future integration.
func (s *AssetRegistryServer) ListAssetIdentifiers(ctx context.Context, req *servicesv1.ListAssetIdentifiersRequest) (*servicesv1.ListAssetIdentifiersResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListAssetIdentifiers not yet implemented")
}

// Venue Methods (Commit 9c)

// CreateVenue registers a new trading venue.
// Venues can be centralized exchanges, DEXs, DEX aggregators, bridges, or lending protocols.
func (s *AssetRegistryServer) CreateVenue(ctx context.Context, req *servicesv1.CreateVenueRequest) (*servicesv1.CreateVenueResponse, error) {
	// Validate required fields
	if req.Name == nil || *req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	if req.VenueType == nil {
		return nil, status.Error(codes.InvalidArgument, "venue_type is required")
	}

	// Construct Venue domain object
	// Note: VenueId is generated by manager, not provided in request
	venue := &venuesv1.Venue{
		Name:        req.Name,
		VenueType:   req.VenueType,
		ChainId:     req.ChainId,
		WebsiteUrl:  req.WebsiteUrl,
		ApiEndpoint: req.ApiEndpoint,
	}

	// Manager validates and creates venue
	if err := s.venueManager.CreateVenue(ctx, venue); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.CreateVenueResponse{Venue: venue}, nil
}

// GetVenue retrieves a specific venue by ID.
func (s *AssetRegistryServer) GetVenue(ctx context.Context, req *servicesv1.GetVenueRequest) (*servicesv1.GetVenueResponse, error) {
	// Validate required fields
	if req.VenueId == nil || *req.VenueId == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}

	// Manager handles retrieval and error wrapping
	venue, err := s.venueManager.GetVenue(ctx, *req.VenueId)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.GetVenueResponse{Venue: venue}, nil
}

// ListVenues lists all registered trading venues with optional filtering.
func (s *AssetRegistryServer) ListVenues(ctx context.Context, req *servicesv1.ListVenuesRequest) (*servicesv1.ListVenuesResponse, error) {
	// Build filter from request
	filter := &repository.VenueFilter{
		Limit: 100, // default limit
	}

	// Apply optional filters
	if req.VenueType != nil {
		typeStr := req.VenueType.String()
		filter.VenueType = &typeStr
	}
	if req.ChainId != nil && *req.ChainId != "" {
		filter.ChainID = req.ChainId
	}

	// Manager handles listing and error wrapping
	venues, err := s.venueManager.ListVenues(ctx, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.ListVenuesResponse{Venues: venues}, nil
}

// CreateVenueAsset registers asset availability on a venue.
// Maps which assets are available for trading, deposit, or withdrawal on each venue.
func (s *AssetRegistryServer) CreateVenueAsset(ctx context.Context, req *servicesv1.CreateVenueAssetRequest) (*servicesv1.CreateVenueAssetResponse, error) {
	// Validate required fields
	if req.VenueId == nil || *req.VenueId == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}
	if req.VenueAssetSymbol == nil || *req.VenueAssetSymbol == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_asset_symbol is required")
	}

	// Construct VenueAsset domain object
	venueAsset := &venuesv1.VenueAsset{
		VenueId:          req.VenueId,
		AssetId:          req.AssetId,
		VenueAssetSymbol: req.VenueAssetSymbol,
		DeploymentId:     req.DeploymentId,
		DepositEnabled:   req.DepositEnabled,
		WithdrawEnabled:  req.WithdrawEnabled,
		TradingEnabled:   req.TradingEnabled,
	}

	// Manager validates and creates venue asset mapping
	if err := s.venueManager.CreateVenueAsset(ctx, venueAsset); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.CreateVenueAssetResponse{VenueAsset: venueAsset}, nil
}

// GetVenueAsset retrieves venue asset availability information.
func (s *AssetRegistryServer) GetVenueAsset(ctx context.Context, req *servicesv1.GetVenueAssetRequest) (*servicesv1.GetVenueAssetResponse, error) {
	// Validate required fields
	if req.VenueId == nil || *req.VenueId == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}
	if req.AssetId == nil || *req.AssetId == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Manager handles retrieval and error wrapping
	venueAsset, err := s.venueManager.GetVenueAsset(ctx, *req.VenueId, *req.AssetId)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.GetVenueAssetResponse{VenueAsset: venueAsset}, nil
}

// ListVenueAssets lists all assets available on a venue or all venues for an asset.
// Supports bidirectional queries: "which assets on Binance?" or "which venues trade BTC?"
func (s *AssetRegistryServer) ListVenueAssets(ctx context.Context, req *servicesv1.ListVenueAssetsRequest) (*servicesv1.ListVenueAssetsResponse, error) {
	// Build filter from request
	filter := &repository.VenueAssetFilter{
		Limit: 100, // default limit
	}

	// Filter by venue_id OR asset_id (or both)
	if req.VenueId != nil && *req.VenueId != "" {
		filter.VenueID = req.VenueId
	}
	if req.AssetId != nil && *req.AssetId != "" {
		filter.AssetID = req.AssetId
	}

	// Manager handles listing and error wrapping
	venueAssets, err := s.venueManager.ListVenueAssets(ctx, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.ListVenueAssetsResponse{VenueAssets: venueAssets}, nil
}

// CreateVenueSymbol maps a venue's trading symbol to a canonical symbol.
// Example: Binance "BTCUSDT" → canonical BTC/USDT spot symbol
func (s *AssetRegistryServer) CreateVenueSymbol(ctx context.Context, req *servicesv1.CreateVenueSymbolRequest) (*servicesv1.CreateVenueSymbolResponse, error) {
	// Validate required fields
	if req.VenueId == nil || *req.VenueId == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}
	if req.SymbolId == nil || *req.SymbolId == "" {
		return nil, status.Error(codes.InvalidArgument, "symbol_id is required")
	}
	if req.VenueSymbol == nil || *req.VenueSymbol == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_symbol is required")
	}

	// Construct VenueSymbol domain object
	venueSymbol := &venuesv1.VenueSymbol{
		VenueId:     req.VenueId,
		SymbolId:    req.SymbolId,
		VenueSymbol: req.VenueSymbol,
		MakerFee:    req.MakerFee,
		TakerFee:    req.TakerFee,
	}

	// Manager validates and creates venue symbol mapping
	if err := s.venueManager.CreateVenueSymbol(ctx, venueSymbol); err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.CreateVenueSymbolResponse{VenueSymbol: venueSymbol}, nil
}

// GetVenueSymbol retrieves a venue symbol mapping.
// Primary use case: cqmd needs to resolve "BTCUSDT" on Binance to canonical symbol.
func (s *AssetRegistryServer) GetVenueSymbol(ctx context.Context, req *servicesv1.GetVenueSymbolRequest) (*servicesv1.GetVenueSymbolResponse, error) {
	// Validate required fields
	if req.VenueId == nil || *req.VenueId == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}
	if req.VenueSymbol == nil || *req.VenueSymbol == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_symbol is required")
	}

	// Manager handles retrieval with enriched symbol data
	// Note: Manager returns both VenueSymbol and canonical Symbol, but response only includes VenueSymbol
	venueSymbol, _, err := s.venueManager.GetVenueSymbol(ctx, *req.VenueId, *req.VenueSymbol)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.GetVenueSymbolResponse{
		VenueSymbol: venueSymbol,
	}, nil
}

// ListVenueSymbols lists all symbol mappings for a venue or symbol.
// Supports queries like "all symbols on Binance" or "all venues trading BTC/USDT"
func (s *AssetRegistryServer) ListVenueSymbols(ctx context.Context, req *servicesv1.ListVenueSymbolsRequest) (*servicesv1.ListVenueSymbolsResponse, error) {
	// Build filter from request
	filter := &repository.VenueSymbolFilter{
		Limit: 100, // default limit
	}

	// Filter by venue_id OR symbol_id (or both)
	if req.VenueId != nil && *req.VenueId != "" {
		filter.VenueID = req.VenueId
	}
	if req.SymbolId != nil && *req.SymbolId != "" {
		filter.SymbolID = req.SymbolId
	}

	// Manager handles listing and error wrapping
	venueSymbols, err := s.venueManager.ListVenueSymbols(ctx, filter)
	if err != nil {
		return nil, err // Manager already wrapped error
	}

	return &servicesv1.ListVenueSymbolsResponse{VenueSymbols: venueSymbols}, nil
}
