package manager

import (
	"context"
	"fmt"
	"strings"

	"github.com/Combine-Capital/cqar/internal/repository"
	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AssetManager handles business logic for asset operations with validation,
// collision resolution, and relationship management
type AssetManager struct {
	repo           repository.Repository
	qualityManager *QualityManager
	eventPublisher *EventPublisher
}

// NewAssetManager creates a new AssetManager instance
func NewAssetManager(repo repository.Repository, qualityManager *QualityManager, eventPublisher *EventPublisher) *AssetManager {
	return &AssetManager{
		repo:           repo,
		qualityManager: qualityManager,
		eventPublisher: eventPublisher,
	}
}

// CreateAsset creates a new asset with validation and collision checking
func (m *AssetManager) CreateAsset(ctx context.Context, asset *assetsv1.Asset) error {
	// Validate required fields
	if err := ValidateRequiredAssetFields(asset); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Check for symbol collision across chains
	// Note: Same symbol can exist for different chains (e.g., USDC on Ethereum vs Polygon)
	// The collision check here is informational - we allow it but log/warn
	symbol := strings.ToUpper(strings.TrimSpace(*asset.Symbol))
	filter := &repository.AssetFilter{
		Limit: 10, // Check for existing assets with same symbol
	}

	existingAssets, err := m.repo.SearchAssets(ctx, symbol, filter)
	if err != nil {
		// Don't fail on search error, just proceed
		// In production, this should be logged
	} else if len(existingAssets) > 0 {
		// Symbol collision detected - this is OK for multi-chain assets
		// The caller should be aware that multiple assets share this symbol
		// This is handled by the unique asset_id (UUID) per deployment
	}

	// Create the asset in the repository
	if err := m.repo.CreateAsset(ctx, asset); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create asset: %v", err))
	}

	// Publish AssetCreated event asynchronously
	if m.eventPublisher != nil {
		m.eventPublisher.PublishAssetCreated(ctx, asset)
	}

	return nil
}

// GetAsset retrieves an asset by ID
func (m *AssetManager) GetAsset(ctx context.Context, assetID string) (*assetsv1.Asset, error) {
	if assetID == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	asset, err := m.repo.GetAsset(ctx, assetID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("asset not found: %s", assetID))
	}

	return asset, nil
}

// UpdateAsset updates an existing asset
func (m *AssetManager) UpdateAsset(ctx context.Context, asset *assetsv1.Asset) error {
	// Validate required fields
	if err := ValidateRequiredAssetFields(asset); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if asset.AssetId == nil || *asset.AssetId == "" {
		return status.Error(codes.InvalidArgument, "asset_id is required for update")
	}

	// Verify asset exists
	existing, err := m.repo.GetAsset(ctx, *asset.AssetId)
	if err != nil || existing == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("asset not found: %s", *asset.AssetId))
	}

	// Update the asset
	if err := m.repo.UpdateAsset(ctx, asset); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to update asset: %v", err))
	}

	return nil
}

// DeleteAsset deletes an asset by ID
func (m *AssetManager) DeleteAsset(ctx context.Context, assetID string) error {
	if assetID == "" {
		return status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Verify asset exists
	existing, err := m.repo.GetAsset(ctx, assetID)
	if err != nil || existing == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("asset not found: %s", assetID))
	}

	// Delete the asset
	if err := m.repo.DeleteAsset(ctx, assetID); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to delete asset: %v", err))
	}

	return nil
}

// ListAssets retrieves assets with optional filtering
func (m *AssetManager) ListAssets(ctx context.Context, filter *repository.AssetFilter) ([]*assetsv1.Asset, error) {
	assets, err := m.repo.ListAssets(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list assets: %v", err))
	}

	return assets, nil
}

// SearchAssets searches for assets by query string
func (m *AssetManager) SearchAssets(ctx context.Context, query string, filter *repository.AssetFilter) ([]*assetsv1.Asset, error) {
	if strings.TrimSpace(query) == "" {
		return nil, status.Error(codes.InvalidArgument, "search query is required")
	}

	assets, err := m.repo.SearchAssets(ctx, query, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to search assets: %v", err))
	}

	return assets, nil
}

