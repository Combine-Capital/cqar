# Implementation Roadmap

## Progress Checklist
- [ ] **Commit 1**: Project Foundation & Configuration
- [ ] **Commit 2**: Database Schema - Core Tables
- [ ] **Commit 3**: Database Schema - Relationships & Indexes
- [ ] **Commit 4**: Domain Models & Errors
- [ ] **Commit 5**: Repository Interfaces
- [ ] **Commit 6**: Asset Repository Implementation
- [ ] **Commit 7**: Chain & Venue Repository Implementation
- [ ] **Commit 8**: Deployment & Identifier Repository Implementation
- [ ] **Commit 9**: Relationship & Group Repository Implementation
- [ ] **Commit 10**: Quality Flag Repository Implementation
- [ ] **Commit 11**: Event Publisher Integration
- [ ] **Commit 12**: gRPC Service - Asset Operations
- [ ] **Commit 13**: gRPC Service - Deployment & Chain Operations
- [ ] **Commit 14**: gRPC Service - Relationship & Group Operations
- [ ] **Commit 15**: gRPC Service - Quality & Venue Operations
- [ ] **Commit 16**: Redis Cache Layer
- [ ] **Commit 17**: Integration Tests & Fixtures
- [ ] **Commit 18**: Performance Optimization & Documentation

## Implementation Sequence

### Commit 1: Project Foundation & Configuration

**Goal**: Bootstrap Go project with CQC/CQI dependencies and build infrastructure.
**Depends**: None

**Deliverables**:
- [ ] `go.mod` with Go 1.21+, CQC, CQI, pgx/v5, go-playground/validator, testify dependencies
- [ ] `cmd/server/main.go` with config loading and graceful shutdown
- [ ] `Makefile` with build, test, migrate, and run targets
- [ ] `.github/copilot-instructions.md` with development guidelines
- [ ] `README.md` with setup instructions and architecture overview

**Success**:
- `go build ./cmd/server` compiles without errors
- `make help` displays available commands

---

### Commit 2: Database Schema - Core Tables

**Goal**: Create PostgreSQL migrations for assets, chains, and venues tables.
**Depends**: Commit 1

**Deliverables**:
- [ ] `migrations/001_create_assets.up.sql` with assets table, symbol/type/category indexes, soft-delete support
- [ ] `migrations/001_create_assets.down.sql` with rollback
- [ ] `migrations/006_create_chains.up.sql` with chains table and chain_id PK (numbered 006 to allow FK dependencies)
- [ ] `migrations/006_create_chains.down.sql` with rollback
- [ ] `migrations/007_create_venues.up.sql` with venues table and venue_type field (numbered 007 for consistency)
- [ ] `migrations/007_create_venues.down.sql` with rollback

**Success**:
- `migrate up` creates tables in test PostgreSQL instance
- `migrate down` removes tables cleanly

---

### Commit 3: Database Schema - Relationships & Indexes

**Goal**: Create migrations for deployments, identifiers, relationships, groups, flags, and venue symbols.
**Depends**: Commit 2

**Deliverables**:
- [ ] `migrations/002_create_deployments.up.sql` with asset_deployments table, FKs to assets/chains, unique constraint on (chain_id, contract_address)
- [ ] `migrations/002_create_deployments.down.sql` with rollback
- [ ] `migrations/003_create_relationships.up.sql` with asset_relationships table, parent/child FKs, composite PK
- [ ] `migrations/003_create_relationships.down.sql` with rollback
- [ ] `migrations/004_create_groups.up.sql` with asset_groups and asset_group_members tables
- [ ] `migrations/004_create_groups.down.sql` with rollback
- [ ] `migrations/005_create_quality_flags.up.sql` with asset_quality_flags table, severity/cleared_at indexes
- [ ] `migrations/005_create_quality_flags.down.sql` with rollback
- [ ] `migrations/007_create_venue_symbols.up.sql` with venue_symbols table, (venue_id, symbol) unique constraint
- [ ] `migrations/007_create_venue_symbols.down.sql` with rollback
- [ ] `migrations/008_create_indexes.up.sql` with performance indexes for (symbol, asset_type), (source), (venue_id, symbol)
- [ ] `migrations/008_create_indexes.down.sql` with rollback

**Success**:
- All 10 tables exist after `migrate up`
- FK constraints enforced (inserting deployment with invalid asset_id fails)

---

### Commit 4: Domain Models & Errors

**Goal**: Define domain models and business error types for all entities.
**Depends**: Commit 1

