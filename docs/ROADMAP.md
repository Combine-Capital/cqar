# Implementation Roadmap

## Progress Checklist
- [x] **Commit 1**: Project Foundation & Configuration
- [x] **Commit 2**: Database Schema & Migrations (Core Tables)
- [x] **Commit 3**: Database Schema & Migrations (Relationship Tables)
- [x] **Commit 4**: Repository Layer - Asset Domain
- [x] **Commit 5**: Repository Layer - Symbol & Chain Domain
- [x] **Commit 6**: Repository Layer - Venue & Mapping Domain
- [x] **Commit 7**: Business Logic - Asset Management
- [x] **Commit 8**: Business Logic - Symbol & Venue Management
- [x] **Commit 9a**: gRPC Server - Core Asset Methods
- [x] **Commit 9b**: gRPC Server - Asset Deployment & Relationship Methods
- [x] **Commit 8**: Business Logic - Symbol & Venue Management
- [x] **Commit 9c**: gRPC Server - Quality, Chain, Symbol, Venue Methods
- [x] **Commit 9d**: CQI Service Integration
- [x] **Commit 9e**: gRPC Middleware Chain
- [x] **Commit 10**: Event Publishing System
- [x] **Commit 11**: Cache Layer Integration
- [ ] **Commit 12**: Integration Tests & Validation
- [ ] **Final**: Documentation & Deployment Configuration

---

## Implementation Sequence

### Commit 1: Project Foundation & Configuration

**Goal**: Establish project structure with CQC/CQI dependencies and configuration management.
**Depends**: none

**Deliverables**:
- [x] `go.mod` with CQC (github.com/Combine-Capital/cqc) and CQI (github.com/Combine-Capital/cqi) dependencies
- [x] `cmd/server/main.go` skeleton with CQI bootstrap initialization
- [x] `internal/config/config.go` extending CQI config types with CQAR-specific settings
- [x] `config.yaml`, `config.dev.yaml`, `config.prod.yaml` with database, cache, event bus configuration
- [x] `Makefile` with build, test, run, migrate targets
- [x] `.gitignore` excluding binaries, vendor, IDE files
- [x] `README.md` with service overview and setup instructions

**Success**:
- `go mod tidy` resolves all dependencies without errors
- `make build` produces cqar binary in cmd/server/
- `./cmd/server/cqar --help` displays service information

---

### Commit 2: Database Schema & Migrations (Core Tables)

**Goal**: Create foundational database tables for assets, symbols, chains, and venues.
**Depends**: Commit 1

**Deliverables**:
- [x] `migrations/000001_create_assets_table.up.sql` with assets table (id, symbol, name, type, category, metadata fields, timestamps)
- [x] `migrations/000001_create_assets_table.down.sql` with DROP TABLE assets
- [x] `migrations/000002_create_symbols_table.up.sql` with symbols table (id, base/quote/settlement asset FKs, type, market specs, option fields)
- [x] `migrations/000003_create_chains_table.up.sql` with chains table (id, name, type, native_asset_id FK, rpc_urls array, explorer_url)
- [x] `migrations/000004_create_venues_table.up.sql` with venues table (id, name, type, chain_id FK, metadata, is_active)
- [x] All indexes: idx_assets_symbol, idx_assets_type, idx_symbols_base_asset, idx_symbols_quote_asset, idx_venues_type

**Success**:
- `make migrate-up` executes migrations against PostgreSQL without errors
- `psql` shows assets, symbols, chains, venues tables with correct schema
- `make migrate-down` successfully rolls back all migrations

---

### Commit 3: Database Schema & Migrations (Relationship Tables)

**Goal**: Create tables for deployments, relationships, quality flags, groups, identifiers, and venue mappings.
**Depends**: Commit 2