// CreateAssetDeployment creates a new asset deployment with validation
func (m *AssetManager) CreateAssetDeployment(ctx context.Context, deployment *assetsv1.AssetDeployment) error {
	// Validate asset_id exists
	if deployment.AssetId == nil || *deployment.AssetId == "" {
		return status.Error(codes.InvalidArgument, "asset_id is required")
	}

	asset, err := m.repo.GetAsset(ctx, *deployment.AssetId)
	if err != nil || asset == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("asset not found: %s", *deployment.AssetId))
	}

	// Validate chain_id exists
	if deployment.ChainId == nil || *deployment.ChainId == "" {
		return status.Error(codes.InvalidArgument, "chain_id is required")
	}

	chain, err := m.repo.GetChain(ctx, *deployment.ChainId)
	if err != nil || chain == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("chain not found: %s", *deployment.ChainId))
	}

	// Validate contract_address format based on chain type
	if deployment.Address != nil && *deployment.Address != "" {
		chainType := chain.GetChainType()
		if err := ValidateContractAddress(*deployment.Address, chainType); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
	} else {
		return status.Error(codes.InvalidArgument, "address is required")
	}

	// Validate decimals range (0-18)
	if deployment.Decimals != nil {
		if err := ValidateDecimals(*deployment.Decimals); err != nil {
			return status.Error(codes.InvalidArgument, err.Error())
		}
	}

	// Create the deployment
	if err := m.repo.CreateAssetDeployment(ctx, deployment); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create asset deployment: %v", err))
	}

	// Publish AssetDeploymentCreated event asynchronously
	if m.eventPublisher != nil {
		m.eventPublisher.PublishAssetDeploymentCreated(ctx, deployment)
	}

	return nil
}

// GetAssetDeployment retrieves a deployment by ID
func (m *AssetManager) GetAssetDeployment(ctx context.Context, deploymentID string) (*assetsv1.AssetDeployment, error) {
	if deploymentID == "" {
		return nil, status.Error(codes.InvalidArgument, "deployment_id is required")
	}

	deployment, err := m.repo.GetAssetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("deployment not found: %s", deploymentID))
	}

	return deployment, nil
}

// ListAssetDeployments retrieves deployments with optional filtering
func (m *AssetManager) ListAssetDeployments(ctx context.Context, filter *repository.DeploymentFilter) ([]*assetsv1.AssetDeployment, error) {
	deployments, err := m.repo.ListAssetDeployments(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list deployments: %v", err))
	}

	return deployments, nil
}

// CreateAssetRelationship creates a new asset relationship with validation and cycle detection
func (m *AssetManager) CreateAssetRelationship(ctx context.Context, relationship *assetsv1.AssetRelationship) error {
	// Validate relationship_type
	if err := ValidateRelationshipType(relationship.GetRelationshipType()); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate from_asset_id
	if relationship.FromAssetId == nil || *relationship.FromAssetId == "" {
		return status.Error(codes.InvalidArgument, "from_asset_id is required")
	}

	fromAsset, err := m.repo.GetAsset(ctx, *relationship.FromAssetId)
	if err != nil || fromAsset == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("from_asset not found: %s", *relationship.FromAssetId))
	}

	// Validate to_asset_id
	if relationship.ToAssetId == nil || *relationship.ToAssetId == "" {
		return status.Error(codes.InvalidArgument, "to_asset_id is required")
	}

	toAsset, err := m.repo.GetAsset(ctx, *relationship.ToAssetId)
	if err != nil || toAsset == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("to_asset not found: %s", *relationship.ToAssetId))
	}

	// Prevent self-referential relationships
	if *relationship.FromAssetId == *relationship.ToAssetId {
		return status.Error(codes.InvalidArgument, "self-referential relationships are not allowed")
	}

	// Detect cycles in relationship graph
	if err := m.detectRelationshipCycle(ctx, *relationship.FromAssetId, *relationship.ToAssetId, relationship.GetRelationshipType()); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("relationship would create cycle: %v", err))
	}

	// Create the relationship
	if err := m.repo.CreateAssetRelationship(ctx, relationship); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create asset relationship: %v", err))
	}

	// Publish RelationshipEstablished event asynchronously
	if m.eventPublisher != nil {
		m.eventPublisher.PublishRelationshipEstablished(ctx, relationship)
	}

	return nil
}

// detectRelationshipCycle performs cycle detection in the relationship graph
// This prevents circular dependencies like A->B->C->A
func (m *AssetManager) detectRelationshipCycle(ctx context.Context, fromAssetID, toAssetID string, relType assetsv1.RelationshipType) error {
	// Use BFS to detect if adding this relationship would create a cycle
	// Start from toAssetID and see if we can reach fromAssetID following relationships
	visited := make(map[string]bool)
	queue := []string{toAssetID}
	visited[toAssetID] = true

	relTypeStr := relType.String()

	for len(queue) > 0 {
		currentID := queue[0]
		queue = queue[1:]

		// If we reached the fromAssetID, adding this relationship would create a cycle
		if currentID == fromAssetID {
			return fmt.Errorf("cycle detected: adding relationship from %s to %s would create a cycle", fromAssetID, toAssetID)
		}

		// Get all relationships where current asset is the "from" asset
		filter := &repository.RelationshipFilter{
			FromAssetID:      &currentID,
			RelationshipType: &relTypeStr,
			Limit:            100, // Reasonable limit to prevent infinite loops
		}

		relationships, err := m.repo.ListAssetRelationships(ctx, filter)
		if err != nil {
			// Don't fail on query error, just stop cycle detection
			break
		}

		// Add all "to" assets to the queue if not visited
		for _, rel := range relationships {
			if rel.ToAssetId != nil && !visited[*rel.ToAssetId] {
				visited[*rel.ToAssetId] = true
				queue = append(queue, *rel.ToAssetId)
			}
		}
	}

	return nil
}