**Deliverables**:
- [ ] `internal/domain/asset.go` with Asset struct, NewAsset constructor, validation methods
- [ ] `internal/domain/deployment.go` with AssetDeployment struct, chain/address validation
- [ ] `internal/domain/relationship.go` with AssetRelationship struct, RelationshipType constants
- [ ] `internal/domain/group.go` with AssetGroup and AssetGroupMember structs
- [ ] `internal/domain/quality.go` with AssetQualityFlag struct, FlagType/FlagSeverity constants
- [ ] `internal/domain/chain.go` with Chain struct, ChainType constants
- [ ] `internal/domain/venue.go` with Venue and VenueSymbol structs, VenueType constants
- [ ] `internal/domain/errors.go` with ErrAssetNotFound, ErrDeploymentExists, ErrInvalidRelationship domain errors

**Success**:
- Domain structs compile with proper field tags
- NewAsset("BTC", "Bitcoin", ASSET_TYPE_NATIVE) returns valid Asset
- ErrAssetNotFound is distinct error type (not wrapping sql.ErrNoRows)

---

### Commit 5: Repository Interfaces

**Goal**: Define repository interfaces for data access abstraction.
**Depends**: Commit 4

**Deliverables**:
- [ ] `internal/repository/interface.go` with AssetRepository, DeploymentRepository, RelationshipRepository, GroupRepository, QualityFlagRepository, ChainRepository, VenueRepository interfaces
- [ ] Each interface defines CRUD methods: Create, Get, Update, Delete, List, Search with context.Context as first parameter
- [ ] List methods accept pagination (limit, offset) and filter parameters
- [ ] Search methods accept query string and filter options

**Success**:
- Interfaces compile without implementation
- Method signatures enforce context.Context first parameter pattern
- Return types use domain models from Commit 4

---

### Commit 6: Asset Repository Implementation

**Goal**: Implement PostgreSQL repository for assets with CRUD operations.
**Depends**: Commits 3, 4, 5

**Deliverables**:
- [ ] `internal/repository/postgres/asset.go` implementing AssetRepository interface using CQI Database
- [ ] CreateAsset with UUID generation, INSERT, event data return
- [ ] GetAsset with WHERE deleted_at IS NULL filter
- [ ] UpdateAsset with UPDATE and updated_at timestamp
- [ ] DeleteAsset with soft-delete (SET deleted_at = NOW())
- [ ] ListAssets with pagination, filters (asset_type, category), ORDER BY symbol
- [ ] SearchAssets with ILIKE on symbol/name, collision detection (return multiple matches)
- [ ] `internal/repository/postgres/queries.sql` with documented SQL queries

**Success**:
- CreateAsset inserts row, returns Asset with generated UUID
- GetAsset("deleted-id") returns ErrAssetNotFound (soft-deleted excluded)
- SearchAssets("USDC") returns multiple assets if collision exists
- ListAssets with pagination returns correct page and total count

---

### Commit 7: Chain & Venue Repository Implementation

**Goal**: Implement repositories for blockchain and venue registries.
**Depends**: Commits 3, 4, 5

**Deliverables**:
- [ ] `internal/repository/postgres/chain.go` implementing ChainRepository interface
- [ ] CreateChain, GetChain, ListChains methods with metadata JSONB support
- [ ] `internal/repository/postgres/venue.go` implementing VenueRepository interface
- [ ] CreateVenue, GetVenue, ListVenues methods
- [ ] CreateVenueSymbol, GetVenueSymbol, ListVenueSymbols methods with (venue_id, symbol) unique constraint handling
- [ ] VenueSymbol lookups support asset_id FK joins

**Success**:
- CreateChain("ethereum", "Ethereum", EVM) inserts chain
- CreateVenueSymbol("binance", "BTCUSDT", btc-asset-id) maps ticker to canonical asset
- GetVenueSymbol("binance", "BTCUSDT") returns asset_id <10ms (indexed lookup)
- Duplicate (venue_id, symbol) returns error

---

### Commit 8: Deployment & Identifier Repository Implementation

**Goal**: Implement repositories for asset deployments and external identifiers.
**Depends**: Commits 3, 4, 5, 6

**Deliverables**:
- [ ] `internal/repository/postgres/deployment.go` implementing DeploymentRepository interface
- [ ] CreateAssetDeployment with (chain_id, contract_address) uniqueness check
- [ ] GetAssetDeployment, ListAssetDeployments with asset_id and chain_id filters
- [ ] GetDeploymentByAddress with (chain_id, address) lookup
- [ ] Asset identifier CRUD in asset.go with external_id mappings (CoinGecko, CMC, DeFiLlama)
- [ ] CreateAssetIdentifier, GetAssetIdentifier, ListAssetIdentifiers methods

