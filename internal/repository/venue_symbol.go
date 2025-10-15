package repository

import (
	"context"
	"fmt"
	"time"

	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateVenueSymbol inserts a new venue symbol mapping
func (r *PostgresRepository) CreateVenueSymbol(ctx context.Context, venueSymbol *venuesv1.VenueSymbol) error {
	// Validate venue exists
	var venueExists bool
	err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM venues WHERE id = $1)", venueSymbol.GetVenueId()).Scan(&venueExists)
	if err != nil {
		return fmt.Errorf("check venue exists: %w", err)
	}
	if !venueExists {
		return fmt.Errorf("venue not found: %s", venueSymbol.GetVenueId())
	}

	// Validate symbol exists
	var symbolExists bool
	err = r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM symbols WHERE id = $1)", venueSymbol.GetSymbolId()).Scan(&symbolExists)
	if err != nil {
		return fmt.Errorf("check symbol exists: %w", err)
	}
	if !symbolExists {
		return fmt.Errorf("symbol not found: %s", venueSymbol.GetSymbolId())
	}

	// Set timestamp
	now := timestamppb.Now()
	if venueSymbol.ListedAt == nil {
		venueSymbol.ListedAt = now
	}

	// Convert metadata to JSON if present
	var metadataJSON interface{}
	if venueSymbol.Metadata != nil {
		metadataBytes, err := venueSymbol.Metadata.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		metadataJSON = metadataBytes
	}

	var listedAtTime interface{}
	if venueSymbol.ListedAt != nil {
		listedAtTime = venueSymbol.ListedAt.AsTime()
	}

	var delistedAtTime interface{}
	if venueSymbol.DelistedAt != nil {
		delistedAtTime = venueSymbol.DelistedAt.AsTime()
	}

	query := `
		INSERT INTO venue_symbols (
			venue_id, symbol_id, venue_symbol, is_active,
			maker_fee, taker_fee, min_order_value, max_order_value,
			min_notional, max_leverage,
			listed_at, delisted_at, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`

	_, err = r.exec(ctx, query,
		venueSymbol.GetVenueId(),
		venueSymbol.GetSymbolId(),
		venueSymbol.GetVenueSymbol(),
		venueSymbol.GetIsActive(),
		venueSymbol.MakerFee,
		venueSymbol.TakerFee,
		venueSymbol.MinOrderValue,
		venueSymbol.MaxOrderValue,
		venueSymbol.MinNotional,
		venueSymbol.MaxLeverage,
		listedAtTime,
		delistedAtTime,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("create venue symbol: %w", err)
	}

	return nil
}

// GetVenueSymbol retrieves a venue symbol by venue_id and venue_symbol string
// This is the critical method for cqmd to resolve venue-specific symbols to canonical symbols
func (r *PostgresRepository) GetVenueSymbol(ctx context.Context, venueID, venueSymbolStr string) (*venuesv1.VenueSymbol, error) {
	query := `
		SELECT
			venue_id, symbol_id, venue_symbol, is_active,
			maker_fee, taker_fee, min_order_value, max_order_value,
			min_notional, max_leverage,
			listed_at, delisted_at, metadata
		FROM venue_symbols
		WHERE venue_id = $1 AND venue_symbol = $2
	`

	var venueId, symbolId, venueSymbol string
	var isActive bool
	var makerFee, takerFee, minOrderValue, maxOrderValue, minNotional, maxLeverage *float64
	var listedAt, delistedAt *time.Time
	var metadataJSON []byte

	err := r.queryRow(ctx, query, venueID, venueSymbolStr).Scan(
		&venueId,
		&symbolId,
		&venueSymbol,
		&isActive,
		&makerFee,
		&takerFee,
		&minOrderValue,
		&maxOrderValue,
		&minNotional,
		&maxLeverage,
		&listedAt,
		&delistedAt,
		&metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("venue symbol not found: venue=%s symbol=%s", venueID, venueSymbolStr)
		}
		return nil, fmt.Errorf("get venue symbol: %w", err)
	}

	vs := &venuesv1.VenueSymbol{
		VenueId:       &venueId,
		SymbolId:      &symbolId,
		VenueSymbol:   &venueSymbol,
		IsActive:      &isActive,
		MakerFee:      makerFee,
		TakerFee:      takerFee,
		MinOrderValue: minOrderValue,
		MaxOrderValue: maxOrderValue,
		MinNotional:   minNotional,
		MaxLeverage:   maxLeverage,
	}

	if listedAt != nil {
		vs.ListedAt = timestamppb.New(*listedAt)
	}

	if delistedAt != nil {
		vs.DelistedAt = timestamppb.New(*delistedAt)
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		metadata := &structpb.Struct{}
		if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
		vs.Metadata = metadata
	}

	return vs, nil
}