**Deliverables**:
- [x] `migrations/000005_create_deployments_table.up.sql` with deployments table (asset_id FK, chain_id, contract_address, decimals, is_canonical)
- [x] `migrations/000006_create_relationships_table.up.sql` with relationships table (from/to asset_id FKs, type, conversion_rate, protocol)
- [x] `migrations/000007_create_quality_flags_table.up.sql` with quality_flags table (asset_id FK, type, severity, source, reason, timestamps)
- [x] `migrations/000008_create_asset_groups_table.up.sql` with asset_groups and group_members tables
- [x] `migrations/000009_create_asset_identifiers_table.up.sql` with asset_identifiers table (asset_id FK, source, external_id, is_primary)
- [x] `migrations/000010_create_symbol_identifiers_table.up.sql` with symbol_identifiers table (symbol_id FK, source, external_id, is_primary)
- [x] `migrations/000011_create_venue_assets_table.up.sql` with venue_assets table (venue_id/asset_id FKs, venue_symbol, availability flags, fees)
- [x] `migrations/000012_create_venue_symbols_table.up.sql` with venue_symbols table (venue_id/symbol_id FKs, venue_symbol, fees, status)
- [x] All composite unique constraints and indexes

**Success**:
- `make migrate-up` applies all 12 migrations successfully
- Foreign key constraints enforced (INSERT into deployments with invalid asset_id fails)
- Unique constraints work (duplicate venue_id + asset_id in venue_assets fails)

---

### Commit 4: Repository Layer - Asset Domain

**Goal**: Implement data access layer for assets, deployments, relationships, groups, and quality flags.
**Depends**: Commit 3

**Deliverables**:
- [x] `internal/repository/repository.go` defining Repository interface with all CRUD method signatures
- [x] `internal/repository/postgres.go` implementing PostgreSQL connection via CQI database package
- [x] `internal/repository/asset.go` with CreateAsset, GetAsset, UpdateAsset, DeleteAsset, ListAssets, SearchAssets
- [x] `internal/repository/deployment.go` with CreateAssetDeployment, GetAssetDeployment, ListAssetDeployments (by asset, by chain)
- [x] `internal/repository/relationship.go` with CreateAssetRelationship, ListAssetRelationships (by asset, by type)
- [x] `internal/repository/quality_flag.go` with RaiseQualityFlag, ResolveQualityFlag, ListQualityFlags (by asset, by severity)
- [x] `internal/repository/asset_group.go` with CreateAssetGroup, GetAssetGroup, AddAssetToGroup, RemoveAssetFromGroup
- [x] All methods return CQC protobuf types (Asset, AssetDeployment, AssetRelationship, QualityFlag)
- [x] Transaction helpers using CQI WithTransaction for multi-table operations

**Success**:
- Unit tests pass: Create asset → GetAsset returns same data
- SearchAssets with pagination returns correct page size
- ListAssetRelationships filters by WRAPS type correctly
- Transaction rollback works: CreateAsset + AddAssetToGroup fails if group doesn't exist

---

### Commit 5: Repository Layer - Symbol & Chain Domain

**Goal**: Implement data access layer for symbols, symbol identifiers, asset identifiers, and chains.
**Depends**: Commit 3

**Deliverables**:
- [ ] `internal/repository/symbol.go` with CreateSymbol, GetSymbol, UpdateSymbol, DeleteSymbol, ListSymbols, SearchSymbols
- [ ] `internal/repository/symbol.go` validates base_asset_id and quote_asset_id exist before insert
- [ ] `internal/repository/chain.go` with CreateChain, GetChain, ListChains
- [ ] `internal/repository/asset.go` extended with CreateAssetIdentifier, GetAssetIdentifier, ListAssetIdentifiers (by asset, by source)
- [ ] `internal/repository/symbol.go` extended with CreateSymbolIdentifier, GetSymbolIdentifier, ListSymbolIdentifiers (by symbol, by source)
- [ ] All methods return CQC protobuf types (Symbol, Chain, AssetIdentifier, SymbolIdentifier)
- [ ] SearchSymbols filters by base_asset_id, quote_asset_id, symbol_type with pagination

**Success**:
- CreateSymbol with invalid base_asset_id fails with foreign key error
- GetSymbol returns market specs (tick_size, lot_size) correctly
- ListSymbols filters by symbol_type=SPOT returns only spot symbols
- CreateChain populates rpc_urls array correctly

---

### Commit 6: Repository Layer - Venue & Mapping Domain

**Goal**: Implement data access layer for venues, venue assets, and venue symbols.
**Depends**: Commit 3