**Success**:
- CreateAssetDeployment(asset-id, "ethereum", "0x123", 18) inserts deployment
- GetDeploymentByAddress("ethereum", "0x123") returns deployment <20ms
- Duplicate (chain_id, address) returns ErrDeploymentExists
- ListAssetDeployments(asset-id) returns all chains for asset

---

### Commit 9: Relationship & Group Repository Implementation

**Goal**: Implement repositories for asset relationships and groupings.
**Depends**: Commits 3, 4, 5, 6

**Deliverables**:
- [ ] `internal/repository/postgres/relationship.go` implementing RelationshipRepository interface
- [ ] CreateAssetRelationship with parent/child FK validation, relationship_type
- [ ] ListAssetRelationships with filters (parent_id, child_id, relationship_type)
- [ ] GetRelationshipsByType(asset-id, RELATIONSHIP_TYPE_WRAPS) for querying variants
- [ ] `internal/repository/postgres/group.go` implementing GroupRepository interface
- [ ] CreateAssetGroup, GetAssetGroup with canonical_symbol lookup
- [ ] AddAssetToGroup, RemoveAssetFromGroup with membership management
- [ ] ListAssetGroupMembers with is_canonical flag

**Success**:
- CreateAssetRelationship(eth-id, weth-id, WRAPS) establishes relationship
- ListAssetRelationships(eth-id) returns WETH, stETH, cbETH children
- CreateAssetGroup("eth-native", "ETH") creates group
- AddAssetToGroup(group-id, weth-id, false) adds member
- GetAssetGroup("ETH") returns all variants for aggregation

---

### Commit 10: Quality Flag Repository Implementation

**Goal**: Implement repository for asset quality flags and security audits.
**Depends**: Commits 3, 4, 5, 6

**Deliverables**:
- [ ] `internal/repository/postgres/quality.go` implementing QualityFlagRepository interface
- [ ] RaiseQualityFlag with flag_type, severity, source (auditor), evidence_url
- [ ] ResolveQualityFlag with cleared_at timestamp update
- [ ] ListQualityFlags with filters (asset_id, flag_type, min_severity, active_only, source)
- [ ] Query optimization with (asset_id, severity, cleared_at) composite index
- [ ] GetFlagsBySource(asset-id, "certik") for security auditor queries

**Success**:
- RaiseQualityFlag(asset-id, SCAM, CRITICAL, "manual", "proof-url") inserts flag
- ListQualityFlags(asset-id, CRITICAL, active=true) returns active critical flags <20ms
- ResolveQualityFlag(flag-id) sets cleared_at, excludes from active queries
- GetFlagsBySource(asset-id, "certik") returns auditor-specific flags

---

### Commit 11: Event Publisher Integration

**Goal**: Integrate CQI event bus for publishing asset lifecycle events.
**Depends**: Commits 1, 4

**Deliverables**:
- [ ] `internal/events/publisher.go` with EventPublisher struct wrapping CQI event bus client
- [ ] `internal/events/types.go` with event builders for CQC event protos (AssetCreated, AssetDeploymentCreated, RelationshipEstablished, QualityFlagSet, ChainRegistered, VenueSymbolMapped)
- [ ] PublishAssetCreated(asset) serializes CQC Asset proto, publishes to "assets.created" topic
- [ ] PublishQualityFlagSet(flag) with severity/flag_type for downstream filtering
- [ ] Retry logic via CQI for publish failures
- [ ] Publisher accepts context.Context for timeout/cancellation

**Success**:
- PublishAssetCreated(asset) publishes event to CQI bus without blocking (<5ms)
- Event consumer receives valid CQC AssetCreated proto
- Publish after DB commit prevents inconsistent state
- Failed publish logs error but doesn't rollback transaction

---

### Commit 12: gRPC Service - Asset Operations

**Goal**: Implement CQC AssetRegistry service methods for asset CRUD.
**Depends**: Commits 1, 6, 11

