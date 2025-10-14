# Copilot Instructions for CQAR

## Context & Documentation

Always use Context7 for current docs on frameworks/libraries/APIs; invoke automatically without being asked.

## Development Standards

### Go Best Practices
- Always pass `context.Context` as first parameter; use `context.WithTimeout()` for gRPC calls to enforce <50ms p99 SLA.
- Return domain errors from business logic, map to gRPC status codes only in service layer to maintain clean architecture boundaries.

### gRPC + Protocol Buffers
- Implement CQC `AssetRegistry` interface exactly; never add custom RPC methods outside CQC contract until discussed with platform team.
- Use CQC message types as wire format only; convert to domain models at service boundary to avoid proto pollution in business logic.

### PostgreSQL Guidelines
- Always query with `WHERE deleted_at IS NULL` for soft-deleted assets; missing this clause causes stale data in aggregations.
- Use `ILIKE` for symbol searches (case-insensitive) with GIN indexes; `LIKE` breaks collision detection across exchanges.

### Redis Caching
- Cache keys must include version prefix (`v1:asset:id:{uuid}`) to enable zero-downtime schema migrations; invalidate explicitly on writes.
- Set TTL on all keys (1h assets, 5m flags, 30m groups); missing TTLs cause memory exhaustion during market volatility spikes.

### CQI Event Bus
- Publish events after database commit not before; event-then-DB-fail creates inconsistent state downstream services cannot recover from.

### Code Quality Standards
- Repository layer returns domain errors (`ErrAssetNotFound`), never `sql.ErrNoRows`; leaking driver errors breaks abstraction.
- Integration tests must seed relationships before querying groups; orphaned test data causes flaky aggregation tests.

### Project Conventions
- Domain models in `internal/domain/`, repos in `internal/repository/`, gRPC in `internal/service/`; never mix layers (no DB calls from service).
- Migration files must have matching `.up.sql` and `.down.sql`; missing down migrations block rollbacks in production incidents.

### Agentic AI Guidelines
- Never create "summary" documents; direct action is more valuable than summarization.
