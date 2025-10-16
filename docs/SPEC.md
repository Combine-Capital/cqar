# MVP Technical Specification: CQAR - Crypto Quant Asset Registry

**Project Type:** gRPC Microservice (Reference Data Authority)

## Core Requirements (from Brief)

### MVP Scope

#### Asset Domain
- CRUD operations for Assets: CreateAsset, GetAsset, UpdateAsset, DeleteAsset, ListAssets, SearchAssets
- Assign canonical UUID per token with metadata (symbol, name, type, category)
- Resolve symbol collisions across chains (USDC on Ethereum ≠ USDC on Polygon)
- Track multi-chain deployments: asset_id, chain_id, contract_address, decimals, is_canonical flag
- Map asset relationships: WRAPS (WETH ↔ ETH), STAKES (stETH → ETH), BRIDGES (USDC.e → USDC)
- Organize assets into groups for portfolio aggregation
- Quality flags: SCAM, RUGPULL, EXPLOITED with severity levels (INFO → CRITICAL)
- Map canonical assets to external provider IDs (CoinGecko, CoinMarketCap, DeFiLlama)

#### Symbol Domain
- CRUD operations for Symbols: CreateSymbol, GetSymbol, UpdateSymbol, DeleteSymbol, ListSymbols, SearchSymbols
- Assign canonical UUID per trading pair/market
- Symbol types: SPOT, PERPETUAL, FUTURE, OPTION, MARGIN
- Store market specs: base/quote/settlement asset IDs, tick_size, lot_size, min/max order sizes
- Option-specific fields: strike_price, expiry, option_type
- Map canonical symbols to external provider IDs

#### Chain Domain
- Operations: CreateChain, GetChain, ListChains
- Store chain metadata: chain_id, name, type, native_asset_id, rpc_urls, explorer_url
- Enable asset and deployment queries by chain

#### Venue Domain
- Operations: CreateVenue, GetVenue, ListVenues
- Store venue data: venue_id, name, type (CEX/DEX/DEX_AGGREGATOR/BRIDGE/LENDING)
- Venue metadata: chain_id (DEX), protocol_address, website_url, api_endpoint, is_active flag

#### Venue Mapping Domain
- VenueAsset operations: CreateVenueAsset, GetVenueAsset, ListVenueAssets
- Map asset availability per venue with venue-specific symbols
- Availability flags: deposit_enabled, withdraw_enabled, trading_enabled
- Fees: withdraw_fee, min_withdraw amount
- VenueSymbol operations: CreateVenueSymbol, GetVenueSymbol, ListVenueSymbols
- Map canonical symbol to venue format (BTC/USDT spot → "BTCUSDT" Binance)
- Store venue-specific fees: maker_fee, taker_fee
- Track market status: is_active, listed_at, delisted_at

#### Event Publishing
- Publish domain events: AssetCreated, AssetDeploymentCreated, RelationshipEstablished, QualityFlagRaised
- Publish symbol events: SymbolCreated
- Publish availability events: VenueAssetListed, VenueSymbolListed
- Publish infrastructure events: ChainRegistered
- Use CQC protobuf types with CQI event bus auto-serialization

#### Service Infrastructure
- Implement cqi.Service interface (Start, Stop, Name, Health)
- Bootstrap via CQI: configuration, logging, metrics, tracing initialization
- Graceful shutdown: SIGTERM/SIGINT handling with cleanup hooks
- Database: CQI connection pooling and transaction helpers
- Cache: CQI Redis with automatic CQC protobuf serialization
- Health checks: /health/live, /health/ready with component status
- Structured logging via zerolog with trace context
- Prometheus metrics with standard gRPC metrics
- OpenTelemetry tracing with automatic span creation
- Configuration via viper from environment + YAML with validation

### Performance Requirements (from Success Metrics)
- Symbol resolution: <10ms p50, <50ms p99
- Asset lookup: <20ms p99
- 99.9% uptime with automatic failover
- 100% relationship coverage (zero missed variants)
- Zero trades on CRITICAL-flagged assets

### Post-MVP Scope
- External metadata sync (CoinGecko, DeFiLlama, Token Lists)
- Auto-detect suspicious tokens via contract analysis
- Version deployment history for contract upgrades
- Asset recommendation engine
- Multi-sig approval for critical flag overrides
- Service discovery registration (Redis backend)
- Relationship graph visualization

## Technology Stack

### Core Technologies
- **Go 1.21+** - Primary language for all CQ platform services, excellent concurrency, strong typing
- **Protocol Buffers v3** - Message serialization via CQC dependency, type-safe contracts
- **gRPC** - RPC framework for AssetRegistry service interface, HTTP/2 multiplexing, bidirectional streaming
- **PostgreSQL 14+** - Primary data store for reference data, ACID compliance, JSONB for flexible metadata
- **Redis 7+** - Cache layer for fast lookups, protobuf serialization via CQI
- **NATS JetStream 2.10+** - Event bus for lifecycle events, at-least-once delivery via CQI

### Platform Dependencies
- **CQC (Crypto Quant Contracts)** - `github.com/Combine-Capital/cqc` - Provides all protobuf types (Asset, Symbol, Venue, VenueAsset, VenueSymbol, etc.) and AssetRegistry gRPC service interface
- **CQI (Crypto Quant Infrastructure)** - `github.com/Combine-Capital/cqi` - Provides service lifecycle, database, cache, event bus, logging, metrics, tracing, configuration, error handling

### Observability Stack
- **zerolog** - Zero-allocation structured logging via CQI
- **Prometheus** - Metrics collection via CQI (gRPC call duration, cache hit rate, database query latency)
- **OpenTelemetry** - Distributed tracing via CQI (span creation, context propagation)

### Justification for Choices
- **PostgreSQL**: Strong consistency required for reference data; JSONB for flexible asset metadata; full-text search for SearchAssets
- **Redis**: Low-latency reads (<10ms p50) require cache layer; protobuf serialization via CQI maintains type safety
- **NATS JetStream**: Lightweight event bus via CQI; sufficient for lifecycle events (low volume)
- **gRPC**: Native protobuf support; efficient binary protocol; code generation for clients

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                              CQAR Service                                │
│                  (github.com/Combine-Capital/cqar)                      │
└─────────────────────────────────────────────────────────────────────────┘
                                   │
                                   │ imports
                 ┌─────────────────┼─────────────────┐
                 │                 │                 │
                 ▼                 ▼                 ▼
