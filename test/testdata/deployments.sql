-- Seed data for asset deployments integration tests

-- WETH on Ethereum
INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, is_canonical, deployed_at, metadata, created_at, updated_at)
VALUES (
    'd1111111-1111-1111-1111-111111111111',
    'a3333333-3333-3333-3333-333333333333',  -- WETH
    'c1111111-1111-1111-1111-111111111111',  -- Ethereum
    '0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2',
    18,
    true,
    NOW(),
    '{"contract_name": "WETH9", "verified": true}',
    NOW(),
    NOW()
);

-- USDT on Ethereum
INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, is_canonical, deployed_at, metadata, created_at, updated_at)
VALUES (
    'd2222222-2222-2222-2222-222222222222',
    'a5555555-5555-5555-5555-555555555555',  -- USDT
    'c1111111-1111-1111-1111-111111111111',  -- Ethereum
    '0xdAC17F958D2ee523a2206206994597C13D831ec7',
    6,
    true,
    NOW(),
    '{"contract_name": "TetherToken", "verified": true}',
    NOW(),
    NOW()
);

-- USDC on Ethereum
INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, is_canonical, deployed_at, metadata, created_at, updated_at)
VALUES (
    'd3333333-3333-3333-3333-333333333333',
    'a6666666-6666-6666-6666-666666666666',  -- USDC Ethereum
    'c1111111-1111-1111-1111-111111111111',  -- Ethereum
    '0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48',
    6,
    true,
    NOW(),
    '{"contract_name": "USD Coin", "verified": true}',
    NOW(),
    NOW()
);

-- USDC on Polygon (native)
INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, is_canonical, deployed_at, metadata, created_at, updated_at)
VALUES (
    'd4444444-4444-4444-4444-444444444444',
    'a7777777-7777-7777-7777-777777777777',  -- USDC Polygon
    'c2222222-2222-2222-2222-222222222222',  -- Polygon
    '0x3c499c542cEF5E3811e1192ce70d8cC03d5c3359',
    6,
    true,
    NOW(),
    '{"contract_name": "USD Coin", "verified": true}',
    NOW(),
    NOW()
);

-- USDC.e on Polygon (bridged)
INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, is_canonical, deployed_at, metadata, created_at, updated_at)
VALUES (
    'd5555555-5555-5555-5555-555555555555',
    'a8888888-8888-8888-8888-888888888888',  -- USDC.e
    'c2222222-2222-2222-2222-222222222222',  -- Polygon
    '0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174',
    6,
    true,
    NOW(),
    '{"contract_name": "USD Coin (PoS)", "verified": true, "bridge": "polygon_pos"}',
    NOW(),
    NOW()
);

-- stETH on Ethereum
INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, is_canonical, deployed_at, metadata, created_at, updated_at)
VALUES (
    'd6666666-6666-6666-6666-666666666666',
    'a4444444-4444-4444-4444-444444444444',  -- stETH
    'c1111111-1111-1111-1111-111111111111',  -- Ethereum
    '0xae7ab96520DE3A18E5e111B5EaAb095312D7fE84',
    18,
    true,
    NOW(),
    '{"contract_name": "Lido: stETH Token", "verified": true, "protocol": "lido"}',
    NOW(),
    NOW()
);

-- DAI on Ethereum
INSERT INTO deployments (id, asset_id, chain_id, contract_address, decimals, is_canonical, deployed_at, metadata, created_at, updated_at)
VALUES (
    'd7777777-7777-7777-7777-777777777777',
    'a9999999-9999-9999-9999-999999999999',  -- DAI
    'c1111111-1111-1111-1111-111111111111',  -- Ethereum
    '0x6B175474E89094C44Da98b954EedeAC495271d0F',
    18,
    true,
    NOW(),
    '{"contract_name": "Dai Stablecoin", "verified": true, "protocol": "makerdao"}',
    NOW(),
    NOW()
);
