package repository

import (
	"context"
	"fmt"
	"time"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateAssetGroup creates a new asset group
func (r *PostgresRepository) CreateAssetGroup(ctx context.Context, group *assetsv1.AssetGroup) error {
	// Generate ID if not provided
	if group.GroupId == nil || *group.GroupId == "" {
		id := uuid.New().String()
		group.GroupId = &id
	}

	// Set timestamps
	now := timestamppb.Now()
	if group.CreatedAt == nil {
		group.CreatedAt = now
	}
	group.UpdatedAt = now

	query := `
		INSERT INTO asset_groups (
			id, name, description, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`

	_, err := r.exec(ctx, query,
		group.GetGroupId(),
		group.GetName(),
		group.GetDescription(),
		group.CreatedAt.AsTime(),
		group.UpdatedAt.AsTime(),
	)

	if err != nil {
		return fmt.Errorf("create asset group: %w", err)
	}

	return nil
}

// GetAssetGroup retrieves an asset group by ID with its members
func (r *PostgresRepository) GetAssetGroup(ctx context.Context, id string) (*assetsv1.AssetGroup, error) {
	// Get group info
	query := `
		SELECT
			id, name, description, created_at, updated_at
		FROM asset_groups
		WHERE id = $1
	`

	var groupId, name, description string
	var createdAt, updatedAt time.Time

	err := r.queryRow(ctx, query, id).Scan(
		&groupId,
		&name,
		&description,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get asset group: %w", err)
	}

	group := &assetsv1.AssetGroup{
		GroupId:     &groupId,
		Name:        ptrIfNotEmpty(name),
		Description: ptrIfNotEmpty(description),
		CreatedAt:   timestamppb.New(createdAt),
		UpdatedAt:   timestamppb.New(updatedAt),
	}

	// Get group members
	members, err := r.getGroupMembers(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get group members: %w", err)
	}
	group.Members = members

	return group, nil
}

// GetAssetGroupByName retrieves an asset group by name
func (r *PostgresRepository) GetAssetGroupByName(ctx context.Context, name string) (*assetsv1.AssetGroup, error) {
	// Get group info
	query := `
		SELECT
			id, name, description, created_at, updated_at
		FROM asset_groups
		WHERE name = $1
	`

	var groupId, groupName, description string
	var createdAt, updatedAt time.Time

	err := r.queryRow(ctx, query, name).Scan(
		&groupId,
		&groupName,
		&description,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("get asset group by name: %w", err)
	}

	group := &assetsv1.AssetGroup{
		GroupId:     &groupId,
		Name:        ptrIfNotEmpty(groupName),
		Description: ptrIfNotEmpty(description),
		CreatedAt:   timestamppb.New(createdAt),
		UpdatedAt:   timestamppb.New(updatedAt),
	}

	// Get group members
	members, err := r.getGroupMembers(ctx, groupId)
	if err != nil {
		return nil, fmt.Errorf("get group members: %w", err)
	}
	group.Members = members

	return group, nil
}

// getGroupMembers retrieves all members of a group
func (r *PostgresRepository) getGroupMembers(ctx context.Context, groupId string) ([]*assetsv1.AssetGroupMember, error) {
	query := `
		SELECT
			id, group_id, asset_id, weight, added_at
		FROM group_members
		WHERE group_id = $1
		ORDER BY added_at ASC
	`

	rows, err := r.query(ctx, query, groupId)
	if err != nil {
		return nil, fmt.Errorf("query group members: %w", err)
	}
	defer rows.Close()

	var members []*assetsv1.AssetGroupMember
	for rows.Next() {
		var memberId, memberGroupId, assetId string
		var weight float64
		var addedAt time.Time

		err := rows.Scan(
			&memberId,
			&memberGroupId,
			&assetId,
			&weight,
			&addedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan group member row: %w", err)
		}

		member := &assetsv1.AssetGroupMember{
			MemberId: &memberId,
			GroupId:  &memberGroupId,
			AssetId:  &assetId,
			Weight:   &weight,
			AddedAt:  timestamppb.New(addedAt),
		}

		members = append(members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group members: %w", err)
	}

	return members, nil
}

// AddAssetToGroup adds an asset to a group with an optional weight
func (r *PostgresRepository) AddAssetToGroup(ctx context.Context, groupID, assetID string, weight float64) error {
	// Validate that the group exists
	var exists bool
	err := r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM asset_groups WHERE id = $1)", groupID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check group exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("asset group not found: %s", groupID)
	}

	// Validate that the asset exists
	err = r.queryRow(ctx, "SELECT EXISTS(SELECT 1 FROM assets WHERE id = $1)", assetID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check asset exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("asset not found: %s", assetID)
	}

	// If weight is 0, default to 1.0
	if weight == 0 {
		weight = 1.0
	}

	memberID := uuid.New().String()
	query := `
		INSERT INTO group_members (
			id, group_id, asset_id, weight, added_at
		) VALUES (
			$1, $2, $3, $4, $5
		)
	`

	_, err = r.exec(ctx, query,
		memberID,
		groupID,
		assetID,
		weight,
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("add asset to group: %w", err)
	}

	return nil
}

// RemoveAssetFromGroup removes an asset from a group
func (r *PostgresRepository) RemoveAssetFromGroup(ctx context.Context, groupID, assetID string) error {
	query := `
		DELETE FROM group_members
		WHERE group_id = $1 AND asset_id = $2
	`

	result, err := r.exec(ctx, query, groupID, assetID)
	if err != nil {
		return fmt.Errorf("remove asset from group: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("asset %s not found in group %s", assetID, groupID)
	}

	return nil
}

// ListAssetGroups retrieves a list of asset groups with optional filtering and pagination
func (r *PostgresRepository) ListAssetGroups(ctx context.Context, filter *AssetGroupFilter) ([]*assetsv1.AssetGroup, error) {
	// Build query with filters
	query := `
		SELECT
			id, name, description, created_at, updated_at
		FROM asset_groups
		WHERE 1=1
	`
	args := []interface{}{}
	argPos := 1

	// Apply filters
	if filter != nil {
		if filter.Name != nil {
			query += fmt.Sprintf(" AND name = $%d", argPos)
			args = append(args, *filter.Name)
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
		return nil, fmt.Errorf("list asset groups: %w", err)
	}
	defer rows.Close()

	var groups []*assetsv1.AssetGroup
	for rows.Next() {
		var groupId, name, description string
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&groupId,
			&name,
			&description,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan asset group row: %w", err)
		}

		group := &assetsv1.AssetGroup{
			GroupId:     &groupId,
			Name:        ptrIfNotEmpty(name),
			Description: ptrIfNotEmpty(description),
			CreatedAt:   timestamppb.New(createdAt),
			UpdatedAt:   timestamppb.New(updatedAt),
		}

		// Get group members
		members, err := r.getGroupMembers(ctx, groupId)
		if err != nil {
			return nil, fmt.Errorf("get group members: %w", err)
		}
		group.Members = members

		groups = append(groups, group)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate asset groups: %w", err)
	}

	return groups, nil
}
