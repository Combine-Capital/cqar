package repository

import (
	"context"
	"fmt"
	"time"

	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateVenueAsset inserts a new venue asset mapping
func (r *PostgresRepository) CreateVenueAsset(ctx context.Context, venueAsset *venuesv1.VenueAsset) error {
	// Validate venue exists
	var venueExists bool
	err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM venues WHERE id = $1)", venueAsset.GetVenueId()).Scan(&venueExists)
	if err != nil {
		return fmt.Errorf("check venue exists: %w", err)
	}
	if !venueExists {
		return fmt.Errorf("venue not found: %s", venueAsset.GetVenueId())
	}

	// Validate asset exists
	var assetExists bool
	err = r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", venueAsset.GetAssetId()).Scan(&assetExists)
	if err != nil {
		return fmt.Errorf("check asset exists: %w", err)
	}
	if !assetExists {
		return fmt.Errorf("asset not found: %s", venueAsset.GetAssetId())
	}

	// Validate deployment_id if provided
	if venueAsset.DeploymentId != nil && *venueAsset.DeploymentId != "" {
		var deploymentExists bool
		err = r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM deployments WHERE id = $1)", *venueAsset.DeploymentId).Scan(&deploymentExists)
		if err != nil {
			return fmt.Errorf("check deployment exists: %w", err)
		}
		if !deploymentExists {
			return fmt.Errorf("deployment not found: %s", *venueAsset.DeploymentId)
		}
	}

	// Set timestamp
	now := timestamppb.Now()
	if venueAsset.ListedAt == nil {
		venueAsset.ListedAt = now
	}

	// Convert metadata to JSON if present
	var metadataJSON interface{}
	if venueAsset.Metadata != nil {
		metadataBytes, err := venueAsset.Metadata.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		metadataJSON = metadataBytes
	}

	var listedAtTime interface{}
	if venueAsset.ListedAt != nil {
		listedAtTime = venueAsset.ListedAt.AsTime()
	}

	var delistedAtTime interface{}
	if venueAsset.DelistedAt != nil {
		delistedAtTime = venueAsset.DelistedAt.AsTime()
	}

	query := `
		INSERT INTO venue_assets (
			venue_id, asset_id, venue_asset_symbol, deployment_id,
			deposit_enabled, withdraw_enabled, trading_enabled,
			min_deposit, min_withdrawal, withdrawal_fee, deposit_fee,
			listed_at, delisted_at, is_active, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
	`

	_, err = r.exec(ctx, query,
		venueAsset.GetVenueId(),
		venueAsset.GetAssetId(),
		venueAsset.VenueAssetSymbol,
		venueAsset.DeploymentId,
		venueAsset.GetDepositEnabled(),
		venueAsset.GetWithdrawEnabled(),
		venueAsset.GetTradingEnabled(),
		venueAsset.MinDeposit,
		venueAsset.MinWithdrawal,
		venueAsset.WithdrawalFee,
		venueAsset.DepositFee,
		listedAtTime,
		delistedAtTime,
		venueAsset.GetIsActive(),
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("create venue asset: %w", err)
	}

	return nil
}

