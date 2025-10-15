package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateSymbol inserts a new symbol into the database
// Validates that base_asset_id and quote_asset_id exist before insert
func (r *PostgresRepository) CreateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error {
	// Generate ID if not provided
	if symbol.SymbolId == nil || *symbol.SymbolId == "" {
		id := uuid.New().String()
		symbol.SymbolId = &id
	}

	// Set timestamps
	now := timestamppb.Now()
	if symbol.CreatedAt == nil {
		symbol.CreatedAt = now
	}
	symbol.UpdatedAt = now

	// Validate that base_asset_id exists (foreign key check)
	if symbol.BaseAssetId != nil && *symbol.BaseAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *symbol.BaseAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check base_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("base_asset_id does not exist: %s", *symbol.BaseAssetId)
		}
	}

	// Validate that quote_asset_id exists (foreign key check)
	if symbol.QuoteAssetId != nil && *symbol.QuoteAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *symbol.QuoteAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check quote_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("quote_asset_id does not exist: %s", *symbol.QuoteAssetId)
		}
	}

	// Validate settlement_asset_id if provided
	if symbol.SettlementAssetId != nil && *symbol.SettlementAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *symbol.SettlementAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check settlement_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("settlement_asset_id does not exist: %s", *symbol.SettlementAssetId)
		}
	}

	query := `
		INSERT INTO symbols (
			id, base_asset_id, quote_asset_id, settlement_asset_id,
			symbol_type, tick_size, lot_size, min_order_size, max_order_size,
			strike_price, expiry, option_type,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	var expiryTime *time.Time
	if symbol.Expiry != nil {
		t := symbol.Expiry.AsTime()
		expiryTime = &t
	}

	var optionTypeStr *string
	if symbol.OptionType != nil {
		s := symbol.OptionType.String()
		optionTypeStr = &s
	}

	_, err := r.exec(ctx, query,
		symbol.GetSymbolId(),
		symbol.BaseAssetId,
		symbol.QuoteAssetId,
		symbol.SettlementAssetId,
		symbol.GetSymbolType().String(),
		symbol.TickSize,
		symbol.LotSize,
		symbol.MinOrderSize,
		symbol.MaxOrderSize,
		symbol.StrikePrice,
		expiryTime,
		optionTypeStr,
		symbol.CreatedAt.AsTime(),
		symbol.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create symbol: %w", err)
	}

	return nil
}

// GetSymbol retrieves a symbol by ID
func (r *PostgresRepository) GetSymbol(ctx context.Context, id string) (*marketsv1.Symbol, error) {
	query := `
		SELECT
			id, base_asset_id, quote_asset_id, settlement_asset_id,
			symbol_type, tick_size, lot_size, min_order_size, max_order_size,
			strike_price, expiry, option_type,
			created_at, updated_at
		FROM symbols
		WHERE id = $1
	`

	var symbolId, baseAssetId, quoteAssetId string
	var settlementAssetId, optionTypeStr *string
	var symbolTypeStr string
	var tickSize, lotSize, minOrderSize, maxOrderSize float64
	var strikePrice *float64
	var expiry *time.Time
	var createdAt, updatedAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&symbolId,
		&baseAssetId,
		&quoteAssetId,
		&settlementAssetId,
		&symbolTypeStr,
		&tickSize,
		&lotSize,
		&minOrderSize,
		&maxOrderSize,
		&strikePrice,
		&expiry,
		&optionTypeStr,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("symbol not found: %s", id)
		}
		return nil, fmt.Errorf("get symbol: %w", err)
	}

	// Parse symbol type enum
	symbolType := parseSymbolType(symbolTypeStr)

	// Parse option type enum
	var optionType *marketsv1.OptionType
	if optionTypeStr != nil {
		ot := parseOptionType(*optionTypeStr)
		optionType = &ot
	}

	// Convert expiry to protobuf timestamp
	var expiryPb *timestamppb.Timestamp
	if expiry != nil {
		expiryPb = timestamppb.New(*expiry)
	}

	symbol := &marketsv1.Symbol{
		SymbolId:          &symbolId,
		BaseAssetId:       &baseAssetId,
		QuoteAssetId:      &quoteAssetId,
		SettlementAssetId: settlementAssetId,
		SymbolType:        &symbolType,
		TickSize:          &tickSize,
		LotSize:           &lotSize,
		MinOrderSize:      &minOrderSize,
		MaxOrderSize:      &maxOrderSize,
		StrikePrice:       strikePrice,
		Expiry:            expiryPb,
		OptionType:        optionType,
		CreatedAt:         timestamppb.New(createdAt),
		UpdatedAt:         timestamppb.New(updatedAt),
	}

	return symbol, nil
}

// UpdateSymbol updates an existing symbol
func (r *PostgresRepository) UpdateSymbol(ctx context.Context, symbol *marketsv1.Symbol) error {
	// Update timestamp
	symbol.UpdatedAt = timestamppb.Now()

	query := `
		UPDATE symbols
		SET
			base_asset_id = $2,
			quote_asset_id = $3,
			settlement_asset_id = $4,
			symbol_type = $5,
			tick_size = $6,
			lot_size = $7,
			min_order_size = $8,
			max_order_size = $9,
			strike_price = $10,
			expiry = $11,
			option_type = $12,
			updated_at = $13
		WHERE id = $1
	`

	var expiryTime *time.Time
	if symbol.Expiry != nil {
		t := symbol.Expiry.AsTime()
		expiryTime = &t
	}

	var optionTypeStr *string
	if symbol.OptionType != nil {
		s := symbol.OptionType.String()
		optionTypeStr = &s
	}

	result, err := r.exec(ctx, query,
		symbol.GetSymbolId(),
		symbol.BaseAssetId,
		symbol.QuoteAssetId,
		symbol.SettlementAssetId,
		symbol.GetSymbolType().String(),
		symbol.TickSize,
		symbol.LotSize,
		symbol.MinOrderSize,
		symbol.MaxOrderSize,
		symbol.StrikePrice,
		expiryTime,
		optionTypeStr,
		symbol.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("update symbol: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("symbol not found: %s", symbol.GetSymbolId())
	}

	return nil
}

// DeleteSymbol removes a symbol from the database
func (r *PostgresRepository) DeleteSymbol(ctx context.Context, id string) error {
	query := `DELETE FROM symbols WHERE id = $1`

	result, err := r.exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete symbol: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("symbol not found: %s", id)
	}

	return nil
}

// ListSymbols retrieves symbols with optional filtering and pagination
func (r *PostgresRepository) ListSymbols(ctx context.Context, filter *SymbolFilter) ([]*marketsv1.Symbol, error) {
	query := `
		SELECT
			id, base_asset_id, quote_asset_id, settlement_asset_id,
			symbol_type, tick_size, lot_size, min_order_size, max_order_size,
			strike_price, expiry, option_type,
			created_at, updated_at
		FROM symbols
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.BaseAssetID != nil {
			query += fmt.Sprintf(" AND base_asset_id = $%d", argPos)
			args = append(args, *filter.BaseAssetID)
			argPos++
		}
		if filter.QuoteAssetID != nil {
			query += fmt.Sprintf(" AND quote_asset_id = $%d", argPos)
			args = append(args, *filter.QuoteAssetID)
			argPos++
		}
		if filter.SymbolType != nil {
			query += fmt.Sprintf(" AND symbol_type = $%d", argPos)
			args = append(args, *filter.SymbolType)
			argPos++
		}
		if filter.SettlementAssetID != nil {
			query += fmt.Sprintf(" AND settlement_asset_id = $%d", argPos)
			args = append(args, *filter.SettlementAssetID)
			argPos++
		}

		// Sorting
		sortBy := "created_at"
		if filter.SortBy != "" {
			sortBy = filter.SortBy
		}
		sortOrder := "DESC"
		if filter.SortOrder != "" {
			sortOrder = strings.ToUpper(filter.SortOrder)
		}
		query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

		// Pagination
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
		return nil, fmt.Errorf("list symbols: %w", err)
	}
	defer rows.Close()

	symbols := []*marketsv1.Symbol{}
	for rows.Next() {
		var symbolId, baseAssetId, quoteAssetId string
		var settlementAssetId, optionTypeStr *string
		var symbolTypeStr string
		var tickSize, lotSize, minOrderSize, maxOrderSize float64
		var strikePrice *float64
		var expiry *time.Time
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&symbolId,
			&baseAssetId,
			&quoteAssetId,
			&settlementAssetId,
			&symbolTypeStr,
			&tickSize,
			&lotSize,
			&minOrderSize,
			&maxOrderSize,
			&strikePrice,
			&expiry,
			&optionTypeStr,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}

		symbolType := parseSymbolType(symbolTypeStr)

		var optionType *marketsv1.OptionType
		if optionTypeStr != nil {
			ot := parseOptionType(*optionTypeStr)
			optionType = &ot
		}

		var expiryPb *timestamppb.Timestamp
		if expiry != nil {
			expiryPb = timestamppb.New(*expiry)
		}

		symbol := &marketsv1.Symbol{
			SymbolId:          &symbolId,
			BaseAssetId:       &baseAssetId,
			QuoteAssetId:      &quoteAssetId,
			SettlementAssetId: settlementAssetId,
			SymbolType:        &symbolType,
			TickSize:          &tickSize,
			LotSize:           &lotSize,
			MinOrderSize:      &minOrderSize,
			MaxOrderSize:      &maxOrderSize,
			StrikePrice:       strikePrice,
			Expiry:            expiryPb,
			OptionType:        optionType,
			CreatedAt:         timestamppb.New(createdAt),
			UpdatedAt:         timestamppb.New(updatedAt),
		}

		symbols = append(symbols, symbol)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate symbols: %w", err)
	}

	return symbols, nil
}

