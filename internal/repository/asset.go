package repository

import (
	"context"
	"fmt"
	"time"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateAsset inserts a new asset into the database
func (r *PostgresRepository) CreateAsset(ctx context.Context, asset *assetsv1.Asset) error {
	// Generate ID if not provided
	if asset.AssetId == nil || *asset.AssetId == "" {
		id := uuid.New().String()
		asset.AssetId = &id
	}

	// Set timestamps
	now := timestamppb.Now()
	if asset.CreatedAt == nil {
		asset.CreatedAt = now
	}
	asset.UpdatedAt = now

	query := `
		INSERT INTO assets (
			id, symbol, name, type, category, description, logo_url, website_url,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err := r.exec(ctx, query,
		asset.GetAssetId(),
		asset.GetSymbol(),
		asset.GetName(),
		asset.GetAssetType().String(),
		asset.GetCategory(),
		asset.GetDescription(),
		asset.GetLogoUrl(),
		asset.GetWebsiteUrl(),
		asset.CreatedAt.AsTime(),
		asset.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create asset: %w", err)
	}

	return nil
}

// GetAsset retrieves an asset by ID
func (r *PostgresRepository) GetAsset(ctx context.Context, id string) (*assetsv1.Asset, error) {
	query := `
		SELECT
			id, symbol, name, type, category, description, logo_url, website_url,
			created_at, updated_at
		FROM assets
		WHERE id = $1
	`

	var assetId, symbol, name, assetTypeStr, category, description, logoUrl, websiteUrl string
	var createdAt, updatedAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&assetId,
		&symbol,
		&name,
		&assetTypeStr,
		&category,
		&description,
		&logoUrl,
		&websiteUrl,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}

	// Parse asset type enum
	assetType := parseAssetType(assetTypeStr)

	asset := &assetsv1.Asset{
		AssetId:     &assetId,
		Symbol:      ptrIfNotEmpty(symbol),
		Name:        ptrIfNotEmpty(name),
		AssetType:   &assetType,
		Category:    ptrIfNotEmpty(category),
		Description: ptrIfNotEmpty(description),
		LogoUrl:     ptrIfNotEmpty(logoUrl),
		WebsiteUrl:  ptrIfNotEmpty(websiteUrl),
		CreatedAt:   timestamppb.New(createdAt),
		UpdatedAt:   timestamppb.New(updatedAt),
	}

	return asset, nil
}

// UpdateAsset updates an existing asset
func (r *PostgresRepository) UpdateAsset(ctx context.Context, asset *assetsv1.Asset) error {
	// Update timestamp
	asset.UpdatedAt = timestamppb.Now()

	query := `
		UPDATE assets
		SET
			symbol = $2,
			name = $3,
			type = $4,
			category = $5,
			description = $6,
			logo_url = $7,
			website_url = $8,
			updated_at = $9
		WHERE id = $1
	`

	result, err := r.exec(ctx, query,
		asset.GetAssetId(),
		asset.GetSymbol(),
		asset.GetName(),
		asset.GetAssetType().String(),
		asset.GetCategory(),
		asset.GetDescription(),
		asset.GetLogoUrl(),
		asset.GetWebsiteUrl(),
		asset.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("update asset: %w", err)
	}

	// Check if any rows were affected
	if result.RowsAffected() == 0 {
		return fmt.Errorf("asset not found: %s", asset.GetAssetId())
	}

	return nil
}

// DeleteAsset removes an asset from the database
func (r *PostgresRepository) DeleteAsset(ctx context.Context, id string) error {
	query := `DELETE FROM assets WHERE id = $1`

	result, err := r.exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("asset not found: %s", id)
	}

	return nil
}

// ListAssets retrieves a list of assets with optional filtering and pagination
func (r *PostgresRepository) ListAssets(ctx context.Context, filter *AssetFilter) ([]*assetsv1.Asset, error) {
	// Build query with filters
	query := `
		SELECT
			id, symbol, name, type, category, description, logo_url, website_url,
			created_at, updated_at
		FROM assets
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.Type != nil {
			query += fmt.Sprintf(" AND type = $%d", argPos)
			args = append(args, *filter.Type)
			argPos++
		}

		if filter.Category != nil {
			query += fmt.Sprintf(" AND category = $%d", argPos)
			args = append(args, *filter.Category)
			argPos++
		}

		// Add sorting
		sortBy := filter.SortBy
		if sortBy == "" {
			sortBy = "created_at"
		}
		sortOrder := filter.SortOrder
		if sortOrder == "" {
			sortOrder = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

		// Add pagination
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argPos)
			args = append(args, filter.Limit)
			argPos++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	var assets []*assetsv1.Asset
	for rows.Next() {
		var assetId, symbol, name, assetTypeStr, category, description, logoUrl, websiteUrl string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&assetId,
			&symbol,
			&name,
			&assetTypeStr,
			&category,
			&description,
			&logoUrl,
			&websiteUrl,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan asset row: %w", err)
		}

		assetType := parseAssetType(assetTypeStr)
		asset := &assetsv1.Asset{
			AssetId:     &assetId,
			Symbol:      ptrIfNotEmpty(symbol),
			Name:        ptrIfNotEmpty(name),
			AssetType:   &assetType,
			Category:    ptrIfNotEmpty(category),
			Description: ptrIfNotEmpty(description),
			LogoUrl:     ptrIfNotEmpty(logoUrl),
			WebsiteUrl:  ptrIfNotEmpty(websiteUrl),
			CreatedAt:   timestamppb.New(createdAt),
			UpdatedAt:   timestamppb.New(updatedAt),
		}

		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assets: %w", err)
	}

	return assets, nil
}

