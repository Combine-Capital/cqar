-- Seed data for venue assets (asset availability on venues)

-- Binance - BTC
INSERT INTO venue_assets (id, venue_id, asset_id, venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled, withdraw_fee, min_withdraw_amount, metadata, created_at, updated_at)
VALUES (
    'va111111-1111-1111-1111-111111111111',
    'v1111111-1111-1111-1111-111111111111',  -- Binance
    'a1111111-1111-1111-1111-111111111111',  -- BTC
    'BTC',
    true,
    true,
    true,
    0.0005,
    0.001,
    '{"networks": ["BTC", "BEP20", "BEP2"]}',
    NOW(),
    NOW()
);

-- Binance - ETH
INSERT INTO venue_assets (id, venue_id, asset_id, venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled, withdraw_fee, min_withdraw_amount, metadata, created_at, updated_at)
VALUES (
    'va222222-2222-2222-2222-222222222222',
    'v1111111-1111-1111-1111-111111111111',  -- Binance
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    'ETH',
    true,
    true,
    true,
    0.005,
    0.01,
    '{"networks": ["ETH", "BEP20", "ARBITRUM"]}',
    NOW(),
    NOW()
);

-- Binance - USDT
INSERT INTO venue_assets (id, venue_id, asset_id, venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled, withdraw_fee, min_withdraw_amount, metadata, created_at, updated_at)
VALUES (
    'va333333-3333-3333-3333-333333333333',
    'v1111111-1111-1111-1111-111111111111',  -- Binance
    'a5555555-5555-5555-5555-555555555555',  -- USDT
    'USDT',
    true,
    true,
    true,
    1.0,
    10.0,
    '{"networks": ["ETH", "TRC20", "BEP20", "POLYGON"]}',
    NOW(),
    NOW()
);

-- Coinbase - BTC
INSERT INTO venue_assets (id, venue_id, asset_id, venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled, withdraw_fee, min_withdraw_amount, metadata, created_at, updated_at)
VALUES (
    'va444444-4444-4444-4444-444444444444',
    'v2222222-2222-2222-2222-222222222222',  -- Coinbase
    'a1111111-1111-1111-1111-111111111111',  -- BTC
    'BTC',
    true,
    true,
    true,
    0.0,
    0.0001,
    '{"network_fee_dynamic": true}',
    NOW(),
    NOW()
);

-- Coinbase - ETH
INSERT INTO venue_assets (id, venue_id, asset_id, venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled, withdraw_fee, min_withdraw_amount, metadata, created_at, updated_at)
VALUES (
    'va555555-5555-5555-5555-555555555555',
    'v2222222-2222-2222-2222-222222222222',  -- Coinbase
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    'ETH',
    true,
    true,
    true,
    0.0,
    0.001,
    '{"network_fee_dynamic": true}',
    NOW(),
    NOW()
);

-- Uniswap V3 - WETH
INSERT INTO venue_assets (id, venue_id, asset_id, venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled, withdraw_fee, min_withdraw_amount, metadata, created_at, updated_at)
VALUES (
    'va666666-6666-6666-6666-666666666666',
    'v3333333-3333-3333-3333-333333333333',  -- Uniswap V3
    'a3333333-3333-3333-3333-333333333333',  -- WETH
    'WETH',
    NULL,  -- DEX doesn't have deposit/withdraw concept
    NULL,
    true,
    NULL,
    NULL,
    '{"is_dex": true}',
    NOW(),
    NOW()
);

-- Uniswap V3 - USDC
INSERT INTO venue_assets (id, venue_id, asset_id, venue_symbol, deposit_enabled, withdraw_enabled, trading_enabled, withdraw_fee, min_withdraw_amount, metadata, created_at, updated_at)
VALUES (
    'va777777-7777-7777-7777-777777777777',
    'v3333333-3333-3333-3333-333333333333',  -- Uniswap V3
    'a6666666-6666-6666-6666-666666666666',  -- USDC
    'USDC',
    NULL,
    NULL,
    true,
    NULL,
    NULL,
    '{"is_dex": true}',
    NOW(),
    NOW()
);