// SearchSymbols performs a text search on symbols with filtering
func (r *PostgresRepository) SearchSymbols(ctx context.Context, query string, filter *SymbolFilter) ([]*marketsv1.Symbol, error) {
	// For now, implement as a simple LIKE search on base/quote asset IDs
	// In production, this could use full-text search or join with assets table to search by symbol names
	sqlQuery := `
		SELECT
			id, base_asset_id, quote_asset_id, settlement_asset_id,
			symbol_type, tick_size, lot_size, min_order_size, max_order_size,
			strike_price, expiry, option_type,
			created_at, updated_at
		FROM symbols
		WHERE (
			base_asset_id ILIKE $1
			OR quote_asset_id ILIKE $1
		)
	`

	args := []interface{}{fmt.Sprintf("%%%s%%", query)}
	argPos := 2

	// Apply additional filters
	if filter != nil {
		if filter.BaseAssetID != nil {
			sqlQuery += fmt.Sprintf(" AND base_asset_id = $%d", argPos)
			args = append(args, *filter.BaseAssetID)
			argPos++
		}
		if filter.QuoteAssetID != nil {
			sqlQuery += fmt.Sprintf(" AND quote_asset_id = $%d", argPos)
			args = append(args, *filter.QuoteAssetID)
			argPos++
		}
		if filter.SymbolType != nil {
			sqlQuery += fmt.Sprintf(" AND symbol_type = $%d", argPos)
			args = append(args, *filter.SymbolType)
			argPos++
		}

		// Sorting
		sortBy := "created_at"
		if filter.SortBy != "" {
			sortBy = filter.SortBy
		}
		sortOrder := "DESC"
		if filter.SortOrder != "" {
			sortOrder = strings.ToUpper(filter.SortOrder)
		}
		sqlQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

		// Pagination
		if filter.Limit > 0 {
			sqlQuery += fmt.Sprintf(" LIMIT $%d", argPos)
			args = append(args, filter.Limit)
			argPos++
		}
		if filter.Offset > 0 {
			sqlQuery += fmt.Sprintf(" OFFSET $%d", argPos)
			args = append(args, filter.Offset)
			argPos++
		}
	} else {
		sqlQuery += " ORDER BY created_at DESC LIMIT 100"
	}

	rows, err := r.query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search symbols: %w", err)
	}
	defer rows.Close()

	symbols := []*marketsv1.Symbol{}
	for rows.Next() {
		var symbolId, baseAssetId, quoteAssetId string
		var settlementAssetId, optionTypeStr *string
		var symbolTypeStr string
		var tickSize, lotSize, minOrderSize, maxOrderSize float64
		var strikePrice *float64
		var expiry *time.Time
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&symbolId,
			&baseAssetId,
			&quoteAssetId,
			&settlementAssetId,
			&symbolTypeStr,
			&tickSize,
			&lotSize,
			&minOrderSize,
			&maxOrderSize,
			&strikePrice,
			&expiry,
			&optionTypeStr,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan symbol: %w", err)
		}

		symbolType := parseSymbolType(symbolTypeStr)

		var optionType *marketsv1.OptionType
		if optionTypeStr != nil {
			ot := parseOptionType(*optionTypeStr)
			optionType = &ot
		}

		var expiryPb *timestamppb.Timestamp
		if expiry != nil {
			expiryPb = timestamppb.New(*expiry)
		}

		symbol := &marketsv1.Symbol{
			SymbolId:          &symbolId,
			BaseAssetId:       &baseAssetId,
			QuoteAssetId:      &quoteAssetId,
			SettlementAssetId: settlementAssetId,
			SymbolType:        &symbolType,
			TickSize:          &tickSize,
			LotSize:           &lotSize,
			MinOrderSize:      &minOrderSize,
			MaxOrderSize:      &maxOrderSize,
			StrikePrice:       strikePrice,
			Expiry:            expiryPb,
			OptionType:        optionType,
			CreatedAt:         timestamppb.New(createdAt),
			UpdatedAt:         timestamppb.New(updatedAt),
		}

		symbols = append(symbols, symbol)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate symbols: %w", err)
	}

	return symbols, nil
}