// SearchAssets performs a text search on assets
func (r *PostgresRepository) SearchAssets(ctx context.Context, searchQuery string, filter *AssetFilter) ([]*assetsv1.Asset, error) {
	// Build search query using ILIKE for case-insensitive search
	query := `
		SELECT
			id, symbol, name, type, category, description, logo_url, website_url,
			created_at, updated_at
		FROM assets
		WHERE (
			symbol ILIKE $1 OR
			name ILIKE $1 OR
			description ILIKE $1
		)
	`
	args := []interface{}{fmt.Sprintf("%%%s%%", searchQuery)}
	argPos := 2

	// Apply filters
	if filter != nil {
		if filter.Type != nil {
			query += fmt.Sprintf(" AND type = $%d", argPos)
			args = append(args, *filter.Type)
			argPos++
		}

		if filter.Category != nil {
			query += fmt.Sprintf(" AND category = $%d", argPos)
			args = append(args, *filter.Category)
			argPos++
		}

		// Add sorting
		sortBy := filter.SortBy
		if sortBy == "" {
			sortBy = "created_at"
		}
		sortOrder := filter.SortOrder
		if sortOrder == "" {
			sortOrder = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

		// Add pagination
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argPos)
			args = append(args, filter.Limit)
			argPos++
		}

		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filter.Offset)
		}
	}

	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search assets: %w", err)
	}
	defer rows.Close()

	var assets []*assetsv1.Asset
	for rows.Next() {
		var assetId, symbol, name, assetTypeStr, category, description, logoUrl, websiteUrl string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&assetId,
			&symbol,
			&name,
			&assetTypeStr,
			&category,
			&description,
			&logoUrl,
			&websiteUrl,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan asset row: %w", err)
		}

		assetType := parseAssetType(assetTypeStr)
		asset := &assetsv1.Asset{
			AssetId:     &assetId,
			Symbol:      ptrIfNotEmpty(symbol),
			Name:        ptrIfNotEmpty(name),
			AssetType:   &assetType,
			Category:    ptrIfNotEmpty(category),
			Description: ptrIfNotEmpty(description),
			LogoUrl:     ptrIfNotEmpty(logoUrl),
			WebsiteUrl:  ptrIfNotEmpty(websiteUrl),
			CreatedAt:   timestamppb.New(createdAt),
			UpdatedAt:   timestamppb.New(updatedAt),
		}

		assets = append(assets, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assets: %w", err)
	}

	return assets, nil
}

