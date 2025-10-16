-- Seed data for symbols integration tests

-- BTC/USDT Spot
INSERT INTO symbols (id, base_asset_id, quote_asset_id, settlement_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, metadata, created_at, updated_at)
VALUES (
    's1111111-1111-1111-1111-111111111111',
    'a1111111-1111-1111-1111-111111111111',  -- BTC
    'a5555555-5555-5555-5555-555555555555',  -- USDT
    NULL,
    'SPOT',
    0.01,
    0.00001,
    0.0001,
    1000.0,
    '{"canonical_format": "BTC/USDT"}',
    NOW(),
    NOW()
);

-- ETH/USDT Spot
INSERT INTO symbols (id, base_asset_id, quote_asset_id, settlement_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, metadata, created_at, updated_at)
VALUES (
    's2222222-2222-2222-2222-222222222222',
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    'a5555555-5555-5555-5555-555555555555',  -- USDT
    NULL,
    'SPOT',
    0.01,
    0.0001,
    0.001,
    10000.0,
    '{"canonical_format": "ETH/USDT"}',
    NOW(),
    NOW()
);

-- ETH/USD Perpetual
INSERT INTO symbols (id, base_asset_id, quote_asset_id, settlement_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, contract_size, funding_interval, metadata, created_at, updated_at)
VALUES (
    's3333333-3333-3333-3333-333333333333',
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    NULL,  -- USD (not asset, just quote unit)
    'a5555555-5555-5555-5555-555555555555',  -- USDT settlement
    'PERPETUAL',
    0.01,
    0.001,
    0.01,
    100000.0,
    1.0,
    '8h',
    '{"canonical_format": "ETH-PERP", "index": "ETH/USD"}',
    NOW(),
    NOW()
);

-- BTC/USD Future (March 2026)
INSERT INTO symbols (id, base_asset_id, quote_asset_id, settlement_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, contract_size, expiry, metadata, created_at, updated_at)
VALUES (
    's4444444-4444-4444-4444-444444444444',
    'a1111111-1111-1111-1111-111111111111',  -- BTC
    NULL,  -- USD
    'a5555555-5555-5555-5555-555555555555',  -- USDT settlement
    'FUTURE',
    0.1,
    0.001,
    0.001,
    10000.0,
    1.0,
    '2026-03-27T08:00:00Z',
    '{"canonical_format": "BTC-0327", "delivery_type": "cash_settled"}',
    NOW(),
    NOW()
);

-- ETH Call Option (strike $3000, expiry Dec 2025)
INSERT INTO symbols (id, base_asset_id, quote_asset_id, settlement_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, contract_size, strike_price, expiry, option_type, metadata, created_at, updated_at)
VALUES (
    's5555555-5555-5555-5555-555555555555',
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    'a6666666-6666-6666-6666-666666666666',  -- USDC
    'a6666666-6666-6666-6666-666666666666',  -- USDC settlement
    'OPTION',
    0.01,
    0.01,
    0.1,
    1000.0,
    1.0,
    3000.0,
    '2025-12-26T08:00:00Z',
    'CALL',
    '{"canonical_format": "ETH-26DEC25-3000-C", "exercise_type": "european"}',
    NOW(),
    NOW()
);

-- SOL/USDT Spot
INSERT INTO symbols (id, base_asset_id, quote_asset_id, settlement_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, metadata, created_at, updated_at)
VALUES (
    's6666666-6666-6666-6666-666666666666',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',  -- SOL
    'a5555555-5555-5555-5555-555555555555',  -- USDT
    NULL,
    'SPOT',
    0.001,
    0.01,
    0.1,
    100000.0,
    '{"canonical_format": "SOL/USDT"}',
    NOW(),
    NOW()
);

-- ETH/BTC Spot
INSERT INTO symbols (id, base_asset_id, quote_asset_id, settlement_asset_id, symbol_type, tick_size, lot_size, min_order_size, max_order_size, metadata, created_at, updated_at)
VALUES (
    's7777777-7777-7777-7777-777777777777',
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    'a1111111-1111-1111-1111-111111111111',  -- BTC
    NULL,
    'SPOT',
    0.000001,
    0.0001,
    0.001,
    10000.0,
    '{"canonical_format": "ETH/BTC"}',
    NOW(),
    NOW()
);
