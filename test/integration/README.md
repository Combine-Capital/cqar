# Integration Tests

This directory contains end-to-end integration tests for CQAR that validate complete user workflows with real infrastructure.

## Setup

### Prerequisites

1. **Docker & Docker Compose** - Required for test infrastructure
2. **Go 1.21+** - For running tests
3. **PostgreSQL, Redis, NATS** - Provided via Docker Compose

### Quick Start

```bash
# Start test infrastructure
cd test
docker-compose up -d

# Wait for services to be healthy
docker-compose ps

# Run migrations on test database
cd ..
export DATABASE_URL="postgres://cqar_test:cqar_test_password@localhost:5433/cqar_test?sslmode=disable"
make migrate-up

# Run integration tests
go test ./test/integration/... -v

# Cleanup
cd test
docker-compose down -v
```

## Test Infrastructure

### Docker Compose Services

- **PostgreSQL** (port 5433): Test database with same schema as production
- **Redis** (port 6380): Cache layer for performance tests  
- **NATS JetStream** (port 4223): Event bus for event publishing tests

### Configuration

Test configuration is in `test/config.test.yaml` with:
- Lower resource limits (10 DB connections vs 25 in prod)
- Shorter TTLs (5min cache vs 60min in prod)
- Debug logging enabled
- Metrics disabled

## Test Structure

### Test Files

- `helpers.go` - Test fixture and infrastructure setup
- `asset_test.go` - Asset lifecycle, validation, collision handling
- `symbol_test.go` - Symbol creation, market specs, options
- `venue_test.go` - Venue symbol resolution (cqmd workflow)

### Test Data

Seed data in `testdata/*.sql`:
- `assets.sql` - BTC, ETH, USDC (multiple chains), stablecoins
- `chains.sql` - Ethereum, Polygon, Solana, Bitcoin
- `deployments.sql` - Token contracts on various chains
- `relationships.sql` - WRAPS, STAKES, BRIDGES relationships
- `symbols.sql` - Spot, perpetual, future, option symbols
- `venues.sql` - Binance, Coinbase, Uniswap, dYdX
- `venue_assets.sql` - Asset availability per venue
- `venue_symbols.sql` - Trading pairs per venue

## Test Scenarios

### Asset Lifecycle (`TestAssetLifecycle`)
Tests complete CRUD flow:
1. CreateAsset → verify ID generated
2. GetAsset → verify data persisted
3. UpdateAsset → verify changes applied
4. CreateAssetDeployment → link to chain
5. CreateAssetRelationship → establish WRAPS relationship
6. CreateAssetGroup → portfolio aggregation
7. DeleteAsset → verify dependency blocking

**Acceptance Criteria:**
- ✅ All operations succeed in sequence
- ✅ Generated IDs are valid UUIDs
- ✅ Relationships maintained correctly
- ✅ Delete blocked when dependencies exist

### Asset Validation (`TestAssetValidation`)
Tests input validation:
- Missing required fields (symbol, name, type) → InvalidArgument error
- Invalid asset type enum → validation error
- Empty strings → validation error

**Acceptance Criteria:**
- ✅ All validation errors return `codes.InvalidArgument`
- ✅ Error messages are descriptive

### Symbol Collision (`TestAssetSymbolCollision`)
Tests multi-chain asset handling:
- Search for "USDC" finds Ethereum + Polygon versions
- Each chain deployment has unique asset_id
- GetAsset by ID returns correct chain-specific asset

**Acceptance Criteria:**
- ✅ SearchAssets("USDC") returns ≥ 2 results
- ✅ Each USDC variant has unique asset_id
- ✅ Metadata distinguishes chains

### Deployment Validation (`TestAssetDeploymentValidation`)
Tests contract deployment rules:
- Invalid contract address format → validation error
- Decimals out of range (< 0 or > 18) → validation error
- Nonexistent asset_id → foreign key error

**Acceptance Criteria:**
- ✅ EVM address validation (0x + 40 hex chars)
- ✅ Decimals constrained to [0, 18]
- ✅ Asset existence checked before deployment

### Relationship Graph (`TestAssetRelationshipGraph`)
Tests asset relationship queries:
- ListAssetRelationships finds all ETH relationships
- Filter by relationship_type returns only WRAPS
- Cycle detection prevents circular dependencies

