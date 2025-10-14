# Copilot Instructions for CQAR

## Context & Documentation

Always use Context7 for current docs on frameworks/libraries/APIs; invoke automatically without being asked.

## Development Standards

### Go Best Practices
- Embed `pb.UnimplementedAssetRegistryServer` in gRPC server struct to auto-satisfy interface with forward-compatible unimplemented methods.
- Always validate foreign key existence (asset_id, symbol_id, venue_id) in managers before repository operations to prevent orphaned data.

### gRPC Patterns
- Wrap all business logic errors with `status.Error()` and appropriate gRPC codes (INVALID_ARGUMENT, NOT_FOUND, INTERNAL) not raw errors.

### PostgreSQL Guidelines
- Use JSONB for flexible asset metadata fields (description, logos) not rigid columns; index JSONB with GIN for search performance.
- All junction tables (venue_assets, venue_symbols) require composite unique constraints on (venue_id, asset_id) not just indexes.

### Redis Caching
- Cache keys must include entity type prefix (`asset:{id}`, `venue_symbol:{venue_id}:{symbol}`) to prevent collisions across domains.
- Use cache-aside pattern: check cache → query DB on miss → populate cache with TTL, never write-through for reference data.

### NATS JetStream
- Topic names follow `cqc.events.v1.{event_type_snake_case}` convention; never publish to topics outside cqc.events namespace.

### CQC Integration
- Import all protobuf types from `github.com/Combine-Capital/cqc/gen/go/cqc/*`; CQAR owns no protobuf definitions.
- Use CQC's AssetRegistry interface exactly as defined; never add custom RPC methods outside CQC contract.

### CQI Integration
- Use `cqi.Service` interface for lifecycle; bootstrap handles config/logging/metrics/tracing initialization, never manual setup.
- All event publishing goes through CQI event bus with automatic protobuf serialization; never manually marshal protobuf to bytes.

### Code Quality Standards
- Performance requirement: <10ms p50 reads demand cache-first strategy; measure with Prometheus histogram `cqar_grpc_request_duration_seconds`.
- CRITICAL-severity quality flags must block trades; AssetManager validates this before any asset operation that affects trading.

### Project Conventions
- Business logic lives in `internal/manager/` (AssetManager, SymbolManager, VenueManager), data access in `internal/repository/`; never mix concerns.
- Each repository method returns CQC protobuf types not custom structs; database rows scan directly into protobuf messages.

### Agentic AI Guidelines
- Never create "summary" documents; direct action is more valuable than summarization.