// GetVenueAsset retrieves a venue asset mapping by venue_id and asset_id
func (r *PostgresRepository) GetVenueAsset(ctx context.Context, venueID, assetID string) (*venuesv1.VenueAsset, error) {
	query := `
		SELECT
			venue_id, asset_id, venue_asset_symbol, deployment_id,
			deposit_enabled, withdraw_enabled, trading_enabled,
			min_deposit, min_withdrawal, withdrawal_fee, deposit_fee,
			listed_at, delisted_at, is_active, metadata
		FROM venue_assets
		WHERE venue_id = $1 AND asset_id = $2
	`

	var venueId, assetId string
	var venueAssetSymbol, deploymentId *string
	var depositEnabled, withdrawEnabled, tradingEnabled, isActive bool
	var minDeposit, minWithdrawal, withdrawalFee, depositFee *float64
	var listedAt, delistedAt *time.Time
	var metadataJSON []byte

	err := r.queryRow(ctx, query, venueID, assetID).Scan(
		&venueId,
		&assetId,
		&venueAssetSymbol,
		&deploymentId,
		&depositEnabled,
		&withdrawEnabled,
		&tradingEnabled,
		&minDeposit,
		&minWithdrawal,
		&withdrawalFee,
		&depositFee,
		&listedAt,
		&delistedAt,
		&isActive,
		&metadataJSON,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("venue asset not found: venue=%s asset=%s", venueID, assetID)
		}
		return nil, fmt.Errorf("get venue asset: %w", err)
	}

	venueAsset := &venuesv1.VenueAsset{
		VenueId:          &venueId,
		AssetId:          &assetId,
		VenueAssetSymbol: venueAssetSymbol,
		DeploymentId:     deploymentId,
		DepositEnabled:   &depositEnabled,
		WithdrawEnabled:  &withdrawEnabled,
		TradingEnabled:   &tradingEnabled,
		MinDeposit:       minDeposit,
		MinWithdrawal:    minWithdrawal,
		WithdrawalFee:    withdrawalFee,
		DepositFee:       depositFee,
		IsActive:         &isActive,
	}

	if listedAt != nil {
		venueAsset.ListedAt = timestamppb.New(*listedAt)
	}

	if delistedAt != nil {
		venueAsset.DelistedAt = timestamppb.New(*delistedAt)
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		metadata := &structpb.Struct{}
		if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
		venueAsset.Metadata = metadata
	}

	return venueAsset, nil
}

// ListVenueAssets retrieves venue assets with optional filtering
// Supports queries like "which venues trade BTC?" and "which assets on Binance?"
func (r *PostgresRepository) ListVenueAssets(ctx context.Context, filter *VenueAssetFilter) ([]*venuesv1.VenueAsset, error) {
	query := `
		SELECT
			venue_id, asset_id, venue_asset_symbol, deployment_id,
			deposit_enabled, withdraw_enabled, trading_enabled,
			min_deposit, min_withdrawal, withdrawal_fee, deposit_fee,
			listed_at, delisted_at, is_active, metadata
		FROM venue_assets
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

		if filter.AssetID != nil {
			query += fmt.Sprintf(" AND asset_id = $%d", argPos)
			args = append(args, *filter.AssetID)
			argPos++
		}

		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argPos)
			args = append(args, *filter.IsActive)
			argPos++
		}

		if filter.TradingEnabled != nil {
			query += fmt.Sprintf(" AND trading_enabled = $%d", argPos)
			args = append(args, *filter.TradingEnabled)
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
		return nil, fmt.Errorf("list venue assets: %w", err)
	}
	defer rows.Close()

	venueAssets := []*venuesv1.VenueAsset{}
	for rows.Next() {
		var venueId, assetId string
		var venueAssetSymbol, deploymentId *string
		var depositEnabled, withdrawEnabled, tradingEnabled, isActive bool
		var minDeposit, minWithdrawal, withdrawalFee, depositFee *float64
		var listedAt, delistedAt *time.Time
		var metadataJSON []byte

		err := rows.Scan(
			&venueId,
			&assetId,
			&venueAssetSymbol,
			&deploymentId,
			&depositEnabled,
			&withdrawEnabled,
			&tradingEnabled,
			&minDeposit,
			&minWithdrawal,
			&withdrawalFee,
			&depositFee,
			&listedAt,
			&delistedAt,
			&isActive,
			&metadataJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("scan venue asset: %w", err)
		}

		venueAsset := &venuesv1.VenueAsset{
			VenueId:          &venueId,
			AssetId:          &assetId,
			VenueAssetSymbol: venueAssetSymbol,
			DeploymentId:     deploymentId,
			DepositEnabled:   &depositEnabled,
			WithdrawEnabled:  &withdrawEnabled,
			TradingEnabled:   &tradingEnabled,
			MinDeposit:       minDeposit,
			MinWithdrawal:    minWithdrawal,
			WithdrawalFee:    withdrawalFee,
			DepositFee:       depositFee,
			IsActive:         &isActive,
		}

		if listedAt != nil {
			venueAsset.ListedAt = timestamppb.New(*listedAt)
		}

		if delistedAt != nil {
			venueAsset.DelistedAt = timestamppb.New(*delistedAt)
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			metadata := &structpb.Struct{}
			if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
			venueAsset.Metadata = metadata
		}

		venueAssets = append(venueAssets, venueAsset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate venue assets: %w", err)
	}

	return venueAssets, nil
}
