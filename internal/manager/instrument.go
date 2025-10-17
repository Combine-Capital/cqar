package manager

import (
	"context"
	"fmt"

	"github.com/Combine-Capital/cqar/internal/repository"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// InstrumentManager handles business logic for instrument operations with validation
type InstrumentManager struct {
	repo           repository.Repository
	assetManager   *AssetManager
	eventPublisher *EventPublisher
}

// NewInstrumentManager creates a new InstrumentManager instance
func NewInstrumentManager(repo repository.Repository, assetManager *AssetManager, eventPublisher *EventPublisher) *InstrumentManager {
	return &InstrumentManager{
		repo:           repo,
		assetManager:   assetManager,
		eventPublisher: eventPublisher,
	}
}

// CreateInstrument creates a new instrument with validation
func (m *InstrumentManager) CreateInstrument(ctx context.Context, instrument *marketsv1.Instrument) error {
	// Validate required fields
	if err := ValidateRequiredInstrumentFields(instrument); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the instrument in the repository
	if err := m.repo.CreateInstrument(ctx, instrument); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create instrument: %v", err))
	}

	// Publish InstrumentCreated event asynchronously
	if m.eventPublisher != nil {
		m.eventPublisher.PublishInstrumentCreated(ctx, instrument)
	}

	return nil
}

// GetInstrument retrieves an instrument by ID
func (m *InstrumentManager) GetInstrument(ctx context.Context, instrumentID string) (*marketsv1.Instrument, error) {
	if instrumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "instrument_id is required")
	}

	instrument, err := m.repo.GetInstrument(ctx, instrumentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("instrument not found: %s", instrumentID))
	}

	return instrument, nil
}

// CreateSpotInstrument creates a spot instrument with validation
// Validates that base_asset_id and quote_asset_id exist before creation
func (m *InstrumentManager) CreateSpotInstrument(ctx context.Context, instrument *marketsv1.Instrument, spot *marketsv1.SpotInstrument) error {
	// First create the base instrument
	if err := m.CreateInstrument(ctx, instrument); err != nil {
		return err
	}

	// Link spot details to the instrument
	spot.InstrumentId = instrument.Id

	// Validate base_asset_id exists
	if spot.BaseAssetId != nil && *spot.BaseAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *spot.BaseAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("base_asset_id does not exist: %s", *spot.BaseAssetId))
		}
	}

	// Validate quote_asset_id exists
	if spot.QuoteAssetId != nil && *spot.QuoteAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *spot.QuoteAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("quote_asset_id does not exist: %s", *spot.QuoteAssetId))
		}
	}

	// Create the spot instrument details
	if err := m.repo.CreateSpotInstrument(ctx, spot); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create spot instrument: %v", err))
	}

	return nil
}

// GetSpotInstrument retrieves a spot instrument by instrument_id
func (m *InstrumentManager) GetSpotInstrument(ctx context.Context, instrumentID string) (*marketsv1.SpotInstrument, error) {
	if instrumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "instrument_id is required")
	}

	spot, err := m.repo.GetSpotInstrument(ctx, instrumentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("spot instrument not found: %s", instrumentID))
	}

	return spot, nil
}

// CreatePerpContract creates a perpetual contract with validation
// Validates that underlying_asset_id exists before creation
func (m *InstrumentManager) CreatePerpContract(ctx context.Context, instrument *marketsv1.Instrument, perp *marketsv1.PerpContract) error {
	// First create the base instrument
	if err := m.CreateInstrument(ctx, instrument); err != nil {
		return err
	}

	// Link perp details to the instrument
	perp.InstrumentId = instrument.Id

	// Validate underlying_asset_id exists
	if perp.UnderlyingAssetId != nil && *perp.UnderlyingAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *perp.UnderlyingAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("underlying_asset_id does not exist: %s", *perp.UnderlyingAssetId))
		}
	}

	// Validate contract multiplier
	if err := ValidatePerpContractFields(perp); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the perp contract details
	if err := m.repo.CreatePerpContract(ctx, perp); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create perp contract: %v", err))
	}

	return nil
}

// GetPerpContract retrieves a perpetual contract by instrument_id
func (m *InstrumentManager) GetPerpContract(ctx context.Context, instrumentID string) (*marketsv1.PerpContract, error) {
	if instrumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "instrument_id is required")
	}

	perp, err := m.repo.GetPerpContract(ctx, instrumentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("perp contract not found: %s", instrumentID))
	}

	return perp, nil
}

