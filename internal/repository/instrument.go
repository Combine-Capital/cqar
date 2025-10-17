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

// CreateInstrument inserts a new instrument into the database
func (r *PostgresRepository) CreateInstrument(ctx context.Context, instrument *marketsv1.Instrument) error {
	// Generate ID if not provided
	if instrument.Id == nil || *instrument.Id == "" {
		id := uuid.New().String()
		instrument.Id = &id
	}

	// Set timestamps
	now := timestamppb.Now()
	if instrument.CreatedAt == nil {
		instrument.CreatedAt = now
	}
	instrument.UpdatedAt = now

	query := `
		INSERT INTO instruments (
			id, instrument_type, code, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`

	_, err := r.exec(ctx, query,
		instrument.GetId(),
		instrument.GetInstrumentType(),
		instrument.GetCode(),
		instrument.CreatedAt.AsTime(),
		instrument.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create instrument: %w", err)
	}

	return nil
}

// GetInstrument retrieves an instrument by ID
func (r *PostgresRepository) GetInstrument(ctx context.Context, id string) (*marketsv1.Instrument, error) {
	query := `
		SELECT id, instrument_type, code, created_at, updated_at
		FROM instruments
		WHERE id = $1
	`

	var instrument marketsv1.Instrument
	var createdAt, updatedAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&instrument.Id,
		&instrument.InstrumentType,
		&instrument.Code,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("instrument not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get instrument: %w", err)
	}

	instrument.CreatedAt = timestamppb.New(createdAt)
	instrument.UpdatedAt = timestamppb.New(updatedAt)

	return &instrument, nil
}

// CreateSpotInstrument inserts a new spot instrument
// Validates that instrument_id, base_asset_id, and quote_asset_id exist before insert
func (r *PostgresRepository) CreateSpotInstrument(ctx context.Context, spot *marketsv1.SpotInstrument) error {
	// Validate that instrument_id exists
	if spot.InstrumentId != nil && *spot.InstrumentId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM instruments WHERE id = $1)", *spot.InstrumentId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check instrument_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("instrument_id does not exist: %s", *spot.InstrumentId)
		}
	}

	// Validate that base_asset_id exists
	if spot.BaseAssetId != nil && *spot.BaseAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *spot.BaseAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check base_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("base_asset_id does not exist: %s", *spot.BaseAssetId)
		}
	}

	// Validate that quote_asset_id exists
	if spot.QuoteAssetId != nil && *spot.QuoteAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *spot.QuoteAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check quote_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("quote_asset_id does not exist: %s", *spot.QuoteAssetId)
		}
	}

	// Set timestamps
	now := timestamppb.Now()
	if spot.CreatedAt == nil {
		spot.CreatedAt = now
	}
	spot.UpdatedAt = now

	var extensionsJSON []byte
	var err error
	if spot.Extensions != nil {
		extensionsJSON, err = spot.Extensions.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal extensions: %w", err)
		}
	}

	query := `
		INSERT INTO spot_instruments (
			instrument_id, base_asset_id, quote_asset_id, extensions, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		)
	`

	_, err = r.exec(ctx, query,
		spot.GetInstrumentId(),
		spot.BaseAssetId,
		spot.QuoteAssetId,
		extensionsJSON,
		spot.CreatedAt.AsTime(),
		spot.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create spot instrument: %w", err)
	}

	return nil
}

