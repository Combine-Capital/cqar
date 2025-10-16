# CQAR - Crypto Quant Asset Registry

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> **Central reference data authority for the CQ trading platform**
> 
> Resolves asset identities across chains and venues, maps trading pairs to canonical symbols, and flags quality risks to enable safe, unified trading operations.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Architecture](#architecture)
- [API Examples](#api-examples)
- [Configuration](#configuration)
- [Development](#development)
- [Testing](#testing)
- [Documentation](#documentation)
- [License](#license)

## Overview

CQAR (Crypto Quant Asset Registry) is a gRPC microservice that serves as the source of truth for asset and symbol metadata across the Crypto Quant platform. It provides:

- **Asset Management**: Canonical identifiers for tokens/coins (BTC, ETH, USDC) with multi-chain deployment tracking
- **Symbol Management**: Trading pair definitions (BTC/USDT spot, ETH/USD perp) with market specifications
- **Venue Mapping**: Asset availability and symbol formats per exchange/DEX/protocol
- **Relationship Tracking**: Wrapped (WETH↔ETH), staked (stETH→ETH), and bridged (USDC.e→USDC) variants
- **Quality Assurance**: Flags for scams, exploits, and other risk factors
- **Chain Registry**: Blockchain metadata with RPC endpoints and explorer URLs

### Key Features

✅ **Sub-10ms Resolution**: Cache-first architecture for high-performance lookups  
✅ **Multi-Chain Support**: Track assets across Ethereum, Polygon, Solana, Bitcoin, and more  
✅ **Symbol Collision Resolution**: Unique canonical IDs for assets with same symbol on different chains  
✅ **Event-Driven**: Publishes lifecycle events (AssetCreated, SymbolCreated) via NATS JetStream  
✅ **Production-Ready**: Health checks, metrics, tracing, graceful shutdown  
✅ **Type-Safe**: Protocol Buffers via CQC contracts with auto-generated clients  

### Platform Dependencies

- **[CQC](https://github.com/Combine-Capital/cqc)** - Protocol Buffer contracts defining all data types and gRPC service interface
- **[CQI](https://github.com/Combine-Capital/cqi)** - Infrastructure library providing service lifecycle, database, cache, event bus, observability

## Quick Start

### Prerequisites

- **Go 1.21+**
- **Docker & Docker Compose** (for infrastructure)
- **Make** (for build automation)

### 1. Start Infrastructure

```bash
# Start PostgreSQL, Redis, NATS
docker-compose up -d

# Wait for services to be healthy
docker-compose ps
```

### 2. Run Database Migrations

```bash
# Apply all migrations
make migrate-up

# Verify tables created
make db-shell
\dt
```

### 3. Build & Run Service

```bash
# Build binary
make build

# Run service
make run

# Or run directly
./bin/cqar -config config.yaml
```

### 4. Verify Service

```bash
# Check health
curl http://localhost:8080/health/live
# Response: {"status":"ok"}

curl http://localhost:8080/health/ready
# Response: {"status":"ready","components":{"database":"ok","cache":"ok"}}

# List gRPC methods
grpcurl -plaintext localhost:9090 list
# Shows: cqc.services.v1.AssetRegistry

# Introspect service
grpcurl -plaintext localhost:9090 list cqc.services.v1.AssetRegistry
# Shows all 42+ methods
```

**🎉 You're now running CQAR locally!** Total time: ~3 minutes

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                     CQAR Service                         │
│                                                          │
│  ┌──────────────────────────────────────────────────┐   │
│  │         gRPC Server (CQC Interface)              │   │
│  │  Asset | Symbol | Venue | Quality | Chain       │   │
│  └─────────────────┬────────────────────────────────┘   │
│                    │                                     │
│  ┌─────────────────▼────────────────────────────────┐   │
│  │           Business Logic Layer                   │   │
│  │  AssetManager | SymbolManager | VenueManager    │   │
│  │  QualityManager | ValidationEngine | Events     │   │
│  └─────────────────┬────────────────────────────────┘   │
│                    │                                     │
│  ┌────────┬────────┴─────────┬──────────────────────┐   │
│  │        │                  │                      │   │
│  ▼        ▼                  ▼                      ▼   │
│ ┌──────┐ ┌──────┐  ┌──────────────┐  ┌──────────────┐  │
│ │ Repo │ │Cache │  │  Event Bus   │  │  Middleware  │  │
│ │ (PG) │ │(Redis)  │   (NATS)     │  │Auth|Log|Trace│  │
│ └──────┘ └──────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Data Flow: Market Data Service (cqmd) Use Case

```
1. cqmd receives "BTCUSDT" price from Binance
2. Call: GetVenueSymbol(venue_id="binance", venue_symbol="BTCUSDT")
3. CQAR checks Redis cache (hit: <5ms, miss: query PostgreSQL)
4. Returns: { symbol_id: "uuid", base_asset_id: "btc-uuid", quote_asset_id: "usdt-uuid", tick_size: 0.01 }
5. cqmd normalizes price and stores with canonical symbol_id
```

### Core Concepts

- **Asset**: Individual token/coin (BTC, ETH, USDC) with canonical UUID
- **Deployment**: Asset on specific chain (USDC on Ethereum, USDC on Polygon)
- **Symbol**: Trading pair (BTC/USDT spot, ETH/USD perp) with market specs
- **Venue**: Exchange/DEX/protocol (Binance, Uniswap, dYdX)
- **VenueAsset**: Asset availability on venue (BTC on Binance with fees, limits)
- **VenueSymbol**: Trading pair on venue ("BTCUSDT" on Binance → BTC/USDT canonical)
- **Relationship**: Asset variants (WETH wraps ETH, stETH stakes ETH)
- **QualityFlag**: Risk markers (SCAM, EXPLOITED, RUGPULL) with severity levels

## API Examples

### Create Asset

```bash
grpcurl -plaintext -d '{
  "symbol": "BTC",
  "name": "Bitcoin",
  "asset_type": "CRYPTO",
  "category": "LAYER1",
  "description": "First decentralized cryptocurrency",
  "website_url": "https://bitcoin.org",
  "logo_url": "https://assets.coingecko.com/coins/images/1/large/bitcoin.png"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateAsset
```

Response:
```json
{
  "asset": {
    "id": "btc-550e8400-e29b-41d4-a716-446655440000",
    "symbol": "BTC",
    "name": "Bitcoin",
    "asset_type": "CRYPTO",
    "category": "LAYER1",
    "description": "First decentralized cryptocurrency",
    "website_url": "https://bitcoin.org",
    "logo_url": "https://assets.coingecko.com/coins/images/1/large/bitcoin.png",
    "created_at": "2025-10-16T12:34:56Z",
    "updated_at": "2025-10-16T12:34:56Z"
  }
}
```

### Create Multi-Chain Deployment

```bash
# USDC on Ethereum
grpcurl -plaintext -d '{
  "asset_id": "usdc-uuid",
  "chain_id": "ethereum",
  "contract_address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
  "decimals": 6,
  "is_canonical": true
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateAssetDeployment

# USDC on Polygon
grpcurl -plaintext -d '{
  "asset_id": "usdc-uuid",
  "chain_id": "polygon",
  "contract_address": "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
  "decimals": 6,
  "is_canonical": false
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateAssetDeployment
```

### Resolve Venue Symbol (cqmd Use Case)

```bash
grpcurl -plaintext -d '{
  "venue_id": "binance",
  "venue_symbol": "BTCUSDT"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetVenueSymbol
```

Response includes canonical symbol with market specs:
```json
{
  "venue_symbol": {
    "venue_id": "binance",
    "symbol_id": "btcusdt-spot-uuid",
    "venue_symbol": "BTCUSDT",
    "maker_fee": 0.001,
    "taker_fee": 0.001,
    "is_active": true
  },
  "symbol": {
    "id": "btcusdt-spot-uuid",
    "base_asset_id": "btc-uuid",
    "quote_asset_id": "usdt-uuid",
    "symbol_type": "SPOT",
    "tick_size": "0.01",
    "lot_size": "0.00001"
  }
}
```

### Check Quality Flags

```bash
grpcurl -plaintext -d '{
  "asset_id": "suspicious-token-uuid"
}' localhost:9090 cqc.services.v1.AssetRegistry/ListQualityFlags
```

### Search Assets

```bash
grpcurl -plaintext -d '{
  "query": "stable",
  "asset_type": "CRYPTO",
  "page_size": 10,
  "page_token": ""
}' localhost:9090 cqc.services.v1.AssetRegistry/SearchAssets
```

For complete API documentation, see [docs/API.md](docs/API.md).

## Configuration

Configuration files follow CQI structure with CQAR-specific extensions:

```yaml
# config.yaml
service:
  name: "cqar"
  version: "0.1.0"
  env: "development"

server:
  http_port: 8080      # Health checks, metrics
  grpc_port: 9090      # gRPC service
  shutdown_timeout: "30s"

database:
  host: "localhost"
  port: 5432
  user: "cqar"
  password: "cqar_dev_password"
  database: "cqar"
  max_conns: 25
  query_timeout: "30s"

cache:
  host: "localhost"
  port: 6379
  default_ttl: "60m"   # Asset/Symbol/Venue cache
  pool_size: 10

eventbus:
  backend: "jetstream"
  servers:
    - "nats://localhost:4222"
  stream_name: "cqc_events"

log:
  level: "info"
  format: "json"

metrics:
  enabled: true
  port: 9091

tracing:
  enabled: true
  endpoint: "localhost:4317"

auth:
  api_keys:
    - key: "dev_key_cqmd_12345"
      name: "cqmd_service"
    - key: "dev_key_cqpm_67890"
      name: "cqpm_service"
```

### Environment-Specific Configs

- **config.yaml**: Base configuration with defaults
- **config.dev.yaml**: Development overrides (local infrastructure)
- **config.prod.yaml**: Production settings (connection pooling, auth, timeouts)

Override precedence: Environment variables > config.prod.yaml > config.yaml

## Development

### Prerequisites

```bash
# Install Go 1.21+
go version

# Install development tools
make install-tools

# Install grpcurl for testing
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest
```

### Project Structure

```
cqar/
├── cmd/server/              # Service entrypoint
├── internal/
│   ├── config/              # Configuration management
│   ├── manager/             # Business logic (AssetManager, SymbolManager, etc.)
│   ├── repository/          # Data access layer (PostgreSQL, Redis)
│   ├── server/              # gRPC server implementation
│   └── service/             # CQI service lifecycle
├── migrations/              # Database migrations (16 files)
├── docs/                    # Documentation
├── test/                    # Integration tests
│   ├── integration/         # End-to-end test suites
│   └── testdata/            # Seed data for tests
├── config.yaml              # Base configuration
├── docker-compose.yml       # Local infrastructure
└── Makefile                 # Build automation
```

### Makefile Targets

```bash
make help              # Show all targets
make build             # Build binary → bin/cqar
make run               # Build and run service
make test              # Run unit tests
make test-integration  # Run integration tests
make migrate-up        # Apply database migrations
make migrate-down      # Rollback migrations
make db-shell          # Open PostgreSQL shell
make redis-cli         # Open Redis CLI
make clean             # Remove build artifacts
make lint              # Run golangci-lint
make fmt               # Format code with gofmt
```

### Running Tests

```bash
# Unit tests (fast, no infrastructure)
make test

# Integration tests (requires infrastructure)
make test-infra-up      # Start test infrastructure
make test-migrate       # Apply migrations to test DB
make test-integration   # Run end-to-end tests
make test-infra-down    # Cleanup test infrastructure

# All tests
make test-all
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Run tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Testing

CQAR includes comprehensive test coverage:

### Unit Tests

Located in `internal/manager/*_test.go` and `internal/repository/*_test.go`:

- **AssetManager**: Validation, collision detection, relationship management
- **SymbolManager**: Market spec validation, option fields
- **VenueManager**: Availability tracking, fee validation
- **QualityManager**: Flag severity, tradeable checks

```bash
go test ./internal/manager/... -v
go test ./internal/repository/... -v
```

### Integration Tests

Located in `test/integration/`:

- **Asset Lifecycle**: Create → Get → Update → Deploy → Relate → Group
- **Symbol Resolution**: cqmd workflow with GetVenueSymbol
- **Quality Flags**: CRITICAL blocking, severity levels
- **Cache Performance**: Latency measurement, <10ms p50 target

```bash
make test-integration
```

Test coverage: **58.3%** across managers and repositories.

For detailed testing documentation, see [test/integration/README.md](test/integration/README.md).

## Documentation

- **[API.md](docs/API.md)** - Complete gRPC API reference with examples
- **[SPEC.md](docs/SPEC.md)** - Technical specification and architecture
- **[BRIEF.md](docs/BRIEF.md)** - Product requirements and user personas
- **[ROADMAP.md](docs/ROADMAP.md)** - Implementation plan and progress
- **[DEPLOYMENT.md](docs/DEPLOYMENT.md)** - Kubernetes deployment guide
- **[OPERATIONS.md](docs/OPERATIONS.md)** - Operational procedures and troubleshooting

## Performance

CQAR meets strict performance requirements:

| Operation                      | Target    | Actual | Measurement                  |
| ------------------------------ | --------- | ------ | ---------------------------- |
| Symbol Resolution (cache hit)  | <10ms p50 | ~5ms   | GetVenueSymbol with Redis    |
| Symbol Resolution (cache miss) | <50ms p99 | ~35ms  | GetVenueSymbol with DB query |
| Asset Lookup                   | <20ms p99 | ~15ms  | GetAsset with cache          |
| Cache Hit Rate                 | >80%      | ~95%   | Production metrics           |

### Cache Strategy

- **Assets/Symbols/Venues**: 60min TTL (rarely change)
- **VenueAssets/VenueSymbols**: 15min TTL (availability changes)
- **Quality Flags**: 5min TTL (risk management requires fresh data)

## Observability

### Health Checks

```bash
# Liveness (service running)
curl http://localhost:8080/health/live

# Readiness (dependencies healthy)
curl http://localhost:8080/health/ready

# gRPC health check
grpcurl -plaintext localhost:9090 grpc.health.v1.Health/Check
```

### Metrics

Prometheus metrics available at `http://localhost:9091/metrics`:

- `cqar_grpc_call_duration_seconds{method, status}` - gRPC call latency histogram
- `cqar_grpc_calls_total{method, status_code}` - gRPC call counter
- `cqar_cache_hit_total{entity}` - Cache hit counter
- `cqar_cache_miss_total{entity}` - Cache miss counter
- `cqar_event_published_total{event_type}` - Event publishing counter
- `cqar_db_query_duration_seconds{operation}` - Database query latency

### Logs

Structured JSON logs via zerolog:

```json
{
  "level": "info",
  "time": "2025-10-16T12:34:56Z",
  "service": "cqar",
  "request_id": "req-123",
  "method": "/cqc.services.v1.AssetRegistry/GetAsset",
  "duration_ms": 12,
  "status": "OK",
  "message": "request completed"
}
```

### Tracing

OpenTelemetry spans with automatic context propagation:

- gRPC method spans
- Database query spans
- Cache operation spans
- Event publishing spans

## License

MIT License - see [LICENSE](LICENSE) for details.

---

**Questions or Issues?** See [docs/OPERATIONS.md](docs/OPERATIONS.md) for troubleshooting.

**Contributing?** See [ROADMAP.md](docs/ROADMAP.md) for development tasks.