package manager

import (
	"context"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	eventsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/events/v1"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
	venuesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/venues/v1"
	"github.com/Combine-Capital/cqi/pkg/bus"
	"github.com/Combine-Capital/cqi/pkg/logging"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// EventPublisher publishes domain lifecycle events to the NATS JetStream event bus.
// All events use CQC protobuf types and are published asynchronously.
// Publishing failures are logged but do not fail the originating operation.
type EventPublisher struct {
	bus    bus.EventBus
	logger *logging.Logger
}

// NewEventPublisher creates a new EventPublisher instance.
func NewEventPublisher(eventBus bus.EventBus, logger *logging.Logger) *EventPublisher {
	return &EventPublisher{
		bus:    eventBus,
		logger: logger,
	}
}

// PublishAssetCreated publishes an AssetCreated event when a new asset is created.
// Topic: cqc.events.v1.asset_created
func (p *EventPublisher) PublishAssetCreated(ctx context.Context, asset *assetsv1.Asset) {
	if p.bus == nil {
		p.logger.Warn().Msg("Event bus not initialized, skipping AssetCreated event")
		return
	}

	eventID := uuid.New().String()
	actorID := "service:cqar"
	source := "cqar"
	event := &eventsv1.AssetCreated{
		EventId:   &eventID,
		Timestamp: timestamppb.Now(),
		ActorId:   &actorID,
		Asset:     asset,
		Source:    &source,
	}

	// Publish asynchronously - don't block the operation on event publishing
	go func() {
		if err := p.bus.Publish(ctx, "cqc.events.v1.asset_created", event); err != nil {
			p.logger.Error().
				Err(err).
				Str("asset_id", ptrStr(asset.AssetId)).
				Str("symbol", ptrStr(asset.Symbol)).
				Str("event_id", eventID).
				Msg("Failed to publish AssetCreated event")
		} else {
			p.logger.Info().
				Str("asset_id", ptrStr(asset.AssetId)).
				Str("symbol", ptrStr(asset.Symbol)).
				Str("event_id", eventID).
				Msg("Published AssetCreated event")
		}
	}()
}

// PublishAssetDeploymentCreated publishes an AssetDeploymentCreated event when a deployment is registered.
// Topic: cqc.events.v1.asset_deployment_created
func (p *EventPublisher) PublishAssetDeploymentCreated(ctx context.Context, deployment *assetsv1.AssetDeployment) {
	if p.bus == nil {
		p.logger.Warn().Msg("Event bus not initialized, skipping AssetDeploymentCreated event")
		return
	}

	eventID := uuid.New().String()
	actorID := "service:cqar"
	source := "cqar"
	autoDetected := false // Manual creation via CQAR
	event := &eventsv1.AssetDeploymentCreated{
		EventId:      &eventID,
		Timestamp:    timestamppb.Now(),
		ActorId:      &actorID,
		Deployment:   deployment,
		Source:       &source,
		AutoDetected: &autoDetected,
	}

	go func() {
		if err := p.bus.Publish(ctx, "cqc.events.v1.asset_deployment_created", event); err != nil {
			p.logger.Error().
				Err(err).
				Str("asset_id", ptrStr(deployment.AssetId)).
				Str("chain_id", ptrStr(deployment.ChainId)).
				Str("event_id", eventID).
				Msg("Failed to publish AssetDeploymentCreated event")
		} else {
			p.logger.Info().
				Str("asset_id", ptrStr(deployment.AssetId)).
				Str("chain_id", ptrStr(deployment.ChainId)).
				Str("event_id", eventID).
				Msg("Published AssetDeploymentCreated event")
		}
	}()
}

// PublishRelationshipEstablished publishes a RelationshipEstablished event when an asset relationship is created.
// Topic: cqc.events.v1.relationship_established
func (p *EventPublisher) PublishRelationshipEstablished(ctx context.Context, relationship *assetsv1.AssetRelationship) {
	if p.bus == nil {
		p.logger.Warn().Msg("Event bus not initialized, skipping RelationshipEstablished event")
		return
	}

	eventID := uuid.New().String()
	actorID := "service:cqar"
	source := "cqar"
	event := &eventsv1.RelationshipEstablished{
		EventId:      &eventID,
		Timestamp:    timestamppb.Now(),
		ActorId:      &actorID,
		Relationship: relationship,
		Source:       &source,
		Protocol:     relationship.Protocol, // Duplicate for event filtering
	}

	go func() {
		if err := p.bus.Publish(ctx, "cqc.events.v1.relationship_established", event); err != nil {
			p.logger.Error().
				Err(err).
				Str("from_asset_id", ptrStr(relationship.FromAssetId)).
				Str("to_asset_id", ptrStr(relationship.ToAssetId)).
				Str("event_id", eventID).
				Msg("Failed to publish RelationshipEstablished event")
		} else {
			p.logger.Info().
				Str("from_asset_id", ptrStr(relationship.FromAssetId)).
				Str("to_asset_id", ptrStr(relationship.ToAssetId)).
				Str("event_id", eventID).
				Msg("Published RelationshipEstablished event")
		}
	}()
}

// PublishVenueAssetListed publishes a VenueAssetListed event when an asset is listed on a venue.
// Topic: cqc.events.v1.venue_asset_listed
func (p *EventPublisher) PublishVenueAssetListed(ctx context.Context, venueAsset *venuesv1.VenueAsset) {
	if p.bus == nil {
		p.logger.Warn().Msg("Event bus not initialized, skipping VenueAssetListed event")
		return
	}

	eventID := uuid.New().String()
	actorID := "service:cqar"
	isNewListing := true // All listings through CreateVenueAsset are new
	event := &eventsv1.VenueAssetListed{
		EventId:      &eventID,
		Timestamp:    timestamppb.Now(),
		ActorId:      &actorID,
		VenueAsset:   venueAsset,
		IsNewListing: &isNewListing,
	}

	go func() {
		if err := p.bus.Publish(ctx, "cqc.events.v1.venue_asset_listed", event); err != nil {
			p.logger.Error().
				Err(err).
				Str("venue_id", ptrStr(venueAsset.VenueId)).
				Str("asset_id", ptrStr(venueAsset.AssetId)).
				Str("event_id", eventID).
				Msg("Failed to publish VenueAssetListed event")
		} else {
			p.logger.Info().
				Str("venue_id", ptrStr(venueAsset.VenueId)).
				Str("asset_id", ptrStr(venueAsset.AssetId)).
				Str("event_id", eventID).
				Msg("Published VenueAssetListed event")
		}
	}()
}

// PublishQualityFlagRaised publishes a QualityFlagRaised event when a quality flag is raised for an asset.
// Topic: cqc.events.v1.quality_flag_raised
func (p *EventPublisher) PublishQualityFlagRaised(ctx context.Context, flag *assetsv1.AssetQualityFlag) {
	if p.bus == nil {
		p.logger.Warn().Msg("Event bus not initialized, skipping QualityFlagRaised event")
		return
	}

	eventID := uuid.New().String()
	actorID := "service:cqar"
	event := &eventsv1.QualityFlagRaised{
		EventId:   &eventID,
		Timestamp: timestamppb.Now(),
		ActorId:   &actorID,
		Flag:      flag,
	}

	go func() {
		if err := p.bus.Publish(ctx, "cqc.events.v1.quality_flag_raised", event); err != nil {
			p.logger.Error().
				Err(err).
				Str("asset_id", ptrStr(flag.AssetId)).
				Str("flag_type", flag.FlagType.String()).
				Str("severity", flag.Severity.String()).
				Str("event_id", eventID).
				Msg("Failed to publish QualityFlagRaised event")
		} else {
			p.logger.Info().
				Str("asset_id", ptrStr(flag.AssetId)).
				Str("flag_type", flag.FlagType.String()).
				Str("severity", flag.Severity.String()).
				Str("event_id", eventID).
				Msg("Published QualityFlagRaised event")
		}
	}()
}

// Helper function to safely dereference string pointer
func ptrStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// PublishInstrumentCreated publishes an InstrumentCreated event when a new instrument is created.
// Topic: cqc.events.v1.instrument_created
// TODO: Update to use proper protobuf event when added to CQC
func (p *EventPublisher) PublishInstrumentCreated(ctx context.Context, instrument *marketsv1.Instrument) {
	if p.bus == nil {
		p.logger.Warn().Msg("Event bus not initialized, skipping InstrumentCreated event")
		return
	}

	eventID := uuid.New().String()

	// Log the event for now until proper protobuf event is added to CQC
	p.logger.Info().
		Str("instrument_id", ptrStr(instrument.Id)).
		Str("instrument_type", ptrStr(instrument.InstrumentType)).
		Str("code", ptrStr(instrument.Code)).
		Str("event_id", eventID).
		Msg("Instrument created (event publishing pending protobuf definition)")
}

// PublishMarketListed publishes a MarketListed event when a market is listed.
// Topic: cqc.events.v1.market_listed
// TODO: Update to use proper protobuf event when added to CQC
func (p *EventPublisher) PublishMarketListed(ctx context.Context, market *marketsv1.Market) {
	if p.bus == nil {
		p.logger.Warn().Msg("Event bus not initialized, skipping MarketListed event")
		return
	}

	eventID := uuid.New().String()

	// Log the event for now until proper protobuf event is added to CQC
	p.logger.Info().
		Str("market_id", ptrStr(market.Id)).
		Str("venue_id", ptrStr(market.VenueId)).
		Str("venue_symbol", ptrStr(market.VenueSymbol)).
		Str("event_id", eventID).
		Msg("Market listed (event publishing pending protobuf definition)")
}
