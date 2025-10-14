# MVP Technical Specification: CQAR - Crypto Quant Asset Registry

## Core Requirements (from Brief)

### MVP Scope
**Asset & Token Identification:**
- Assign and resolve canonical asset IDs (CQC Asset, AssetIdentifier, AssetDeployment)
- Resolve symbols to addresses with collision detection (multiple "USDC" tokens across chains)
- Store metadata: symbol, name, AssetType, category, logo, description, decimals, DataSource
- Track multi-chain deployments: chain_id, address, decimals, canonical flag
- Define relationships: wraps, bridges, stakes, synthetic, LP pairs, migrations
- Organize into groups for aggregation (AssetGroup: "all ETH variants")
- Query: assets by chain, venue, deployment addresses, wrapped/bridged variants, LST variants, LP tokens, CEX ticker mappings

**Asset Quality & Filtering:**
- Flag quality issues with 10+ FlagTypes and 5 severity levels
- Query tokens by flag type and severity threshold
- Query tokens flagged by security auditors (filter by source)
- Track flag lifecycle (raised, cleared, evidence)

**Infrastructure & Operations:**
- Register chains and venues with metadata
- Implement 30+ gRPC operations from CQC AssetRegistry interface
- Search with pagination and filtering (name, symbol, address, venue ticker)
- Publish 6 event types via CQI event bus

**Performance Targets:**
- Symbol → canonical ID resolution: <50ms p99, <10ms p50
- 100% coverage of traded assets within 7 days

### Post-MVP (Future)
- Sync metadata from external sources (CoinGecko, DeFiLlama, Token Lists)
- Auto-detect suspicious tokens via contract analysis
- Version deployment history for contract upgrades
- Asset recommendations
- Multi-sig approval for flag overrides
- Relationship graph visualization

## Technology Stack

### Core Technologies
- **Language:** Go 1.21+ (platform standard, type-safe, excellent concurrency)
- **Protocol:** gRPC + Protocol Buffers (CQC contract enforcement)
- **Database:** PostgreSQL 15+ (ACID compliance, complex queries, JSON support)
- **Cache:** Redis 7+ (sub-10ms lookups, relationship graph queries)
- **Event Bus:** NATS/RabbitMQ via CQI (async event publishing)

### Platform Dependencies
- **CQC (Crypto Quant Contracts):** `github.com/Combine-Capital/cqc` - Proto definitions, gRPC interfaces
- **CQI (Crypto Quant Infrastructure):** `github.com/Combine-Capital/cqi` - Database, events, logging, metrics, tracing

### Supporting Libraries
- **Database:** `jackc/pgx/v5` - PostgreSQL driver (used by CQI)
- **Migration:** `golang-migrate/migrate` - Schema versioning
- **Validation:** `go-playground/validator/v10` - Request validation
- **Testing:** `stretchr/testify` - Test assertions

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         External Clients                             │
│  (cqpm, cqmd, cqex, cqvx, cqdefi, cqrisk, cqrep, cqstrat)          │
└─────────────────────────┬───────────────────────────────────────────┘
                          │ gRPC (CQC AssetRegistry Interface)
                          ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      CQAR gRPC Server                                │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │  AssetRegistry Service Implementation                         │  │
│  │  - Asset CRUD (Create, Get, Update, Delete, List, Search)    │  │
│  │  - Deployment Management (Create, Get, List)                 │  │
│  │  - Identifier Mapping (Create, Get, List)                    │  │
│  │  - Relationship Management (Create, List)                    │  │
│  │  - Group Management (Create, Get, Add, Remove)               │  │
│  │  - Quality Flags (Raise, Resolve, List)                      │  │
│  │  - Chain Registry (Create, Get, List)                        │  │
│  │  - Venue Symbols (Create, Get, List)                         │  │
│  └─────────────┬────────────────────────────────────────────────┘  │
└────────────────┼───────────────────────────────────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
        ▼                 ▼
