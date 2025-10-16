-- Seed data for venues integration tests

-- Binance (CEX)
INSERT INTO venues (id, name, venue_type, chain_id, protocol_address, website_url, api_endpoint, is_active, metadata, created_at, updated_at)
VALUES (
    'v1111111-1111-1111-1111-111111111111',
    'Binance',
    'CEX',
    NULL,
    NULL,
    'https://www.binance.com',
    'https://api.binance.com',
    true,
    '{"country": "global", "kyc_required": true, "spot_trading": true, "derivatives_trading": true}',
    NOW(),
    NOW()
);

-- Coinbase (CEX)
INSERT INTO venues (id, name, venue_type, chain_id, protocol_address, website_url, api_endpoint, is_active, metadata, created_at, updated_at)
VALUES (
    'v2222222-2222-2222-2222-222222222222',
    'Coinbase',
    'CEX',
    NULL,
    NULL,
    'https://www.coinbase.com',
    'https://api.coinbase.com',
    true,
    '{"country": "USA", "kyc_required": true, "spot_trading": true, "derivatives_trading": false}',
    NOW(),
    NOW()
);

-- Uniswap V3 (DEX)
INSERT INTO venues (id, name, venue_type, chain_id, protocol_address, website_url, api_endpoint, is_active, metadata, created_at, updated_at)
VALUES (
    'v3333333-3333-3333-3333-333333333333',
    'Uniswap V3',
    'DEX',
    'c1111111-1111-1111-1111-111111111111',  -- Ethereum
    '0x1F98431c8aD98523631AE4a59f267346ea31F984',  -- Factory
    'https://uniswap.org',
    'https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3',
    true,
    '{"version": "3", "dex_type": "AMM", "fee_tiers": [100, 500, 3000, 10000]}',
    NOW(),
    NOW()
);

-- Curve Finance (DEX)
INSERT INTO venues (id, name, venue_type, chain_id, protocol_address, website_url, api_endpoint, is_active, metadata, created_at, updated_at)
VALUES (
    'v4444444-4444-4444-4444-444444444444',
    'Curve',
    'DEX',
    'c1111111-1111-1111-1111-111111111111',  -- Ethereum
    '0xbEbc44782C7dB0a1A60Cb6fe97d0b483032FF1C7',  -- 3pool
    'https://curve.fi',
    'https://api.curve.fi',
    true,
    '{"dex_type": "stable_swap", "focus": "stablecoins"}',
    NOW(),
    NOW()
);

-- dYdX (DEX - Derivatives)
INSERT INTO venues (id, name, venue_type, chain_id, protocol_address, website_url, api_endpoint, is_active, metadata, created_at, updated_at)
VALUES (
    'v5555555-5555-5555-5555-555555555555',
    'dYdX',
    'DEX',
    'c1111111-1111-1111-1111-111111111111',  -- Ethereum (Layer 2)
    NULL,
    'https://dydx.exchange',
    'https://api.dydx.exchange',
    true,
    '{"dex_type": "orderbook", "derivatives": true, "layer": "layer2"}',
    NOW(),
    NOW()
);
