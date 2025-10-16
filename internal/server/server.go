package server

import (
	"context"
	"strconv"

	"github.com/Combine-Capital/cqar/internal/manager"
	"github.com/Combine-Capital/cqar/internal/repository"
	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
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
