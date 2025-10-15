package repository

import (
	"context"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
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

	// Symbol operations
	CreateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error
	GetSymbol(ctx context.Context, id string) (*marketsv1.Symbol, error)
	UpdateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error
	DeleteSymbol(ctx context.Context, id string) error
	ListSymbols(ctx context.Context, filter *SymbolFilter) ([]*marketsv1.Symbol, error)
	SearchSymbols(ctx context.Context, query string, filter *SymbolFilter) ([]*marketsv1.Symbol, error)

	// SymbolIdentifier operations
	CreateSymbolIdentifier(ctx context.Context, identifier *marketsv1.SymbolIdentifier) error
	GetSymbolIdentifier(ctx context.Context, id string) (*marketsv1.SymbolIdentifier, error)
	ListSymbolIdentifiers(ctx context.Context, filter *SymbolIdentifierFilter) ([]*marketsv1.SymbolIdentifier, error)

	// Chain operations
	CreateChain(ctx context.Context, chain *assetsv1.Chain) error
	GetChain(ctx context.Context, id string) (*assetsv1.Chain, error)
	ListChains(ctx context.Context, filter *ChainFilter) ([]*assetsv1.Chain, error)

	// Venue operations
	CreateVenue(ctx context.Context, venue *venuesv1.Venue) error
	GetVenue(ctx context.Context, id string) (*venuesv1.Venue, error)
	ListVenues(ctx context.Context, filter *VenueFilter) ([]*venuesv1.Venue, error)

	// VenueAsset operations
	CreateVenueAsset(ctx context.Context, venueAsset *venuesv1.VenueAsset) error
	GetVenueAsset(ctx context.Context, venueID, assetID string) (*venuesv1.VenueAsset, error)
	ListVenueAssets(ctx context.Context, filter *VenueAssetFilter) ([]*venuesv1.VenueAsset, error)

	// VenueSymbol operations
	CreateVenueSymbol(ctx context.Context, venueSymbol *venuesv1.VenueSymbol) error
	GetVenueSymbol(ctx context.Context, venueID, venueSymbol string) (*venuesv1.VenueSymbol, error)
	GetVenueSymbolByID(ctx context.Context, venueID, symbolID string) (*venuesv1.VenueSymbol, error)
	ListVenueSymbols(ctx context.Context, filter *VenueSymbolFilter) ([]*venuesv1.VenueSymbol, error)
	GetVenueSymbolEnriched(ctx context.Context, venueID, venueSymbol string) (*venuesv1.VenueSymbol, *marketsv1.Symbol, error)

	// AssetIdentifier operations
	CreateAssetIdentifier(ctx context.Context, identifier *assetsv1.AssetIdentifier) error
	GetAssetIdentifier(ctx context.Context, id string) (*assetsv1.AssetIdentifier, error)
	ListAssetIdentifiers(ctx context.Context, filter *AssetIdentifierFilter) ([]*assetsv1.AssetIdentifier, error)

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
	AssetID *string // Filter by asset ID
	ChainID *string // Filter by chain ID
	Limit   int     // Maximum number of results
	Offset  int     // Number of results to skip
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

// SymbolFilter defines filtering options for symbol queries
type SymbolFilter struct {
	BaseAssetID       *string // Filter by base asset ID
	QuoteAssetID      *string // Filter by quote asset ID
	SymbolType        *string // Filter by symbol type (SPOT, PERPETUAL, FUTURE, OPTION, MARGIN)
	SettlementAssetID *string // Filter by settlement asset ID
	Limit             int     // Maximum number of results
	Offset            int     // Number of results to skip (for pagination)
	SortBy            string  // Field to sort by (default: created_at)
	SortOrder         string  // Sort order: ASC or DESC (default: DESC)
}

// SymbolIdentifierFilter defines filtering options for symbol identifier queries
type SymbolIdentifierFilter struct {
	SymbolID  *string // Filter by symbol ID
	Source    *string // Filter by source (coingecko, coinmarketcap, etc.)
	IsPrimary *bool   // Filter by primary flag
	Limit     int     // Maximum number of results
	Offset    int     // Number of results to skip
}

// ChainFilter defines filtering options for chain queries
type ChainFilter struct {
	ChainType     *string // Filter by chain type (EVM, COSMOS, SOLANA, etc.)
	NativeAssetID *string // Filter by native asset ID
	Limit         int     // Maximum number of results
	Offset        int     // Number of results to skip
	SortBy        string  // Field to sort by (default: created_at)
	SortOrder     string  // Sort order: ASC or DESC (default: DESC)
}

// AssetIdentifierFilter defines filtering options for asset identifier queries
type AssetIdentifierFilter struct {
	AssetID   *string // Filter by asset ID
	Source    *string // Filter by source (coingecko, coinmarketcap, etc.)
	IsPrimary *bool   // Filter by primary flag
	Limit     int     // Maximum number of results
	Offset    int     // Number of results to skip
}

// VenueFilter defines filtering options for venue queries
type VenueFilter struct {
	VenueType *string // Filter by venue type (CEX, DEX, DEX_AGGREGATOR, BRIDGE, LENDING)
	ChainID   *string // Filter by chain ID (for DEX/Bridge venues)
	IsActive  *bool   // Filter by active status
	Limit     int     // Maximum number of results
	Offset    int     // Number of results to skip
	SortBy    string  // Field to sort by (default: created_at)
	SortOrder string  // Sort order: ASC or DESC (default: DESC)
}

// VenueAssetFilter defines filtering options for venue asset queries
type VenueAssetFilter struct {
	VenueID        *string // Filter by venue ID (e.g., "which assets on Binance?")
	AssetID        *string // Filter by asset ID (e.g., "which venues trade BTC?")
	IsActive       *bool   // Filter by active status
	TradingEnabled *bool   // Filter by trading enabled flag
	Limit          int     // Maximum number of results
	Offset         int     // Number of results to skip
}

// VenueSymbolFilter defines filtering options for venue symbol queries
type VenueSymbolFilter struct {
	VenueID  *string // Filter by venue ID
	SymbolID *string // Filter by canonical symbol ID
	IsActive *bool   // Filter by active status
	Limit    int     // Maximum number of results
	Offset   int     // Number of results to skip
}