// parseAssetType converts a database string to an AssetType enum
func parseAssetType(s string) assetsv1.AssetType {
	switch s {
	case "NATIVE":
		return assetsv1.AssetType_ASSET_TYPE_NATIVE
	case "ERC20", "CRYPTOCURRENCY", "STABLECOIN", "GOVERNANCE", "MEME":
		// Legacy mappings for backwards compatibility
		return assetsv1.AssetType_ASSET_TYPE_ERC20
	case "SPL":
		return assetsv1.AssetType_ASSET_TYPE_SPL
	case "ERC721", "NFT":
		return assetsv1.AssetType_ASSET_TYPE_ERC721
	case "ERC1155":
		return assetsv1.AssetType_ASSET_TYPE_ERC1155
	case "SYNTHETIC":
		return assetsv1.AssetType_ASSET_TYPE_SYNTHETIC
	case "LP_TOKEN":
		return assetsv1.AssetType_ASSET_TYPE_LP_TOKEN
	case "RECEIPT_TOKEN":
		return assetsv1.AssetType_ASSET_TYPE_RECEIPT_TOKEN
	case "WRAPPED":
		return assetsv1.AssetType_ASSET_TYPE_WRAPPED
	default:
		return assetsv1.AssetType_ASSET_TYPE_UNSPECIFIED
	}
}

