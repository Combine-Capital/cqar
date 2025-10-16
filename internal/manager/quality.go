package manager

import (
	"context"
	"fmt"

	"github.com/Combine-Capital/cqar/internal/repository"
	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// QualityManager handles business logic for quality flags and asset tradeability
type QualityManager struct {
	repo           repository.Repository
	eventPublisher *EventPublisher
}

// NewQualityManager creates a new QualityManager instance
func NewQualityManager(repo repository.Repository, eventPublisher *EventPublisher) *QualityManager {
	return &QualityManager{
		repo:           repo,
		eventPublisher: eventPublisher,
	}
}

// RaiseQualityFlag creates a new quality flag with validation
func (m *QualityManager) RaiseQualityFlag(ctx context.Context, flag *assetsv1.AssetQualityFlag) error {
	// Validate flag type
	if err := ValidateFlagType(flag.GetFlagType()); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate severity
	if err := ValidateFlagSeverity(flag.GetSeverity()); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate source
	if err := ValidateSource(flag.GetSource()); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate reason
	if err := ValidateReason(flag.GetReason()); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate asset_id is provided
	if flag.AssetId == nil || *flag.AssetId == "" {
		return status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Verify asset exists
	asset, err := m.repo.GetAsset(ctx, *flag.AssetId)
	if err != nil {
		return status.Error(codes.NotFound, fmt.Sprintf("asset not found: %s", *flag.AssetId))
	}
	if asset == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("asset not found: %s", *flag.AssetId))
	}

	// Create the flag in the repository
	if err := m.repo.RaiseQualityFlag(ctx, flag); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to raise quality flag: %v", err))
	}

	// Publish QualityFlagRaised event asynchronously
	if m.eventPublisher != nil {
		m.eventPublisher.PublishQualityFlagRaised(ctx, flag)
	}

	return nil
}

// ResolveQualityFlag marks a quality flag as resolved
func (m *QualityManager) ResolveQualityFlag(ctx context.Context, flagID, resolvedBy, resolutionNotes string) error {
	if flagID == "" {
		return status.Error(codes.InvalidArgument, "flag_id is required")
	}

	if resolvedBy == "" {
		return status.Error(codes.InvalidArgument, "resolved_by is required")
	}

	// Verify flag exists
	flag, err := m.repo.GetQualityFlag(ctx, flagID)
	if err != nil {
		return status.Error(codes.NotFound, fmt.Sprintf("quality flag not found: %s", flagID))
	}
	if flag == nil {
		return status.Error(codes.NotFound, fmt.Sprintf("quality flag not found: %s", flagID))
	}

	// Check if already resolved
	if flag.ResolvedAt != nil {
		return status.Error(codes.FailedPrecondition, "quality flag already resolved")
	}

	// Resolve the flag
	if err := m.repo.ResolveQualityFlag(ctx, flagID, resolvedBy, resolutionNotes); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to resolve quality flag: %v", err))
	}

	return nil
}

// IsAssetTradeable checks if an asset has any active CRITICAL quality flags
// Returns false if any active CRITICAL flags exist, true otherwise
func (m *QualityManager) IsAssetTradeable(ctx context.Context, assetID string) (bool, error) {
	if assetID == "" {
		return false, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	// Query for active CRITICAL flags
	severityCritical := assetsv1.FlagSeverity_FLAG_SEVERITY_CRITICAL.String()
	filter := &repository.QualityFlagFilter{
		AssetID:    &assetID,
		Severity:   &severityCritical,
		ActiveOnly: true, // Only check unresolved flags
		Limit:      1,    // We only need to know if any exist
	}

	flags, err := m.repo.ListQualityFlags(ctx, filter)
	if err != nil {
		return false, status.Error(codes.Internal, fmt.Sprintf("failed to check quality flags: %v", err))
	}

	// If any active CRITICAL flags exist, asset is not tradeable
	if len(flags) > 0 {
		return false, nil
	}

	return true, nil
}

// GetQualityFlag retrieves a quality flag by ID
func (m *QualityManager) GetQualityFlag(ctx context.Context, flagID string) (*assetsv1.AssetQualityFlag, error) {
	if flagID == "" {
		return nil, status.Error(codes.InvalidArgument, "flag_id is required")
	}

	flag, err := m.repo.GetQualityFlag(ctx, flagID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("quality flag not found: %s", flagID))
	}

	return flag, nil
}

// ListQualityFlags retrieves quality flags with filtering
func (m *QualityManager) ListQualityFlags(ctx context.Context, filter *repository.QualityFlagFilter) ([]*assetsv1.AssetQualityFlag, error) {
	flags, err := m.repo.ListQualityFlags(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list quality flags: %v", err))
	}

	return flags, nil
}