**Acceptance Criteria:**
- ✅ ETH has WRAPS (WETH) and STAKES (stETH) relationships
- ✅ RelationshipType filter works correctly
- ✅ Cyclic relationships rejected (if implemented)

### Quality Flag Blocking (`TestQualityFlagBlocking`)
Tests CRITICAL flag enforcement:
- RaiseQualityFlag with CRITICAL severity
- ListQualityFlags shows active CRITICAL flag
- Consumers can detect tradeable status

**Acceptance Criteria:**
- ✅ CRITICAL flags persist correctly
- ✅ Multiple flags per asset supported
- ✅ ResolvedAt distinguishes active vs resolved

### Asset Group Aggregation (`TestAssetGroupAggregation`)
Tests portfolio management (cqpm use case):
- CreateAssetGroup for "ETH Family"
- AddAssetToGroup for ETH, WETH, stETH
- GetAssetGroup returns all members with weights
- RemoveAssetFromGroup updates membership

**Acceptance Criteria:**
- ✅ Group created with description
- ✅ All 3 ETH variants added successfully
- ✅ GetAssetGroup includes member details
- ✅ Remove updates group atomically

### Cache Performance (`TestAssetCachePerformance`)
Tests performance requirements:
- First GetAsset primes cache (slower)
- Subsequent GetAsset hits cache (<10ms p50)
- 100 iterations measure average latency

**Acceptance Criteria:**
- ✅ Cache hit avg latency < 10ms (p50 requirement)
- ✅ Performance consistent across iterations

## Running Specific Tests

```bash
# Run only asset tests
go test ./test/integration/ -run TestAsset -v

# Run only performance tests
go test ./test/integration/ -run Performance -v

# Skip performance tests
go test ./test/integration/ -short -v

# Run with race detector
go test ./test/integration/ -race -v
```

## Troubleshooting

### Database Connection Errors
```
Error: failed to ping test database
```
**Solution**: Ensure Docker Compose services are running and healthy:
```bash
cd test && docker-compose ps
# Should show all services with status "Up (healthy)"
```

### Migration Errors
```
Error: relation "assets" does not exist
```
**Solution**: Run migrations against test database:
```bash
export DATABASE_URL="postgres://cqar_test:cqar_test_password@localhost:5433/cqar_test?sslmode=disable"
make migrate-up
```

### Cache Connection Errors
```
Error: failed to create cache client
```
**Solution**: Check Redis is running on port 6380:
```bash
redis-cli -p 6380 -a cqar_test_password PING
# Should return: PONG
```

### Event Bus Errors
```
Error: failed to create event bus
```
**Solution**: Check NATS is running:
```bash
curl http://localhost:8223/healthz
# Should return: ok
```

## Performance Benchmarks

Expected performance on development machine:

| Metric                | Target     | Notes                    |
| --------------------- | ---------- | ------------------------ |
| GetAsset (cache hit)  | < 10ms p50 | After initial prime      |
| GetAsset (cache miss) | < 50ms p99 | DB query + cache write   |
| CreateAsset           | < 100ms    | DB write + event publish |
| SearchAssets          | < 100ms    | Full-text search         |

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Integration Tests

on: [push, pull_request]

jobs:
  integration:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: cqar_test
          POSTGRES_PASSWORD: cqar_test_password
          POSTGRES_DB: cqar_test
        ports:
          - 5433:5432
      redis:
        image: redis:7
        ports:
          - 6380:6379
      nats:
        image: nats:2.10
        ports:
          - 4223:4222
    
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run migrations
        run: make migrate-up
        env:
          DATABASE_URL: postgres://cqar_test:cqar_test_password@localhost:5433/cqar_test?sslmode=disable
      
      - name: Run integration tests
        run: go test ./test/integration/... -v
```

## Future Enhancements

- [ ] Symbol integration tests (market specs, options)
- [ ] Venue integration tests (cqmd symbol resolution workflow)
- [ ] Performance regression tests with baseline tracking
- [ ] Load testing (concurrent operations, connection pooling)
- [ ] Chaos testing (DB failover, cache eviction)
- [ ] Contract testing with cqmd/cqpm consumers