┌───────────────┐  ┌──────────────────┐
│  Domain Layer │  │  Event Publisher │
│               │  │   (via CQI)      │
│ - Repository  │  └────────┬─────────┘
│   Interface   │           │
│ - Business    │           │ Events:
│   Logic       │           │ - AssetCreated
│ - Validation  │           │ - AssetDeploymentCreated
│               │           │ - RelationshipEstablished
└───────┬───────┘           │ - QualityFlagSet
        │                   │ - ChainRegistered
        ▼                   │ - VenueSymbolMapped
┌──────────────────────┐    │
│  Repository Layer    │    │
│                      │    ▼
│ - PostgreSQL Impl    │  ┌─────────────┐
│ - Redis Cache        │  │  Event Bus  │
│ - CQI Database       │  │  (CQI NATS) │
│                      │  └─────────────┘
└──────┬───────────────┘
       │
       ▼
┌──────────────────────────────────────┐
│         Data Storage                  │
│  ┌────────────────┐  ┌─────────────┐ │
│  │   PostgreSQL   │  │    Redis    │ │
│  │                │  │             │ │
│  │ - assets       │  │ - symbol    │ │
│  │ - deployments  │  │   lookup    │ │
│  │ - identifiers  │  │ - group     │ │
│  │ - relationships│  │   cache     │ │
│  │ - groups       │  │ - flag      │ │
│  │ - quality_flags│  │   index     │ │
│  │ - chains       │  └─────────────┘ │
│  │ - venues       │                   │
│  │ - venue_symbols│                   │
│  └────────────────┘                   │
└───────────────────────────────────────┘
```

## Data Flow

### Primary Flow: Resolve Symbol to Canonical Asset

1. **Client Request** (e.g., cqpm queries "USDC" on Ethereum)
   ```
   SearchAssets(query="USDC", chain_id="ethereum")
   ```

2. **Service Layer** validates request, checks cache
   - Redis cache hit → return immediately (p50 <10ms)
   - Cache miss → query PostgreSQL

3. **Repository Layer** executes optimized query
   ```sql
   SELECT a.* FROM assets a
   JOIN deployments d ON a.asset_id = d.asset_id
   WHERE a.symbol ILIKE 'USDC' 
     AND d.chain_id = 'ethereum'
   LIMIT 10;
   ```

4. **Response** returns Asset with collision warnings
   - Single match: canonical asset
   - Multiple matches: list with disambiguation metadata
   - Cache result in Redis (TTL: 1 hour)

5. **Performance**: <50ms p99, <10ms p50

### Secondary Flow: Aggregate Asset Variants (Portfolio Manager)

1. **Client Request** (cqpm aggregates all ETH exposure)
   ```
   GetAssetGroup(canonical_symbol="ETH")
   ListAssetRelationships(parent_asset_id=eth_uuid)
   ```

2. **Service Layer** retrieves asset group
   - Query group members: ETH, WETH, stETH, cbETH, rETH
   - Include relationships: wraps, stakes

3. **Client** sums positions across all variants
   - Result: Total ETH exposure across chains/forms

### Tertiary Flow: Block Trade on Flagged Asset

1. **Client Request** (cqex validates asset before trade)
   ```
   ListQualityFlags(asset_id=token_uuid, min_severity=CRITICAL)
   ```

2. **Service Layer** queries active flags
   - Check cache first (flag index)
   - PostgreSQL: `WHERE cleared_at IS NULL AND severity >= 5`

3. **Response** returns active critical flags
   - Empty: safe to trade
   - Non-empty: block trade, log reason

4. **Performance**: <5ms from cache, <20ms from DB

## System Components

### gRPC Server
**Purpose:** Expose CQC AssetRegistry interface to platform services  
**Inputs:** gRPC requests (30+ operations)  
**Outputs:** CQC proto responses, gRPC errors  
**Dependencies:** CQI logging/metrics/tracing, Domain Layer  
**Key Responsibilities:**
- Parse and validate gRPC requests
- Route to appropriate domain handlers
- Map domain errors to gRPC status codes
- Record metrics (latency, error rates)
- Trace distributed requests

### Domain Layer
**Purpose:** Business logic and orchestration  
**Inputs:** Validated requests from gRPC layer  
**Outputs:** Domain models, business errors  
**Dependencies:** Repository interface, Event Publisher, CQI validation  
**Key Responsibilities:**
- Enforce business rules (collision detection, relationship validity)
- Coordinate multi-entity operations (CreateAsset + CreateDeployment)
- Generate canonical IDs (UUIDs)
- Trigger event publishing
- Cache invalidation decisions

### Repository Layer
**Purpose:** Data persistence and retrieval  
**Inputs:** Domain queries, models  
**Outputs:** Persisted/retrieved models  
**Dependencies:** CQI Database, Redis client  
**Key Responsibilities:**
- Execute optimized PostgreSQL queries
- Manage cache read-through/write-through patterns
- Handle transactions for multi-table operations
- Implement pagination and filtering
- Build complex queries (relationships, groups)

### Event Publisher
**Purpose:** Async notification to platform services  
**Inputs:** Domain events  
**Outputs:** Published events to CQI event bus  
**Dependencies:** CQI event bus client  
**Key Responsibilities:**
- Serialize CQC event protos
- Publish to appropriate topics
- Handle publish failures (retry logic via CQI)
- Event types:
  - `AssetCreated`: new canonical asset registered
  - `AssetDeploymentCreated`: on-chain deployment added
  - `RelationshipEstablished`: asset relationship defined
  - `QualityFlagSet`: quality issue flagged
  - `ChainRegistered`: new blockchain added
  - `VenueSymbolMapped`: venue ticker mapped

### PostgreSQL Schema
**Purpose:** ACID-compliant persistent storage  
**Tables:**
- `assets`: canonical assets (asset_id PK, symbol, name, type, category, metadata)
- `asset_deployments`: on-chain deployments (deployment_id PK, asset_id FK, chain_id FK, address, decimals)
- `asset_identifiers`: external IDs (asset_id FK, source, external_id, is_primary)
- `asset_relationships`: parent-child links (parent_id FK, child_id FK, type, conversion_rate, protocol)
- `asset_groups`: logical groupings (group_id PK, canonical_symbol, issuer)
- `asset_group_members`: group membership (group_id FK, asset_id FK, is_canonical)
- `asset_quality_flags`: quality issues (asset_id FK, flag_type, severity, source, flagged_at, cleared_at, evidence_url)
- `chains`: blockchain registry (chain_id PK, name, type, native_asset, rpc_url, explorer_url)
- `venues`: trading venues (venue_id PK, name, venue_type, api_url)
- `venue_symbols`: ticker mappings (venue_id FK, symbol, asset_id FK, is_active)

**Indexes:**
- `assets`: (symbol), (asset_type), (category), (symbol, asset_type)
- `asset_deployments`: (asset_id), (chain_id), (chain_id, address) UNIQUE
- `asset_relationships`: (parent_id), (child_id), (relationship_type)
- `asset_quality_flags`: (asset_id, severity, cleared_at), (flag_type), (source)
- `venue_symbols`: (venue_id, symbol) UNIQUE, (asset_id)

### Redis Cache
**Purpose:** Sub-10ms lookups for hot paths  
**Keys:**
- `asset:id:{uuid}` → Asset JSON (TTL: 1h)
- `asset:symbol:{symbol}:{chain_id}` → Asset UUID list (TTL: 1h)
- `asset:address:{chain_id}:{address}` → Asset UUID (TTL: 1h)
- `group:{group_id}:members` → Asset UUID set (TTL: 30m)
- `flags:{asset_id}:critical` → Flag count (TTL: 5m)
- `venue:{venue_id}:symbol:{symbol}` → Asset UUID (TTL: 1h)

**Cache Invalidation:**
- Write-through on Create/Update operations
- Explicit delete on asset modifications
- TTL-based expiration for eventual consistency

## File Structure

```
services/cqar/
├── cmd/
│   └── server/
│       └── main.go                    # Server bootstrap, config loading
├── internal/
│   ├── domain/
│   │   ├── asset.go                   # Asset domain logic
│   │   ├── deployment.go              # Deployment domain logic
│   │   ├── relationship.go            # Relationship domain logic
│   │   ├── group.go                   # Group domain logic
│   │   ├── quality.go                 # Quality flag domain logic
│   │   ├── chain.go                   # Chain registry domain logic
│   │   ├── venue.go                   # Venue domain logic
│   │   └── errors.go                  # Domain error types
│   ├── repository/
│   │   ├── interface.go               # Repository interface definitions
│   │   ├── postgres/
│   │   │   ├── asset.go               # Asset PostgreSQL implementation
│   │   │   ├── deployment.go          # Deployment queries
│   │   │   ├── relationship.go        # Relationship queries
│   │   │   ├── group.go               # Group queries
│   │   │   ├── quality.go             # Quality flag queries
│   │   │   ├── chain.go               # Chain queries
│   │   │   ├── venue.go               # Venue queries
│   │   │   └── queries.sql            # SQL query definitions
│   │   └── cache/
│   │       ├── redis.go               # Redis cache implementation
│   │       └── keys.go                # Cache key patterns
│   ├── service/
│   │   ├── asset_registry.go          # gRPC service implementation
│   │   ├── validators.go              # Request validation
│   │   └── mappers.go                 # Proto <-> Domain mapping
│   └── events/
│       ├── publisher.go               # Event publishing logic
│       └── types.go                   # Event type definitions
├── pkg/
│   └── (empty - internal-only service)
├── test/
│   ├── integration/
│   │   ├── asset_test.go              # Asset flow tests
│   │   ├── deployment_test.go         # Deployment tests
│   │   ├── relationship_test.go       # Relationship tests
│   │   └── fixtures.go                # Test data builders
│   └── testdata/
│       ├── schema.sql                 # Test database schema
│       └── seed.sql                   # Test data seeds
├── migrations/
│   ├── 001_create_assets.up.sql
│   ├── 001_create_assets.down.sql
│   ├── 002_create_deployments.up.sql
│   ├── 002_create_deployments.down.sql
│   ├── 003_create_relationships.up.sql
│   ├── 003_create_relationships.down.sql
│   ├── 004_create_groups.up.sql
│   ├── 004_create_groups.down.sql
│   ├── 005_create_quality_flags.up.sql
│   ├── 005_create_quality_flags.down.sql
│   ├── 006_create_chains.up.sql
│   ├── 006_create_chains.down.sql
│   ├── 007_create_venues.up.sql
│   ├── 007_create_venues.down.sql
│   ├── 008_create_indexes.up.sql
│   └── 008_create_indexes.down.sql
├── docs/
│   ├── BRIEF.md                       # Project brief (existing)
│   ├── SPEC.md                        # This technical spec
│   └── ROADMAP.md                     # Implementation phases
├── go.mod
├── go.sum
├── Makefile                           # Build, test, migrate commands
└── README.md

