# Project Brief: CQAR - Crypto Quant Asset Registry

## Vision
Central reference data authority serving as the "phonebook" for the CQ trading platform. Resolves asset identities across chains and venues, maps trading pairs to canonical symbols, and flags quality risks to enable safe, unified trading operations.

## Platform Dependencies
This service depends on two foundational CQ platform packages:

- **CQC (Crypto Quant Contracts)**: `github.com/Combine-Capital/cqc` - Protocol Buffer definitions providing all shared data types (Asset, Symbol, Venue, VenueAsset, VenueSymbol, AssetDeployment, AssetRelationship, Chain, etc.) and the AssetRegistry gRPC service interface. CQAR implements the CQC AssetRegistry service interface and uses CQC message types as its data models.

- **CQI (Crypto Quant Infrastructure)**: `github.com/Combine-Capital/cqi` - Shared infrastructure library providing service lifecycle, database, cache, event bus, logging, metrics, tracing, configuration, and error handling. CQAR uses CQI for all infrastructure concerns including service bootstrap, database transactions, event publishing, and graceful shutdown.

## User Personas
### Primary: Market Data Service (cqmd)
- **Role:** Resolves venue-specific symbols for price feed ingestion
- **Needs:** Venue symbol → canonical symbol resolution; market specifications (tick size, lot size)
- **Pain Points:** Cannot normalize venue-specific symbols ("BTCUSDT" vs "BTC-USDT"); missing specs break price calculations
- **Success:** <10ms p50 resolution of venue symbol to canonical symbol with complete market specs

### Secondary: Portfolio Manager Service (cqpm)
- **Role:** Aggregates positions across chains and venues
- **Needs:** Canonical asset IDs; asset relationships for variant grouping (WETH/stETH → ETH); quality flags
- **Pain Points:** Symbol collisions across chains ("USDC" on 5+ networks); cannot aggregate variants; risky asset exposure
- **Success:** <50ms p99 asset resolution; automatic variant grouping; zero trades on critical-flagged assets

### Tertiary: Venue Exchange Gateway (cqvx)
- **Role:** Executes orders on trading venues
- **Needs:** Asset availability per venue; canonical → venue-specific symbol mapping; venue API configuration
- **Pain Points:** Unknown asset availability; incorrect venue symbol formats; missing venue metadata
- **Success:** <20ms mapping of canonical symbol to venue-specific format with availability confirmation

## Core Requirements

### Asset Management (Individual Tokens/Coins)
- [MVP] CRUD operations: CreateAsset, GetAsset, UpdateAsset, DeleteAsset, ListAssets, SearchAssets
- [MVP] Assign canonical UUID per token (BTC, ETH, USDT, WETH, stETH)
- [MVP] Store metadata: symbol, name, type, category, description, logo_url, website_url
- [MVP] Search by symbol/name with pagination and filtering (type, category, flags)
- [MVP] Resolve symbol collisions across chains (USDC on Ethereum ≠ USDC on Polygon)

### Asset Deployment Tracking (Multi-Chain)
- [MVP] Operations: CreateAssetDeployment, GetAssetDeployment, ListAssetDeployments
- [MVP] Track deployments: asset_id, chain_id, contract_address, decimals, is_canonical flag
- [MVP] List all deployments per asset (ETH → Ethereum native, Polygon bridged, Arbitrum bridged)
- [MVP] List all assets per chain (Ethereum → ETH, WETH, USDC, DAI)
- [MVP] Identify canonical deployment per chain

### Asset Relationships (Wrapped/Staked/Bridged Variants)
- [MVP] Operations: CreateAssetRelationship, ListAssetRelationships
- [MVP] Relationship types: WRAPS (WETH ↔ ETH), STAKES (stETH → ETH), BRIDGES (USDC.e → USDC), SYNTHETIC, LP_PAIR
- [MVP] Store conversion rates for fixed relationships (1:1 wrapping)
- [MVP] Track facilitating protocol (Lido for stETH, Wormhole for bridges)
- [MVP] Query all variants (ETH → ETH, WETH, stETH, cbETH, rETH)
- [MVP] Filter by relationship type (staked ETH variants → stETH, cbETH, rETH)

### Asset Groups (Aggregation)
- [MVP] Operations: CreateAssetGroup, GetAssetGroup, AddAssetToGroup, RemoveAssetFromGroup
- [MVP] Group assets for aggregation ("all_eth_variants" → [ETH, WETH, stETH, cbETH, rETH])
- [MVP] Enable portfolio exposure rollups across asset variants

### Asset Quality Flags (Risk Management)
- [MVP] Operations: RaiseQualityFlag, ResolveQualityFlag, ListQualityFlags
- [MVP] Flag types: SCAM, RUGPULL, EXPLOITED, DEPRECATED, PAUSED, UNVERIFIED, LOW_LIQUIDITY, HONEYPOT, TAX_TOKEN
- [MVP] Severity levels: INFO, LOW, MEDIUM, HIGH, CRITICAL
- [MVP] Track flag source (certik, tokensniffer, manual, coingecko)
- [MVP] Query by flag type and severity
- [MVP] Block trades on CRITICAL-flagged assets

