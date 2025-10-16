-- Seed data for chains integration tests

-- Ethereum Mainnet
INSERT INTO chains (id, name, chain_type, native_asset_id, rpc_urls, explorer_url, metadata, created_at, updated_at)
VALUES (
    'c1111111-1111-1111-1111-111111111111',
    'Ethereum',
    'EVM',
    'a2222222-2222-2222-2222-222222222222',  -- ETH
    ARRAY['https://eth.llamarpc.com', 'https://rpc.ankr.com/eth'],
    'https://etherscan.io',
    '{"chain_id": 1, "network": "mainnet"}',
    NOW(),
    NOW()
);

-- Polygon Mainnet
INSERT INTO chains (id, name, chain_type, native_asset_id, rpc_urls, explorer_url, metadata, created_at, updated_at)
VALUES (
    'c2222222-2222-2222-2222-222222222222',
    'Polygon',
    'EVM',
    NULL,  -- MATIC asset not in test seed data
    ARRAY['https://polygon-rpc.com', 'https://rpc.ankr.com/polygon'],
    'https://polygonscan.com',
    '{"chain_id": 137, "network": "mainnet"}',
    NOW(),
    NOW()
);

-- Solana Mainnet
INSERT INTO chains (id, name, chain_type, native_asset_id, rpc_urls, explorer_url, metadata, created_at, updated_at)
VALUES (
    'c3333333-3333-3333-3333-333333333333',
    'Solana',
    'SOLANA',
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',  -- SOL
    ARRAY['https://api.mainnet-beta.solana.com'],
    'https://explorer.solana.com',
    '{"network": "mainnet-beta"}',
    NOW(),
    NOW()
);

-- Bitcoin Mainnet
INSERT INTO chains (id, name, chain_type, native_asset_id, rpc_urls, explorer_url, metadata, created_at, updated_at)
VALUES (
    'c4444444-4444-4444-4444-444444444444',
    'Bitcoin',
    'BITCOIN',
    'a1111111-1111-1111-1111-111111111111',  -- BTC
    ARRAY['https://btc.getblock.io/mainnet'],
    'https://blockchair.com/bitcoin',
    '{"network": "mainnet"}',
    NOW(),
    NOW()
);