// GetSpotInstrument retrieves a spot instrument by instrument_id
func (r *PostgresRepository) GetSpotInstrument(ctx context.Context, instrumentID string) (*marketsv1.SpotInstrument, error) {
	query := `
		SELECT instrument_id, base_asset_id, quote_asset_id, extensions, created_at, updated_at
		FROM spot_instruments
		WHERE instrument_id = $1
	`

	var spot marketsv1.SpotInstrument
	var createdAt, updatedAt time.Time
	var extensionsJSON []byte

	err := r.queryRow(ctx, query, instrumentID).Scan(
		&spot.InstrumentId,
		&spot.BaseAssetId,
		&spot.QuoteAssetId,
		&extensionsJSON,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("spot instrument not found: %s", instrumentID)
	}
	if err != nil {
		return nil, fmt.Errorf("get spot instrument: %w", err)
	}

	if extensionsJSON != nil {
		spot.Extensions = &structpb.Struct{}
		if err := spot.Extensions.UnmarshalJSON(extensionsJSON); err != nil {
			return nil, fmt.Errorf("unmarshal extensions: %w", err)
		}
	}

	spot.CreatedAt = timestamppb.New(createdAt)
	spot.UpdatedAt = timestamppb.New(updatedAt)

	return &spot, nil
}

// CreatePerpContract inserts a new perpetual contract
// Validates that instrument_id and underlying_asset_id exist before insert
func (r *PostgresRepository) CreatePerpContract(ctx context.Context, perp *marketsv1.PerpContract) error {
	// Validate that instrument_id exists
	if perp.InstrumentId != nil && *perp.InstrumentId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM instruments WHERE id = $1)", *perp.InstrumentId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check instrument_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("instrument_id does not exist: %s", *perp.InstrumentId)
		}
	}

	// Validate that underlying_asset_id exists
	if perp.UnderlyingAssetId != nil && *perp.UnderlyingAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *perp.UnderlyingAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check underlying_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("underlying_asset_id does not exist: %s", *perp.UnderlyingAssetId)
		}
	}

	// Set timestamps
	now := timestamppb.Now()
	if perp.CreatedAt == nil {
		perp.CreatedAt = now
	}
	perp.UpdatedAt = now

	var extensionsJSON []byte
	var err error
	if perp.Extensions != nil {
		extensionsJSON, err = perp.Extensions.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal extensions: %w", err)
		}
	}

	query := `
		INSERT INTO perp_contracts (
			instrument_id, underlying_asset_id, is_inverse, is_quanto, 
			contract_multiplier, extensions, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	_, err = r.exec(ctx, query,
		perp.GetInstrumentId(),
		perp.UnderlyingAssetId,
		perp.GetIsInverse(),
		perp.GetIsQuanto(),
		perp.ContractMultiplier,
		extensionsJSON,
		perp.CreatedAt.AsTime(),
		perp.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create perp contract: %w", err)
	}

	return nil
}

// GetPerpContract retrieves a perpetual contract by instrument_id
func (r *PostgresRepository) GetPerpContract(ctx context.Context, instrumentID string) (*marketsv1.PerpContract, error) {
	query := `
		SELECT instrument_id, underlying_asset_id, is_inverse, is_quanto,
		       contract_multiplier, extensions, created_at, updated_at
		FROM perp_contracts
		WHERE instrument_id = $1
	`

	var perp marketsv1.PerpContract
	var createdAt, updatedAt time.Time
	var extensionsJSON []byte

	err := r.queryRow(ctx, query, instrumentID).Scan(
		&perp.InstrumentId,
		&perp.UnderlyingAssetId,
		&perp.IsInverse,
		&perp.IsQuanto,
		&perp.ContractMultiplier,
		&extensionsJSON,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("perp contract not found: %s", instrumentID)
	}
	if err != nil {
		return nil, fmt.Errorf("get perp contract: %w", err)
	}

	if extensionsJSON != nil {
		perp.Extensions = &structpb.Struct{}
		if err := perp.Extensions.UnmarshalJSON(extensionsJSON); err != nil {
			return nil, fmt.Errorf("unmarshal extensions: %w", err)
		}
	}

	perp.CreatedAt = timestamppb.New(createdAt)
	perp.UpdatedAt = timestamppb.New(updatedAt)

	return &perp, nil
}

// CreateFutureContract inserts a new future contract
// Validates that instrument_id and underlying_asset_id exist before insert
func (r *PostgresRepository) CreateFutureContract(ctx context.Context, future *marketsv1.FutureContract) error {
	// Validate that instrument_id exists
	if future.InstrumentId != nil && *future.InstrumentId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM instruments WHERE id = $1)", *future.InstrumentId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check instrument_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("instrument_id does not exist: %s", *future.InstrumentId)
		}
	}

	// Validate that underlying_asset_id exists
	if future.UnderlyingAssetId != nil && *future.UnderlyingAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *future.UnderlyingAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check underlying_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("underlying_asset_id does not exist: %s", *future.UnderlyingAssetId)
		}
	}

	// Set timestamps
	now := timestamppb.Now()
	if future.CreatedAt == nil {
		future.CreatedAt = now
	}
	future.UpdatedAt = now

	var extensionsJSON []byte
	var err error
	if future.Extensions != nil {
		extensionsJSON, err = future.Extensions.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal extensions: %w", err)
		}
	}

	var expiryTime *time.Time
	if future.Expiry != nil {
		t := future.Expiry.AsTime()
		expiryTime = &t
	}

	query := `
		INSERT INTO future_contracts (
			instrument_id, underlying_asset_id, expiry, is_inverse, is_quanto,
			contract_multiplier, extensions, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	_, err = r.exec(ctx, query,
		future.GetInstrumentId(),
		future.UnderlyingAssetId,
		expiryTime,
		future.GetIsInverse(),
		future.GetIsQuanto(),
		future.ContractMultiplier,
		extensionsJSON,
		future.CreatedAt.AsTime(),
		future.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create future contract: %w", err)
	}

	return nil
}