Generated by CQC (via buf or protoc):
├── vendor/
│   └── github.com/Combine-Capital/cqc/
│       └── gen/go/cqc/
│           ├── assets/v1/             # Asset protos
│           ├── services/v1/           # AssetRegistry gRPC interface
│           └── events/v1/             # Event protos
```

## Database Schema

### Core Tables

```sql
-- Canonical Assets
CREATE TABLE assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    asset_type VARCHAR(50) NOT NULL,
    category VARCHAR(100),
    description TEXT,
    logo_url VARCHAR(500),
    website_url VARCHAR(500),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_assets_symbol ON assets(symbol) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_type ON assets(asset_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_assets_category ON assets(category) WHERE deleted_at IS NULL;

-- On-Chain Deployments
CREATE TABLE asset_deployments (
    deployment_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES assets(asset_id),
    chain_id VARCHAR(50) NOT NULL REFERENCES chains(chain_id),
    contract_address VARCHAR(100) NOT NULL,
    decimals INT NOT NULL,
    is_native BOOLEAN DEFAULT FALSE,
    is_canonical BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(chain_id, contract_address)
);

CREATE INDEX idx_deployments_asset ON asset_deployments(asset_id);
CREATE INDEX idx_deployments_chain ON asset_deployments(chain_id);
CREATE INDEX idx_deployments_address ON asset_deployments(chain_id, contract_address);

-- External Identifiers
CREATE TABLE asset_identifiers (
    asset_id UUID NOT NULL REFERENCES assets(asset_id),
    source VARCHAR(50) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY(asset_id, source, external_id)
);

CREATE INDEX idx_identifiers_external ON asset_identifiers(source, external_id);

-- Asset Relationships
CREATE TABLE asset_relationships (
    parent_asset_id UUID NOT NULL REFERENCES assets(asset_id),
    child_asset_id UUID NOT NULL REFERENCES assets(asset_id),
    relationship_type VARCHAR(50) NOT NULL,
    conversion_rate NUMERIC(36,18),
    protocol VARCHAR(100),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY(parent_asset_id, child_asset_id, relationship_type)
);

CREATE INDEX idx_relationships_parent ON asset_relationships(parent_asset_id);
CREATE INDEX idx_relationships_child ON asset_relationships(child_asset_id);
CREATE INDEX idx_relationships_type ON asset_relationships(relationship_type);

-- Asset Groups
CREATE TABLE asset_groups (
    group_id VARCHAR(100) PRIMARY KEY,
    canonical_symbol VARCHAR(20) NOT NULL,
    issuer VARCHAR(100),
    description TEXT,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE asset_group_members (
    group_id VARCHAR(100) NOT NULL REFERENCES asset_groups(group_id),
    asset_id UUID NOT NULL REFERENCES assets(asset_id),
    is_canonical BOOLEAN DEFAULT FALSE,
    added_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY(group_id, asset_id)
);

CREATE INDEX idx_group_members_asset ON asset_group_members(asset_id);

-- Quality Flags
CREATE TABLE asset_quality_flags (
    flag_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    asset_id UUID NOT NULL REFERENCES assets(asset_id),
    flag_type VARCHAR(50) NOT NULL,
    severity INT NOT NULL,
    source VARCHAR(100) NOT NULL,
    flagged_at TIMESTAMP NOT NULL DEFAULT NOW(),
    cleared_at TIMESTAMP,
    notes TEXT,
    evidence_url VARCHAR(500),
    metadata JSONB
);

CREATE INDEX idx_flags_asset_active ON asset_quality_flags(asset_id, severity, cleared_at);
CREATE INDEX idx_flags_type ON asset_quality_flags(flag_type) WHERE cleared_at IS NULL;

-- Blockchain Registry
CREATE TABLE chains (
    chain_id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    chain_type VARCHAR(50) NOT NULL,
    native_asset_id UUID REFERENCES assets(asset_id),
    rpc_url VARCHAR(500),
    explorer_url VARCHAR(500),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Venue Registry
CREATE TABLE venues (
    venue_id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    venue_type VARCHAR(50) NOT NULL,
    api_url VARCHAR(500),
    website_url VARCHAR(500),
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Venue Symbol Mappings
CREATE TABLE venue_symbols (
    venue_id VARCHAR(100) NOT NULL REFERENCES venues(venue_id),
    symbol VARCHAR(50) NOT NULL,
    asset_id UUID NOT NULL REFERENCES assets(asset_id),
    is_active BOOLEAN DEFAULT TRUE,
    metadata JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY(venue_id, symbol)
);

CREATE INDEX idx_venue_symbols_asset ON venue_symbols(asset_id);
```

## Integration Patterns

### MVP Usage Pattern: Portfolio Manager Aggregation

**Scenario:** cqpm needs to calculate total ETH exposure across all forms

```go
// 1. Get asset group for ETH
groupResp := client.GetAssetGroup(ctx, &GetAssetGroupRequest{
    CanonicalSymbol: "ETH",
})

// 2. List all members in the group
membersResp := client.ListAssetGroupMembers(ctx, &ListAssetGroupMembersRequest{
    GroupId: groupResp.Group.GroupId,
})

// 3. For each member, get relationships
for _, member := range membersResp.Members {
    relationshipsResp := client.ListAssetRelationships(ctx, &ListAssetRelationshipsRequest{
        ParentAssetId: member.AssetId,
    })
    
    // Process positions for member.AssetId and all child assets
}

// 4. Sum positions across all variants
// Result: Total ETH exposure = native ETH + WETH + stETH + cbETH + rETH + ...
```

**Performance:** 3-5 RPC calls, <100ms total with caching

### MVP Usage Pattern: Market Data Symbol Resolution

**Scenario:** cqmd receives price tick from Binance for "BTCUSDT"

```go
// 1. Resolve venue symbol to canonical asset
symbolResp := client.GetVenueSymbol(ctx, &GetVenueSymbolRequest{
    VenueId: "binance",
    Symbol:  "BTCUSDT",
})

// 2. Get asset details
assetResp := client.GetAsset(ctx, &GetAssetRequest{
    AssetId: symbolResp.AssetId,
})

// 3. Store price with canonical asset_id
priceService.StorePrice(symbolResp.AssetId, price, timestamp)
```

**Performance:** <10ms from cache, <30ms from DB

### MVP Usage Pattern: Pre-Trade Quality Check

**Scenario:** cqex validates asset before executing trade

```go
// 1. Check for active critical flags
flagsResp := client.ListQualityFlags(ctx, &ListQualityFlagsRequest{
    AssetId:      assetId,
    MinSeverity:  FLAG_SEVERITY_CRITICAL,
    ActiveOnly:   true,
})

// 2. Block trade if critical flags exist
if len(flagsResp.Flags) > 0 {
    return fmt.Errorf("asset %s has critical quality flags: %v", 
        assetId, flagsResp.Flags)
}

// 3. Proceed with trade
return executeTrade(assetId, amount)
```

**Performance:** <5ms from cache

### Post-MVP Extensions

**External Metadata Sync:**
- Scheduled jobs pull from CoinGecko, DeFiLlama APIs
- Enrich asset metadata, detect new deployments
- Auto-create relationships based on token lists

**Contract Analysis:**
- Background service analyzes new deployments
- Detect honeypots, unusual permissions, tax mechanisms
- Auto-raise quality flags for suspicious contracts

**Multi-Sig Approval:**
- Critical flag overrides require 2-of-3 admin signatures
- Implement approval workflow for high-stakes operations

**Relationship Graphs:**
- Generate D3.js/Graphviz visualizations
- Show asset relationship networks
- Aid in debugging aggregation logic

## Implementation Phases

### Phase 1: Foundation (Week 1-2)
- [ ] Project scaffolding (directory structure, go.mod)
- [ ] Database schema and migrations
- [ ] CQI integration (database, logging, metrics)
- [ ] Basic gRPC server setup

### Phase 2: Core Entities (Week 3-4)
- [ ] Asset CRUD operations
- [ ] Deployment management
- [ ] Chain and venue registry
- [ ] PostgreSQL repository implementations

### Phase 3: Relationships & Groups (Week 5)
- [ ] Asset relationships
- [ ] Asset groups and membership
- [ ] Complex queries (aggregation, filtering)

### Phase 4: Quality & Filtering (Week 6)
- [ ] Quality flag management
- [ ] Flag queries and filters
- [ ] Cache implementation (Redis)

### Phase 5: Integration & Performance (Week 7-8)
- [ ] Event publishing (CQI event bus)
- [ ] Search optimization and caching
- [ ] Integration tests
- [ ] Performance testing (<50ms p99)
- [ ] Documentation and deployment

### Phase 6: Production Readiness (Week 9-10)
- [ ] Observability (metrics, traces, dashboards)
- [ ] Error handling and retries
- [ ] Load testing (validate performance targets)
- [ ] Client libraries for downstream services
- [ ] Runbook and operational docs

## Success Criteria

1. **API Completeness:** All 30+ CQC AssetRegistry operations implemented
2. **Performance:** Symbol lookups <50ms p99, <10ms p50 from cache
3. **Coverage:** Support 10+ chains, 50+ venues at launch
4. **Adoption:** All 8 platform services (cqmd, cqpm, cqex, cqvx, cqdefi, cqrisk, cqrep, cqstrat) use cqar exclusively; zero local asset mappings
5. **Aggregation:** Zero missed relationships in portfolio queries (all ETH variants correctly grouped)
6. **Quality Protection:** Zero trades on critical-severity flagged tokens