// ptrIfNotEmpty returns a pointer to the string if not empty, nil otherwise
func ptrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// CreateAssetIdentifier inserts a new asset identifier mapping
func (r *PostgresRepository) CreateAssetIdentifier(ctx context.Context, identifier *assetsv1.AssetIdentifier) error {
	// Generate ID if not provided
	if identifier.IdentifierId == nil || *identifier.IdentifierId == "" {
		id := uuid.New().String()
		identifier.IdentifierId = &id
	}

	// Set timestamps
	now := timestamppb.Now()
	if identifier.CreatedAt == nil {
		identifier.CreatedAt = now
	}

	// Convert DataSource enum to string for database storage
	var source string
	if identifier.Source != nil {
		source = identifier.Source.String()
	}

	query := `
		INSERT INTO asset_identifiers (
			id, asset_id, source, external_id, is_primary, metadata,
			verified_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	// For now, verified_at and updated_at are set to created_at
	createdAtTime := identifier.CreatedAt.AsTime()

	// Convert metadata struct to JSONB
	// Note: In production, this should properly marshal the protobuf Struct to JSONB
	var metadataJSON []byte
	if identifier.Metadata != nil {
		// For now, store as nil - proper implementation would marshal the struct
		metadataJSON = nil
	}

	_, err := r.exec(ctx, query,
		identifier.GetIdentifierId(),
		identifier.GetAssetId(),
		source,
		identifier.GetExternalId(),
		identifier.GetIsPrimary(),
		metadataJSON,
		createdAtTime, // verified_at defaults to created_at
		createdAtTime,
		createdAtTime, // updated_at defaults to created_at
	)

	if err != nil {
		return fmt.Errorf("create asset identifier: %w", err)
	}

	return nil
}

// GetAssetIdentifier retrieves an asset identifier by ID
func (r *PostgresRepository) GetAssetIdentifier(ctx context.Context, id string) (*assetsv1.AssetIdentifier, error) {
	query := `
		SELECT
			id, asset_id, source, external_id, is_primary, metadata,
			created_at
		FROM asset_identifiers
		WHERE id = $1
	`

	var identifierId, assetId, sourceStr, externalId string
	var isPrimary bool
	var metadata []byte
	var createdAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&identifierId,
		&assetId,
		&sourceStr,
		&externalId,
		&isPrimary,
		&metadata,
		&createdAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("asset identifier not found: %s", id)
		}
		return nil, fmt.Errorf("get asset identifier: %w", err)
	}

	// Parse DataSource enum
	dataSource := parseAssetDataSource(sourceStr)

	// Convert metadata - for now nil, proper implementation would unmarshal to protobuf Struct
	var metadataStruct *structpb.Struct
	// TODO: Unmarshal metadata JSON to protobuf Struct

	identifier := &assetsv1.AssetIdentifier{
		IdentifierId: &identifierId,
		AssetId:      &assetId,
		Source:       &dataSource,
		ExternalId:   &externalId,
		IsPrimary:    &isPrimary,
		Metadata:     metadataStruct,
		CreatedAt:    timestamppb.New(createdAt),
	}

	return identifier, nil
}

// ListAssetIdentifiers retrieves asset identifiers with optional filtering
func (r *PostgresRepository) ListAssetIdentifiers(ctx context.Context, filter *AssetIdentifierFilter) ([]*assetsv1.AssetIdentifier, error) {
	query := `
		SELECT
			id, asset_id, source, external_id, is_primary, metadata,
			created_at
		FROM asset_identifiers
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if filter != nil {
		if filter.AssetID != nil {
			query += fmt.Sprintf(" AND asset_id = $%d", argPos)
			args = append(args, *filter.AssetID)
			argPos++
		}
		if filter.Source != nil {
			query += fmt.Sprintf(" AND source = $%d", argPos)
			args = append(args, *filter.Source)
			argPos++
		}
		if filter.IsPrimary != nil {
			query += fmt.Sprintf(" AND is_primary = $%d", argPos)
			args = append(args, *filter.IsPrimary)
			argPos++
		}

		query += " ORDER BY created_at DESC"

		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT $%d", argPos)
			args = append(args, filter.Limit)
			argPos++
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filter.Offset)
			argPos++
		}
	} else {
		query += " ORDER BY created_at DESC"
	}

	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list asset identifiers: %w", err)
	}
	defer rows.Close()

	identifiers := []*assetsv1.AssetIdentifier{}
	for rows.Next() {
		var identifierId, assetId, sourceStr, externalId string
		var isPrimary bool
		var metadata []byte
		var createdAt time.Time

		err := rows.Scan(
			&identifierId,
			&assetId,
			&sourceStr,
			&externalId,
			&isPrimary,
			&metadata,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan asset identifier: %w", err)
		}

		// Parse DataSource enum
		dataSource := parseAssetDataSource(sourceStr)

		// Convert metadata - for now nil
		var metadataStruct *structpb.Struct
		// TODO: Unmarshal metadata JSON to protobuf Struct

		identifier := &assetsv1.AssetIdentifier{
			IdentifierId: &identifierId,
			AssetId:      &assetId,
			Source:       &dataSource,
			ExternalId:   &externalId,
			IsPrimary:    &isPrimary,
			Metadata:     metadataStruct,
			CreatedAt:    timestamppb.New(createdAt),
		}

		identifiers = append(identifiers, identifier)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate asset identifiers: %w", err)
	}

	return identifiers, nil
}

// parseAssetDataSource converts a database string to a DataSource enum for assets
func parseAssetDataSource(s string) assetsv1.DataSource {
	switch s {
	case "coingecko":
		return assetsv1.DataSource_DATA_SOURCE_COINGECKO
	case "coinmarketcap":
		return assetsv1.DataSource_DATA_SOURCE_COINMARKETCAP
	case "defillama":
		return assetsv1.DataSource_DATA_SOURCE_DEFILLAMA
	case "messari":
		return assetsv1.DataSource_DATA_SOURCE_MESSARI
	case "internal":
		return assetsv1.DataSource_DATA_SOURCE_INTERNAL
	default:
		return assetsv1.DataSource_DATA_SOURCE_UNSPECIFIED
	}
}