### Asset Identifiers (External Data Providers)
- [MVP] Operations: CreateAssetIdentifier, GetAssetIdentifier, ListAssetIdentifiers
- [MVP] Map canonical assets to external IDs (CoinGecko, CoinMarketCap, DeFiLlama)
- [MVP] Support multiple identifiers per provider with primary flag
- [MVP] Enable metadata sync from external sources

### Symbol Management (Trading Pairs/Markets)
- [MVP] CRUD operations: CreateSymbol, GetSymbol, UpdateSymbol, DeleteSymbol, ListSymbols, SearchSymbols
- [MVP] Assign canonical UUID per trading pair/market
- [MVP] Symbol types: SPOT, PERPETUAL, FUTURE, OPTION, MARGIN
- [MVP] Store market specs: base/quote/settlement asset IDs, tick_size, lot_size, min/max order sizes
- [MVP] Option-specific fields: strike_price, expiry, option_type (CALL/PUT)
- [MVP] Search by base/quote assets and type with pagination

### Symbol Identifiers (External Data Providers)
- [MVP] Operations: CreateSymbolIdentifier, GetSymbolIdentifier, ListSymbolIdentifiers
- [MVP] Map canonical symbols to external provider IDs for market data aggregation

### Chain Registry (Blockchain Networks)
- [MVP] Operations: CreateChain, GetChain, ListChains
- [MVP] Store chain metadata: chain_id, name, type, native_asset_id, rpc_urls, explorer_url
- [MVP] Enable asset and deployment queries by chain

### Venue Registry (Exchanges/Protocols)
- [MVP] Operations: CreateVenue, GetVenue, ListVenues
- [MVP] Store venue data: venue_id, name, type (CEX/DEX/DEX_AGGREGATOR/BRIDGE/LENDING)
- [MVP] Venue metadata: chain_id (DEX), protocol_address, website_url, api_endpoint, is_active flag

### Venue Asset Availability (Which Assets on Which Venues)
- [MVP] Operations: CreateVenueAsset, GetVenueAsset, ListVenueAssets
- [MVP] Map asset availability per venue (Binance → BTC, ETH, USDT; Uniswap V3 → WETH, USDC, DAI)
- [MVP] Store venue-specific symbol (Binance "BTC" for Bitcoin)
- [MVP] Availability flags: deposit_enabled, withdraw_enabled, trading_enabled
- [MVP] Fees: withdraw_fee, min_withdraw amount
- [MVP] Listing lifecycle: listed_at, delisted_at timestamps
- [MVP] Query assets by venue and venues by asset

### Venue Symbol Availability (Which Markets on Which Venues)
- [MVP] Operations: CreateVenueSymbol, GetVenueSymbol, ListVenueSymbols
- [MVP] Map canonical symbol to venue format (BTC/USDT spot → "BTCUSDT" Binance, "BTC-USDT" Coinbase)
- [MVP] Store venue-specific fees: maker_fee, taker_fee
- [MVP] Track market status: is_active, listed_at, delisted_at
- [MVP] Query canonical → venue symbol and venues by canonical symbol
- [MVP] Support divergent venue formats for same canonical symbol

### Event Publishing (Lifecycle Events)
- [MVP] Publish domain events: AssetCreated, AssetDeploymentCreated, RelationshipEstablished, QualityFlagRaised
- [MVP] Symbol events: SymbolCreated
- [MVP] Availability events: VenueAssetListed, VenueSymbolListed
- [MVP] Infrastructure events: ChainRegistered
- [MVP] Use CQC protobuf types with CQI event bus auto-serialization

### Service Infrastructure (via CQI)
- [MVP] Implement cqi.Service interface (Start, Stop, Name, Health)
- [MVP] Bootstrap via CQI: configuration, logging, metrics, tracing initialization
- [MVP] Graceful shutdown: SIGTERM/SIGINT handling with cleanup hooks
- [MVP] Database: CQI connection pooling and transaction helpers
- [MVP] Cache: CQI Redis with automatic CQC protobuf serialization
- [MVP] Health checks: /health/live, /health/ready with component status
- [MVP] Logging: structured logs via zerolog with trace context
- [MVP] Metrics: Prometheus with standard gRPC metrics
- [MVP] Tracing: OpenTelemetry with automatic span creation
- [MVP] Configuration: viper-based from environment + YAML with validation

### Post-MVP
- External metadata sync (CoinGecko, DeFiLlama, Token Lists)
- Auto-detect suspicious tokens via contract analysis
- Version deployment history for contract upgrades
- Asset recommendation engine
- Multi-sig approval for critical flag overrides
- Service discovery registration (Redis backend)
- Relationship graph visualization

## Success Metrics
1. **Coverage**: 100% of traded assets registered within 7 days of first trade
2. **Performance**: <10ms p50, <50ms p99 symbol resolution; <20ms p99 asset lookup
3. **Aggregation**: 100% relationship coverage (zero missed variants in portfolio rollups)
4. **Quality Protection**: Zero trades on CRITICAL-flagged assets
5. **Adoption**: 8 services (cqmd, cqpm, cqex, cqvx, cqdefi, cqrisk, cqrep, cqstrat) depend on CQAR; zero local mappings
6. **Availability**: 99.9% uptime with automatic failover
