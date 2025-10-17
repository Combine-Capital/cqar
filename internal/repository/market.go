package repository

import (
	"context"
	"fmt"
	"time"

	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateMarket inserts a new market into the database
// Validates that instrument_id and venue_id exist before insert
func (r *PostgresRepository) CreateMarket(ctx context.Context, market *marketsv1.Market) error {
	// Generate ID if not provided
	if market.Id == nil || *market.Id == "" {
		id := uuid.New().String()
		market.Id = &id
	}

	// Validate that instrument_id exists (foreign key check)
	if market.InstrumentId != nil && *market.InstrumentId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM instruments WHERE id = $1)", *market.InstrumentId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check instrument_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("instrument_id does not exist: %s", *market.InstrumentId)
		}
	}

	// Validate that venue_id exists (foreign key check)
	if market.VenueId != nil && *market.VenueId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM venues WHERE id = $1)", *market.VenueId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check venue_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("venue_id does not exist: %s", *market.VenueId)
		}
	}

	// Validate settlement_asset_id if provided
	if market.SettlementAssetId != nil && *market.SettlementAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *market.SettlementAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check settlement_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("settlement_asset_id does not exist: %s", *market.SettlementAssetId)
		}
	}

	// Validate price_currency_asset_id if provided
	if market.PriceCurrencyAssetId != nil && *market.PriceCurrencyAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *market.PriceCurrencyAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check price_currency_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("price_currency_asset_id does not exist: %s", *market.PriceCurrencyAssetId)
		}
	}

	// Set timestamps
	now := timestamppb.Now()
	if market.CreatedAt == nil {
		market.CreatedAt = now
	}
	market.UpdatedAt = now

	var metadataJSON []byte
	var err error
	if market.Metadata != nil {
		metadataJSON, err = market.Metadata.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	var listedAtTime, delistedAtTime *time.Time
	if market.ListedAt != nil {
		t := market.ListedAt.AsTime()
		listedAtTime = &t
	}
	if market.DelistedAt != nil {
		t := market.DelistedAt.AsTime()
		delistedAtTime = &t
	}

	query := `
		INSERT INTO markets (
			id, instrument_id, venue_id, venue_symbol,
			settlement_asset_id, price_currency_asset_id,
			tick_size, lot_size, min_order_size, max_order_size, min_notional,
			maker_fee, taker_fee, funding_interval_secs,
			status, listed_at, delisted_at, metadata,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`

	_, err = r.exec(ctx, query,
		market.GetId(),
		market.InstrumentId,
		market.VenueId,
		market.VenueSymbol,
		market.SettlementAssetId,
		market.PriceCurrencyAssetId,
		market.TickSize,
		market.LotSize,
		market.MinOrderSize,
		market.MaxOrderSize,
		market.MinNotional,
		market.MakerFee,
		market.TakerFee,
		market.FundingIntervalSecs,
		market.Status,
		listedAtTime,
		delistedAtTime,
		metadataJSON,
		market.CreatedAt.AsTime(),
		market.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create market: %w", err)
	}

	return nil
}