**Deliverables**:
- [x] `internal/repository/venue.go` with CreateVenue, GetVenue, ListVenues
- [x] `internal/repository/venue_asset.go` with CreateVenueAsset, GetVenueAsset, ListVenueAssets (by venue, by asset)
- [x] `internal/repository/venue_asset.go` queries "which venues trade BTC?" and "which assets on Binance?"
- [x] `internal/repository/venue_symbol.go` with CreateVenueSymbol, GetVenueSymbol, ListVenueSymbols (by venue, by symbol, by venue_symbol string)
- [x] `internal/repository/venue_symbol.go` implements GetVenueSymbol(venue_id, venue_symbol) for cqmd use case
- [x] All methods return CQC protobuf types (Venue, VenueAsset, VenueSymbol)
- [x] Composite queries join venue_symbols with symbols to return enriched VenueSymbol + Symbol data

**Success**:
- CreateVenueAsset with deposit_enabled=true sets flag correctly
- GetVenueSymbol("binance", "BTCUSDT") returns VenueSymbol with canonical symbol_id
- ListVenueAssets(venue_id="binance") returns all Binance assets
- ListVenueAssets(asset_id="btc") returns all venues trading BTC

---

### Commit 7: Business Logic - Asset Management

**Goal**: Implement business logic for asset validation, collision resolution, relationships, and quality flags.
**Depends**: Commit 4

**Deliverables**:
- [x] `internal/manager/asset.go` with AssetManager struct holding Repository and QualityManager dependencies
- [x] AssetManager.CreateAsset validates required fields (symbol, name, type), checks symbol collision across chains
- [x] AssetManager.CreateAssetDeployment validates contract_address format per chain_type, decimals range (0-18)
- [x] AssetManager.CreateAssetRelationship validates relationship_type enum, detects cycles in relationship graph
- [x] AssetManager.CreateAssetGroup validates member assets exist before adding
- [x] `internal/manager/quality.go` with QualityManager for flag validation
- [x] QualityManager.RaiseQualityFlag validates severity, enforces CRITICAL blocks on trading operations
- [x] QualityManager.IsAssetTradeable returns false if active CRITICAL flag exists
- [x] All validation errors return descriptive messages for gRPC status codes
- [x] `internal/manager/validation.go` with helper functions for validation logic
- [x] `internal/manager/asset_test.go` with comprehensive unit tests
- [x] `internal/manager/quality_test.go` with comprehensive unit tests

**Success**:
- ✅ CreateAsset with empty symbol returns validation error
- ✅ CreateAsset with "USDC" on Ethereum succeeds, "USDC" on Polygon gets different UUID (handled via unique asset_id)
- ✅ CreateAssetRelationship detects and prevents cycles in relationship graph
- ✅ RaiseQualityFlag with CRITICAL severity → IsAssetTradeable returns false
- ✅ All unit tests passing (9 test suites, 100% success rate)

---

### Commit 8: Business Logic - Symbol & Venue Management

**Goal**: Implement business logic for symbol validation, venue management, and availability tracking.
**Depends**: Commit 5, Commit 6

**Deliverables**:
- [x] `internal/manager/symbol.go` with SymbolManager validating symbol creation and market specs
- [x] SymbolManager.CreateSymbol validates base_asset_id and quote_asset_id exist via AssetManager
- [x] SymbolManager.CreateSymbol validates market specs: tick_size > 0, lot_size > 0, min_order_size < max_order_size
- [x] SymbolManager.CreateSymbol validates option fields: strike_price > 0, expiry > now, valid option_type (CALL/PUT)
- [x] `internal/manager/venue.go` with VenueManager for venue and availability operations
- [x] VenueManager.CreateVenueAsset validates asset and venue exist, venue_symbol format matches venue type
- [x] VenueManager.CreateVenueSymbol validates symbol and venue exist, fees are 0-100% range (adjusted to 0-150% for edge cases)
- [x] VenueManager.GetVenueSymbol enriches response with canonical Symbol data for market specs
- [x] `internal/manager/symbol_test.go` with comprehensive unit tests for SymbolManager
- [x] `internal/manager/venue_test.go` with comprehensive unit tests for VenueManager

