# Project Brief: CQAR - Crypto Quant Asset Registry

## Vision
Canonical asset registry resolving identity across 10+ chains and 50+ venues—eliminating symbol collisions, mapping asset relationships (wrapped/bridged/staked variants), and flagging quality risks.

## Platform Dependencies
This service depends on two foundational CQ platform packages:

- **CQC (Crypto Quant Contracts)**: `github.com/Combine-Capital/cqc` - Protocol Buffer definitions providing all shared data types (Asset, AssetDeployment, AssetRelationship, Chain, Venue, etc.) and gRPC service interfaces (AssetRegistry). CQAR implements the CQC AssetRegistry service interface and uses CQC message types as its data models.

- **CQI (Crypto Quant Infrastructure)**: `github.com/Combine-Capital/cqi` - Shared infrastructure library providing event bus, database, cache, logging, metrics, tracing, configuration, and error handling. CQAR uses CQI for all infrastructure concerns including publishing CQC events and managing database connections.

## User Personas
### Primary: Portfolio Manager Service (cqpm)
- **Needs:** Canonical IDs to aggregate positions across chains/venues; asset relationships to group variants (WETH/stETH/cbETH → ETH); quality flags to block risky trades
- **Pain:** Symbol collisions ("USDC" exists on 5+ chains); cannot aggregate 15+ ETH variants; risk exposure to flagged tokens
- **Success:** Query "ETH" → auto-aggregate all variants; zero trades on critical-severity flagged assets

### Secondary: Market Data Service (cqmd)
- **Needs:** Venue ticker → canonical ID mapping; deployment addresses for chain-specific price feeds; asset availability per venue
- **Pain:** Inconsistent formats (Binance "BTCUSDT" vs Coinbase "BTC-USD"); missing contract addresses; ambiguous canonical chains
- **Success:** Map any venue price tick to canonical asset <10ms; route queries to correct deployment

## Core Requirements

### Asset & Token Identification
- [MVP] Assign and resolve canonical asset IDs (CQC Asset, AssetIdentifier, AssetDeployment)
- [MVP] Resolve symbols to addresses with collision detection (multiple "USDC" tokens across chains)
- [MVP] Store metadata: symbol, name, AssetType, category, logo, description, decimals, DataSource (with conflict resolution)
- [MVP] Track multi-chain deployments: chain_id, address, decimals, canonical flag
- [MVP] Define relationships: wraps, bridges, stakes, synthetic, LP pairs, migrations (CQC RelationshipType)
- [MVP] Organize into groups for aggregation (AssetGroup: "all ETH variants" → WETH/stETH/cbETH/rETH)
- [MVP] List all assets on a chain
- [MVP] List all tradeable assets on a venue
- [MVP] Get deployment addresses across all chains
- [MVP] Get all wrapped/bridged variants of underlying asset
- [MVP] Get all LST variants of base asset
- [MVP] Get all LP tokens containing specific asset
- [MVP] Map CEX tickers to on-chain addresses
- [MVP] Identify rebasing vs non-rebasing variants
- [MVP] Identify synthetic/derivative representations

### Asset Quality & Filtering
- [MVP] Flag quality issues: FlagType (scam, exploited, deprecated, paused, low_liquidity, oracle_failed, unverified, tax_token, transfer_restricted, admin_keys, circuit_breaker); FlagSeverity (info → critical)
- [MVP] Query tokens by flag type: deprecated, scam/rug-pull, exploited, paused, admin keys, circuit breaker failures, low liquidity, oracle failures, unverified contracts, tax/fee mechanisms, transfer restrictions
- [MVP] Query tokens flagged by security auditors
- [MVP] Filter assets by severity threshold

### Infrastructure & Operations
- [MVP] Register chains: chain_id, name, type, native_asset, RPC, explorer
- [MVP] Map venue symbols: venue_id, symbol, asset_id, VenueType (CEX/DEX/aggregator/bridge/lending)
- [MVP] Implement CQC AssetRegistry gRPC interface: 30+ operations (Create, Get, Update, List, Search for Asset, Deployment, Relationship, Group, Flag, Chain, Venue)
- [MVP] Search by name, symbol, address, venue ticker with pagination and filters (type, category, chain, venue, flags)
- [MVP] Publish events via CQI: AssetCreated, AssetDeploymentCreated, RelationshipEstablished, QualityFlagSet, ChainRegistered, VenueSymbolMapped
- [Post-MVP] Sync metadata from external sources (CoinGecko, DeFiLlama, Token Lists)
- [Post-MVP] Auto-detect suspicious tokens via contract analysis (honeypot detection, unusual permissions)
- [Post-MVP] Version deployment history for contract upgrades
- [Post-MVP] Recommend similar/alternative assets
- [Post-MVP] Multi-sig approval for critical flag overrides
- [Post-MVP] Generate relationship graphs for visualization

## Success Metrics
1. **Coverage**: 100% of traded assets registered with canonical IDs within 7 days of first platform trade
2. **Aggregation**: Zero missed relationships in portfolio queries (all ETH variants correctly grouped)
3. **Quality Protection**: Zero trades on critical-severity flagged tokens
4. **Performance**: Symbol → canonical ID resolution <50ms p99, <10ms p50
5. **Adoption**: 8 services (cqmd, cqpm, cqex, cqvx, cqdefi, cqrisk, cqrep, cqstrat) use cqar exclusively; zero local asset mappings