// GetMarket retrieves a market by ID
func (r *PostgresRepository) GetMarket(ctx context.Context, id string) (*marketsv1.Market, error) {
	query := `
		SELECT id, instrument_id, venue_id, venue_symbol,
		       settlement_asset_id, price_currency_asset_id,
		       tick_size, lot_size, min_order_size, max_order_size, min_notional,
		       maker_fee, taker_fee, funding_interval_secs,
		       status, listed_at, delisted_at, metadata,
		       created_at, updated_at
		FROM markets
		WHERE id = $1
	`

	var market marketsv1.Market
	var createdAt, updatedAt time.Time
	var listedAt, delistedAt *time.Time
	var metadataJSON []byte

	err := r.queryRow(ctx, query, id).Scan(
		&market.Id,
		&market.InstrumentId,
		&market.VenueId,
		&market.VenueSymbol,
		&market.SettlementAssetId,
		&market.PriceCurrencyAssetId,
		&market.TickSize,
		&market.LotSize,
		&market.MinOrderSize,
		&market.MaxOrderSize,
		&market.MinNotional,
		&market.MakerFee,
		&market.TakerFee,
		&market.FundingIntervalSecs,
		&market.Status,
		&listedAt,
		&delistedAt,
		&metadataJSON,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("market not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get market: %w", err)
	}

	if listedAt != nil {
		market.ListedAt = timestamppb.New(*listedAt)
	}
	if delistedAt != nil {
		market.DelistedAt = timestamppb.New(*delistedAt)
	}

	if metadataJSON != nil {
		market.Metadata = &structpb.Struct{}
		if err := market.Metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	market.CreatedAt = timestamppb.New(createdAt)
	market.UpdatedAt = timestamppb.New(updatedAt)

	return &market, nil
}

// ResolveMarket retrieves a market by venue_id and venue_symbol
// This is the critical method for resolving venue-specific symbols to market_id
func (r *PostgresRepository) ResolveMarket(ctx context.Context, venueID, venueSymbol string) (*marketsv1.Market, error) {
	query := `
		SELECT id, instrument_id, venue_id, venue_symbol,
		       settlement_asset_id, price_currency_asset_id,
		       tick_size, lot_size, min_order_size, max_order_size, min_notional,
		       maker_fee, taker_fee, funding_interval_secs,
		       status, listed_at, delisted_at, metadata,
		       created_at, updated_at
		FROM markets
		WHERE venue_id = $1 AND venue_symbol = $2
	`

	var market marketsv1.Market
	var createdAt, updatedAt time.Time
	var listedAt, delistedAt *time.Time
	var metadataJSON []byte

	err := r.queryRow(ctx, query, venueID, venueSymbol).Scan(
		&market.Id,
		&market.InstrumentId,
		&market.VenueId,
		&market.VenueSymbol,
		&market.SettlementAssetId,
		&market.PriceCurrencyAssetId,
		&market.TickSize,
		&market.LotSize,
		&market.MinOrderSize,
		&market.MaxOrderSize,
		&market.MinNotional,
		&market.MakerFee,
		&market.TakerFee,
		&market.FundingIntervalSecs,
		&market.Status,
		&listedAt,
		&delistedAt,
		&metadataJSON,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("market not found for venue %s symbol %s", venueID, venueSymbol)
	}
	if err != nil {
		return nil, fmt.Errorf("resolve market: %w", err)
	}

	if listedAt != nil {
		market.ListedAt = timestamppb.New(*listedAt)
	}
	if delistedAt != nil {
		market.DelistedAt = timestamppb.New(*delistedAt)
	}

	if metadataJSON != nil {
		market.Metadata = &structpb.Struct{}
		if err := market.Metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	market.CreatedAt = timestamppb.New(createdAt)
	market.UpdatedAt = timestamppb.New(updatedAt)

	return &market, nil
}

// ListMarketsByInstrument retrieves all markets for a given instrument
func (r *PostgresRepository) ListMarketsByInstrument(ctx context.Context, instrumentID string) ([]*marketsv1.Market, error) {
	query := `
		SELECT id, instrument_id, venue_id, venue_symbol,
		       settlement_asset_id, price_currency_asset_id,
		       tick_size, lot_size, min_order_size, max_order_size, min_notional,
		       maker_fee, taker_fee, funding_interval_secs,
		       status, listed_at, delisted_at, metadata,
		       created_at, updated_at
		FROM markets
		WHERE instrument_id = $1
		ORDER BY venue_symbol
	`

	rows, err := r.query(ctx, query, instrumentID)
	if err != nil {
		return nil, fmt.Errorf("list markets by instrument: %w", err)
	}
	defer rows.Close()

	var markets []*marketsv1.Market
	for rows.Next() {
		var market marketsv1.Market
		var createdAt, updatedAt time.Time
		var listedAt, delistedAt *time.Time
		var metadataJSON []byte

		err := rows.Scan(
			&market.Id,
			&market.InstrumentId,
			&market.VenueId,
			&market.VenueSymbol,
			&market.SettlementAssetId,
			&market.PriceCurrencyAssetId,
			&market.TickSize,
			&market.LotSize,
			&market.MinOrderSize,
			&market.MaxOrderSize,
			&market.MinNotional,
			&market.MakerFee,
			&market.TakerFee,
			&market.FundingIntervalSecs,
			&market.Status,
			&listedAt,
			&delistedAt,
			&metadataJSON,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan market: %w", err)
		}

		if listedAt != nil {
			market.ListedAt = timestamppb.New(*listedAt)
		}
		if delistedAt != nil {
			market.DelistedAt = timestamppb.New(*delistedAt)
		}

		if metadataJSON != nil {
			market.Metadata = &structpb.Struct{}
			if err := market.Metadata.UnmarshalJSON(metadataJSON); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		market.CreatedAt = timestamppb.New(createdAt)
		market.UpdatedAt = timestamppb.New(updatedAt)

		markets = append(markets, &market)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return markets, nil
}

// ListMarketsByVenue retrieves all markets for a given venue
func (r *PostgresRepository) ListMarketsByVenue(ctx context.Context, venueID string) ([]*marketsv1.Market, error) {
	query := `
		SELECT id, instrument_id, venue_id, venue_symbol,
		       settlement_asset_id, price_currency_asset_id,
		       tick_size, lot_size, min_order_size, max_order_size, min_notional,
		       maker_fee, taker_fee, funding_interval_secs,
		       status, listed_at, delisted_at, metadata,
		       created_at, updated_at
		FROM markets
		WHERE venue_id = $1
		ORDER BY venue_symbol
	`

	rows, err := r.query(ctx, query, venueID)
	if err != nil {
		return nil, fmt.Errorf("list markets by venue: %w", err)
	}
	defer rows.Close()

	var markets []*marketsv1.Market
	for rows.Next() {
		var market marketsv1.Market
		var createdAt, updatedAt time.Time
		var listedAt, delistedAt *time.Time
		var metadataJSON []byte

		err := rows.Scan(
			&market.Id,
			&market.InstrumentId,
			&market.VenueId,
			&market.VenueSymbol,
			&market.SettlementAssetId,
			&market.PriceCurrencyAssetId,
			&market.TickSize,
			&market.LotSize,
			&market.MinOrderSize,
			&market.MaxOrderSize,
			&market.MinNotional,
			&market.MakerFee,
			&market.TakerFee,
			&market.FundingIntervalSecs,
			&market.Status,
			&listedAt,
			&delistedAt,
			&metadataJSON,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan market: %w", err)
		}

		if listedAt != nil {
			market.ListedAt = timestamppb.New(*listedAt)
		}
		if delistedAt != nil {
			market.DelistedAt = timestamppb.New(*delistedAt)
		}

		if metadataJSON != nil {
			market.Metadata = &structpb.Struct{}
			if err := market.Metadata.UnmarshalJSON(metadataJSON); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		market.CreatedAt = timestamppb.New(createdAt)
		market.UpdatedAt = timestamppb.New(updatedAt)

		markets = append(markets, &market)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return markets, nil
}