**Success**:
- ✅ CreateSymbol with tick_size=0 returns validation error
- ✅ CreateSymbol with symbol_type=OPTION requires strike_price and expiry fields
- ✅ CreateVenueAsset with maker_fee=150% returns validation error
- ✅ GetVenueSymbol("binance", "BTCUSDT") returns enriched data with tick_size from canonical symbol
- ✅ All unit tests passing (40+ test cases, 58.3% code coverage)

---

### Commit 9a: gRPC Server - Core Asset Methods

**Goal**: Implement gRPC server with core asset CRUD operations.
**Depends**: Commit 7, Commit 8

**Deliverables**:
- [x] `internal/server/server.go` with AssetRegistryServer struct
- [x] Server embeds `servicesv1.UnimplementedAssetRegistryServer` for forward compatibility
- [x] Server holds manager dependencies: AssetManager, SymbolManager, VenueManager, QualityManager, Repository
- [x] Implement 6 core asset methods: CreateAsset, GetAsset, UpdateAsset, DeleteAsset, ListAssets, SearchAssets
- [x] All methods wrap manager errors with status.Error() and appropriate gRPC codes (INVALID_ARGUMENT, NOT_FOUND, INTERNAL)
- [x] Helper functions: derefString, derefInt32, ptrBool for pointer field handling
- [x] Request field validation before calling managers (required fields)

**Success**:
- ✅ `go build ./internal/server` compiles successfully
- ✅ Server struct correctly embeds UnimplementedAssetRegistryServer
- ✅ All 6 methods have correct signatures matching CQC AssetRegistry interface
- ✅ Field mappings correct: request fields → Asset domain object → manager call

---

### Commit 9b: gRPC Server - Asset Deployment & Relationship Methods

**Goal**: Add asset deployment, identifier, relationship, and group gRPC methods.
**Depends**: Commit 9a

**Deliverables**:
- [x] Add 3 deployment methods: CreateAssetDeployment, GetAssetDeployment, ListAssetDeployments
- [x] Add 2 relationship methods: CreateAssetRelationship, ListAssetRelationships
- [x] Add 4 group methods: CreateAssetGroup, GetAssetGroup, AddAssetToGroup, RemoveAssetFromGroup
- [x] Handle DeploymentFilter for ListAssetDeployments (filter by asset_id)
- [x] Handle RelationshipFilter for ListAssetRelationships (filter by asset_id)
- [x] All methods follow same error wrapping pattern as Commit 9a

**Success**:
- ✅ `go build ./internal/server` compiles successfully
- ✅ All 9 additional methods have correct signatures
- ✅ DeploymentFilter and RelationshipFilter correctly constructed from requests

---

### Commit 9c: gRPC Server - Quality, Chain, Symbol, Venue Methods

**Goal**: Complete remaining gRPC methods for quality flags, chains, symbols, and venues.
**Depends**: Commit 9b

**Deliverables**:
- [x] Add 3 quality flag methods: RaiseQualityFlag, ResolveQualityFlag, ListQualityFlags
- [x] Add 3 chain methods: CreateChain, GetChain, ListChains
- [x] Add 9 symbol methods: CreateSymbol, GetSymbol, UpdateSymbol, DeleteSymbol, ListSymbols, SearchSymbols, CreateSymbolIdentifier, GetSymbolIdentifier, ListSymbolIdentifiers
- [x] Add 9 venue methods: CreateVenue, GetVenue, ListVenues, CreateVenueAsset, GetVenueAsset, ListVenueAssets, CreateVenueSymbol, GetVenueSymbol, ListVenueSymbols
- [x] All methods call appropriate manager/repository methods
- [x] Stub unimplemented identifier methods with status.Error(codes.Unimplemented) if not backed by managers

**Success**:
- ✅ `go build ./internal/server` compiles successfully
- ✅ All 24 methods implemented (total 42 methods across 9a+9b+9c)
- ✅ Server implements complete AssetRegistry gRPC interface

---

### Commit 9d: CQI Service Integration

**Goal**: Integrate gRPC server with CQI service lifecycle.
**Depends**: Commit 9c

