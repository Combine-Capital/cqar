package repository

import (
	"context"
	"fmt"
	"time"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateAssetDeployment creates a new asset deployment record
func (r *PostgresRepository) CreateAssetDeployment(ctx context.Context, deployment *assetsv1.AssetDeployment) error {
	// Validate that the asset exists
	var assetExists bool
	err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", deployment.GetAssetId()).Scan(&assetExists)
	if err != nil {
		return fmt.Errorf("check asset exists: %w", err)
	}
	if !assetExists {
		return fmt.Errorf("asset not found: %s", deployment.GetAssetId())
	}

	// Validate that the chain exists
	var chainExists bool
	err = r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM chains WHERE id = $1)", deployment.GetChainId()).Scan(&chainExists)
	if err != nil {
		return fmt.Errorf("check chain exists: %w", err)
	}
	if !chainExists {
		return fmt.Errorf("chain not found: %s", deployment.GetChainId())
	}

	// Generate ID if not provided
	if deployment.DeploymentId == nil || *deployment.DeploymentId == "" {
		// Use format: {chain}:{address}
		id := fmt.Sprintf("%s:%s", deployment.GetChainId(), deployment.GetAddress())
		deployment.DeploymentId = &id
	}

	// Set timestamps
	now := timestamppb.Now()
	if deployment.CreatedAt == nil {
		deployment.CreatedAt = now
	}
	deployment.UpdatedAt = now

	query := `
		INSERT INTO deployments (
			id, asset_id, chain_id, contract_address, decimals, is_canonical,
			deployment_block, deployer_address, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err = r.exec(ctx, query,
		deployment.GetDeploymentId(),
		deployment.GetAssetId(),
		deployment.GetChainId(),
		deployment.GetAddress(),
		deployment.GetDecimals(),
		deployment.GetIsCanonical(),
		nullableInt64(deployment.DeploymentBlock),
		nullableString(deployment.DeployerAddress),
		deployment.CreatedAt.AsTime(),
		deployment.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create asset deployment: %w", err)
	}

	return nil
}

// GetAssetDeployment retrieves a deployment by ID
func (r *PostgresRepository) GetAssetDeployment(ctx context.Context, id string) (*assetsv1.AssetDeployment, error) {
	query := `
		SELECT
			id, asset_id, chain_id, contract_address, decimals, is_canonical,
			deployment_block, deployer_address, created_at, updated_at
		FROM deployments
		WHERE id = $1
	`

	var deploymentId, assetId, chainId, contractAddress string
	var decimals int32
	var isCanonical bool
	var deploymentBlock *int64
	var deployerAddress *string
	var createdAt, updatedAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&deploymentId,
		&assetId,
		&chainId,
		&contractAddress,
		&decimals,
		&isCanonical,
		&deploymentBlock,
		&deployerAddress,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get asset deployment: %w", err)
	}

	deployment := &assetsv1.AssetDeployment{
		DeploymentId:    &deploymentId,
		AssetId:         &assetId,
		ChainId:         &chainId,
		Address:         &contractAddress,
		Decimals:        &decimals,
		IsCanonical:     &isCanonical,
		DeploymentBlock: deploymentBlock,
		DeployerAddress: deployerAddress,
		CreatedAt:       timestamppb.New(createdAt),
		UpdatedAt:       timestamppb.New(updatedAt),
	}

	return deployment, nil
}

// GetAssetDeploymentByChain retrieves a deployment for a specific asset on a specific chain
func (r *PostgresRepository) GetAssetDeploymentByChain(ctx context.Context, assetID, chainID string) (*assetsv1.AssetDeployment, error) {
	query := `
		SELECT
			id, asset_id, chain_id, contract_address, decimals, is_canonical,
			deployment_block, deployer_address, created_at, updated_at
		FROM deployments
		WHERE asset_id = $1 AND chain_id = $2
	`

	var deploymentId, assetId, chainId, contractAddress string
	var decimals int32
	var isCanonical bool
	var deploymentBlock *int64
	var deployerAddress *string
	var createdAt, updatedAt time.Time

	err := r.queryRow(ctx, query, assetID, chainID).Scan(
		&deploymentId,
		&assetId,
		&chainId,
		&contractAddress,
		&decimals,
		&isCanonical,
		&deploymentBlock,
		&deployerAddress,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get asset deployment by chain: %w", err)
	}

	deployment := &assetsv1.AssetDeployment{
		DeploymentId:    &deploymentId,
		AssetId:         &assetId,
		ChainId:         &chainId,
		Address:         &contractAddress,
		Decimals:        &decimals,
		IsCanonical:     &isCanonical,
		DeploymentBlock: deploymentBlock,
		DeployerAddress: deployerAddress,
		CreatedAt:       timestamppb.New(createdAt),
		UpdatedAt:       timestamppb.New(updatedAt),
	}

	return deployment, nil
}

// ListAssetDeployments retrieves a list of deployments with optional filtering
func (r *PostgresRepository) ListAssetDeployments(ctx context.Context, filter *DeploymentFilter) ([]*assetsv1.AssetDeployment, error) {
	// Build query with filters
	query := `
		SELECT
			id, asset_id, chain_id, contract_address, decimals, is_canonical,
			deployment_block, deployer_address, created_at, updated_at
		FROM deployments
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.AssetID != nil {
			query += fmt.Sprintf(" AND asset_id = $%d", argPos)
			args = append(args, *filter.AssetID)
			argPos++
		}

		if filter.ChainID != nil {
			query += fmt.Sprintf(" AND chain_id = $%d", argPos)
			args = append(args, *filter.ChainID)
			argPos++
		}

		if filter.IsCanonical != nil {
			query += fmt.Sprintf(" AND is_canonical = $%d", argPos)
			args = append(args, *filter.IsCanonical)
			argPos++
		}

		// Add sorting
		query += " ORDER BY created_at DESC"

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
		return nil, fmt.Errorf("list asset deployments: %w", err)
	}
	defer rows.Close()

	var deployments []*assetsv1.AssetDeployment
	for rows.Next() {
		var deploymentId, assetId, chainId, contractAddress string
		var decimals int32
		var isCanonical bool
		var deploymentBlock *int64
		var deployerAddress *string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&deploymentId,
			&assetId,
			&chainId,
			&contractAddress,
			&decimals,
			&isCanonical,
			&deploymentBlock,
			&deployerAddress,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan deployment row: %w", err)
		}

		deployment := &assetsv1.AssetDeployment{
			DeploymentId:    &deploymentId,
			AssetId:         &assetId,
			ChainId:         &chainId,
			Address:         &contractAddress,
			Decimals:        &decimals,
			IsCanonical:     &isCanonical,
			DeploymentBlock: deploymentBlock,
			DeployerAddress: deployerAddress,
			CreatedAt:       timestamppb.New(createdAt),
			UpdatedAt:       timestamppb.New(updatedAt),
		}

		deployments = append(deployments, deployment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate deployments: %w", err)
	}

	return deployments, nil
}

// Helper function to handle nullable int64 values
func nullableInt64(ptr *int64) interface{} {
	if ptr == nil {
		return nil
	}
	return *ptr
}

// Helper function to handle nullable string values
func nullableString(ptr *string) interface{} {
	if ptr == nil {
		return nil
	}
	return *ptr
}