// CreateFutureContract creates a future contract with validation
func (m *InstrumentManager) CreateFutureContract(ctx context.Context, instrument *marketsv1.Instrument, future *marketsv1.FutureContract) error {
	// First create the base instrument
	if err := m.CreateInstrument(ctx, instrument); err != nil {
		return err
	}

	// Link future details to the instrument
	future.InstrumentId = instrument.Id

	// Validate underlying_asset_id exists
	if future.UnderlyingAssetId != nil && *future.UnderlyingAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *future.UnderlyingAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("underlying_asset_id does not exist: %s", *future.UnderlyingAssetId))
		}
	}

	// Validate future-specific fields
	if err := ValidateFutureContractFields(future); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the future contract details
	if err := m.repo.CreateFutureContract(ctx, future); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create future contract: %v", err))
	}

	return nil
}

// GetFutureContract retrieves a future contract by instrument_id
func (m *InstrumentManager) GetFutureContract(ctx context.Context, instrumentID string) (*marketsv1.FutureContract, error) {
	if instrumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "instrument_id is required")
	}

	future, err := m.repo.GetFutureContract(ctx, instrumentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("future contract not found: %s", instrumentID))
	}

	return future, nil
}

// CreateOptionSeries creates an option series with validation
func (m *InstrumentManager) CreateOptionSeries(ctx context.Context, instrument *marketsv1.Instrument, option *marketsv1.OptionSeries) error {
	// First create the base instrument
	if err := m.CreateInstrument(ctx, instrument); err != nil {
		return err
	}

	// Link option details to the instrument
	option.InstrumentId = instrument.Id

	// Validate underlying_asset_id exists
	if option.UnderlyingAssetId != nil && *option.UnderlyingAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *option.UnderlyingAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("underlying_asset_id does not exist: %s", *option.UnderlyingAssetId))
		}
	}

	// Validate option-specific fields
	if err := ValidateOptionSeriesFields(option); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	// Create the option series details
	if err := m.repo.CreateOptionSeries(ctx, option); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create option series: %v", err))
	}

	return nil
}

// GetOptionSeries retrieves an option series by instrument_id
func (m *InstrumentManager) GetOptionSeries(ctx context.Context, instrumentID string) (*marketsv1.OptionSeries, error) {
	if instrumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "instrument_id is required")
	}

	option, err := m.repo.GetOptionSeries(ctx, instrumentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("option series not found: %s", instrumentID))
	}

	return option, nil
}

// CreateLendingDeposit creates a lending deposit instrument with validation
func (m *InstrumentManager) CreateLendingDeposit(ctx context.Context, instrument *marketsv1.Instrument, deposit *marketsv1.LendingDeposit) error {
	// First create the base instrument
	if err := m.CreateInstrument(ctx, instrument); err != nil {
		return err
	}

	// Link deposit details to the instrument
	deposit.InstrumentId = instrument.Id

	// Validate underlying_asset_id exists
	if deposit.UnderlyingAssetId != nil && *deposit.UnderlyingAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *deposit.UnderlyingAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("underlying_asset_id does not exist: %s", *deposit.UnderlyingAssetId))
		}
	}

	// Create the lending deposit details
	if err := m.repo.CreateLendingDeposit(ctx, deposit); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create lending deposit: %v", err))
	}

	return nil
}

// GetLendingDeposit retrieves a lending deposit by instrument_id
func (m *InstrumentManager) GetLendingDeposit(ctx context.Context, instrumentID string) (*marketsv1.LendingDeposit, error) {
	if instrumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "instrument_id is required")
	}

	deposit, err := m.repo.GetLendingDeposit(ctx, instrumentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("lending deposit not found: %s", instrumentID))
	}

	return deposit, nil
}

// CreateLendingBorrow creates a lending borrow instrument with validation
func (m *InstrumentManager) CreateLendingBorrow(ctx context.Context, instrument *marketsv1.Instrument, borrow *marketsv1.LendingBorrow) error {
	// First create the base instrument
	if err := m.CreateInstrument(ctx, instrument); err != nil {
		return err
	}

	// Link borrow details to the instrument
	borrow.InstrumentId = instrument.Id

	// Validate underlying_asset_id exists
	if borrow.UnderlyingAssetId != nil && *borrow.UnderlyingAssetId != "" {
		if _, err := m.assetManager.GetAsset(ctx, *borrow.UnderlyingAssetId); err != nil {
			return status.Error(codes.InvalidArgument, fmt.Sprintf("underlying_asset_id does not exist: %s", *borrow.UnderlyingAssetId))
		}
	}

	// Create the lending borrow details
	if err := m.repo.CreateLendingBorrow(ctx, borrow); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to create lending borrow: %v", err))
	}

	return nil
}

// GetLendingBorrow retrieves a lending borrow by instrument_id
func (m *InstrumentManager) GetLendingBorrow(ctx context.Context, instrumentID string) (*marketsv1.LendingBorrow, error) {
	if instrumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "instrument_id is required")
	}

	borrow, err := m.repo.GetLendingBorrow(ctx, instrumentID)
	if err != nil {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("lending borrow not found: %s", instrumentID))
	}

	return borrow, nil
}