// GetFutureContract retrieves a future contract by instrument_id
func (r *PostgresRepository) GetFutureContract(ctx context.Context, instrumentID string) (*marketsv1.FutureContract, error) {
	query := `
		SELECT instrument_id, underlying_asset_id, expiry, is_inverse, is_quanto,
		       contract_multiplier, extensions, created_at, updated_at
		FROM future_contracts
		WHERE instrument_id = $1
	`

	var future marketsv1.FutureContract
	var createdAt, updatedAt time.Time
	var expiryTime *time.Time
	var extensionsJSON []byte

	err := r.queryRow(ctx, query, instrumentID).Scan(
		&future.InstrumentId,
		&future.UnderlyingAssetId,
		&expiryTime,
		&future.IsInverse,
		&future.IsQuanto,
		&future.ContractMultiplier,
		&extensionsJSON,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("future contract not found: %s", instrumentID)
	}
	if err != nil {
		return nil, fmt.Errorf("get future contract: %w", err)
	}

	if expiryTime != nil {
		future.Expiry = timestamppb.New(*expiryTime)
	}

	if extensionsJSON != nil {
		future.Extensions = &structpb.Struct{}
		if err := future.Extensions.UnmarshalJSON(extensionsJSON); err != nil {
			return nil, fmt.Errorf("unmarshal extensions: %w", err)
		}
	}

	future.CreatedAt = timestamppb.New(createdAt)
	future.UpdatedAt = timestamppb.New(updatedAt)

	return &future, nil
}

