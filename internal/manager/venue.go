package manager

import (
	"context"
	"fmt"

	"github.com/Combine-Capital/cqar/internal/repository"
	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// VenueManager handles business logic for venue operations and venue asset mapping
type VenueManager struct {
	repo           repository.Repository
	assetManager   *AssetManager
	eventPublisher *EventPublisher
}

// NewVenueManager creates a new VenueManager instance
func NewVenueManager(repo repository.Repository, assetManager *AssetManager, eventPublisher *EventPublisher) *VenueManager {
	return &VenueManager{
		repo:           repo,
		assetManager:   assetManager,
		eventPublisher: eventPublisher,
	}
}

// CreateVenue creates a new venue with validation
func (m *VenueManager) CreateVenue(ctx context.Context, venue *venuesv1.Venue) error {
	// Validate required fields
	if err := ValidateRequiredVenueFields(venue); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate chain_id exists if provided (for DEX venues)
	if venue.ChainId != nil && *venue.ChainId != "" {
		chain, err := m.repo.GetChain(ctx, *venue.ChainId)
		if err != nil || chain == nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("chain_id does not exist: %s", *venue.ChainId))
		}
	}

	// Create the venue in the repository
	if err := m.repo.CreateVenue(ctx, venue); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create venue: %v", err))
	}

	return nil
}

// GetVenue retrieves a venue by ID
func (m *VenueManager) GetVenue(ctx context.Context, venueID string) (*venuesv1.Venue, error) {
	if venueID == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}

	venue, err := m.repo.GetVenue(ctx, venueID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("venue not found: %s", venueID))
	}

	return venue, nil
}

// ListVenues retrieves venues with optional filtering
func (m *VenueManager) ListVenues(ctx context.Context, filter *repository.VenueFilter) ([]*venuesv1.Venue, error) {
	venues, err := m.repo.ListVenues(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list venues: %v", err))
	}

	return venues, nil
}

// CreateVenueAsset creates a new venue asset mapping with validation
func (m *VenueManager) CreateVenueAsset(ctx context.Context, venueAsset *venuesv1.VenueAsset) error {
	// Validate required fields
	if err := ValidateRequiredVenueAssetFields(venueAsset); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Validate venue_id exists
	if _, err := m.GetVenue(ctx, *venueAsset.VenueId); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("venue_id does not exist: %s", *venueAsset.VenueId))
	}

	// Validate asset_id exists
	if _, err := m.assetManager.GetAsset(ctx, *venueAsset.AssetId); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("asset_id does not exist: %s", *venueAsset.AssetId))
	}

	// Validate fees
	if err := ValidateFees(venueAsset.WithdrawalFee); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid withdrawal_fee: %v", err))
	}

	if err := ValidateFees(venueAsset.DepositFee); err != nil {
		return status.Error(codes.InvalidArgument, fmt.Sprintf("invalid deposit_fee: %v", err))
	}

	// Create the venue asset in the repository
	if err := m.repo.CreateVenueAsset(ctx, venueAsset); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create venue asset: %v", err))
	}

	// Publish VenueAssetListed event asynchronously
	if m.eventPublisher != nil {
		m.eventPublisher.PublishVenueAssetListed(ctx, venueAsset)
	}

	return nil
}

// GetVenueAsset retrieves a venue asset by venue_id and asset_id
func (m *VenueManager) GetVenueAsset(ctx context.Context, venueID, assetID string) (*venuesv1.VenueAsset, error) {
	if venueID == "" {
		return nil, status.Error(codes.InvalidArgument, "venue_id is required")
	}

	if assetID == "" {
		return nil, status.Error(codes.InvalidArgument, "asset_id is required")
	}

	venueAsset, err := m.repo.GetVenueAsset(ctx, venueID, assetID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("venue asset not found: venue=%s asset=%s", venueID, assetID))
	}

	return venueAsset, nil
}

// ListVenueAssets retrieves venue assets with optional filtering
func (m *VenueManager) ListVenueAssets(ctx context.Context, filter *repository.VenueAssetFilter) ([]*venuesv1.VenueAsset, error) {
	venueAssets, err := m.repo.ListVenueAssets(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to list venue assets: %v", err))
	}

	return venueAssets, nil
}

// Validation functions

// ValidateRequiredVenueFields validates required venue fields
func ValidateRequiredVenueFields(venue *venuesv1.Venue) error {
	if venue.VenueId == nil || *venue.VenueId == "" {
		return fmt.Errorf("venue_id is required")
	}
	if venue.Name == nil || *venue.Name == "" {
		return fmt.Errorf("name is required")
	}
	if venue.VenueType == nil {
		return fmt.Errorf("venue_type is required")
	}
	return nil
}

// ValidateRequiredVenueAssetFields validates required venue asset fields
func ValidateRequiredVenueAssetFields(venueAsset *venuesv1.VenueAsset) error {
	if venueAsset.VenueId == nil || *venueAsset.VenueId == "" {
		return fmt.Errorf("venue_id is required")
	}
	if venueAsset.AssetId == nil || *venueAsset.AssetId == "" {
		return fmt.Errorf("asset_id is required")
	}
	return nil
}

// ValidateFees validates fee values are in acceptable range
func ValidateFees(fee *float64) error {
	if fee == nil {
		return nil // Fees are optional
	}

	if *fee < 0 {
		return fmt.Errorf("fee cannot be negative, got %f", *fee)
	}

	// Allow up to 150% to support some edge cases (very high fees)
	if *fee > 150.0 {
		return fmt.Errorf("fee cannot exceed 150%%, got %f", *fee)
	}

	return nil
}