// CreateSymbolIdentifier inserts a new symbol identifier mapping
func (r *PostgresRepository) CreateSymbolIdentifier(ctx context.Context, identifier *marketsv1.SymbolIdentifier) error {
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
	if identifier.DataSource != nil {
		source = identifier.DataSource.String()
	}

	query := `
		INSERT INTO symbol_identifiers (
			id, symbol_id, source, external_id, is_primary, metadata,
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
		identifier.GetSymbolId(),
		source,
		identifier.GetExternalId(),
		identifier.GetIsPrimary(),
		metadataJSON,
		createdAtTime, // verified_at defaults to created_at
		createdAtTime,
		createdAtTime, // updated_at defaults to created_at
	)

	if err != nil {
		return fmt.Errorf("create symbol identifier: %w", err)
	}

	return nil
}

// GetSymbolIdentifier retrieves a symbol identifier by ID
func (r *PostgresRepository) GetSymbolIdentifier(ctx context.Context, id string) (*marketsv1.SymbolIdentifier, error) {
	query := `
		SELECT
			id, symbol_id, source, external_id, is_primary, metadata,
			created_at
		FROM symbol_identifiers
		WHERE id = $1
	`

	var identifierId, symbolId, sourceStr, externalId string
	var isPrimary bool
	var metadata []byte
	var createdAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&identifierId,
		&symbolId,
		&sourceStr,
		&externalId,
		&isPrimary,
		&metadata,
		&createdAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("symbol identifier not found: %s", id)
		}
		return nil, fmt.Errorf("get symbol identifier: %w", err)
	}

	// Parse DataSource enum
	dataSource := parseDataSource(sourceStr)

	// Convert metadata - for now nil, proper implementation would unmarshal to protobuf Struct
	var metadataStruct *structpb.Struct
	// TODO: Unmarshal metadata JSON to protobuf Struct

	identifier := &marketsv1.SymbolIdentifier{
		IdentifierId: &identifierId,
		SymbolId:     &symbolId,
		DataSource:   &dataSource,
		ExternalId:   &externalId,
		IsPrimary:    &isPrimary,
		Metadata:     metadataStruct,
		CreatedAt:    timestamppb.New(createdAt),
	}

	return identifier, nil
}

// ListSymbolIdentifiers retrieves symbol identifiers with optional filtering
func (r *PostgresRepository) ListSymbolIdentifiers(ctx context.Context, filter *SymbolIdentifierFilter) ([]*marketsv1.SymbolIdentifier, error) {
	query := `
		SELECT
			id, symbol_id, source, external_id, is_primary, metadata,
			created_at
		FROM symbol_identifiers
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	if filter != nil {
		if filter.SymbolID != nil {
			query += fmt.Sprintf(" AND symbol_id = $%d", argPos)
			args = append(args, *filter.SymbolID)
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
		return nil, fmt.Errorf("list symbol identifiers: %w", err)
	}
	defer rows.Close()

	identifiers := []*marketsv1.SymbolIdentifier{}
	for rows.Next() {
		var identifierId, symbolId, sourceStr, externalId string
		var isPrimary bool
		var metadata []byte
		var createdAt time.Time

		err := rows.Scan(
			&identifierId,
			&symbolId,
			&sourceStr,
			&externalId,
			&isPrimary,
			&metadata,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan symbol identifier: %w", err)
		}

		// Parse DataSource enum
		dataSource := parseDataSource(sourceStr)

		// Convert metadata - for now nil
		var metadataStruct *structpb.Struct
		// TODO: Unmarshal metadata JSON to protobuf Struct

		identifier := &marketsv1.SymbolIdentifier{
			IdentifierId: &identifierId,
			SymbolId:     &symbolId,
			DataSource:   &dataSource,
			ExternalId:   &externalId,
			IsPrimary:    &isPrimary,
			Metadata:     metadataStruct,
			CreatedAt:    timestamppb.New(createdAt),
		}

		identifiers = append(identifiers, identifier)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate symbol identifiers: %w", err)
	}

	return identifiers, nil
}

// parseSymbolType converts a database string to a SymbolType enum
func parseSymbolType(s string) marketsv1.SymbolType {
	switch s {
	case "SPOT":
		return marketsv1.SymbolType_SYMBOL_TYPE_SPOT
	case "PERPETUAL":
		return marketsv1.SymbolType_SYMBOL_TYPE_PERPETUAL
	case "FUTURE":
		return marketsv1.SymbolType_SYMBOL_TYPE_FUTURE
	case "OPTION":
		return marketsv1.SymbolType_SYMBOL_TYPE_OPTION
	case "MARGIN":
		return marketsv1.SymbolType_SYMBOL_TYPE_MARGIN
	default:
		return marketsv1.SymbolType_SYMBOL_TYPE_UNSPECIFIED
	}
}

// parseOptionType converts a database string to an OptionType enum
func parseOptionType(s string) marketsv1.OptionType {
	switch s {
	case "CALL":
		return marketsv1.OptionType_OPTION_TYPE_CALL
	case "PUT":
		return marketsv1.OptionType_OPTION_TYPE_PUT
	default:
		return marketsv1.OptionType_OPTION_TYPE_UNSPECIFIED
	}
}

// parseDataSource converts a database string to a DataSource enum
func parseDataSource(s string) marketsv1.DataSource {
	switch s {
	case "coingecko":
		return marketsv1.DataSource_DATA_SOURCE_COINGECKO
	case "coinmarketcap":
		return marketsv1.DataSource_DATA_SOURCE_COINMARKETCAP
	case "defillama":
		return marketsv1.DataSource_DATA_SOURCE_DEFILLAMA
	case "messari":
		return marketsv1.DataSource_DATA_SOURCE_MESSARI
	case "glassnode":
		return marketsv1.DataSource_DATA_SOURCE_GLASSNODE
	default:
		return marketsv1.DataSource_DATA_SOURCE_UNSPECIFIED
	}
}
