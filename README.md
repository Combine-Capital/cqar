# CQAR - Crypto Quant Asset Registry

Part of the **Crypto Quant Platform** - Professional-grade crypto trading infrastructure.

## Overview

CQAR is a gRPC microservice that provides canonical reference data for the Crypto Quant trading platform. It acts as the single source of truth for:

- **Assets**: Individual tokens and coins (BTC, ETH, USDT, WETH) with multi-chain deployments
- **Symbols**: Trading pairs and markets (BTC/USDT spot, ETH-PERP, options)
- **Chains**: Blockchain networks (Ethereum, Polygon, Arbitrum)
- **Venues**: Exchanges and protocols (Binance, Uniswap V3, dYdX)
- **Mappings**: Asset/symbol availability per venue, cross-chain relationships

### Key Features

- **Domain Separation**: Clear boundaries between Assets, Symbols, and Venues
- **Symbol Resolution**: Fast lookups (<10ms p50) for trading pair identification
- **Multi-Chain Support**: Track token deployments across multiple blockchains
- **Relationship Graph**: Map wrapped tokens, staking derivatives, and bridges
- **Quality Flags**: Track scams, exploits, and deprecated assets with severity levels
- **Event Publishing**: Lifecycle events via NATS JetStream for downstream services

## Architecture

CQAR is built on the CQ platform infrastructure:

- **CQC**: Protocol Buffer definitions for all data types and gRPC interfaces
- **CQI**: Infrastructure packages (logging, metrics, tracing, config, database, cache, event bus)
- **PostgreSQL**: Primary data store with JSONB for flexible metadata
- **Redis**: Cache layer for sub-10ms read performance
- **NATS JetStream**: Event bus for publishing domain events

## Repository Structure

```
cqar/
├── cmd/
│   └── server/         # Main application entry point
├── internal/
│   ├── config/         # Configuration (extends CQI config)
│   ├── manager/        # Business logic layer (to be implemented)
│   ├── repository/     # Data access layer (to be implemented)
│   └── server/         # gRPC server implementation (to be implemented)
├── migrations/         # Database schema migrations (to be implemented)
├── test/              # Integration tests (to be implemented)
├── docs/              # Documentation
│   ├── BRIEF.md       # Project requirements
│   ├── SPEC.md        # Technical specification
│   └── ROADMAP.md     # Implementation roadmap
├── config.yaml        # Base configuration
├── config.dev.yaml    # Development overrides
├── config.prod.yaml   # Production overrides
└── Makefile           # Build automation
```

## Getting Started

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 14+
- Redis 7+
- NATS JetStream 2.10+
- [golang-migrate](https://github.com/golang-migrate/migrate) for database migrations

### Installation

1. Clone the repository:
```bash
git clone https://github.com/Combine-Capital/cqar.git
cd cqar
```

2. Install dependencies:
```bash
make deps
```

3. Set up configuration:
```bash
# Copy and edit configuration
cp config.yaml config.local.yaml
# Edit config.local.yaml with your database, cache, and event bus settings
```

4. Run database migrations (once migrations are implemented):
```bash
make migrate-up
```

5. Build the service:
```bash
make build
```

### Running Locally

```bash
# Run from source
make run

# Or run the compiled binary
./bin/cqar --config config.yaml

# Show help
./bin/cqar --help

# Show version
./bin/cqar --version
```

### Configuration

CQAR uses CQI's configuration management with YAML files and environment variable overrides:

```yaml
service:
  name: "cqar"
  version: "0.1.0"
  env: "development"

server:
  http_port: 8080
  grpc_port: 9090

database:
  host: "localhost"
  port: 5432
  database: "cqar"
  user: "cqar"
  password: "your_password"

cache:
  host: "localhost"
  port: 6379

eventbus:
  backend: "jetstream"
  servers:
    - "nats://localhost:4222"
```

Environment variables use the `CQAR_` prefix:
- `CQAR_DATABASE_HOST`
- `CQAR_DATABASE_PASSWORD`
- `CQAR_CACHE_HOST`
- etc.

### Development Commands

```bash
# Format code
make fmt

# Run linters
make lint

# Run tests
make test

# Generate coverage report
make test-coverage

# Run integration tests
make test-integration

# Clean build artifacts
make clean

# See all available commands
make help
```

## API

CQAR implements the `AssetRegistry` gRPC service defined in [CQC](https://github.com/Combine-Capital/cqc). See the CQC repository for complete protobuf definitions.

### Example: Resolve Venue Symbol

```go
import (
    servicespb "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
)

// Get symbol for Binance's "BTCUSDT" market
resp, err := client.GetVenueSymbol(ctx, &servicespb.GetVenueSymbolRequest{
    VenueId:     "binance",
    VenueSymbol: "BTCUSDT",
})

// Response includes:
// - Canonical symbol ID
// - Base asset (BTC) and quote asset (USDT) IDs
// - Market specs (tick_size, lot_size, etc.)
// - Venue-specific fees
```

## Project Status

**Current Phase**: Commit 1 - Project Foundation & Configuration ✅

See [ROADMAP.md](docs/ROADMAP.md) for the complete implementation plan.

## Related Services

- [CQ Hub](https://github.com/Combine-Capital/cqhub) - Platform Documentation
- [CQC](https://github.com/Combine-Capital/cqc) - Platform Contracts (Protobuf Definitions)
- [CQI](https://github.com/Combine-Capital/cqi) - Platform Infrastructure (Shared Libraries)

## License

Proprietary - Combine Capital
