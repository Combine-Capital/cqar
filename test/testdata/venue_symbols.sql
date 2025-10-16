-- Seed data for venue symbols (trading pairs on venues)

-- Binance - BTCUSDT Spot
INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, is_active, listed_at, metadata, created_at, updated_at)
VALUES (
    'vs111111-1111-1111-1111-111111111111',
    'v1111111-1111-1111-1111-111111111111',  -- Binance
    's1111111-1111-1111-1111-111111111111',  -- BTC/USDT
    'BTCUSDT',
    0.001,
    0.001,
    true,
    NOW() - INTERVAL '5 years',
    '{"market_type": "SPOT"}',
    NOW(),
    NOW()
);

-- Binance - ETHUSDT Spot
INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, is_active, listed_at, metadata, created_at, updated_at)
VALUES (
    'vs222222-2222-2222-2222-222222222222',
    'v1111111-1111-1111-1111-111111111111',  -- Binance
    's2222222-2222-2222-2222-222222222222',  -- ETH/USDT
    'ETHUSDT',
    0.001,
    0.001,
    true,
    NOW() - INTERVAL '4 years',
    '{"market_type": "SPOT"}',
    NOW(),
    NOW()
);

-- Binance - ETHUSDT Perpetual
INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, is_active, listed_at, metadata, created_at, updated_at)
VALUES (
    'vs333333-3333-3333-3333-333333333333',
    'v1111111-1111-1111-1111-111111111111',  -- Binance
    's3333333-3333-3333-3333-333333333333',  -- ETH/USD Perp
    'ETHUSDT',
    0.0002,
    0.0005,
    true,
    NOW() - INTERVAL '3 years',
    '{"market_type": "PERPETUAL", "contract_type": "USDⓈ-M"}',
    NOW(),
    NOW()
);

-- Coinbase - BTC-USDT
INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, is_active, listed_at, metadata, created_at, updated_at)
VALUES (
    'vs444444-4444-4444-4444-444444444444',
    'v2222222-2222-2222-2222-222222222222',  -- Coinbase
    's1111111-1111-1111-1111-111111111111',  -- BTC/USDT
    'BTC-USDT',
    0.004,
    0.006,
    true,
    NOW() - INTERVAL '2 years',
    '{"market_type": "SPOT"}',
    NOW(),
    NOW()
);

-- Coinbase - ETH-USDT
INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, is_active, listed_at, metadata, created_at, updated_at)
VALUES (
    'vs555555-5555-5555-5555-555555555555',
    'v2222222-2222-2222-2222-222222222222',  -- Coinbase
    's2222222-2222-2222-2222-222222222222',  -- ETH/USDT
    'ETH-USDT',
    0.004,
    0.006,
    true,
    NOW() - INTERVAL '2 years',
    '{"market_type": "SPOT"}',
    NOW(),
    NOW()
);

-- Uniswap V3 - WETH/USDC Pool (0.05% fee tier)
INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, is_active, listed_at, metadata, created_at, updated_at)
VALUES (
    'vs666666-6666-6666-6666-666666666666',
    'v3333333-3333-3333-3333-333333333333',  -- Uniswap V3
    's2222222-2222-2222-2222-222222222222',  -- ETH/USDT (using as proxy for WETH/USDC)
    'WETH-USDC-500',
    0.0005,
    0.0005,
    true,
    NOW() - INTERVAL '3 years',
    '{"pool_address": "0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640", "fee_tier": 500, "is_dex": true}',
    NOW(),
    NOW()
);

-- dYdX - ETH-USD Perpetual
INSERT INTO venue_symbols (id, venue_id, symbol_id, venue_symbol, maker_fee, taker_fee, is_active, listed_at, metadata, created_at, updated_at)
VALUES (
    'vs777777-7777-7777-7777-777777777777',
    'v5555555-5555-5555-5555-555555555555',  -- dYdX
    's3333333-3333-3333-3333-333333333333',  -- ETH/USD Perp
    'ETH-USD',
    0.0002,
    0.0005,
    true,
    NOW() - INTERVAL '2 years',
    '{"market_type": "PERPETUAL", "leverage_max": 20}',
    NOW(),
    NOW()
);
