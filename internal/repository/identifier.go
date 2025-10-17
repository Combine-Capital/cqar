package repository

import (
	"context"
	"fmt"
	"time"

	identifiersv1 "github.com/Combine-Capital/cqc/gen/go/cqc/identifiers/v1"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateIdentifier inserts a new identifier mapping
// Validates foreign key existence based on entity_type
func (r *PostgresRepository) CreateIdentifier(ctx context.Context, identifier *identifiersv1.Identifier) error {
	// Generate ID if not provided
	if identifier.Id == nil || *identifier.Id == "" {
		id := uuid.New().String()
		identifier.Id = &id
	}

	// Validate foreign key existence based on entity_type
	entityType := identifier.GetEntityType()
	switch entityType {
	case "ASSET":
		if identifier.AssetId != nil && *identifier.AssetId != "" {
			var exists bool
			err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", *identifier.AssetId).Scan(&exists)
			if err != nil {
				return fmt.Errorf("check asset_id existence: %w", err)
			}
			if !exists {
				return fmt.Errorf("asset_id does not exist: %s", *identifier.AssetId)
			}
		}
	case "INSTRUMENT":
		if identifier.InstrumentId != nil && *identifier.InstrumentId != "" {
			var exists bool
			err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM instruments WHERE id = $1)", *identifier.InstrumentId).Scan(&exists)
			if err != nil {
				return fmt.Errorf("check instrument_id existence: %w", err)
			}
			if !exists {
				return fmt.Errorf("instrument_id does not exist: %s", *identifier.InstrumentId)
			}
		}
	case "MARKET":
		if identifier.MarketId != nil && *identifier.MarketId != "" {
			var exists bool
			err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM markets WHERE id = $1)", *identifier.MarketId).Scan(&exists)
			if err != nil {
				return fmt.Errorf("check market_id existence: %w", err)
			}
			if !exists {
				return fmt.Errorf("market_id does not exist: %s", *identifier.MarketId)
			}
		}
	}

	// Set timestamps
	now := timestamppb.Now()
	if identifier.CreatedAt == nil {
		identifier.CreatedAt = now
	}
	identifier.UpdatedAt = now

	var metadataJSON []byte
	var err error
	if identifier.Metadata != nil {
		metadataJSON, err = identifier.Metadata.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	var verifiedAtTime *time.Time
	if identifier.VerifiedAt != nil {
		t := identifier.VerifiedAt.AsTime()
		verifiedAtTime = &t
	}

	query := `
		INSERT INTO identifiers (
			id, entity_type, asset_id, instrument_id, market_id,
			source, external_id, is_primary, metadata, verified_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err = r.exec(ctx, query,
		identifier.GetId(),
		identifier.EntityType,
		identifier.AssetId,
		identifier.InstrumentId,
		identifier.MarketId,
		identifier.Source,
		identifier.ExternalId,
		identifier.GetIsPrimary(),
		metadataJSON,
		verifiedAtTime,
		identifier.CreatedAt.AsTime(),
		identifier.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create identifier: %w", err)
	}

	return nil
}

// GetIdentifier retrieves an identifier by ID
func (r *PostgresRepository) GetIdentifier(ctx context.Context, id string) (*identifiersv1.Identifier, error) {
	query := `
		SELECT id, entity_type, asset_id, instrument_id, market_id,
		       source, external_id, is_primary, metadata, verified_at,
		       created_at, updated_at
		FROM identifiers
		WHERE id = $1
	`

	var identifier identifiersv1.Identifier
	var createdAt, updatedAt time.Time
	var verifiedAt *time.Time
	var metadataJSON []byte

	err := r.queryRow(ctx, query, id).Scan(
		&identifier.Id,
		&identifier.EntityType,
		&identifier.AssetId,
		&identifier.InstrumentId,
		&identifier.MarketId,
		&identifier.Source,
		&identifier.ExternalId,
		&identifier.IsPrimary,
		&metadataJSON,
		&verifiedAt,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("identifier not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get identifier: %w", err)
	}

	if verifiedAt != nil {
		identifier.VerifiedAt = timestamppb.New(*verifiedAt)
	}

	if metadataJSON != nil {
		identifier.Metadata = &structpb.Struct{}
		if err := identifier.Metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	identifier.CreatedAt = timestamppb.New(createdAt)
	identifier.UpdatedAt = timestamppb.New(updatedAt)

	return &identifier, nil
}

// ResolveIdentifierByExternalID retrieves an identifier by source and external_id
// Returns the entity_type and corresponding ID (asset_id, instrument_id, or market_id)
func (r *PostgresRepository) ResolveIdentifierByExternalID(ctx context.Context, source, externalID string) (*identifiersv1.Identifier, error) {
	query := `
		SELECT id, entity_type, asset_id, instrument_id, market_id,
		       source, external_id, is_primary, metadata, verified_at,
		       created_at, updated_at
		FROM identifiers
		WHERE source = $1 AND external_id = $2
	`

	var identifier identifiersv1.Identifier
	var createdAt, updatedAt time.Time
	var verifiedAt *time.Time
	var metadataJSON []byte

	err := r.queryRow(ctx, query, source, externalID).Scan(
		&identifier.Id,
		&identifier.EntityType,
		&identifier.AssetId,
		&identifier.InstrumentId,
		&identifier.MarketId,
		&identifier.Source,
		&identifier.ExternalId,
		&identifier.IsPrimary,
		&metadataJSON,
		&verifiedAt,
		&createdAt,
		&updatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("identifier not found for source %s external_id %s", source, externalID)
	}
	if err != nil {
		return nil, fmt.Errorf("resolve identifier by external id: %w", err)
	}

	if verifiedAt != nil {
		identifier.VerifiedAt = timestamppb.New(*verifiedAt)
	}

	if metadataJSON != nil {
		identifier.Metadata = &structpb.Struct{}
		if err := identifier.Metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}

	identifier.CreatedAt = timestamppb.New(createdAt)
	identifier.UpdatedAt = timestamppb.New(updatedAt)

	return &identifier, nil
}

// ListIdentifiersByEntity retrieves all identifiers for a given entity
func (r *PostgresRepository) ListIdentifiersByEntity(ctx context.Context, entityType, entityID string) ([]*identifiersv1.Identifier, error) {
	var query string
	var args []interface{}

	switch entityType {
	case "ASSET":
		query = `
			SELECT id, entity_type, asset_id, instrument_id, market_id,
			       source, external_id, is_primary, metadata, verified_at,
			       created_at, updated_at
			FROM identifiers
			WHERE entity_type = 'ASSET' AND asset_id = $1
			ORDER BY source, is_primary DESC
		`
		args = []interface{}{entityID}
	case "INSTRUMENT":
		query = `
			SELECT id, entity_type, asset_id, instrument_id, market_id,
			       source, external_id, is_primary, metadata, verified_at,
			       created_at, updated_at
			FROM identifiers
			WHERE entity_type = 'INSTRUMENT' AND instrument_id = $1
			ORDER BY source, is_primary DESC
		`
		args = []interface{}{entityID}
	case "MARKET":
		query = `
			SELECT id, entity_type, asset_id, instrument_id, market_id,
			       source, external_id, is_primary, metadata, verified_at,
			       created_at, updated_at
			FROM identifiers
			WHERE entity_type = 'MARKET' AND market_id = $1
			ORDER BY source, is_primary DESC
		`
		args = []interface{}{entityID}
	default:
		return nil, fmt.Errorf("invalid entity_type: %s", entityType)
	}

	rows, err := r.query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list identifiers by entity: %w", err)
	}
	defer rows.Close()

	var identifiers []*identifiersv1.Identifier
	for rows.Next() {
		var identifier identifiersv1.Identifier
		var createdAt, updatedAt time.Time
		var verifiedAt *time.Time
		var metadataJSON []byte

		err := rows.Scan(
			&identifier.Id,
			&identifier.EntityType,
			&identifier.AssetId,
			&identifier.InstrumentId,
			&identifier.MarketId,
			&identifier.Source,
			&identifier.ExternalId,
			&identifier.IsPrimary,
			&metadataJSON,
			&verifiedAt,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan identifier: %w", err)
		}

		if verifiedAt != nil {
			identifier.VerifiedAt = timestamppb.New(*verifiedAt)
		}

		if metadataJSON != nil {
			identifier.Metadata = &structpb.Struct{}
			if err := identifier.Metadata.UnmarshalJSON(metadataJSON); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		identifier.CreatedAt = timestamppb.New(createdAt)
		identifier.UpdatedAt = timestamppb.New(updatedAt)

		identifiers = append(identifiers, &identifier)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return identifiers, nil
}

// ListIdentifiersBySource retrieves all identifiers for a given source
func (r *PostgresRepository) ListIdentifiersBySource(ctx context.Context, source string) ([]*identifiersv1.Identifier, error) {
	query := `
		SELECT id, entity_type, asset_id, instrument_id, market_id,
		       source, external_id, is_primary, metadata, verified_at,
		       created_at, updated_at
		FROM identifiers
		WHERE source = $1
		ORDER BY entity_type, external_id
	`

	rows, err := r.query(ctx, query, source)
	if err != nil {
		return nil, fmt.Errorf("list identifiers by source: %w", err)
	}
	defer rows.Close()

	var identifiers []*identifiersv1.Identifier
	for rows.Next() {
		var identifier identifiersv1.Identifier
		var createdAt, updatedAt time.Time
		var verifiedAt *time.Time
		var metadataJSON []byte

		err := rows.Scan(
			&identifier.Id,
			&identifier.EntityType,
			&identifier.AssetId,
			&identifier.InstrumentId,
			&identifier.MarketId,
			&identifier.Source,
			&identifier.ExternalId,
			&identifier.IsPrimary,
			&metadataJSON,
			&verifiedAt,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan identifier: %w", err)
		}

		if verifiedAt != nil {
			identifier.VerifiedAt = timestamppb.New(*verifiedAt)
		}

		if metadataJSON != nil {
			identifier.Metadata = &structpb.Struct{}
			if err := identifier.Metadata.UnmarshalJSON(metadataJSON); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}

		identifier.CreatedAt = timestamppb.New(createdAt)
		identifier.UpdatedAt = timestamppb.New(updatedAt)

		identifiers = append(identifiers, &identifier)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return identifiers, nil
}