// CreateOptionSeries inserts a new option series
// Validates that instrument_id and underlying_asset_id exist before insert
func (r *PostgresRepository) CreateOptionSeries(ctx context.Context, option *marketsv1.OptionSeries) error {
	// Validate that instrument_id exists
	if option.InstrumentId != nil && *option.InstrumentId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM instruments WHERE id = $1)", *option.InstrumentId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check instrument_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("instrument_id does not exist: %s", *option.InstrumentId)
		}
	}

	// Validate that underlying_asset_id exists
	if option.UnderlyingAssetId != nil && *option.UnderlyingAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *option.UnderlyingAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check underlying_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("underlying_asset_id does not exist: %s", *option.UnderlyingAssetId)
		}
	}

	// Set timestamps
	now := timestamppb.Now()
	if option.CreatedAt == nil {
		option.CreatedAt = now
	}
	option.UpdatedAt = now

	var extensionsJSON []byte
	var err error
	if option.Extensions != nil {
		extensionsJSON, err = option.Extensions.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal extensions: %w", err)
		}
	}

	var expiryTime *time.Time
	if option.Expiry != nil {
		t := option.Expiry.AsTime()
		expiryTime = &t
	}

	query := `
		INSERT INTO option_series (
			instrument_id, underlying_asset_id, expiry, strike_price, option_type,
			exercise_style, contract_multiplier, extensions, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err = r.exec(ctx, query,
		option.GetInstrumentId(),
		option.UnderlyingAssetId,
		expiryTime,
		option.StrikePrice,
		option.OptionType,
		option.ExerciseStyle,
		option.ContractMultiplier,
		extensionsJSON,
		option.CreatedAt.AsTime(),
		option.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create option series: %w", err)
	}

	return nil
}

// GetOptionSeries retrieves an option series by instrument_id
func (r *PostgresRepository) GetOptionSeries(ctx context.Context, instrumentID string) (*marketsv1.OptionSeries, error) {
	query := `
		SELECT instrument_id, underlying_asset_id, expiry, strike_price, option_type,
		       exercise_style, contract_multiplier, extensions, created_at, updated_at
		FROM option_series
		WHERE instrument_id = $1
	`

	var option marketsv1.OptionSeries
	var createdAt, updatedAt time.Time
	var expiryTime *time.Time
	var extensionsJSON []byte

	err := r.queryRow(ctx, query, instrumentID).Scan(
		&option.InstrumentId,
		&option.UnderlyingAssetId,
		&expiryTime,
		&option.StrikePrice,
		&option.OptionType,
		&option.ExerciseStyle,
		&option.ContractMultiplier,
		&extensionsJSON,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("option series not found: %s", instrumentID)
	}
	if err != nil {
		return nil, fmt.Errorf("get option series: %w", err)
	}

	if expiryTime != nil {
		option.Expiry = timestamppb.New(*expiryTime)
	}

	if extensionsJSON != nil {
		option.Extensions = &structpb.Struct{}
		if err := option.Extensions.UnmarshalJSON(extensionsJSON); err != nil {
			return nil, fmt.Errorf("unmarshal extensions: %w", err)
		}
	}

	option.CreatedAt = timestamppb.New(createdAt)
	option.UpdatedAt = timestamppb.New(updatedAt)

	return &option, nil
}

// CreateLendingDeposit inserts a new lending deposit
// Validates that instrument_id and underlying_asset_id exist before insert
func (r *PostgresRepository) CreateLendingDeposit(ctx context.Context, deposit *marketsv1.LendingDeposit) error {
	// Validate that instrument_id exists
	if deposit.InstrumentId != nil && *deposit.InstrumentId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM instruments WHERE id = $1)", *deposit.InstrumentId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check instrument_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("instrument_id does not exist: %s", *deposit.InstrumentId)
		}
	}

	// Validate that underlying_asset_id exists
	if deposit.UnderlyingAssetId != nil && *deposit.UnderlyingAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *deposit.UnderlyingAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check underlying_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("underlying_asset_id does not exist: %s", *deposit.UnderlyingAssetId)
		}
	}

	// Set timestamps
	now := timestamppb.Now()
	if deposit.CreatedAt == nil {
		deposit.CreatedAt = now
	}
	deposit.UpdatedAt = now

	var extensionsJSON []byte
	var err error
	if deposit.Extensions != nil {
		extensionsJSON, err = deposit.Extensions.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal extensions: %w", err)
		}
	}

	query := `
		INSERT INTO lending_deposits (
			instrument_id, underlying_asset_id, extensions, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`

	_, err = r.exec(ctx, query,
		deposit.GetInstrumentId(),
		deposit.UnderlyingAssetId,
		extensionsJSON,
		deposit.CreatedAt.AsTime(),
		deposit.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create lending deposit: %w", err)
	}

	return nil
}

// GetLendingDeposit retrieves a lending deposit by instrument_id
func (r *PostgresRepository) GetLendingDeposit(ctx context.Context, instrumentID string) (*marketsv1.LendingDeposit, error) {
	query := `
		SELECT instrument_id, underlying_asset_id, extensions, created_at, updated_at
		FROM lending_deposits
		WHERE instrument_id = $1
	`

	var deposit marketsv1.LendingDeposit
	var createdAt, updatedAt time.Time
	var extensionsJSON []byte

	err := r.queryRow(ctx, query, instrumentID).Scan(
		&deposit.InstrumentId,
		&deposit.UnderlyingAssetId,
		&extensionsJSON,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("lending deposit not found: %s", instrumentID)
	}
	if err != nil {
		return nil, fmt.Errorf("get lending deposit: %w", err)
	}

	if extensionsJSON != nil {
		deposit.Extensions = &structpb.Struct{}
		if err := deposit.Extensions.UnmarshalJSON(extensionsJSON); err != nil {
			return nil, fmt.Errorf("unmarshal extensions: %w", err)
		}
	}

	deposit.CreatedAt = timestamppb.New(createdAt)
	deposit.UpdatedAt = timestamppb.New(updatedAt)

	return &deposit, nil
}

// CreateLendingBorrow inserts a new lending borrow
// Validates that instrument_id and underlying_asset_id exist before insert
func (r *PostgresRepository) CreateLendingBorrow(ctx context.Context, borrow *marketsv1.LendingBorrow) error {
	// Validate that instrument_id exists
	if borrow.InstrumentId != nil && *borrow.InstrumentId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM instruments WHERE id = $1)", *borrow.InstrumentId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check instrument_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("instrument_id does not exist: %s", *borrow.InstrumentId)
		}
	}

	// Validate that underlying_asset_id exists
	if borrow.UnderlyingAssetId != nil && *borrow.UnderlyingAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *borrow.UnderlyingAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check underlying_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("underlying_asset_id does not exist: %s", *borrow.UnderlyingAssetId)
		}
	}

	// Set timestamps
	now := timestamppb.Now()
	if borrow.CreatedAt == nil {
		borrow.CreatedAt = now
	}
	borrow.UpdatedAt = now

	var extensionsJSON []byte
	var err error
	if borrow.Extensions != nil {
		extensionsJSON, err = borrow.Extensions.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal extensions: %w", err)
		}
	}

	query := `
		INSERT INTO lending_borrows (
			instrument_id, underlying_asset_id, extensions, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`

	_, err = r.exec(ctx, query,
		borrow.GetInstrumentId(),
		borrow.UnderlyingAssetId,
		extensionsJSON,
		borrow.CreatedAt.AsTime(),
		borrow.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create lending borrow: %w", err)
	}

	return nil
}

// GetLendingBorrow retrieves a lending borrow by instrument_id
func (r *PostgresRepository) GetLendingBorrow(ctx context.Context, instrumentID string) (*marketsv1.LendingBorrow, error) {
	query := `
		SELECT instrument_id, underlying_asset_id, extensions, created_at, updated_at
		FROM lending_borrows
		WHERE instrument_id = $1
	`

	var borrow marketsv1.LendingBorrow
	var createdAt, updatedAt time.Time
	var extensionsJSON []byte

	err := r.queryRow(ctx, query, instrumentID).Scan(
		&borrow.InstrumentId,
		&borrow.UnderlyingAssetId,
		&extensionsJSON,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("lending borrow not found: %s", instrumentID)
	}
	if err != nil {
		return nil, fmt.Errorf("get lending borrow: %w", err)
	}

	if extensionsJSON != nil {
		borrow.Extensions = &structpb.Struct{}
		if err := borrow.Extensions.UnmarshalJSON(extensionsJSON); err != nil {
			return nil, fmt.Errorf("unmarshal extensions: %w", err)
		}
	}

	borrow.CreatedAt = timestamppb.New(createdAt)
	borrow.UpdatedAt = timestamppb.New(updatedAt)

	return &borrow, nil
}
