package repository

import (
	"context"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
)

// Repository defines the interface for all data access operations in CQAR.
// It abstracts database operations and provides a cache-aside pattern.
// All methods return CQC protobuf types for consistency across the platform.
type Repository interface {
	// Asset operations
	CreateAsset(ctx context.Context, asset *assetsv1.Asset) error
	GetAsset(ctx context.Context, id string) (*assetsv1.Asset, error)
	UpdateAsset(ctx context.Context, asset *assetsv1.Asset) error
	DeleteAsset(ctx context.Context, id string) error
	ListAssets(ctx context.Context, filter *AssetFilter) ([]*assetsv1.Asset, error)
	SearchAssets(ctx context.Context, query string, filter *AssetFilter) ([]*assetsv1.Asset, error)

	// AssetDeployment operations
	CreateAssetDeployment(ctx context.Context, deployment *assetsv1.AssetDeployment) error
	GetAssetDeployment(ctx context.Context, id string) (*assetsv1.AssetDeployment, error)
	ListAssetDeployments(ctx context.Context, filter *DeploymentFilter) ([]*assetsv1.AssetDeployment, error)
	GetAssetDeploymentByChain(ctx context.Context, assetID, chainID string) (*assetsv1.AssetDeployment, error)

	// AssetRelationship operations
	CreateAssetRelationship(ctx context.Context, relationship *assetsv1.AssetRelationship) error
	GetAssetRelationship(ctx context.Context, id string) (*assetsv1.AssetRelationship, error)
	ListAssetRelationships(ctx context.Context, filter *RelationshipFilter) ([]*assetsv1.AssetRelationship, error)

	// QualityFlag operations
	RaiseQualityFlag(ctx context.Context, flag *assetsv1.AssetQualityFlag) error
	ResolveQualityFlag(ctx context.Context, id string, resolvedBy string, resolutionNotes string) error
	GetQualityFlag(ctx context.Context, id string) (*assetsv1.AssetQualityFlag, error)
	ListQualityFlags(ctx context.Context, filter *QualityFlagFilter) ([]*assetsv1.AssetQualityFlag, error)

	// AssetGroup operations
	CreateAssetGroup(ctx context.Context, group *assetsv1.AssetGroup) error
	GetAssetGroup(ctx context.Context, id string) (*assetsv1.AssetGroup, error)
	GetAssetGroupByName(ctx context.Context, name string) (*assetsv1.AssetGroup, error)
	AddAssetToGroup(ctx context.Context, groupID, assetID string, weight float64) error
	RemoveAssetFromGroup(ctx context.Context, groupID, assetID string) error
	ListAssetGroups(ctx context.Context, filter *AssetGroupFilter) ([]*assetsv1.AssetGroup, error)

	// Transaction support
	WithTransaction(ctx context.Context, fn func(repo Repository) error) error

	// Health check
	Ping(ctx context.Context) error
}

// AssetFilter defines filtering options for asset queries
type AssetFilter struct {
	Type      *string // Filter by asset type (CRYPTOCURRENCY, STABLECOIN, etc.)
	Category  *string // Filter by category
	Limit     int     // Maximum number of results
	Offset    int     // Number of results to skip (for pagination)
	SortBy    string  // Field to sort by (default: created_at)
	SortOrder string  // Sort order: ASC or DESC (default: DESC)
}

// DeploymentFilter defines filtering options for deployment queries
type DeploymentFilter struct {
	AssetID     *string // Filter by asset ID
	ChainID     *string // Filter by chain ID
	IsCanonical *bool   // Filter by canonical flag
	Limit       int     // Maximum number of results
	Offset      int     // Number of results to skip
}

// RelationshipFilter defines filtering options for relationship queries
type RelationshipFilter struct {
	FromAssetID      *string // Filter by source asset
	ToAssetID        *string // Filter by target asset
	RelationshipType *string // Filter by relationship type (WRAPS, STAKES, etc.)
	Protocol         *string // Filter by protocol
	Limit            int     // Maximum number of results
	Offset           int     // Number of results to skip
}

// QualityFlagFilter defines filtering options for quality flag queries
type QualityFlagFilter struct {
	AssetID    *string // Filter by asset ID
	FlagType   *string // Filter by flag type (SCAM, RUGPULL, etc.)
	Severity   *string // Filter by severity (INFO, WARNING, CRITICAL)
	ActiveOnly bool    // Only return unresolved flags
	Limit      int     // Maximum number of results
	Offset     int     // Number of results to skip
}

// AssetGroupFilter defines filtering options for asset group queries
type AssetGroupFilter struct {
	Name   *string // Filter by group name
	Limit  int     // Maximum number of results
	Offset int     // Number of results to skip
}