**Deliverables**:
- [ ] `internal/service/asset_registry.go` with AssetRegistryServer struct implementing CQC AssetRegistry interface
- [ ] `internal/service/validators.go` with request validation using go-playground/validator
- [ ] `internal/service/mappers.go` with bidirectional proto ↔ domain conversion
- [ ] CreateAsset gRPC handler: validate → domain.NewAsset → repo.CreateAsset → PublishAssetCreated → map to proto
- [ ] GetAsset handler with asset_id validation, ErrAssetNotFound → NotFound status
- [ ] UpdateAsset handler with partial updates, optimistic locking check
- [ ] DeleteAsset handler with soft-delete
- [ ] ListAssets handler with pagination (page_token encoding), filters (asset_type, category)
- [ ] SearchAssets handler with symbol/name query, collision warnings in response

**Success**:
- CreateAsset("BTC", "Bitcoin") returns Asset proto with generated UUID
- GetAsset(invalid-id) returns gRPC NotFound status
- SearchAssets("USDC") returns list with collision metadata if multiple chains
- ListAssets with page_size=10 returns paginated response with next_page_token

---

### Commit 13: gRPC Service - Deployment & Chain Operations

**Goal**: Implement gRPC methods for deployments, chains, and identifiers.
**Depends**: Commits 7, 8, 11, 12

**Deliverables**:
- [ ] CreateAssetDeployment handler: validate chain_id exists → repo.CreateAssetDeployment → PublishAssetDeploymentCreated
- [ ] GetAssetDeployment, ListAssetDeployments handlers with chain/asset filters
- [ ] CreateAssetIdentifier, GetAssetIdentifier, ListAssetIdentifiers handlers for external ID mapping
- [ ] CreateChain handler: validate chain_id unique → repo.CreateChain → PublishChainRegistered
- [ ] GetChain, ListChains handlers with metadata JSONB
- [ ] Mappers for CQC AssetDeployment, Chain protos

**Success**:
- CreateAssetDeployment(asset-id, "ethereum", "0xabc", 18) creates deployment, publishes event
- ListAssetDeployments(asset-id) returns all chains for asset
- CreateChain("arbitrum", "Arbitrum", EVM) registers chain
- GetChain("ethereum") returns chain with RPC/explorer URLs

---

### Commit 14: gRPC Service - Relationship & Group Operations

**Goal**: Implement gRPC methods for relationships and asset groupings.
**Depends**: Commits 9, 11, 12

**Deliverables**:
- [ ] CreateAssetRelationship handler: validate parent/child exist → repo.CreateAssetRelationship → PublishRelationshipEstablished
- [ ] ListAssetRelationships handler with parent_id/child_id/type filters
- [ ] CreateAssetGroup handler with canonical_symbol uniqueness
- [ ] GetAssetGroup handler with canonical_symbol or group_id lookup
- [ ] AddAssetToGroup, RemoveAssetFromGroup handlers for membership management
- [ ] ListAssetGroupMembers for aggregation queries (returns all variants)
- [ ] Mappers for CQC AssetRelationship, AssetGroup protos

**Success**:
- CreateAssetRelationship(eth-id, weth-id, WRAPS) establishes relationship, publishes event
- ListAssetRelationships(eth-id) returns all wrapped/staked variants
- GetAssetGroup("ETH") returns group with all members (WETH, stETH, cbETH, rETH)
- AddAssetToGroup adds member, removes with RemoveAssetFromGroup

---

### Commit 15: gRPC Service - Quality & Venue Operations

**Goal**: Implement gRPC methods for quality flags and venue symbols.
**Depends**: Commits 7, 10, 11, 12

**Deliverables**:
- [ ] RaiseQualityFlag handler: validate asset exists → repo.RaiseQualityFlag → PublishQualityFlagSet
- [ ] ResolveQualityFlag handler with cleared_at update
- [ ] ListQualityFlags handler with filters (asset_id, flag_type, min_severity, active_only, source for auditor queries)
- [ ] CreateVenue handler: validate venue_id unique → repo.CreateVenue
- [ ] GetVenue, ListVenues handlers
- [ ] CreateVenueSymbol handler: validate venue/asset exist → repo.CreateVenueSymbol → PublishVenueSymbolMapped
- [ ] GetVenueSymbol handler for ticker → asset_id resolution
- [ ] ListVenueSymbols handler with venue_id/asset_id filters
- [ ] Mappers for CQC AssetQualityFlag, Venue, VenueSymbol protos

**Success**:
- RaiseQualityFlag(asset-id, SCAM, CRITICAL, "certik", "url") creates flag, publishes event
- ListQualityFlags(asset-id, min_severity=CRITICAL, active=true) returns active critical flags <20ms
- CreateVenueSymbol("binance", "BTCUSDT", btc-id) maps ticker
- GetVenueSymbol("binance", "BTCUSDT") returns btc-id <10ms (for cqmd price mapping)
- ListQualityFlags with source="certik" filter returns auditor-specific flags