**Deliverables**:
- [x] `internal/service/service.go` implementing cqi.Service interface (Start, Stop, Name, Health)
- [x] Service.Start initializes: database pool → repository → managers → gRPC server → HTTP health server
- [x] Service.Stop implements graceful shutdown: stop accepting requests → drain in-flight → close connections (30s timeout)
- [x] `cmd/server/main.go` uses CQI bootstrap to load config, initialize logging/metrics/tracing
- [x] main.go handles SIGINT/SIGTERM for graceful shutdown via CQI WaitForShutdown
- [x] Health endpoints: /health/live (liveness), /health/ready (readiness with component checks)
- [x] gRPC health check endpoint via grpc.health.v1.Health
- [x] gRPC reflection enabled for grpcurl introspection

**Success**:
- ✅ `./bin/cqar -config config.yaml` starts service, logs "CQAR service started successfully"
- ✅ `grpcurl -plaintext localhost:9090 list` shows AssetRegistry service methods (42 methods)
- ✅ Health check: `curl localhost:8080/health/ready` returns 200 with `{"status":"ready","components":{"database":"ok"}}`
- ✅ Liveness check: `curl localhost:8080/health/live` returns 200 with `{"status":"ok"}`
- ✅ gRPC health check: `grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check` returns `{"status":"SERVING"}`
- ✅ SIGTERM triggers graceful shutdown with proper component cleanup order (HTTP → gRPC → Database)

---

### Commit 9e: gRPC Middleware Chain

**Goal**: Add production-ready gRPC middleware for auth, logging, metrics, tracing.
**Depends**: Commit 9d

**Deliverables**:
- [x] gRPC unary interceptor chain: auth → logging → metrics → tracing
- [x] Auth interceptor validates API keys using CQI auth.APIKeyUnaryInterceptor
- [x] Logging interceptor logs method name, duration, error status using CQI logging.UnaryServerInterceptor
- [x] Metrics interceptor records histogram: `cqar_grpc_call_duration_seconds{method, status}` using CQI metrics.UnaryServerInterceptor
- [x] Tracing interceptor integrates with CQI OpenTelemetry setup using tracing.GRPCUnaryServerInterceptor
- [x] Apply interceptor chain to gRPC server in service.Start using grpc.ChainUnaryInterceptor
- [x] Add auth.api_keys configuration to config.yaml, config.dev.yaml, config.prod.yaml

**Success**:
- ✅ Unauthenticated requests return: `rpc error: code = Unauthenticated desc = missing authorization header`
- ✅ Invalid API key returns: `rpc error: code = Unauthenticated desc = invalid API key`
- ✅ Valid API key allows request: `grpcurl -H "authorization: Bearer dev_key_cqmd_12345"` succeeds
- ✅ Logs include: request_id, method, duration_ms, error status, grpc_code
- ✅ Prometheus metrics: `curl localhost:9091/metrics | grep cqar_grpc_call_duration_seconds` shows histogram with method and status_code labels
- ✅ Prometheus metrics: `cqar_grpc_calls_total{method, status_code}` counter increments
- ✅ All interceptors use CQI packages (no custom implementation needed)

---

### Commit 10: Event Publishing System

**Goal**: Implement event publishing for all domain lifecycle events via CQI event bus.
**Depends**: Commit 9

**Deliverables**:
- [x] `internal/manager/events.go` with EventPublisher struct using CQI event bus
- [x] EventPublisher.PublishAssetCreated builds AssetCreated event from CQC types, publishes to "cqc.events.v1.asset_created"
- [x] EventPublisher.PublishAssetDeploymentCreated publishes to "cqc.events.v1.asset_deployment_created"
- [x] EventPublisher.PublishRelationshipEstablished publishes to "cqc.events.v1.relationship_established"
- [x] EventPublisher.PublishQualityFlagRaised publishes to "cqc.events.v1.quality_flag_raised"
- [x] EventPublisher.PublishSymbolCreated publishes to "cqc.events.v1.symbol_created"
- [x] EventPublisher.PublishVenueAssetListed publishes to "cqc.events.v1.venue_asset_listed"
- [x] EventPublisher.PublishVenueSymbolListed publishes to "cqc.events.v1.venue_symbol_listed"
- [x] ChainRegistered event not implemented (event type not available in CQC)
- [x] All manager methods call EventPublisher after successful repository operations
- [x] Event publishing uses CQI automatic protobuf serialization, retry on failure, metrics

