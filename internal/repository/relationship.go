package repository

import (
	"context"
	"fmt"
	"time"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateAssetRelationship creates a new asset relationship
func (r *PostgresRepository) CreateAssetRelationship(ctx context.Context, relationship *assetsv1.AssetRelationship) error {
	// Validate that the from_asset exists
	var fromAssetExists bool
	err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", relationship.GetFromAssetId()).Scan(&fromAssetExists)
	if err != nil {
		return fmt.Errorf("check from_asset exists: %w", err)
	}
	if !fromAssetExists {
		return fmt.Errorf("from_asset not found: %s", relationship.GetFromAssetId())
	}

	// Validate that the to_asset exists
	var toAssetExists bool
	err = r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", relationship.GetToAssetId()).Scan(&toAssetExists)
	if err != nil {
		return fmt.Errorf("check to_asset exists: %w", err)
	}
	if !toAssetExists {
		return fmt.Errorf("to_asset not found: %s", relationship.GetToAssetId())
	}

	// Prevent self-referential relationships
	if relationship.GetFromAssetId() == relationship.GetToAssetId() {
		return fmt.Errorf("self-referential relationships not allowed")
	}

	// Generate ID if not provided
	if relationship.RelationshipId == nil || *relationship.RelationshipId == "" {
		id := uuid.New().String()
		relationship.RelationshipId = &id
	}

	// Set timestamps
	now := timestamppb.Now()
	if relationship.CreatedAt == nil {
		relationship.CreatedAt = now
	}
	relationship.UpdatedAt = now

	query := `
		INSERT INTO relationships (
			id, from_asset_id, to_asset_id, relationship_type, conversion_rate,
			protocol, description, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	// Convert enum to string for database storage
	relationshipTypeStr := relationship.GetRelationshipType().String()

	_, err = r.exec(ctx, query,
		relationship.GetRelationshipId(),
		relationship.GetFromAssetId(),
		relationship.GetToAssetId(),
		relationshipTypeStr,
		nullableString(relationship.ConversionRate),
		nullableString(relationship.Protocol),
		nullableString(relationship.Description),
		relationship.CreatedAt.AsTime(),
		relationship.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create asset relationship: %w", err)
	}

	return nil
}

// GetAssetRelationship retrieves a relationship by ID
func (r *PostgresRepository) GetAssetRelationship(ctx context.Context, id string) (*assetsv1.AssetRelationship, error) {
	query := `
		SELECT
			id, from_asset_id, to_asset_id, relationship_type, conversion_rate,
			protocol, description, created_at, updated_at
		FROM relationships
		WHERE id = $1
	`

	var relationshipId, fromAssetId, toAssetId, relationshipTypeStr string
	var conversionRate, protocol, description *string
	var createdAt, updatedAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&relationshipId,
		&fromAssetId,
		&toAssetId,
		&relationshipTypeStr,
		&conversionRate,
		&protocol,
		&description,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get asset relationship: %w", err)
	}

	// Parse relationship type enum
	relationshipType := parseRelationshipType(relationshipTypeStr)

	relationship := &assetsv1.AssetRelationship{
		RelationshipId:   &relationshipId,
		FromAssetId:      &fromAssetId,
		ToAssetId:        &toAssetId,
		RelationshipType: &relationshipType,
		ConversionRate:   conversionRate,
		Protocol:         protocol,
		Description:      description,
		CreatedAt:        timestamppb.New(createdAt),
		UpdatedAt:        timestamppb.New(updatedAt),
	}

	return relationship, nil
}

// ListAssetRelationships retrieves a list of relationships with optional filtering
func (r *PostgresRepository) ListAssetRelationships(ctx context.Context, filter *RelationshipFilter) ([]*assetsv1.AssetRelationship, error) {
	// Build query with filters
	query := `
		SELECT
			id, from_asset_id, to_asset_id, relationship_type, conversion_rate,
			protocol, description, created_at, updated_at
		FROM relationships
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.FromAssetID != nil {
			query += fmt.Sprintf(" AND from_asset_id = $%d", argPos)
			args = append(args, *filter.FromAssetID)
			argPos++
		}

		if filter.ToAssetID != nil {
			query += fmt.Sprintf(" AND to_asset_id = $%d", argPos)
			args = append(args, *filter.ToAssetID)
			argPos++
		}

		if filter.RelationshipType != nil {
			query += fmt.Sprintf(" AND relationship_type = $%d", argPos)
			args = append(args, *filter.RelationshipType)
			argPos++
		}

		if filter.Protocol != nil {
			query += fmt.Sprintf(" AND protocol = $%d", argPos)
			args = append(args, *filter.Protocol)
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
		return nil, fmt.Errorf("list asset relationships: %w", err)
	}
	defer rows.Close()

	var relationships []*assetsv1.AssetRelationship
	for rows.Next() {
		var relationshipId, fromAssetId, toAssetId, relationshipTypeStr string
		var conversionRate, protocol, description *string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&relationshipId,
			&fromAssetId,
			&toAssetId,
			&relationshipTypeStr,
			&conversionRate,
			&protocol,
			&description,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan relationship row: %w", err)
		}

		// Parse relationship type enum
		relationshipType := parseRelationshipType(relationshipTypeStr)

		relationship := &assetsv1.AssetRelationship{
			RelationshipId:   &relationshipId,
			FromAssetId:      &fromAssetId,
			ToAssetId:        &toAssetId,
			RelationshipType: &relationshipType,
			ConversionRate:   conversionRate,
			Protocol:         protocol,
			Description:      description,
			CreatedAt:        timestamppb.New(createdAt),
			UpdatedAt:        timestamppb.New(updatedAt),
		}

		relationships = append(relationships, relationship)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relationships: %w", err)
	}

	return relationships, nil
}

// parseRelationshipType converts database string to RelationshipType enum
func parseRelationshipType(s string) assetsv1.RelationshipType {
	switch s {
	case "RELATIONSHIP_TYPE_WRAPS", "WRAPS":
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_WRAPS
	case "RELATIONSHIP_TYPE_BRIDGES", "BRIDGES":
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_BRIDGES
	case "RELATIONSHIP_TYPE_STAKES", "STAKES":
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_STAKES
	case "RELATIONSHIP_TYPE_SYNTHETIC_OF", "SYNTHETIC_OF":
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_SYNTHETIC_OF
	case "RELATIONSHIP_TYPE_LIQUIDITY_PAIR", "LIQUIDITY_PAIR", "LP_TOKEN":
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_LIQUIDITY_PAIR
	case "RELATIONSHIP_TYPE_MIGRATES_TO", "MIGRATES_TO":
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_MIGRATES_TO
	case "RELATIONSHIP_TYPE_FORKS_FROM", "FORKS_FROM":
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_FORKS_FROM
	case "RELATIONSHIP_TYPE_REBASES_WITH", "REBASES_WITH", "REBASES":
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_REBASES_WITH
	default:
		return assetsv1.RelationshipType_RELATIONSHIP_TYPE_UNSPECIFIED
	}
}