┌───────────────────────┐ ┌──────────────┐ ┌─────────────────┐
│  CQC (Contracts)      │ │ CQI (Infra)  │ │ Standard Library│
│  - Asset proto        │ │ - Service    │ │ - context       │
│  - Symbol proto       │ │ - Database   │ │ - net/http      │
│  - Venue proto        │ │ - Cache      │ │ - google.uuid   │
│  - AssetRegistry gRPC │ │ - EventBus   │ └─────────────────┘
│  - Events proto       │ │ - Logging    │
└───────────────────────┘ │ - Metrics    │
                          │ - Tracing    │
                          │ - Config     │
                          │ - Health     │
                          └──────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                        CQAR Service Architecture                         │
│                                                                          │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    gRPC Server Layer                            │    │
│  │  ┌──────────────────────────────────────────────────────────┐  │    │
│  │  │ AssetRegistry Service (implements CQC interface)         │  │    │
│  │  │  - Asset CRUD       - Symbol CRUD      - Chain CRUD      │  │    │
│  │  │  - Deployment CRUD  - Relationship CRUD - Group CRUD     │  │    │
│  │  │  - Quality Flag CRUD - Venue CRUD      - VenueAsset CRUD │  │    │
│  │  │  - VenueSymbol CRUD  - Identifier CRUD                   │  │    │
│  │  └──────────────────────────────────────────────────────────┘  │    │
│  │             │                                                    │    │
│  │             │ (middleware: auth, logging, metrics, tracing)     │    │
│  └─────────────┼────────────────────────────────────────────────────┘    │
│                │                                                         │
│  ┌─────────────▼────────────────────────────────────────────────────┐   │
│  │                    Business Logic Layer                          │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │   │
│  │  │ AssetManager │  │SymbolManager │  │ VenueManager │          │   │
│  │  │              │  │              │  │              │          │   │
│  │  │ -Validation  │  │ -Validation  │  │ -Validation  │          │   │
│  │  │ -Dedup       │  │ -Market Spec │  │ -Availability│          │   │
│  │  │ -Collision   │  │  Validation  │  │  Tracking    │          │   │
│  │  │  Resolution  │  └──────────────┘  └──────────────┘          │   │
│  │  │ -Relationship│                                               │   │
│  │  │  Graph       │  ┌──────────────┐  ┌──────────────┐          │   │
│  │  └──────────────┘  │ QualityMgr   │  │ EventPublisher│         │   │
│  │                    │              │  │              │          │   │
│  │                    │ -Flag Rules  │  │ -Domain      │          │   │
│  │                    │ -Severity    │  │  Events      │          │   │
│  │                    │  Checks      │  │ -NATS Pub    │          │   │
│  │                    └──────────────┘  └──────────────┘          │   │
│  └───────────────────────────────────────────────────────────────┘   │
│                │                      │                               │
│                ▼                      ▼                               │
│  ┌─────────────────────────┐  ┌──────────────────────────┐           │
│  │   Data Access Layer     │  │    Cache Layer (CQI)     │           │
│  │                         │  │                          │           │
│  │  ┌──────────────────┐   │  │  ┌───────────────────┐  │           │
│  │  │ Repository       │   │  │  │ Redis Cache       │  │           │
│  │  │  (via CQI DB)    │   │  │  │  (CQI protobuf)   │  │           │
│  │  │                  │◄──┼──┼──┤                   │  │           │
│  │  │ -Assets          │   │  │  │ Cache Keys:       │  │           │
│  │  │ -Symbols         │   │  │  │ asset:{id}        │  │           │
│  │  │ -Deployments     │   │  │  │ symbol:{id}       │  │           │
│  │  │ -Relationships   │   │  │  │ venue:{id}        │  │           │
│  │  │ -Quality Flags   │   │  │  │ venue_asset:{vid} │  │           │
│  │  │ -Venues          │   │  │  │ venue_symbol:{vs} │  │           │
│  │  │ -VenueAssets     │   │  │  │ TTL: 5-60min      │  │           │
│  │  │ -VenueSymbols    │   │  │  └───────────────────┘  │           │
│  │  │ -Chains          │   │  └──────────────────────────┘           │
│  │  │ -Identifiers     │   │                                         │
│  │  └──────────────────┘   │                                         │
│  └─────────────────────────┘                                         │
│                │                                                      │
│                │                                                      │
└────────────────┼──────────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      External Infrastructure                             │
│                                                                          │
│  ┌──────────────────┐  ┌──────────────┐  ┌──────────────────────────┐  │
│  │   PostgreSQL     │  │    Redis     │  │    NATS JetStream        │  │
│  │                  │  │              │  │                          │  │
│  │  Tables:         │  │  Cache Store │  │  Topics:                 │  │
│  │  -assets         │  │  (Protobuf)  │  │  -cqc.events.v1.asset_*  │  │
│  │  -deployments    │  └──────────────┘  │  -cqc.events.v1.symbol_* │  │
│  │  -relationships  │                    │  -cqc.events.v1.venue_*  │  │
│  │  -quality_flags  │                    │  -cqc.events.v1.chain_*  │  │
│  │  -asset_groups   │                    └──────────────────────────┘  │
│  │  -group_members  │                                                   │
│  │  -symbols        │  ┌──────────────────────────────────────────┐    │
│  │  -symbol_ids     │  │         Prometheus                       │    │
│  │  -chains         │  │  Metrics:                                │    │
│  │  -venues         │  │  -cqar_grpc_request_duration_seconds     │    │
│  │  -venue_assets   │  │  -cqar_cache_hit_total                   │    │
│  │  -venue_symbols  │  │  -cqar_db_query_duration_seconds         │    │
│  │  -asset_ids      │  │  -cqar_event_published_total             │    │
│  └──────────────────┘  └──────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│                         Client Services                                  │
│  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐  ┌────────┐            │
│  │  cqmd  │  │  cqpm  │  │  cqvx  │  │  cqex  │  │  cqrisk│  ...       │
│  │ (price)│  │ (port) │  │(venues)│  │(exec)  │  │ (risk) │            │
│  └────────┘  └────────┘  └────────┘  └────────┘  └────────┘            │
│      │            │           │            │            │                │
│      └────────────┴───────────┴────────────┴────────────┘                │
│                              │                                           │
│                              │ gRPC calls                                │
│                              ▼                                           │
│                  AssetRegistry.GetAsset()                                │
│                  AssetRegistry.GetSymbol()                               │
│                  AssetRegistry.GetVenueSymbol()                          │
└─────────────────────────────────────────────────────────────────────────┘
```

## Data Flow

### Primary User Flow: Market Data Service Resolving Venue Symbol

**Scenario:** cqmd receives "BTCUSDT" price update from Binance and needs to resolve it to canonical BTC/USDT spot symbol with market specs for normalization.

#### 1. Request Arrives at CQAR

```
cqmd → gRPC → CQAR.GetVenueSymbol(venue_id="binance", venue_symbol="BTCUSDT")
```

**gRPC Middleware Chain:**
- OpenTelemetry: Create span, inject trace context
- Auth: Validate API key/JWT from metadata
- Logging: Log request with trace_id
- Metrics: Increment `cqar_grpc_request_total{method="GetVenueSymbol"}`

#### 2. Business Logic Layer

**VenueManager.GetVenueSymbol():**
1. Validate input: venue_id non-empty, venue_symbol non-empty
2. Call Repository.GetVenueSymbol(venue_id, venue_symbol)

#### 3. Data Access with Cache

**Repository.GetVenueSymbol():**
1. Build cache key: `venue_symbol:binance:BTCUSDT`
2. Try Redis cache via CQI:
   ```go
   var vs VenueSymbol
   if err := cache.Get(ctx, key, &vs); err == nil {
       metrics.Increment("cqar_cache_hit_total", "entity", "venue_symbol")
       return &vs, nil // Cache hit - return in <1ms
   }
   ```
3. Cache miss - query PostgreSQL via CQI:
   ```go
   metrics.Increment("cqar_cache_miss_total", "entity", "venue_symbol")
   row := db.QueryRow(ctx, `
       SELECT vs.id, vs.venue_id, vs.symbol_id, vs.venue_symbol, 
              vs.maker_fee, vs.taker_fee, vs.is_active
       FROM venue_symbols vs
       WHERE vs.venue_id = $1 AND vs.venue_symbol = $2
   `, venue_id, venue_symbol)
   ```
4. Scan result into VenueSymbol protobuf message
5. Populate cache with 15min TTL:
   ```go
   cache.Set(ctx, key, &vs, 15*time.Minute)
   ```
6. Return VenueSymbol (contains canonical symbol_id)

#### 4. Enrich with Symbol Data

**VenueManager.GetVenueSymbol()** (continued):
1. Now have VenueSymbol with canonical `symbol_id`
2. Call Repository.GetSymbol(symbol_id) to get market specs
3. Follow same cache pattern: `symbol:{uuid}`
4. Return VenueSymbol + Symbol to caller

#### 5. Response to Client

**gRPC Response:**
```protobuf
VenueSymbol {
  id: "vs_12345"
  venue_id: "binance"
  symbol_id: "sym_btc_usdt_spot"  // canonical UUID
  venue_symbol: "BTCUSDT"
  maker_fee: 0.001
  taker_fee: 0.001
  is_active: true
  
  // Embedded Symbol details
  symbol: {
    id: "sym_btc_usdt_spot"
    base_asset_id: "asset_btc"
    quote_asset_id: "asset_usdt"
    symbol_type: SPOT
    tick_size: 0.01
    lot_size: 0.00001
    min_order_size: 0.0001
    max_order_size: 10000
  }
}
```

**cqmd now has:**
- Canonical symbol ID for database storage/queries
- Base asset (BTC) and quote asset (USDT) IDs
- Market specs (tick_size=0.01) to normalize price to 2 decimals
- Fee structure for P&L calculations

**Total latency: <10ms p50 (cache hit), <50ms p99 (cache miss + DB query)**

### Secondary Flow: Portfolio Manager Aggregating ETH Variants

**Scenario:** cqpm needs to calculate total ETH exposure across WETH, stETH, cbETH positions.

#### 1. Request Asset Group

```
cqpm → gRPC → CQAR.GetAssetGroup(name="all_eth_variants")
```

#### 2. Return Group Members

**AssetManager.GetAssetGroup():**
1. Query `asset_groups` table: id, name, description
2. Query `group_members` table: asset_id, weight
3. For each asset_id, fetch Asset details (cached)
4. Return AssetGroup with member list

**Response:**
```protobuf
AssetGroup {
  id: "group_eth"
  name: "all_eth_variants"
  members: [
    { asset_id: "asset_eth", weight: 1.0 },
    { asset_id: "asset_weth", weight: 1.0 },
    { asset_id: "asset_steth", weight: 0.98 },  // slight discount
    { asset_id: "asset_cbeth", weight: 0.97 },
    { asset_id: "asset_reth", weight: 1.05 },   // staking rewards
  ]
}
```

**cqpm now:**
- Iterates through positions table
- If asset_id matches any member, adds (quantity * weight) to total ETH exposure
- Single query to CQAR (cached) - no N+1 queries per asset

### Tertiary Flow: Creating New Asset with Deployments

**Scenario:** Admin creates USDC asset with deployments on Ethereum, Polygon, Arbitrum.

#### 1. Create Asset Request

```
admin → gRPC → CQAR.CreateAsset(Asset{
  symbol: "USDC",
  name: "USD Coin",
  asset_type: STABLECOIN,
  category: "fiat-backed"
})
```

#### 2. Business Logic Validation

**AssetManager.CreateAsset():**
1. Validate required fields: symbol, name, type
2. Check for symbol collision: query existing assets with same symbol
3. If collision exists, ensure different chain context
4. Generate UUID: `asset_id = uuid.New()`
5. Set timestamps: `created_at = now()`, `updated_at = now()`

#### 3. Database Transaction

**Repository.CreateAsset():**
1. Begin transaction via CQI:
   ```go
   tx, err := db.Begin(ctx)
   defer tx.Rollback() // auto-rollback on error
   ```
2. Insert asset:
   ```sql
   INSERT INTO assets (id, symbol, name, type, category, created_at, updated_at)
   VALUES ($1, $2, $3, $4, $5, $6, $7)
   ```
3. Commit transaction:
   ```go
   tx.Commit()
   ```

#### 4. Event Publishing

**EventPublisher.PublishAssetCreated():**
1. Build event:
   ```go
   event := &events.AssetCreated{
       AssetId: asset.Id,
       Symbol: asset.Symbol,
       Name: asset.Name,
       Type: asset.Type,
       Timestamp: timestamppb.Now(),
   }
   ```
2. Publish via CQI event bus:
   ```go
   eventBus.Publish(ctx, "cqc.events.v1.asset_created", event)
   ```
3. Automatic: logging, metrics, NATS serialization, retry on failure

#### 5. Create Deployments (subsequent requests)

```
admin → CQAR.CreateAssetDeployment(AssetDeployment{
  asset_id: "asset_usdc",
  chain_id: "ethereum",
  contract_address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
  decimals: 6,
  is_canonical: true
})
```

Same pattern: validate → insert → publish event (AssetDeploymentCreated)

**Result:**
- Asset + 3 deployments created
- 4 events published to NATS (1 AssetCreated + 3 AssetDeploymentCreated)
- Other services (cqmd, cqpm, cqvx) receive events and update local caches
- No cache invalidation needed - services subscribe to events

## System Components

### Component 1: gRPC Server Layer

**Purpose:** Expose AssetRegistry service interface defined in CQC, handle client requests

**Inputs:** 
- gRPC requests from client services (cqmd, cqpm, cqvx, etc.)
- Request metadata: API key, JWT, trace context

**Outputs:**
- gRPC responses with Asset, Symbol, Venue, VenueAsset, VenueSymbol protobuf messages
- gRPC status codes: OK, NOT_FOUND, INVALID_ARGUMENT, INTERNAL, UNAUTHENTICATED

**Dependencies:**
- CQC AssetRegistry service interface protobuf definition
- CQI gRPC middleware (auth, logging, metrics, tracing)
- Business Logic Layer (AssetManager, SymbolManager, VenueManager)

**Key Responsibilities:**
- [MVP] Implement all AssetRegistry gRPC methods (48 methods across Asset/Symbol/Venue/Chain domains)
- [MVP] Apply middleware chain: auth → logging → metrics → tracing
- [MVP] Translate business logic errors to gRPC status codes
- [MVP] Validate request messages (required fields, formats)
- [Post-MVP] Rate limiting per client
- [Post-MVP] Request/response caching for idempotent reads

**Implementation Notes:**
- Uses generated code from `cqc/gen/go/cqc/services/v1/asset_registry_grpc.pb.go`
- Struct embedding: `type Server struct { pb.UnimplementedAssetRegistryServer; ... }`
- Each method calls business logic layer, wraps errors in status.Error()

### Component 2: Business Logic Layer

#### AssetManager

**Purpose:** Enforce asset domain rules, manage relationships, handle quality flags

**Inputs:**
- Asset CRUD requests from gRPC layer
- AssetDeployment, AssetRelationship, QualityFlag requests
- AssetGroup requests

**Outputs:**
- Validated Asset protobuf messages
- Boolean validation results
- Error messages for rule violations

**Dependencies:**
- Repository (data access)
- EventPublisher (lifecycle events)
- CQI logger (structured logging)

**Key Responsibilities:**
- [MVP] Validate asset creation: symbol uniqueness per chain, required fields
- [MVP] Resolve symbol collisions: "USDC" on multiple chains → separate asset_ids
- [MVP] Manage asset relationships: validate relationship types, detect cycles
- [MVP] Quality flag rules: validate severity, track source, enforce CRITICAL blocks
- [MVP] Asset group management: validate member assets exist, enforce constraints
- [MVP] Deployment validation: contract_address format per chain, decimals range
- [Post-MVP] Asset similarity recommendations
- [Post-MVP] Automatic relationship inference (wrapped tokens)

#### SymbolManager

**Purpose:** Enforce symbol domain rules, validate market specifications

**Inputs:**
- Symbol CRUD requests from gRPC layer
- SymbolIdentifier requests

**Outputs:**
- Validated Symbol protobuf messages
- Market spec validation results

**Dependencies:**
- Repository (data access)
- AssetManager (verify base/quote/settlement assets exist)
- EventPublisher (SymbolCreated events)

**Key Responsibilities:**
- [MVP] Validate symbol creation: base_asset_id and quote_asset_id exist, unique combination
- [MVP] Validate market specs: tick_size > 0, lot_size > 0, min_order_size < max_order_size
- [MVP] Option-specific validation: strike_price > 0, expiry > now, valid option_type
- [MVP] Symbol search: filter by base/quote assets, type, pagination
- [Post-MVP] Symbol recommendation (similar markets)

#### VenueManager

**Purpose:** Manage venue registry, track asset/symbol availability per venue

**Inputs:**
- Venue CRUD requests from gRPC layer
- VenueAsset, VenueSymbol mapping requests

**Outputs:**
- Validated Venue, VenueAsset, VenueSymbol protobuf messages
- Availability status per venue

**Dependencies:**
- Repository (data access)
- AssetManager (verify assets exist)
- SymbolManager (verify symbols exist)
- EventPublisher (VenueAssetListed, VenueSymbolListed events)

**Key Responsibilities:**
- [MVP] Validate venue creation: unique venue_id, valid type, required metadata
- [MVP] VenueAsset validation: asset exists, venue exists, venue_symbol format per venue type
- [MVP] VenueSymbol validation: symbol exists, venue exists, venue_symbol format matches venue API
- [MVP] Availability tracking: listed_at, delisted_at timestamps, is_active flag
- [MVP] Fee tracking: maker_fee, taker_fee, withdraw_fee with validation (0-100%)
- [MVP] Query optimization: "which venues trade BTC?" → index scan on venue_assets
- [Post-MVP] Venue health monitoring (integration with cqvx)

#### QualityManager

**Purpose:** Manage quality flags, enforce trading blocks on risky assets

**Inputs:**
- RaiseQualityFlag, ResolveQualityFlag requests
- ListQualityFlags queries

**Outputs:**
- QualityFlag protobuf messages
- Boolean: is asset tradeable?

**Dependencies:**
- Repository (data access)
- EventPublisher (QualityFlagRaised events)
- CQI logger (audit trail)

**Key Responsibilities:**
- [MVP] Flag validation: valid flag_type, valid severity, valid source
- [MVP] Flag lifecycle: raised_at timestamp, resolved_at timestamp, reason text
- [MVP] Trading block enforcement: CRITICAL severity → is_tradeable = false
- [MVP] Query by severity: "list all CRITICAL flags" for risk dashboard
- [MVP] Audit trail: log all flag changes with user identity
- [Post-MVP] Automatic flag raising (contract analysis, price anomalies)
- [Post-MVP] Multi-sig approval for flag resolution

#### EventPublisher

**Purpose:** Publish domain events to NATS JetStream for inter-service communication

**Inputs:**
- Domain events from managers (AssetCreated, SymbolCreated, etc.)

**Outputs:**
- Published events to NATS topics
- Metrics: event_published_total

**Dependencies:**
- CQI event bus (NATS JetStream)
- CQC event protobuf messages
- CQI metrics (counters)

**Key Responsibilities:**
- [MVP] Publish lifecycle events: AssetCreated, AssetDeploymentCreated, RelationshipEstablished
- [MVP] Publish quality events: QualityFlagRaised, QualityFlagResolved
- [MVP] Publish symbol events: SymbolCreated, VenueSymbolListed
- [MVP] Topic naming: `cqc.events.v1.{event_type_snake_case}`
- [MVP] Automatic: protobuf serialization, retry on failure, metrics, logging
- [Post-MVP] Event batching for high-volume updates

### Component 3: Data Access Layer (Repository)

**Purpose:** Abstract database operations, provide cache-aside pattern

**Inputs:**
- Entity CRUD requests from business logic layer
- Query filters (asset search, symbol search)

**Outputs:**
- Protobuf messages (Asset, Symbol, Venue, etc.)
- Query results with pagination
- Transaction results (success/error)

**Dependencies:**
- CQI database package (PostgreSQL connection pool)
- CQI cache package (Redis)
- SQL query builder (manual or lightweight library)

**Key Responsibilities:**
- [MVP] Asset CRUD: Insert, Select, Update, Delete in `assets` table
- [MVP] AssetDeployment CRUD: manage `deployments` table with foreign key to assets
- [MVP] AssetRelationship CRUD: manage `relationships` table (self-referential)
- [MVP] QualityFlag CRUD: manage `quality_flags` table with timestamps
- [MVP] Symbol CRUD: manage `symbols` table with foreign keys to assets
- [MVP] Chain CRUD: manage `chains` table
- [MVP] Venue CRUD: manage `venues` table
- [MVP] VenueAsset CRUD: manage `venue_assets` table (junction table)
- [MVP] VenueSymbol CRUD: manage `venue_symbols` table (junction table)
- [MVP] AssetGroup CRUD: manage `asset_groups` and `group_members` tables
- [MVP] Cache-aside pattern: check cache → query DB on miss → populate cache
- [MVP] Cache keys: `asset:{id}`, `symbol:{id}`, `venue:{id}`, `venue_symbol:{venue_id}:{venue_symbol}`
- [MVP] Cache TTLs: 5min (frequently changing), 15min (stable reference data), 60min (rarely changing)
- [MVP] Transactions: use CQI transaction helpers for multi-table operations
- [Post-MVP] Read replicas for query scaling
- [Post-MVP] Full-text search optimization (PostgreSQL tsvector)

**Schema Design (PostgreSQL):**

```sql
-- Assets table
CREATE TABLE assets (
    id UUID PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- CRYPTOCURRENCY, STABLECOIN, NFT, etc.
    category VARCHAR(100),
    description TEXT,
    logo_url TEXT,
    website_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_assets_symbol ON assets(symbol);
CREATE INDEX idx_assets_type ON assets(type);

-- Asset deployments (multi-chain)
CREATE TABLE deployments (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    chain_id VARCHAR(50) NOT NULL,
    contract_address VARCHAR(255) NOT NULL,
    decimals SMALLINT NOT NULL,
    is_canonical BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(chain_id, contract_address)
);
CREATE INDEX idx_deployments_asset_id ON deployments(asset_id);
CREATE INDEX idx_deployments_chain_id ON deployments(chain_id);

-- Asset relationships
CREATE TABLE relationships (
    id UUID PRIMARY KEY,
    from_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    to_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) NOT NULL, -- WRAPS, STAKES, BRIDGES, etc.
    conversion_rate DECIMAL(30, 18),
    protocol VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(from_asset_id, to_asset_id, relationship_type)
);
CREATE INDEX idx_relationships_from ON relationships(from_asset_id);
CREATE INDEX idx_relationships_to ON relationships(to_asset_id);

