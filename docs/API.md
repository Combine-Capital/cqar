# CQAR API Reference

Complete gRPC API documentation for the Crypto Quant Asset Registry service.

## Table of Contents

- [Service Information](#service-information)
- [Asset Methods](#asset-methods)
- [Symbol Methods](#symbol-methods)
- [Chain Methods](#chain-methods)
- [Venue Methods](#venue-methods)
- [Venue Asset Methods](#venue-asset-methods)
- [Venue Symbol Methods](#venue-symbol-methods)
- [Deployment Methods](#deployment-methods)
- [Relationship Methods](#relationship-methods)
- [Quality Flag Methods](#quality-flag-methods)
- [Asset Group Methods](#asset-group-methods)
- [Identifier Methods](#identifier-methods)
- [Error Codes](#error-codes)

## Service Information

**Service Name**: `cqc.services.v1.AssetRegistry`  
**Protocol**: gRPC over HTTP/2  
**Default Port**: 9090  
**Authentication**: API Key via `authorization` header

### Connection Example

```bash
# List all methods
grpcurl -plaintext localhost:9090 list cqc.services.v1.AssetRegistry

# Authenticated request
grpcurl -plaintext \
  -H "authorization: Bearer dev_key_cqmd_12345" \
  -d '{"id": "btc-uuid"}' \
  localhost:9090 cqc.services.v1.AssetRegistry/GetAsset
```

---

## Asset Methods

### CreateAsset

Create a new asset with canonical UUID.

**Request**:
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

**Response**:
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

**Validation**:
- `symbol` (required): 1-20 characters
- `name` (required): 1-200 characters
- `asset_type` (required): CRYPTO, FIAT, COMMODITY, STOCK, etc.

**Error Codes**:
- `INVALID_ARGUMENT`: Missing required fields or invalid format
- `ALREADY_EXISTS`: Symbol collision (use different chain or asset_id)
- `INTERNAL`: Database error

---

### GetAsset

Retrieve asset by canonical ID.

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "btc-550e8400-e29b-41d4-a716-446655440000"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetAsset
```

**Response**:
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

**Performance**: <10ms p50 (cache hit), <20ms p99 (cache miss)

**Error Codes**:
- `INVALID_ARGUMENT`: Invalid UUID format
- `NOT_FOUND`: Asset does not exist
- `INTERNAL`: Cache or database error

---

### UpdateAsset

Update existing asset metadata.

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "btc-550e8400-e29b-41d4-a716-446655440000",
  "name": "Bitcoin (Updated)",
  "description": "The original cryptocurrency"
}' localhost:9090 cqc.services.v1.AssetRegistry/UpdateAsset
```

**Response**:
```json
{
  "asset": {
    "id": "btc-550e8400-e29b-41d4-a716-446655440000",
    "symbol": "BTC",
    "name": "Bitcoin (Updated)",
    "asset_type": "CRYPTO",
    "category": "LAYER1",
    "description": "The original cryptocurrency",
    "website_url": "https://bitcoin.org",
    "logo_url": "https://assets.coingecko.com/coins/images/1/large/bitcoin.png",
    "created_at": "2025-10-16T12:34:56Z",
    "updated_at": "2025-10-16T13:45:12Z"
  }
}
```

**Notes**:
- Only provided fields are updated
- `id` and `symbol` cannot be changed
- Cache is automatically invalidated

---

### DeleteAsset

Soft delete an asset (marks as inactive).

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "spam-token-uuid"
}' localhost:9090 cqc.services.v1.AssetRegistry/DeleteAsset
```

**Response**:
```json
{}
```

**Notes**:
- Soft delete preserves historical data
- Foreign key relationships remain intact
- Asset no longer appears in ListAssets

---

### ListAssets

List assets with pagination and filtering.

**Request**:
```bash
grpcurl -plaintext -d '{
  "asset_type": "CRYPTO",
  "category": "LAYER1",
  "page_size": 20,
  "page_token": ""
}' localhost:9090 cqc.services.v1.AssetRegistry/ListAssets
```

**Response**:
```json
{
  "assets": [
    {
      "id": "btc-uuid",
      "symbol": "BTC",
      "name": "Bitcoin",
      "asset_type": "CRYPTO",
      "category": "LAYER1"
    },
    {
      "id": "eth-uuid",
      "symbol": "ETH",
      "name": "Ethereum",
      "asset_type": "CRYPTO",
      "category": "LAYER1"
    }
  ],
  "next_page_token": "eyJvZmZzZXQiOjIwfQ=="
}
```

**Filters**:
- `asset_type`: CRYPTO, FIAT, COMMODITY
- `category`: LAYER1, LAYER2, STABLECOIN, DEFI
- `page_size`: 1-100 (default 20)

---

### SearchAssets

Full-text search across asset symbol and name.

**Request**:
```bash
grpcurl -plaintext -d '{
  "query": "stable",
  "asset_type": "CRYPTO",
  "page_size": 10
}' localhost:9090 cqc.services.v1.AssetRegistry/SearchAssets
```

**Response**:
```json
{
  "assets": [
    {
      "id": "usdt-uuid",
      "symbol": "USDT",
      "name": "Tether USD",
      "asset_type": "CRYPTO",
      "category": "STABLECOIN"
    },
    {
      "id": "usdc-uuid",
      "symbol": "USDC",
      "name": "USD Coin",
      "asset_type": "CRYPTO",
      "category": "STABLECOIN"
    },
    {
      "id": "dai-uuid",
      "symbol": "DAI",
      "name": "Dai Stablecoin",
      "asset_type": "CRYPTO",
      "category": "STABLECOIN"
    }
  ],
  "next_page_token": ""
}
```

**Search Behavior**:
- Case-insensitive
- Partial matches supported
- Searches both `symbol` and `name` fields
- Results ranked by relevance

---

## Symbol Methods

### CreateSymbol

Create a trading pair with market specifications.

**Request (Spot)**:
```bash
grpcurl -plaintext -d '{
  "base_asset_id": "btc-uuid",
  "quote_asset_id": "usdt-uuid",
  "symbol_type": "SPOT",
  "tick_size": "0.01",
  "lot_size": "0.00001",
  "min_order_size": "0.0001",
  "max_order_size": "1000"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateSymbol
```

**Request (Option)**:
```bash
grpcurl -plaintext -d '{
  "base_asset_id": "eth-uuid",
  "quote_asset_id": "usd-uuid",
  "symbol_type": "OPTION",
  "strike_price": "3000",
  "expiry": "2025-12-31T23:59:59Z",
  "option_type": "CALL",
  "tick_size": "0.01",
  "lot_size": "0.1"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateSymbol
```

**Response**:
```json
{
  "symbol": {
    "id": "btcusdt-spot-uuid",
    "base_asset_id": "btc-uuid",
    "quote_asset_id": "usdt-uuid",
    "symbol_type": "SPOT",
    "tick_size": "0.01",
    "lot_size": "0.00001",
    "min_order_size": "0.0001",
    "max_order_size": "1000",
    "created_at": "2025-10-16T12:34:56Z",
    "updated_at": "2025-10-16T12:34:56Z"
  }
}
```

**Validation**:
- `base_asset_id` and `quote_asset_id` must exist
- `tick_size`, `lot_size` > 0
- `min_order_size` < `max_order_size`
- Option symbols require: `strike_price`, `expiry`, `option_type`

---

### GetSymbol

Retrieve symbol by canonical ID.

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "btcusdt-spot-uuid"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetSymbol
```

**Response**:
```json
{
  "symbol": {
    "id": "btcusdt-spot-uuid",
    "base_asset_id": "btc-uuid",
    "quote_asset_id": "usdt-uuid",
    "symbol_type": "SPOT",
    "tick_size": "0.01",
    "lot_size": "0.00001",
    "min_order_size": "0.0001",
    "max_order_size": "1000",
    "created_at": "2025-10-16T12:34:56Z"
  }
}
```

**Performance**: <10ms p50 (cache hit)

---

### ListSymbols

List symbols with filtering.

**Request**:
```bash
grpcurl -plaintext -d '{
  "base_asset_id": "btc-uuid",
  "symbol_type": "SPOT",
  "page_size": 20
}' localhost:9090 cqc.services.v1.AssetRegistry/ListSymbols
```

**Response**:
```json
{
  "symbols": [
    {
      "id": "btcusdt-spot-uuid",
      "base_asset_id": "btc-uuid",
      "quote_asset_id": "usdt-uuid",
      "symbol_type": "SPOT"
    },
    {
      "id": "btcusdc-spot-uuid",
      "base_asset_id": "btc-uuid",
      "quote_asset_id": "usdc-uuid",
      "symbol_type": "SPOT"
    }
  ],
  "next_page_token": ""
}
```

**Filters**:
- `base_asset_id`: Filter by base asset
- `quote_asset_id`: Filter by quote asset
- `symbol_type`: SPOT, PERPETUAL, FUTURE, OPTION

---

## Chain Methods

### CreateChain

Register a blockchain network.

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "ethereum",
  "name": "Ethereum Mainnet",
  "chain_type": "EVM",
  "native_asset_id": "eth-uuid",
  "rpc_urls": [
    "https://eth.llamarpc.com",
    "https://rpc.ankr.com/eth"
  ],
  "explorer_url": "https://etherscan.io"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateChain
```

**Response**:
```json
{
  "chain": {
    "id": "ethereum",
    "name": "Ethereum Mainnet",
    "chain_type": "EVM",
    "native_asset_id": "eth-uuid",
    "rpc_urls": [
      "https://eth.llamarpc.com",
      "https://rpc.ankr.com/eth"
    ],
    "explorer_url": "https://etherscan.io",
    "created_at": "2025-10-16T12:34:56Z"
  }
}
```

**Chain Types**: EVM, BITCOIN, SOLANA, COSMOS, POLKADOT

---

### GetChain

Retrieve chain metadata.

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "ethereum"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetChain
```

---

### ListChains

List all registered chains.

**Request**:
```bash
grpcurl -plaintext -d '{
  "chain_type": "EVM",
  "page_size": 20
}' localhost:9090 cqc.services.v1.AssetRegistry/ListChains
```

---

## Venue Methods

### CreateVenue

Register a trading venue (CEX/DEX/protocol).

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "binance",
  "name": "Binance",
  "venue_type": "CEX",
  "website_url": "https://www.binance.com",
  "api_endpoint": "https://api.binance.com",
  "is_active": true
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateVenue
```

**Request (DEX)**:
```bash
grpcurl -plaintext -d '{
  "id": "uniswap_v3",
  "name": "Uniswap V3",
  "venue_type": "DEX",
  "chain_id": "ethereum",
  "protocol_address": "0x1F98431c8aD98523631AE4a59f267346ea31F984",
  "website_url": "https://app.uniswap.org"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateVenue
```

**Response**:
```json
{
  "venue": {
    "id": "binance",
    "name": "Binance",
    "venue_type": "CEX",
    "website_url": "https://www.binance.com",
    "api_endpoint": "https://api.binance.com",
    "is_active": true,
    "created_at": "2025-10-16T12:34:56Z"
  }
}
```

**Venue Types**: CEX, DEX, DEX_AGGREGATOR, BRIDGE, LENDING

---

### GetVenue

Retrieve venue metadata.

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "binance"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetVenue
```

---

### ListVenues

List venues with filtering.

**Request**:
```bash
grpcurl -plaintext -d '{
  "venue_type": "CEX",
  "is_active": true,
  "page_size": 20
}' localhost:9090 cqc.services.v1.AssetRegistry/ListVenues
```

---

## Venue Asset Methods

### CreateVenueAsset

Map asset availability to venue.

**Request**:
```bash
grpcurl -plaintext -d '{
  "venue_id": "binance",
  "asset_id": "btc-uuid",
  "venue_symbol": "BTC",
  "deposit_enabled": true,
  "withdraw_enabled": true,
  "trading_enabled": true,
  "withdraw_fee": "0.0005",
  "min_withdraw": "0.001"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateVenueAsset
```

**Response**:
```json
{
  "venue_asset": {
    "venue_id": "binance",
    "asset_id": "btc-uuid",
    "venue_symbol": "BTC",
    "deposit_enabled": true,
    "withdraw_enabled": true,
    "trading_enabled": true,
    "withdraw_fee": "0.0005",
    "min_withdraw": "0.001",
    "created_at": "2025-10-16T12:34:56Z"
  }
}
```

---

### GetVenueAsset

Retrieve venue asset mapping.

**Request**:
```bash
grpcurl -plaintext -d '{
  "venue_id": "binance",
  "asset_id": "btc-uuid"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetVenueAsset
```

---

### ListVenueAssets

List assets on venue or venues for asset.

**Request (Assets on Binance)**:
```bash
grpcurl -plaintext -d '{
  "venue_id": "binance",
  "page_size": 50
}' localhost:9090 cqc.services.v1.AssetRegistry/ListVenueAssets
```

**Request (Venues trading BTC)**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "btc-uuid",
  "page_size": 50
}' localhost:9090 cqc.services.v1.AssetRegistry/ListVenueAssets
```

**Response**:
```json
{
  "venue_assets": [
    {
      "venue_id": "binance",
      "asset_id": "btc-uuid",
      "venue_symbol": "BTC",
      "trading_enabled": true
    },
    {
      "venue_id": "coinbase",
      "asset_id": "btc-uuid",
      "venue_symbol": "BTC",
      "trading_enabled": true
    }
  ],
  "next_page_token": ""
}
```

---

## Venue Symbol Methods

### CreateVenueSymbol

Map canonical symbol to venue-specific format.

**Request**:
```bash
grpcurl -plaintext -d '{
  "venue_id": "binance",
  "symbol_id": "btcusdt-spot-uuid",
  "venue_symbol": "BTCUSDT",
  "maker_fee": 0.001,
  "taker_fee": 0.001,
  "is_active": true
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateVenueSymbol
```

**Response**:
```json
{
  "venue_symbol": {
    "venue_id": "binance",
    "symbol_id": "btcusdt-spot-uuid",
    "venue_symbol": "BTCUSDT",
    "maker_fee": 0.001,
    "taker_fee": 0.001,
    "is_active": true,
    "listed_at": "2025-10-16T12:34:56Z"
  }
}
```

---

### GetVenueSymbol

**PRIMARY USE CASE FOR cqmd**: Resolve venue symbol to canonical with market specs.

**Request**:
```bash
grpcurl -plaintext -d '{
  "venue_id": "binance",
  "venue_symbol": "BTCUSDT"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetVenueSymbol
```

**Response** (enriched with canonical symbol):
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
    "lot_size": "0.00001",
    "min_order_size": "0.0001",
    "max_order_size": "1000"
  }
}
```

**Performance**: <10ms p50 (critical for cqmd price ingestion)

**Workflow**:
1. cqmd receives "BTCUSDT" price from Binance WebSocket
2. Call GetVenueSymbol("binance", "BTCUSDT")
3. Extract canonical `symbol_id` and market specs
4. Store price with canonical ID for cross-venue aggregation

---

### ListVenueSymbols

List symbols on venue or venues for symbol.

**Request**:
```bash
grpcurl -plaintext -d '{
  "venue_id": "binance",
  "is_active": true,
  "page_size": 50
}' localhost:9090 cqc.services.v1.AssetRegistry/ListVenueSymbols
```

---

## Deployment Methods

### CreateAssetDeployment

Track asset deployment on specific chain.

**Request**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "usdc-uuid",
  "chain_id": "ethereum",
  "contract_address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
  "decimals": 6,
  "is_canonical": true
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateAssetDeployment
```

**Response**:
```json
{
  "deployment": {
    "asset_id": "usdc-uuid",
    "chain_id": "ethereum",
    "contract_address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
    "decimals": 6,
    "is_canonical": true,
    "created_at": "2025-10-16T12:34:56Z"
  }
}
```

**Validation**:
- `contract_address` format validated per chain_type
- `decimals` range: 0-18
- `asset_id` and `chain_id` must exist

---

### GetAssetDeployment

Retrieve specific deployment.

**Request**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "usdc-uuid",
  "chain_id": "ethereum"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetAssetDeployment
```

---

### ListAssetDeployments

List deployments for asset or chain.

**Request (All USDC deployments)**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "usdc-uuid"
}' localhost:9090 cqc.services.v1.AssetRegistry/ListAssetDeployments
```

**Response**:
```json
{
  "deployments": [
    {
      "asset_id": "usdc-uuid",
      "chain_id": "ethereum",
      "contract_address": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
      "decimals": 6,
      "is_canonical": true
    },
    {
      "asset_id": "usdc-uuid",
      "chain_id": "polygon",
      "contract_address": "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174",
      "decimals": 6,
      "is_canonical": false
    }
  ]
}
```

---

## Relationship Methods

### CreateAssetRelationship

Define relationship between assets.

**Request (WETH wraps ETH)**:
```bash
grpcurl -plaintext -d '{
  "from_asset_id": "weth-uuid",
  "to_asset_id": "eth-uuid",
  "relationship_type": "WRAPS",
  "conversion_rate": "1.0"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateAssetRelationship
```

**Request (stETH stakes ETH)**:
```bash
grpcurl -plaintext -d '{
  "from_asset_id": "steth-uuid",
  "to_asset_id": "eth-uuid",
  "relationship_type": "STAKES",
  "protocol": "Lido"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateAssetRelationship
```

**Response**:
```json
{
  "relationship": {
    "from_asset_id": "weth-uuid",
    "to_asset_id": "eth-uuid",
    "relationship_type": "WRAPS",
    "conversion_rate": "1.0",
    "created_at": "2025-10-16T12:34:56Z"
  }
}
```

**Relationship Types**: WRAPS, STAKES, BRIDGES, SYNTHETIC, LP_PAIR, DERIVATIVE

**Validation**:
- Detects cycles in relationship graph
- Both assets must exist

---

### ListAssetRelationships

List relationships for asset.

**Request (All ETH variants)**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "eth-uuid",
  "relationship_type": "WRAPS"
}' localhost:9090 cqc.services.v1.AssetRegistry/ListAssetRelationships
```

**Response**:
```json
{
  "relationships": [
    {
      "from_asset_id": "weth-uuid",
      "to_asset_id": "eth-uuid",
      "relationship_type": "WRAPS",
      "conversion_rate": "1.0"
    },
    {
      "from_asset_id": "steth-uuid",
      "to_asset_id": "eth-uuid",
      "relationship_type": "STAKES",
      "protocol": "Lido"
    }
  ]
}
```

---

## Quality Flag Methods

### RaiseQualityFlag

Flag asset with quality concern.

**Request**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "suspicious-token-uuid",
  "flag_type": "SCAM",
  "severity": "CRITICAL",
  "source": "community_report",
  "reason": "Honeypot contract detected"
}' localhost:9090 cqc.services.v1.AssetRegistry/RaiseQualityFlag
```

**Response**:
```json
{
  "quality_flag": {
    "id": "flag-uuid",
    "asset_id": "suspicious-token-uuid",
    "flag_type": "SCAM",
    "severity": "CRITICAL",
    "source": "community_report",
    "reason": "Honeypot contract detected",
    "raised_at": "2025-10-16T12:34:56Z",
    "is_resolved": false
  }
}
```

**Flag Types**: SCAM, RUGPULL, EXPLOITED, DEPRECATED, LOW_LIQUIDITY

**Severity Levels**:
- **INFO**: Informational only
- **WARNING**: Proceed with caution
- **CRITICAL**: Blocks trading operations (via QualityManager.IsAssetTradeable)

---

### ResolveQualityFlag

Mark flag as resolved.

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "flag-uuid",
  "resolution_reason": "False positive, verified by security team"
}' localhost:9090 cqc.services.v1.AssetRegistry/ResolveQualityFlag
```

---

### ListQualityFlags

List flags for asset.

**Request**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "suspicious-token-uuid",
  "is_resolved": false
}' localhost:9090 cqc.services.v1.AssetRegistry/ListQualityFlags
```

**Response**:
```json
{
  "quality_flags": [
    {
      "id": "flag-uuid",
      "asset_id": "suspicious-token-uuid",
      "flag_type": "SCAM",
      "severity": "CRITICAL",
      "raised_at": "2025-10-16T12:34:56Z",
      "is_resolved": false
    }
  ]
}
```

---

## Asset Group Methods

### CreateAssetGroup

Create group for portfolio aggregation.

**Request**:
```bash
grpcurl -plaintext -d '{
  "name": "ETH Variants",
  "description": "All ETH and ETH-derivative assets for portfolio aggregation"
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateAssetGroup
```

---

### AddAssetToGroup

Add asset to group.

**Request**:
```bash
grpcurl -plaintext -d '{
  "group_id": "eth-group-uuid",
  "asset_id": "weth-uuid"
}' localhost:9090 cqc.services.v1.AssetRegistry/AddAssetToGroup
```

---

### GetAssetGroup

Retrieve group with members.

**Request**:
```bash
grpcurl -plaintext -d '{
  "id": "eth-group-uuid"
}' localhost:9090 cqc.services.v1.AssetRegistry/GetAssetGroup
```

**Response**:
```json
{
  "group": {
    "id": "eth-group-uuid",
    "name": "ETH Variants",
    "description": "All ETH and ETH-derivative assets",
    "member_asset_ids": [
      "eth-uuid",
      "weth-uuid",
      "steth-uuid",
      "reth-uuid"
    ],
    "created_at": "2025-10-16T12:34:56Z"
  }
}
```

**Use Case**: cqpm aggregates positions across ETH, WETH, stETH for total ETH exposure

---

## Identifier Methods

### CreateAssetIdentifier

Map canonical asset to external provider ID.

**Request**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "btc-uuid",
  "source": "coingecko",
  "external_id": "bitcoin",
  "is_primary": true
}' localhost:9090 cqc.services.v1.AssetRegistry/CreateAssetIdentifier
```

---

### ListAssetIdentifiers

List external IDs for asset.

**Request**:
```bash
grpcurl -plaintext -d '{
  "asset_id": "btc-uuid"
}' localhost:9090 cqc.services.v1.AssetRegistry/ListAssetIdentifiers
```

**Response**:
```json
{
  "identifiers": [
    {
      "asset_id": "btc-uuid",
      "source": "coingecko",
      "external_id": "bitcoin",
      "is_primary": true
    },
    {
      "asset_id": "btc-uuid",
      "source": "coinmarketcap",
      "external_id": "1",
      "is_primary": false
    }
  ]
}
```

---

## Error Codes

gRPC status codes returned by CQAR:

| Code                  | Description              | Example                                     |
| --------------------- | ------------------------ | ------------------------------------------- |
| `OK`                  | Success                  | Request completed successfully              |
| `INVALID_ARGUMENT`    | Invalid request          | Missing required field, invalid UUID format |
| `NOT_FOUND`           | Resource not found       | Asset ID does not exist                     |
| `ALREADY_EXISTS`      | Duplicate resource       | Symbol collision, duplicate deployment      |
| `FAILED_PRECONDITION` | Precondition failed      | CRITICAL flag blocks trading                |
| `UNAUTHENTICATED`     | Missing/invalid auth     | Missing API key header                      |
| `PERMISSION_DENIED`   | Insufficient permissions | API key lacks required scope                |
| `INTERNAL`            | Server error             | Database connection failure, cache error    |
| `UNAVAILABLE`         | Service unavailable      | Database down, cache down                   |
| `UNIMPLEMENTED`       | Method not implemented   | Feature not yet available                   |

### Error Response Format

```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "asset_id is required",
    "details": [
      {
        "field": "asset_id",
        "description": "field is required but not provided"
      }
    ]
  }
}
```

---

## Rate Limits

No rate limits enforced at service level. Rate limiting should be handled at API gateway or load balancer.

**Recommended Limits**:
- **Read operations**: 1000 req/min per service
- **Write operations**: 100 req/min per service
- **Bulk operations**: 10 req/min per service

---

## Best Practices

### Performance Optimization

1. **Cache Warm-Up**: Pre-load frequently accessed assets/symbols on service start
2. **Batch Requests**: Use List methods instead of repeated Get calls
3. **Pagination**: Always set reasonable `page_size` (20-50)
4. **Filtering**: Use filters to reduce response size

### Data Consistency

1. **Idempotency**: Create operations are idempotent (repeat safe)
2. **Foreign Keys**: Validate asset_id, symbol_id exist before creating relationships
3. **Cache Invalidation**: Automatic on updates (no manual invalidation needed)

### Error Handling

1. **Retry Logic**: Retry on `UNAVAILABLE` with exponential backoff
2. **Fallback**: Cache stale data acceptable for non-critical reads
3. **Validation**: Validate inputs client-side before API call

---

**Related Documentation**:
- [SPEC.md](SPEC.md) - Technical architecture
- [OPERATIONS.md](OPERATIONS.md) - Operational procedures
- [README.md](../README.md) - Quick start guide