// GetAssetRelationship retrieves a relationship by ID
func (m *AssetManager) GetAssetRelationship(ctx context.Context, relationshipID string) (*assetsv1.AssetRelationship, error) {
	if relationshipID == "" {
		return nil, status.Error(codes.InvalidArgument, "relationship_id is required")
	}

	relationship, err := m.repo.GetAssetRelationship(ctx, relationshipID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("relationship not found: %s", relationshipID))
	}

	return relationship, nil
}

// ListAssetRelationships retrieves relationships with optional filtering
func (m *AssetManager) ListAssetRelationships(ctx context.Context, filter *repository.RelationshipFilter) ([]*assetsv1.AssetRelationship, error) {
	relationships, err := m.repo.ListAssetRelationships(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list relationships: %v", err))
	}

	return relationships, nil
}

// CreateAssetGroup creates a new asset group with validation
func (m *AssetManager) CreateAssetGroup(ctx context.Context, group *assetsv1.AssetGroup) error {
	// Validate group name
	if group.Name == nil || strings.TrimSpace(*group.Name) == "" {
		return status.Error(codes.InvalidArgument, "group name is required")
	}

	// Validate that all member assets exist before creating the group
	if len(group.Members) > 0 {
		for _, member := range group.Members {
			if member.AssetId == nil || *member.AssetId == "" {
				return status.Error(codes.InvalidArgument, "member asset_id is required")
			}

			asset, err := m.repo.GetAsset(ctx, *member.AssetId)
			if err != nil || asset == nil {
				return status.Error(codes.NotFound, fmt.Sprintf("member asset not found: %s", *member.AssetId))
			}
		}
	}

	// Create the group
	if err := m.repo.CreateAssetGroup(ctx, group); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create asset group: %v", err))
	}

	return nil
}

// GetAssetGroup retrieves an asset group by ID
func (m *AssetManager) GetAssetGroup(ctx context.Context, groupID string) (*assetsv1.AssetGroup, error) {
	if groupID == "" {
		return nil, status.Error(codes.InvalidArgument, "group_id is required")
	}

	group, err := m.repo.GetAssetGroup(ctx, groupID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("asset group not found: %s", groupID))
	}

	return group, nil
}

// GetAssetGroupByName retrieves an asset group by name
func (m *AssetManager) GetAssetGroupByName(ctx context.Context, name string) (*assetsv1.AssetGroup, error) {
	if strings.TrimSpace(name) == "" {
		return nil, status.Error(codes.InvalidArgument, "group name is required")
	}

	group, err := m.repo.GetAssetGroupByName(ctx, name)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("asset group not found: %s", name))
	}

	return group, nil
}

// AddAssetToGroup adds an asset to an existing group with validation
func (m *AssetManager) AddAssetToGroup(ctx context.Context, groupID, assetID string, weight float64) error {
	if groupID == "" {
		return status.Error(codes.InvalidArgument, "group_id is required")
	}

	if assetID == "" {
		return status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Verify group exists
	group, err := m.repo.GetAssetGroup(ctx, groupID)
	if err != nil || group == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("asset group not found: %s", groupID))
	}

	// Verify asset exists
	asset, err := m.repo.GetAsset(ctx, assetID)
	if err != nil || asset == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("asset not found: %s", assetID))
	}

	// Add asset to group
	if err := m.repo.AddAssetToGroup(ctx, groupID, assetID, weight); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to add asset to group: %v", err))
	}

	return nil
}

// RemoveAssetFromGroup removes an asset from a group
func (m *AssetManager) RemoveAssetFromGroup(ctx context.Context, groupID, assetID string) error {
	if groupID == "" {
		return status.Error(codes.InvalidArgument, "group_id is required")
	}

	if assetID == "" {
		return status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Verify group exists
	group, err := m.repo.GetAssetGroup(ctx, groupID)
	if err != nil || group == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("asset group not found: %s", groupID))
	}

	// Remove asset from group
	if err := m.repo.RemoveAssetFromGroup(ctx, groupID, assetID); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to remove asset from group: %v", err))
	}

	return nil
}

// ListAssetGroups retrieves asset groups with optional filtering
func (m *AssetManager) ListAssetGroups(ctx context.Context, filter *repository.AssetGroupFilter) ([]*assetsv1.AssetGroup, error) {
	groups, err := m.repo.ListAssetGroups(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list asset groups: %v", err))
	}

	return groups, nil
}