-- Quality flags
CREATE TABLE quality_flags (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    flag_type VARCHAR(50) NOT NULL, -- SCAM, RUGPULL, EXPLOITED, etc.
    severity VARCHAR(20) NOT NULL, -- INFO, LOW, MEDIUM, HIGH, CRITICAL
    source VARCHAR(100) NOT NULL,
    reason TEXT NOT NULL,
    raised_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by VARCHAR(255)
);
CREATE INDEX idx_quality_flags_asset_id ON quality_flags(asset_id);
CREATE INDEX idx_quality_flags_severity ON quality_flags(severity);
CREATE INDEX idx_quality_flags_resolved ON quality_flags(resolved_at) WHERE resolved_at IS NULL;

-- Asset groups
CREATE TABLE asset_groups (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE group_members (
    group_id UUID NOT NULL REFERENCES asset_groups(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    weight DECIMAL(10, 6) NOT NULL DEFAULT 1.0,
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, asset_id)
);
CREATE INDEX idx_group_members_asset_id ON group_members(asset_id);

-- Symbols (trading pairs/markets)
CREATE TABLE symbols (
    id UUID PRIMARY KEY,
    base_asset_id UUID NOT NULL REFERENCES assets(id),
    quote_asset_id UUID NOT NULL REFERENCES assets(id),
    settlement_asset_id UUID REFERENCES assets(id),
    symbol_type VARCHAR(50) NOT NULL, -- SPOT, PERPETUAL, FUTURE, OPTION, MARGIN
    tick_size DECIMAL(30, 18) NOT NULL,
    lot_size DECIMAL(30, 18) NOT NULL,
    min_order_size DECIMAL(30, 18) NOT NULL,
    max_order_size DECIMAL(30, 18) NOT NULL,
    -- Option-specific fields
    strike_price DECIMAL(30, 18),
    expiry TIMESTAMPTZ,
    option_type VARCHAR(10), -- CALL, PUT
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(base_asset_id, quote_asset_id, symbol_type, strike_price, expiry)
);
CREATE INDEX idx_symbols_base_asset ON symbols(base_asset_id);
CREATE INDEX idx_symbols_quote_asset ON symbols(quote_asset_id);
CREATE INDEX idx_symbols_type ON symbols(symbol_type);

-- Asset identifiers (external providers)
CREATE TABLE asset_identifiers (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    source VARCHAR(100) NOT NULL, -- coingecko, coinmarketcap, defillama
    external_id VARCHAR(255) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(asset_id, source, external_id)
);
CREATE INDEX idx_asset_identifiers_asset_id ON asset_identifiers(asset_id);
CREATE INDEX idx_asset_identifiers_source ON asset_identifiers(source);

-- Symbol identifiers (external providers)
CREATE TABLE symbol_identifiers (
    id UUID PRIMARY KEY,
    symbol_id UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    source VARCHAR(100) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(symbol_id, source, external_id)
);

-- Chains
CREATE TABLE chains (
    id VARCHAR(50) PRIMARY KEY, -- ethereum, polygon, arbitrum
    name VARCHAR(100) NOT NULL,
    chain_type VARCHAR(50) NOT NULL, -- EVM, COSMOS, SOLANA, etc.
    native_asset_id UUID REFERENCES assets(id),
    rpc_urls TEXT[], -- PostgreSQL array
    explorer_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Venues
CREATE TABLE venues (
    id VARCHAR(100) PRIMARY KEY, -- binance, uniswap_v3_eth, dydx
    name VARCHAR(255) NOT NULL,
    venue_type VARCHAR(50) NOT NULL, -- CEX, DEX, DEX_AGGREGATOR, BRIDGE, LENDING
    chain_id VARCHAR(50) REFERENCES chains(id),
    protocol_address VARCHAR(255),
    website_url TEXT,
    api_endpoint TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_venues_type ON venues(venue_type);
CREATE INDEX idx_venues_chain_id ON venues(chain_id);

-- Venue assets (which assets available on which venues)
CREATE TABLE venue_assets (
    id UUID PRIMARY KEY,
    venue_id VARCHAR(100) NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    venue_symbol VARCHAR(50) NOT NULL, -- venue-specific symbol
    deposit_enabled BOOLEAN NOT NULL DEFAULT true,
    withdraw_enabled BOOLEAN NOT NULL DEFAULT true,
    trading_enabled BOOLEAN NOT NULL DEFAULT true,
    withdraw_fee DECIMAL(30, 18),
    min_withdraw DECIMAL(30, 18),
    listed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delisted_at TIMESTAMPTZ,
    UNIQUE(venue_id, asset_id)
);
CREATE INDEX idx_venue_assets_venue_id ON venue_assets(venue_id);
CREATE INDEX idx_venue_assets_asset_id ON venue_assets(asset_id);
CREATE INDEX idx_venue_assets_venue_symbol ON venue_assets(venue_id, venue_symbol);

-- Venue symbols (which markets on which venues)
CREATE TABLE venue_symbols (
    id UUID PRIMARY KEY,
    venue_id VARCHAR(100) NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
    symbol_id UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    venue_symbol VARCHAR(50) NOT NULL, -- venue-specific symbol format
    maker_fee DECIMAL(10, 6) NOT NULL,
    taker_fee DECIMAL(10, 6) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    listed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delisted_at TIMESTAMPTZ,
    UNIQUE(venue_id, symbol_id)
);
CREATE INDEX idx_venue_symbols_venue_id ON venue_symbols(venue_id);
CREATE INDEX idx_venue_symbols_symbol_id ON venue_symbols(symbol_id);
CREATE INDEX idx_venue_symbols_venue_symbol ON venue_symbols(venue_id, venue_symbol);
```

### Component 4: Cache Layer

**Purpose:** Reduce database load, achieve <10ms p50 read latency

**Inputs:**
- Cache keys from repository
- Protobuf messages to cache

**Outputs:**
- Cached protobuf messages (deserialized)
- Cache hit/miss metrics

**Dependencies:**
- CQI cache package (Redis client)
- CQC protobuf messages
- CQI metrics (cache hit rate)

**Key Responsibilities:**
- [MVP] Store protobuf messages with automatic serialization via CQI
- [MVP] Cache keys: consistent naming `{entity}:{id}` or `{entity}:{composite_key}`
- [MVP] TTL strategy: 5min (volatile), 15min (standard), 60min (stable)
- [MVP] Cache-aside pattern: repository checks cache → DB on miss → populate cache
- [MVP] Metrics: cache_hit_total, cache_miss_total, cache_eviction_total
- [Post-MVP] Cache warming on startup (load frequently accessed entities)
- [Post-MVP] Cache invalidation on updates (publish invalidation events)
- [Post-MVP] Distributed cache with Redis cluster

**Cache Key Design:**
```
asset:{uuid}                          → Asset protobuf, TTL=60min
symbol:{uuid}                         → Symbol protobuf, TTL=60min
venue:{venue_id}                      → Venue protobuf, TTL=60min
venue_asset:{venue_id}:{asset_id}     → VenueAsset protobuf, TTL=15min
venue_symbol:{venue_id}:{venue_symbol}→ VenueSymbol protobuf, TTL=15min
chain:{chain_id}                      → Chain protobuf, TTL=60min
quality_flags:{asset_id}              → []QualityFlag protobuf, TTL=5min
asset_group:{name}                    → AssetGroup protobuf, TTL=15min
```

### Component 5: Service Infrastructure (CQI Integration)

**Purpose:** Bootstrap service with configuration, logging, metrics, tracing, health checks

**Inputs:**
- Configuration from environment + YAML
- Service lifecycle signals (SIGTERM, SIGINT)

**Outputs:**
- Initialized service components
- Health check endpoints (/health/live, /health/ready)
- Metrics endpoint (/metrics)
- Graceful shutdown

**Dependencies:**
- CQI service interface
- CQI bootstrap packages (config, logging, metrics, tracing, health)

**Key Responsibilities:**
- [MVP] Implement cqi.Service interface: Start(ctx), Stop(ctx), Name(), Health()
- [MVP] Load configuration: environment variables (CQI_DATABASE_HOST, CQI_CACHE_HOST, etc.) + config.yaml
- [MVP] Initialize logger: zerolog with trace context, log level from config
- [MVP] Initialize metrics: Prometheus with standard gRPC metrics + custom metrics
- [MVP] Initialize tracing: OpenTelemetry with OTLP exporter, trace sampling
- [MVP] Initialize database: PostgreSQL connection pool with health checks
- [MVP] Initialize cache: Redis client with health checks
- [MVP] Initialize event bus: NATS JetStream with health checks
- [MVP] Register health checkers: database ping, Redis ping, NATS connection
- [MVP] Start gRPC server: port from config, TLS optional
- [MVP] Start HTTP server: health endpoints, metrics endpoint
- [MVP] Graceful shutdown: 30s timeout, stop accepting new requests, drain in-flight requests, close connections
- [Post-MVP] Service discovery: register with Redis registry on startup, heartbeat, deregister on shutdown

**Configuration Structure (config.yaml):**

```yaml
service:
  name: cqar
  version: 0.1.0
  environment: production

grpc:
  port: 9090
  tls_enabled: true
  tls_cert_path: /etc/cqar/tls/cert.pem
  tls_key_path: /etc/cqar/tls/key.pem

http:
  port: 8080

database:
  host: ${CQI_DATABASE_HOST}
  port: 5432
  username: ${CQI_DATABASE_USER}
  password: ${CQI_DATABASE_PASSWORD}
  database: cqar
  pool_size: 20
  max_idle_conns: 5
  conn_max_lifetime: 30m

cache:
  host: ${CQI_CACHE_HOST}
  port: 6379
  password: ${CQI_CACHE_PASSWORD}
  db: 0
  pool_size: 10

event_bus:
  url: ${CQI_NATS_URL}
  cluster_id: cq-platform
  client_id: cqar-${HOSTNAME}
  stream_name: cqc-events

logging:
  level: info
  format: json

metrics:
  enabled: true
  path: /metrics

tracing:
  enabled: true
  otlp_endpoint: ${CQI_OTLP_ENDPOINT}
  sample_rate: 0.1

auth:
  api_keys:
    - ${CQI_API_KEY_CQMD}
    - ${CQI_API_KEY_CQPM}
    - ${CQI_API_KEY_CQVX}
  jwt_public_key_path: /etc/cqar/jwt/public.pem
```

**Environment Variables:**
```bash
# Database
CQI_DATABASE_HOST=postgres.cq-platform.svc.cluster.local
CQI_DATABASE_USER=cqar
CQI_DATABASE_PASSWORD=<secret>

# Cache
CQI_CACHE_HOST=redis.cq-platform.svc.cluster.local
CQI_CACHE_PASSWORD=<secret>

# Event Bus
CQI_NATS_URL=nats://nats.cq-platform.svc.cluster.local:4222

# Tracing
CQI_OTLP_ENDPOINT=http://otel-collector.cq-platform.svc.cluster.local:4317

# Auth
CQI_API_KEY_CQMD=<secret>
CQI_API_KEY_CQPM=<secret>
CQI_API_KEY_CQVX=<secret>
```

## File Structure

```
cqar/
├── cmd/
│   └── server/
│       └── main.go                    # Service entrypoint, CQI bootstrap
│
├── internal/
│   ├── server/
│   │   └── server.go                  # gRPC server, implements AssetRegistry interface
│   │
│   ├── service/
│   │   └── service.go                 # cqi.Service implementation
│   │
│   ├── manager/
│   │   ├── asset.go                   # AssetManager (validation, relationships, groups)
│   │   ├── symbol.go                  # SymbolManager (market spec validation)
│   │   ├── venue.go                   # VenueManager (availability tracking)
│   │   ├── quality.go                 # QualityManager (flag management)
│   │   └── events.go                  # EventPublisher (NATS publishing)
│   │
│   ├── repository/
│   │   ├── repository.go              # Interface definitions
│   │   ├── postgres.go                # PostgreSQL implementation
│   │   ├── cache.go                   # Redis cache-aside helpers
│   │   ├── asset.go                   # Asset CRUD
│   │   ├── deployment.go              # Deployment CRUD
│   │   ├── relationship.go            # Relationship CRUD
│   │   ├── quality_flag.go            # Quality flag CRUD
│   │   ├── asset_group.go             # Group CRUD
│   │   ├── symbol.go                  # Symbol CRUD
│   │   ├── chain.go                   # Chain CRUD
│   │   ├── venue.go                   # Venue CRUD
│   │   ├── venue_asset.go             # VenueAsset CRUD
│   │   └── venue_symbol.go            # VenueSymbol CRUD
│   │
│   └── config/
│       └── config.go                  # Config structs (extends CQI config types)
│
├── pkg/                               # Public packages (if any - likely none for MVP)
│
├── test/
│   ├── integration/                   # Integration tests (real DB, Redis, NATS)
│   │   ├── asset_test.go
│   │   ├── symbol_test.go
│   │   └── venue_test.go
│   │
│   └── testdata/                      # Test fixtures (SQL, YAML)
│       ├── assets.sql
│       ├── symbols.sql
│       └── test_config.yaml
│
├── docs/
│   ├── BRIEF.md                       # Project brief (requirements)
│   ├── SPEC.md                        # This document
│   └── ROADMAP.md                     # Implementation roadmap
│
├── migrations/                        # Database schema migrations (golang-migrate)
│   ├── 000001_create_assets_table.up.sql
│   ├── 000001_create_assets_table.down.sql
│   ├── 000002_create_deployments_table.up.sql
│   ├── 000002_create_deployments_table.down.sql
│   └── ...
│
├── config.yaml                        # Default configuration
├── config.dev.yaml                    # Development overrides
├── config.prod.yaml                   # Production overrides
│
├── Makefile                           # Build, test, run targets
├── go.mod                             # Go module dependencies
├── go.sum
├── README.md                          # Service overview, setup instructions
└── .gitignore
```

**File Count:** ~35 files for MVP

**Key Directories:**
- `cmd/server/`: Service entrypoint, single binary
- `cmd/bootstrap/`: Data seeding utility (separate from service runtime)
- `internal/`: All implementation code (not importable by other services)
- `internal/server/`: gRPC server implementation
- `internal/manager/`: Business logic layer (domain validation, orchestration)
- `internal/repository/`: Data access layer (PostgreSQL + Redis)
- `test/integration/`: End-to-end tests with real infrastructure
- `migrations/`: Database schema versioning

## Data Bootstrap Component

**Purpose:** Seed CQAR database with initial production data from authoritative sources

**Component Location:**
```
cmd/
└── bootstrap/
    └── main.go                    # Bootstrap CLI utility
```

**Data Sources:**
- **Coinbase Top 100**: Authoritative source for initial asset list
- **CoinGecko API**: Asset deployment information (contract addresses, chain deployments, metadata)

**Architecture:**
- Separate CLI utility (not part of service runtime)
- Uses CQAR gRPC client to seed data (CQRS pattern)
- Connects to running CQAR service instance
- Validates data before insertion (never hallucinates)

**Bootstrap Workflow:**
1. Fetch Coinbase Top 100 asset list (symbol, name, type)
2. For each asset, query CoinGecko API for:
   - Contract addresses on supported chains (Ethereum, Polygon, BSC, etc.)
   - Decimals per deployment
   - Basic metadata (description, logo URL, website)
3. Create chains if not exist (Ethereum, Polygon, Solana, Bitcoin, etc.)
4. For each asset:
   - Call CreateAsset via gRPC
   - For each deployment: Call CreateAssetDeployment via gRPC
5. Skip assets with incomplete/unverifiable data

**Error Handling:**
- Log skipped assets with reason (missing contract, unverified data)
- Continue on individual asset failure (don't abort entire process)
- Report summary: successful, failed, skipped

**Configuration:**
```yaml
bootstrap:
  cqar_grpc_endpoint: "localhost:9090"
  coinbase_api_key: "<optional>"
  coingecko_api_key: "<required>"
  rate_limit_per_second: 10  # CoinGecko rate limiting
  chains:
    - ethereum
    - polygon
    - bsc
    - solana
    - bitcoin
```

**Future Enhancements (Post-MVP):**
- Incremental updates (don't re-create existing assets)
- Relationship detection (WETH/ETH, stablecoins)
- Quality flag integration (scam detection via external APIs)
- Dry-run mode for validation

## Integration Patterns

### MVP Usage Pattern: Client Service Integration

**Scenario:** cqmd (Market Data Service) integrates with CQAR to resolve venue symbols.

#### 1. Add Dependencies

```go
// go.mod
require (
    github.com/Combine-Capital/cqc v0.1.0
    google.golang.org/grpc v1.58.0
)
```

#### 2. Initialize gRPC Client

```go
// internal/client/asset_registry.go
import (
    pb "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func NewAssetRegistryClient(addr string) (pb.AssetRegistryClient, error) {
    conn, err := grpc.Dial(addr, 
        grpc.WithTransportCredentials(insecure.NewCredentials()),
        grpc.WithUnaryInterceptor(/* auth, tracing, metrics */),
    )
    if err != nil {
        return nil, err
    }
    return pb.NewAssetRegistryClient(conn), nil
}
```

#### 3. Resolve Venue Symbol

```go
// internal/handler/price_feed.go
func (h *Handler) ProcessBinancePriceUpdate(ctx context.Context, update *BinancePrice) error {
    // Resolve venue symbol to canonical symbol
    resp, err := h.assetRegistry.GetVenueSymbol(ctx, &pb.GetVenueSymbolRequest{
        VenueId:     "binance",
        VenueSymbol: update.Symbol, // "BTCUSDT"
    })
    if err != nil {
        return fmt.Errorf("resolve venue symbol: %w", err)
    }
    
    // Extract canonical symbol ID
    canonicalSymbolID := resp.VenueSymbol.SymbolId
    
    // Extract market specs for normalization
    tickSize := resp.VenueSymbol.Symbol.TickSize
    
    // Normalize price to tick size
    normalizedPrice := roundToTickSize(update.Price, tickSize)
    
    // Store with canonical ID
    return h.db.StorePriceUpdate(ctx, &PriceUpdate{
        SymbolID:  canonicalSymbolID,  // UUID
        Price:     normalizedPrice,
        Timestamp: update.Timestamp,
    })
}
```

**Key Benefits:**
- ✅ Single gRPC call to resolve venue symbol → canonical symbol with specs
- ✅ <10ms p50 latency (cache hit)
- ✅ Type-safe: protobuf messages prevent field errors
- ✅ Automatic: tracing, metrics, error handling via gRPC interceptors
- ✅ No local asset mapping tables in cqmd → all reference data in CQAR

### MVP Usage Pattern: Portfolio Aggregation

**Scenario:** cqpm aggregates ETH exposure across variants.

```go
// internal/portfolio/aggregator.go
func (a *Aggregator) CalculateAssetExposure(ctx context.Context, portfolioID string) (map[string]float64, error) {
    // Get all positions
    positions, err := a.db.GetPositions(ctx, portfolioID)
    if err != nil {
        return nil, err
    }
    
    // Get ETH variants group
    groupResp, err := a.assetRegistry.GetAssetGroup(ctx, &pb.GetAssetGroupRequest{
        Name: "all_eth_variants",
    })
    if err != nil {
        return nil, err
    }
    
    // Build asset ID → weight map
    ethVariants := make(map[string]float64)
    for _, member := range groupResp.Group.Members {
        ethVariants[member.AssetId] = member.Weight
    }
    
    // Aggregate exposure
    totalETH := 0.0
    for _, pos := range positions {
        if weight, ok := ethVariants[pos.AssetID]; ok {
            totalETH += pos.Quantity * weight
        }
    }
    
    return map[string]float64{"ETH": totalETH}, nil
}
```

**Key Benefits:**
- ✅ Single query to get all ETH variants
- ✅ No N+1 queries per asset
- ✅ Weights enable accurate aggregation (stETH ≠ 1:1 with ETH)
- ✅ Centralized group management → update once, all services benefit

### MVP Usage Pattern: Quality Flag Checking

**Scenario:** cqex (Execution Service) checks asset quality before trade.

```go
// internal/execution/validator.go
func (v *Validator) ValidateAsset(ctx context.Context, assetID string) error {
    // Get active quality flags for asset
    resp, err := v.assetRegistry.ListQualityFlags(ctx, &pb.ListQualityFlagsRequest{
        AssetId: assetID,
        ActiveOnly: true,
    })
    if err != nil {
        return fmt.Errorf("check quality flags: %w", err)
    }
    
    // Block trade if CRITICAL flag exists
    for _, flag := range resp.Flags {
        if flag.Severity == pb.FlagSeverity_CRITICAL {
            return fmt.Errorf("asset %s has CRITICAL flag: %s (%s)", 
                assetID, flag.Type, flag.Reason)
        }
    }
    
    return nil // Safe to trade
}
```

**Key Benefits:**
- ✅ Centralized quality flag management
- ✅ Automatic block on CRITICAL flags → zero trades on risky assets
- ✅ Audit trail in CQAR logs

### Post-MVP Extensions

**Future capabilities without building for them now:**

1. **External Metadata Sync**
   - Scheduled jobs to sync from CoinGecko, DeFiLlama
   - Automatic asset creation from token lists
   - Logo/description updates

2. **Contract Analysis Integration**
   - Automatic quality flag raising on honeypot detection
   - Integration with Certik, TokenSniffer APIs
   - Automated scam detection

3. **Service Discovery**
   - Register CQAR with Redis registry on startup
   - Health check heartbeat every 30s
   - Automatic deregistration on shutdown
   - Client-side load balancing

4. **Advanced Caching**
   - Cache warming on startup (top 1000 assets/symbols)
   - Cache invalidation events (publish on update)
   - Distributed cache with Redis Cluster

5. **Multi-Region Deployment**
   - Read replicas per region
   - Cache per region (Redis Cluster)
   - Event replication (NATS clustering)

6. **GraphQL API**
   - Expose GraphQL endpoint for web dashboard
   - Complex queries: asset graph, relationship traversal
   - Real-time subscriptions (venue availability changes)

7. **Bulk Operations**
   - BatchCreateAssets for initial data load
   - BatchUpdateVenueSymbols for exchange API scraping
   - Streaming APIs for large result sets

**Implementation Strategy:**
- Build only MVP features now
- Design interfaces to support extensions (Repository interface, Manager abstraction)
- Document extension points in code comments
- Validate MVP performance before adding features
- Use feature flags for gradual rollout of Post-MVP features

---

## Summary

**CQAR** is a gRPC microservice implementing the CQC AssetRegistry interface, serving as the central reference data authority for the CQ trading platform. It manages Assets (tokens), Symbols (trading pairs), Venues (exchanges), and their relationships/mappings.

**Key Architectural Decisions:**
1. **PostgreSQL for reference data**: Strong consistency, JSONB flexibility, full-text search
2. **Redis cache layer**: <10ms p50 reads via cache-aside pattern, protobuf serialization
3. **NATS JetStream for events**: Lightweight pub/sub for lifecycle events (low volume)
4. **CQI infrastructure library**: Service lifecycle, database, cache, event bus, observability out-of-the-box
5. **CQC contracts**: All protobuf types and service interface from CQC, ensuring platform consistency

**Performance Targets:**
- <10ms p50, <50ms p99 symbol resolution (cache hit)
- <20ms p99 asset lookup (cache miss)
- 99.9% uptime with health checks and automatic failover

**Deployment:**
- Single Go binary
- Kubernetes deployment with horizontal scaling (stateless)
- PostgreSQL primary + read replicas
- Redis cache per region
- NATS JetStream cluster

**Success Criteria:**
- 8 services depend on CQAR (cqmd, cqpm, cqex, cqvx, cqdefi, cqrisk, cqrep, cqstrat)
- Zero local asset mapping tables in consuming services
- 100% relationship coverage (all asset variants grouped)
- Zero trades on CRITICAL-flagged assets
- 100% of traded assets registered within 7 days