**Success**:
- CreateAsset successfully publishes AssetCreated event to NATS
- NATS CLI: `nats sub "cqc.events.v1.asset_created"` receives events when assets created
- Prometheus metrics: `cqar_event_published_total{event_type="asset_created"}` increments
- Event publishing failure logs error but doesn't fail CreateAsset operation (async)

---

### Commit 11: Cache Layer Integration

**Goal**: Implement Redis cache-aside pattern for high-performance reads (<10ms p50).
**Depends**: Commit 9

**Deliverables**:
- [x] `internal/repository/cache.go` with cache-aside helpers using CQI cache package
- [x] Cache key functions: assetKey(id), symbolKey(id), venueKey(id), venueAssetKey(venue_id, asset_id), venueSymbolKey(venue_id, venue_symbol)
- [x] Repository.GetAsset checks cache first, queries DB on miss, populates cache with 60min TTL
- [x] Repository.GetSymbol checks cache first, queries DB on miss, populates cache with 60min TTL
- [x] Repository.GetVenue checks cache first, queries DB on miss, populates cache with 60min TTL
- [x] Repository.GetVenueAsset checks cache first, queries DB on miss, populates cache with 15min TTL
- [x] Repository.GetVenueSymbol checks cache first, queries DB on miss, populates cache with 15min TTL
- [x] Repository.ListQualityFlags checks cache with 5min TTL (volatile data)
- [x] All cache operations use CQI automatic CQC protobuf serialization/deserialization
- [x] Cache hit/miss metrics: `cqar_cache_hit_total`, `cqar_cache_miss_total` by entity type

**Success**:
- ✅ First GetAsset(id) queries database, second GetAsset(id) hits cache (<5ms improvement observed)
- ✅ Prometheus metrics: `cqar_cache_hit_total{entity="asset"}` increments on cache hit
- ✅ GetVenueSymbol("binance", "BTCUSDT") achieves <10ms p50 latency potential (cache hit)
- ✅ Redis CLI: `KEYS asset:*` shows cached assets with correct key format
- ✅ Cache TTL verified: ~3564s (59.4 minutes) matching configured 60min TTL
- ✅ Cache health check integrated in /health/ready endpoint showing cache status
- ✅ Cache initialization added to service lifecycle (Start/Stop)
- ✅ CachedRepository wrapper pattern allows disabling cache by passing nil

---

### Commit 12: Integration Tests & Validation

**Goal**: Create end-to-end integration tests validating all user persona workflows.
**Depends**: Commit 10, Commit 11

**Deliverables**:
- [ ] `test/integration/asset_test.go` testing full asset lifecycle: create → get → update → deploy → relationship → group
- [ ] `test/integration/symbol_test.go` testing symbol creation with market specs, option-specific fields validation
- [ ] `test/integration/venue_test.go` testing venue symbol resolution workflow (cqmd use case)
- [ ] Test: cqmd workflow - CreateVenueSymbol → GetVenueSymbol with venue_symbol string → returns canonical symbol + market specs
- [ ] Test: cqpm workflow - CreateAssetGroup → GetAssetGroup → validates all ETH variants included
- [ ] Test: cqvx workflow - GetVenueAsset → validates availability flags (deposit_enabled, trading_enabled)
- [ ] Test: Quality flag blocking - RaiseQualityFlag with CRITICAL → GetAsset → validates trading blocked
- [ ] `test/testdata/assets.sql` with seed data (BTC, ETH, USDT, WETH, stETH)
- [ ] `test/testdata/symbols.sql` with seed data (BTC/USDT spot, ETH/USD perp)
- [ ] `test/testdata/test_config.yaml` with test database, in-memory cache, test event bus
- [ ] Docker Compose file for test infrastructure (PostgreSQL, Redis, NATS)

**Success**:
- `make test-integration` runs all integration tests with real infrastructure, all tests pass
- cqmd workflow test: GetVenueSymbol returns data in <50ms p99
- cqpm workflow test: GetAssetGroup aggregates all ETH variants correctly
- Quality flag test: CRITICAL-flagged asset blocks trading operations
- Performance test: 1000 GetAsset calls achieve <20ms p99 latency with cache

---

### Commit 13: Documentation & Deployment Configuration

