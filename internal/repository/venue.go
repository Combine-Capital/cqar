package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"github.com/jackc/pgx/v5"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateVenue inserts a new venue into the database
func (r *PostgresRepository) CreateVenue(ctx context.Context, venue *venuesv1.Venue) error {
	// Validate that venue_id is provided
	if venue.VenueId == nil || *venue.VenueId == "" {
		return fmt.Errorf("venue_id is required")
	}

	// Validate chain_id if provided
	if venue.ChainId != nil && *venue.ChainId != "" {
		var exists bool
		err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM chains WHERE id = $1)", *venue.ChainId).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check chain_id existence: %w", err)
		}
		if !exists {
			return fmt.Errorf("chain_id does not exist: %s", *venue.ChainId)
		}
	}

	// Set timestamp
	now := timestamppb.Now()
	if venue.CreatedAt == nil {
		venue.CreatedAt = now
	}

	// Convert metadata to JSON if present
	var metadataJSON interface{}
	if venue.Metadata != nil {
		metadataBytes, err := venue.Metadata.MarshalJSON()
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
		metadataJSON = metadataBytes
	}

	query := `
		INSERT INTO venues (
			id, name, venue_type, chain_id, protocol_address,
			website_url, api_endpoint, is_active, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`

	_, err := r.exec(ctx, query,
		venue.GetVenueId(),
		venue.GetName(),
		venue.GetVenueType().String(),
		venue.ChainId,
		venue.ProtocolAddress,
		venue.WebsiteUrl,
		venue.ApiEndpoint,
		venue.GetIsActive(),
		metadataJSON,
		venue.CreatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create venue: %w", err)
	}

	return nil
}

// GetVenue retrieves a venue by ID
func (r *PostgresRepository) GetVenue(ctx context.Context, id string) (*venuesv1.Venue, error) {
	query := `
		SELECT
			id, name, venue_type, chain_id, protocol_address,
			website_url, api_endpoint, is_active, metadata, created_at
		FROM venues
		WHERE id = $1
	`

	var venueId, name, venueType string
	var chainId, protocolAddress, websiteUrl, apiEndpoint *string
	var isActive bool
	var metadataJSON []byte
	var createdAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&venueId,
		&name,
		&venueType,
		&chainId,
		&protocolAddress,
		&websiteUrl,
		&apiEndpoint,
		&isActive,
		&metadataJSON,
		&createdAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("venue not found: %s", id)
		}
		return nil, fmt.Errorf("get venue: %w", err)
	}

	// Parse venue type enum
	venueTypeEnum, err := parseVenueType(venueType)
	if err != nil {
		return nil, fmt.Errorf("parse venue type: %w", err)
	}

	venue := &venuesv1.Venue{
		VenueId:         &venueId,
		Name:            &name,
		VenueType:       &venueTypeEnum,
		ChainId:         chainId,
		ProtocolAddress: protocolAddress,
		WebsiteUrl:      websiteUrl,
		ApiEndpoint:     apiEndpoint,
		IsActive:        &isActive,
		CreatedAt:       timestamppb.New(createdAt),
	}

	// Unmarshal metadata if present
	if len(metadataJSON) > 0 {
		metadata := &structpb.Struct{}
		if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
		venue.Metadata = metadata
	}

	return venue, nil
}

// ListVenues retrieves venues with optional filtering and pagination
func (r *PostgresRepository) ListVenues(ctx context.Context, filter *VenueFilter) ([]*venuesv1.Venue, error) {
	query := `
		SELECT
			id, name, venue_type, chain_id, protocol_address,
			website_url, api_endpoint, is_active, metadata, created_at
		FROM venues
		WHERE 1=1
	`

	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.VenueType != nil {
			query += fmt.Sprintf(" AND venue_type = $%d", argPos)
			args = append(args, *filter.VenueType)
			argPos++
		}

		if filter.ChainID != nil {
			query += fmt.Sprintf(" AND chain_id = $%d", argPos)
			args = append(args, *filter.ChainID)
			argPos++
		}

		if filter.IsActive != nil {
			query += fmt.Sprintf(" AND is_active = $%d", argPos)
			args = append(args, *filter.IsActive)
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
		return nil, fmt.Errorf("list venues: %w", err)
	}
	defer rows.Close()

	venues := []*venuesv1.Venue{}
	for rows.Next() {
		var venueId, name, venueType string
		var chainId, protocolAddress, websiteUrl, apiEndpoint *string
		var isActive bool
		var metadataJSON []byte
		var createdAt time.Time

		err := rows.Scan(
			&venueId,
			&name,
			&venueType,
			&chainId,
			&protocolAddress,
			&websiteUrl,
			&apiEndpoint,
			&isActive,
			&metadataJSON,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("scan venue: %w", err)
		}

		// Parse venue type enum
		venueTypeEnum, err := parseVenueType(venueType)
		if err != nil {
			return nil, fmt.Errorf("parse venue type: %w", err)
		}

		venue := &venuesv1.Venue{
			VenueId:         &venueId,
			Name:            &name,
			VenueType:       &venueTypeEnum,
			ChainId:         chainId,
			ProtocolAddress: protocolAddress,
			WebsiteUrl:      websiteUrl,
			ApiEndpoint:     apiEndpoint,
			IsActive:        &isActive,
			CreatedAt:       timestamppb.New(createdAt),
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			metadata := &structpb.Struct{}
			if err := metadata.UnmarshalJSON(metadataJSON); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
			venue.Metadata = metadata
		}

		venues = append(venues, venue)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate venues: %w", err)
	}

	return venues, nil
}

// parseVenueType converts a string venue type to the protobuf enum
func parseVenueType(s string) (venuesv1.VenueType, error) {
	switch strings.ToUpper(s) {
	case "CEX", "VENUE_TYPE_CEX":
		return venuesv1.VenueType_VENUE_TYPE_CEX, nil
	case "DEX", "VENUE_TYPE_DEX":
		return venuesv1.VenueType_VENUE_TYPE_DEX, nil
	case "DEX_AGGREGATOR", "VENUE_TYPE_DEX_AGGREGATOR":
		return venuesv1.VenueType_VENUE_TYPE_DEX_AGGREGATOR, nil
	case "BRIDGE", "VENUE_TYPE_BRIDGE":
		return venuesv1.VenueType_VENUE_TYPE_BRIDGE, nil
	case "LENDING", "VENUE_TYPE_LENDING":
		return venuesv1.VenueType_VENUE_TYPE_LENDING, nil
	default:
		return venuesv1.VenueType_VENUE_TYPE_UNSPECIFIED, fmt.Errorf("unknown venue type: %s", s)
	}
}
