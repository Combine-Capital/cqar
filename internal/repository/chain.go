package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"
	"google.golang.org/protobuf/types/known/structpb"
)

// CreateChain inserts a new chain into the database
func (r *PostgresRepository) CreateChain(ctx context.Context, chain *assetsv1.Chain) error {
	// Validate that chain_id is provided
	if chain.ChainId == nil || *chain.ChainId == "" {
		return fmt.Errorf("chain_id is required")
	}

	// Validate native_asset_id if provided
	if chain.NativeAssetId != nil && *chain.NativeAssetId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *chain.NativeAssetId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check native_asset_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("native_asset_id does not exist: %s", *chain.NativeAssetId)
		}
	}

	// Convert protobuf metadata to JSON if present
	var metadataJSON interface{}
	if chain.Metadata != nil {
		metadataBytes, err := chain.Metadata.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		metadataJSON = metadataBytes
	}

	// Note: rpc_urls field exists in database but not in protobuf v0.3.0
	// We'll store an empty array for backward compatibility
	var rpcUrls []string

	query := `
		INSERT INTO chains (
			id, name, chain_type, native_asset_id, rpc_urls, explorer_url,
			network_id, is_testnet, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, NOW()
		)
	`

	_, err := r.exec(ctx, query,
		chain.GetChainId(),
		chain.GetChainName(),
		chain.GetChainType(),
		chain.NativeAssetId,
		pq.Array(rpcUrls),
		chain.ExplorerUrl,
		chain.NetworkId,
		chain.IsTestnet,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("create chain: %w", err)
	}

	return nil
}

// GetChain retrieves a chain by ID
func (r *PostgresRepository) GetChain(ctx context.Context, id string) (*assetsv1.Chain, error) {
	query := `
		SELECT
			id, name, chain_type, native_asset_id, rpc_urls, explorer_url,
			network_id, is_testnet, metadata, created_at
		FROM chains
		WHERE id = $1
	`

	var chainId, name, chainType string
	var nativeAssetId, explorerUrl *string
	var rpcUrls []string
	var networkId *int64
	var isTestnet *bool
	var metadataJSON []byte
	var createdAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&chainId,
		&name,
		&chainType,
		&nativeAssetId,
		pq.Array(&rpcUrls),
		&explorerUrl,
		&networkId,
		&isTestnet,
		&metadataJSON,
		&createdAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("chain not found: %s", id)
		}
		return nil, fmt.Errorf("get chain: %w", err)
	}

	chain := &assetsv1.Chain{
		ChainId:       &chainId,
		ChainName:     &name,
		ChainType:     &chainType,
		NativeAssetId: nativeAssetId,
		ExplorerUrl:   explorerUrl,
		NetworkId:     networkId,
		IsTestnet:     isTestnet,
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		metadata := &structpb.Struct{}
		if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
		chain.Metadata = metadata
	}

	return chain, nil
}

// ListChains retrieves chains with optional filtering and pagination
func (r *PostgresRepository) ListChains(ctx context.Context, filter *ChainFilter) ([]*assetsv1.Chain, error) {
	query := `
		SELECT
			id, name, chain_type, native_asset_id, rpc_urls, explorer_url,
			network_id, is_testnet, metadata, created_at
		FROM chains
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.ChainType != nil {
			query += fmt.Sprintf(" AND chain_type = $%d", argPos)
			args = append(args, *filter.ChainType)
			argPos++
		}
		if filter.NativeAssetID != nil {
			query += fmt.Sprintf(" AND native_asset_id = $%d", argPos)
			args = append(args, *filter.NativeAssetID)
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
		return nil, fmt.Errorf("list chains: %w", err)
	}
	defer rows.Close()

	chains := []*assetsv1.Chain{}
	for rows.Next() {
		var chainId, name, chainType string
		var nativeAssetId, explorerUrl *string
		var rpcUrls []string
		var networkId *int64
		var isTestnet *bool
		var metadataJSON []byte
		var createdAt time.Time

		err := rows.Scan(
			&chainId,
			&name,
			&chainType,
			&nativeAssetId,
			pq.Array(&rpcUrls),
			&explorerUrl,
			&networkId,
			&isTestnet,
			&metadataJSON,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan chain: %w", err)
		}

		chain := &assetsv1.Chain{
			ChainId:       &chainId,
			ChainName:     &name,
			ChainType:     &chainType,
			NativeAssetId: nativeAssetId,
			ExplorerUrl:   explorerUrl,
			NetworkId:     networkId,
			IsTestnet:     isTestnet,
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			metadata := &structpb.Struct{}
			if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
			chain.Metadata = metadata
		}

		chains = append(chains, chain)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chains: %w", err)
	}

	return chains, nil
}