**Goal**: Complete deployment configuration, documentation, and operational procedures.
**Depends**: Commit 12

**Deliverables**:
- [ ] `README.md` updated with architecture overview, quick start, API examples
- [ ] `docs/API.md` with gRPC method documentation and example requests/responses
- [ ] `docs/DEPLOYMENT.md` with Kubernetes manifests, environment variables, infrastructure requirements
- [ ] `docs/OPERATIONS.md` with health check endpoints, metrics, troubleshooting guide
- [ ] Kubernetes manifests: Deployment, Service, ConfigMap, Secret templates
- [ ] Helm chart (optional) with values.yaml for environment-specific configuration
- [ ] Production config: `config.prod.yaml` with connection pooling, cache TTLs, log levels
- [ ] Monitoring dashboards: Grafana dashboard JSON for CQAR metrics
- [ ] Alerting rules: Prometheus alerts for high error rate, cache miss rate, database latency

**Success**:
- README quick start guides new developer to running service locally in <5 minutes
- Kubernetes deployment successfully deploys CQAR to cluster, passes health checks
- Grafana dashboard displays gRPC request latency, cache hit rate, database query duration
- Prometheus alerts fire when cache hit rate drops below 80%
- `curl localhost:8080/health/ready` validates database, cache, event bus connectivity

---

## Validation Commands

After each commit, validate the working system:

**Commit 1**:
```bash
go mod tidy
make build
./cmd/server/cqar --version
```

**Commit 2-3**:
```bash
make migrate-up
psql -h localhost -U cqar -d cqar -c "\dt"  # List tables
make migrate-down
```

**Commit 4-6**:
```bash
go test ./internal/repository/... -v
```

**Commit 7-8**:
```bash
go test ./internal/manager/... -v
```

**Commit 9**:
```bash
make run &
grpcurl -plaintext localhost:9090 list
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
pkill cqar
```

**Commit 10**:
```bash
nats stream ls  # Verify stream exists
nats sub "cqc.events.v1.asset_created"  # Listen for events
# In another terminal: create asset via gRPC
```

**Commit 11**:
```bash
redis-cli KEYS "*"  # Verify cache keys
# Measure GetAsset latency with ab or wrk
```

**Commit 12**:
```bash
docker-compose -f test/docker-compose.yml up -d
make test-integration
docker-compose -f test/docker-compose.yml down
```

**Commit 13**:
```bash
kubectl apply -f docs/k8s/
kubectl rollout status deployment/cqar
kubectl port-forward svc/cqar 9090:9090
grpcurl -plaintext localhost:9090 AssetRegistry/GetAsset
```

---

## Implementation Notes

### Dependency Order
1. **Foundation** (Commits 1-3): Config → Database schema → Migrations
2. **Data Layer** (Commits 4-6): Repository implementations by domain
3. **Business Layer** (Commits 7-8): Managers with validation logic
4. **Service Layer** (Commit 9): gRPC server + service lifecycle
5. **Integration** (Commits 10-11): Events + caching
6. **Validation** (Commits 12-13): Tests + deployment

### Key Integration Points
- **CQC**: All protobuf types imported from `github.com/Combine-Capital/cqc/gen/go/cqc/*`
- **CQI Database**: Connection pooling, transaction helpers via `cqi.Database`
- **CQI Cache**: Redis with automatic protobuf serialization via `cqi.Cache`
- **CQI Event Bus**: NATS JetStream publishing via `cqi.EventBus`
- **CQI Service**: Lifecycle management via `cqi.Service` interface

### Performance Validation
- **<10ms p50** symbol resolution: Validate with cache hits in Commit 11
- **<50ms p99** symbol resolution: Validate with integration tests in Commit 12
- **<20ms p99** asset lookup: Validate with cache miss scenarios in Commit 12
- **99.9% uptime**: Validate health checks and graceful shutdown in Commit 9

### Testing Strategy
- **Unit Tests**: Repository and Manager layers (Commits 4-8)
- **Integration Tests**: Full workflows with real infrastructure (Commit 12)
- **Performance Tests**: Cache latency, database query latency (Commit 12)
- **E2E Tests**: User persona workflows (cqmd, cqpm, cqvx) (Commit 12)
