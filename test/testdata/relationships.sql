-- Seed data for asset relationships integration tests

-- WETH wraps ETH
INSERT INTO relationships (id, from_asset_id, to_asset_id, relationship_type, conversion_rate, protocol, metadata, created_at, updated_at)
VALUES (
    'r1111111-1111-1111-1111-111111111111',
    'a3333333-3333-3333-3333-333333333333',  -- WETH
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    'WRAPS',
    1.0,
    'weth9',
    '{"reversible": true}',
    NOW(),
    NOW()
);

-- stETH stakes ETH
INSERT INTO relationships (id, from_asset_id, to_asset_id, relationship_type, conversion_rate, protocol, metadata, created_at, updated_at)
VALUES (
    'r2222222-2222-2222-2222-222222222222',
    'a4444444-4444-4444-4444-444444444444',  -- stETH
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    'STAKES',
    1.0,
    'lido',
    '{"accrues_rewards": true, "exchange_rate_varies": true}',
    NOW(),
    NOW()
);

-- USDC.e bridges to USDC (Polygon)
INSERT INTO relationships (id, from_asset_id, to_asset_id, relationship_type, conversion_rate, protocol, metadata, created_at, updated_at)
VALUES (
    'r3333333-3333-3333-3333-333333333333',
    'a8888888-8888-8888-8888-888888888888',  -- USDC.e
    'a7777777-7777-7777-7777-777777777777',  -- USDC Polygon
    'BRIDGES',
    1.0,
    'polygon_pos_bridge',
    '{"bridge_type": "pos", "reversible": true}',
    NOW(),
    NOW()
);

-- USDC Polygon derives from USDC Ethereum
INSERT INTO relationships (id, from_asset_id, to_asset_id, relationship_type, conversion_rate, protocol, metadata, created_at, updated_at)
VALUES (
    'r4444444-4444-4444-4444-444444444444',
    'a7777777-7777-7777-7777-777777777777',  -- USDC Polygon
    'a6666666-6666-6666-6666-666666666666',  -- USDC Ethereum
    'DERIVES',
    1.0,
    'circle',
    '{"canonical_source": true}',
    NOW(),
    NOW()
);