---

### Commit 16: Redis Cache Layer

**Goal**: Implement Redis caching for sub-10ms lookups on hot paths.
**Depends**: Commits 6, 7, 8, 9, 10, 12, 13, 14, 15

**Deliverables**:
- [ ] `internal/repository/cache/redis.go` with Redis client wrapper using CQI cache
- [ ] `internal/repository/cache/keys.go` with key pattern constants (v1:asset:id:{uuid}, v1:asset:symbol:{symbol}:{chain}, v1:group:{id}:members, v1:flags:{id}:critical, v1:venue:{id}:symbol:{symbol})
- [ ] CacheAsset, GetCachedAsset with 1h TTL, JSON serialization
- [ ] CacheSymbolLookup with (symbol, chain_id) → asset_id list, 1h TTL
- [ ] CacheGroupMembers with group_id → asset_id set, 30m TTL
- [ ] CacheCriticalFlags with asset_id → flag count, 5m TTL
- [ ] CacheVenueSymbol with (venue_id, symbol) → asset_id, 1h TTL
- [ ] Cache invalidation on Create/Update: explicit DEL, write-through pattern
- [ ] Repository layer integration: check cache → miss → query DB → cache result

**Success**:
- GetAsset(cached-id) returns <10ms (p50 target met)
- GetVenueSymbol(cached) returns <5ms (cqmd requirement)
- ListQualityFlags(cached, CRITICAL) returns <5ms (cqex pre-trade check)
- UpdateAsset invalidates cache, subsequent GetAsset fetches fresh data
- Cache keys include v1 prefix for migration support

---

### Commit 17: Integration Tests & Fixtures

**Goal**: Create end-to-end integration tests validating all user flows.
**Depends**: All previous commits

**Deliverables**:
- [ ] `test/integration/asset_test.go` testing asset lifecycle (create → get → update → search → delete)
- [ ] `test/integration/deployment_test.go` testing multi-chain deployments and address lookups
- [ ] `test/integration/relationship_test.go` testing relationship establishment and group aggregation
- [ ] `test/integration/quality_test.go` testing flag lifecycle and severity filtering
- [ ] `test/fixtures.go` with test data builders (NewTestAsset, NewTestDeployment, NewTestGroup)
- [ ] `test/testdata/schema.sql` with test database schema
- [ ] `test/testdata/seed.sql` with seed data (BTC, ETH, USDC assets; ethereum, arbitrum chains; binance, coinbase venues)
- [ ] Integration tests use real PostgreSQL/Redis (Docker Compose), no mocks
- [ ] Tests validate BRIEF user persona flows: portfolio aggregation, symbol resolution, pre-trade checks

**Success**:
- Portfolio Manager flow: GetAssetGroup("ETH") → ListAssetRelationships → aggregate variants succeeds <100ms
- Market Data flow: GetVenueSymbol("binance", "BTCUSDT") → GetAsset → returns canonical asset <10ms from cache
- Exchange flow: ListQualityFlags(asset-id, CRITICAL) → blocks trade if flags exist <5ms
- All integration tests pass with real infrastructure
- Tests seed relationships before aggregation queries (prevents flaky tests)

---

### Commit 18: Performance Optimization & Documentation

**Goal**: Validate performance targets and complete production documentation.
**Depends**: Commits 16, 17

**Deliverables**:
- [ ] Performance benchmarks validating <50ms p99, <10ms p50 for symbol resolution
- [ ] Load testing script validating performance under concurrent requests
- [ ] Query optimization: EXPLAIN ANALYZE for slow queries, add missing indexes if needed
- [ ] Connection pool tuning (pgxpool max_conns, Redis pool size)
- [ ] `README.md` with architecture diagram, setup guide, API examples
- [ ] API documentation extracted from CQC proto comments
- [ ] Observability: CQI metrics integration (request latency, error rates, cache hit ratio)
- [ ] Runbook with deployment steps, rollback procedures, troubleshooting
- [ ] Update `docs/SPEC.md` with any deviations discovered during implementation

**Success**:
- SearchAssets("USDC") p99 <50ms, p50 <10ms (measured via benchmarks)
- GetVenueSymbol from cache p99 <10ms, p50 <5ms (cqmd requirement met)
- Cache hit ratio >80% for asset lookups after warmup
- All 6 success metrics from BRIEF validated
- Documentation enables new developer to run service locally in <30min
- Load test sustains performance targets under concurrent load