// GetVenueSymbolByID retrieves a venue symbol by venue_id and symbol_id (canonical)
func (r *PostgresRepository) GetVenueSymbolByID(ctx context.Context, venueID, symbolID string) (*venuesv1.VenueSymbol, error) {
	query := `
		SELECT
			venue_id, symbol_id, venue_symbol, is_active,
			maker_fee, taker_fee, min_order_value, max_order_value,
			min_notional, max_leverage,
			listed_at, delisted_at, metadata
		FROM venue_symbols
		WHERE venue_id = $1 AND symbol_id = $2
	`

	var venueId, symbolId, venueSymbol string
	var isActive bool
	var makerFee, takerFee, minOrderValue, maxOrderValue, minNotional, maxLeverage *float64
	var listedAt, delistedAt *time.Time
	var metadataJSON []byte

	err := r.queryRow(ctx, query, venueID, symbolID).Scan(
		&venueId,
		&symbolId,
		&venueSymbol,
		&isActive,
		&makerFee,
		&takerFee,
		&minOrderValue,
		&maxOrderValue,
		&minNotional,
		&maxLeverage,
		&listedAt,
		&delistedAt,
		&metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("venue symbol not found: venue=%s symbol_id=%s", venueID, symbolID)
		}
		return nil, fmt.Errorf("get venue symbol by ID: %w", err)
	}

	vs := &venuesv1.VenueSymbol{
		VenueId:       &venueId,
		SymbolId:      &symbolId,
		VenueSymbol:   &venueSymbol,
		IsActive:      &isActive,
		MakerFee:      makerFee,
		TakerFee:      takerFee,
		MinOrderValue: minOrderValue,
		MaxOrderValue: maxOrderValue,
		MinNotional:   minNotional,
		MaxLeverage:   maxLeverage,
	}

	if listedAt != nil {
		vs.ListedAt = timestamppb.New(*listedAt)
	}

	if delistedAt != nil {
		vs.DelistedAt = timestamppb.New(*delistedAt)
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		metadata := &structpb.Struct{}
		if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
		vs.Metadata = metadata
	}

	return vs, nil
}

// ListVenueSymbols retrieves venue symbols with optional filtering
func (r *PostgresRepository) ListVenueSymbols(ctx context.Context, filter *VenueSymbolFilter) ([]*venuesv1.VenueSymbol, error) {
	query := `
		SELECT
			venue_id, symbol_id, venue_symbol, is_active,
			maker_fee, taker_fee, min_order_value, max_order_value,
			min_notional, max_leverage,
			listed_at, delisted_at, metadata
		FROM venue_symbols
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.VenueID != nil {
			query += fmt.Sprintf(" AND venue_id = $%d", argPos)
			args = append(args, *filter.VenueID)
			argPos++
		}

		if filter.SymbolID != nil {
			query += fmt.Sprintf(" AND symbol_id = $%d", argPos)
			args = append(args, *filter.SymbolID)
			argPos++
		}

		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argPos)
			args = append(args, *filter.IsActive)
			argPos++
		}

		// Sorting
		query += " ORDER BY listed_at DESC"

		// Pagination
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
		return nil, fmt.Errorf("list venue symbols: %w", err)
	}
	defer rows.Close()

	venueSymbols := []*venuesv1.VenueSymbol{}
	for rows.Next() {
		var venueId, symbolId, venueSymbol string
		var isActive bool
		var makerFee, takerFee, minOrderValue, maxOrderValue, minNotional, maxLeverage *float64
		var listedAt, delistedAt *time.Time
		var metadataJSON []byte

		err := rows.Scan(
			&venueId,
			&symbolId,
			&venueSymbol,
			&isActive,
			&makerFee,
			&takerFee,
			&minOrderValue,
			&maxOrderValue,
			&minNotional,
			&maxLeverage,
			&listedAt,
			&delistedAt,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("scan venue symbol: %w", err)
		}

		vs := &venuesv1.VenueSymbol{
			VenueId:       &venueId,
			SymbolId:      &symbolId,
			VenueSymbol:   &venueSymbol,
			IsActive:      &isActive,
			MakerFee:      makerFee,
			TakerFee:      takerFee,
			MinOrderValue: minOrderValue,
			MaxOrderValue: maxOrderValue,
			MinNotional:   minNotional,
			MaxLeverage:   maxLeverage,
		}

		if listedAt != nil {
			vs.ListedAt = timestamppb.New(*listedAt)
		}

		if delistedAt != nil {
			vs.DelistedAt = timestamppb.New(*delistedAt)
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			metadata := &structpb.Struct{}
			if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
			vs.Metadata = metadata
		}

		venueSymbols = append(venueSymbols, vs)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate venue symbols: %w", err)
	}

	return venueSymbols, nil
}

// GetVenueSymbolEnriched retrieves a venue symbol with enriched canonical Symbol data
// This is useful for cqmd to get market specs from the canonical symbol
func (r *PostgresRepository) GetVenueSymbolEnriched(ctx context.Context, venueID, venueSymbolStr string) (*venuesv1.VenueSymbol, *marketsv1.Symbol, error) {
	// First get the venue symbol
	venueSymbol, err := r.GetVenueSymbol(ctx, venueID, venueSymbolStr)
	if err != nil {
		return nil, nil, err
	}

	// Then get the canonical symbol
	symbol, err := r.GetSymbol(ctx, venueSymbol.GetSymbolId())
	if err != nil {
		return nil, nil, fmt.Errorf("get canonical symbol: %w", err)
	}

	return venueSymbol, symbol, nil
}
