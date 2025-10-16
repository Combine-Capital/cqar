-- Seed data for assets integration tests
-- This file contains reference data for common crypto assets used in testing

-- Bitcoin (BTC)
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a1111111-1111-1111-1111-111111111111',
    'BTC',
    'Bitcoin',
    'CRYPTOCURRENCY',
    'NATIVE',
    'Bitcoin is a decentralized digital currency that can be transferred on the peer-to-peer bitcoin network.',
    'https://assets.coingecko.com/coins/images/1/large/bitcoin.png',
    'https://bitcoin.org',
    '{"market_cap_rank": 1, "coingecko_id": "bitcoin"}',
    NOW(),
    NOW()
);

-- Ethereum (ETH)
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a2222222-2222-2222-2222-222222222222',
    'ETH',
    'Ethereum',
    'CRYPTOCURRENCY',
    'NATIVE',
    'Ethereum is a decentralized platform for applications that run exactly as programmed without downtime, fraud or interference.',
    'https://assets.coingecko.com/coins/images/279/large/ethereum.png',
    'https://ethereum.org',
    '{"market_cap_rank": 2, "coingecko_id": "ethereum"}',
    NOW(),
    NOW()
);

-- Wrapped Ethereum (WETH)
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a3333333-3333-3333-3333-333333333333',
    'WETH',
    'Wrapped Ethereum',
    'TOKEN',
    'WRAPPED',
    'WETH is an ERC-20 compatible version of Ethereum that can be traded on decentralized exchanges.',
    'https://assets.coingecko.com/coins/images/2518/large/weth.png',
    'https://weth.io',
    '{"coingecko_id": "weth"}',
    NOW(),
    NOW()
);

-- Staked Ethereum (stETH)
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a4444444-4444-4444-4444-444444444444',
    'stETH',
    'Lido Staked Ether',
    'TOKEN',
    'STAKED',
    'stETH is a token that represents staked ether in Lido, combining the value of initial deposit with staking rewards.',
    'https://assets.coingecko.com/coins/images/13442/large/steth_logo.png',
    'https://lido.fi',
    '{"coingecko_id": "staked-ether", "protocol": "lido"}',
    NOW(),
    NOW()
);

-- Tether (USDT) - Ethereum
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a5555555-5555-5555-5555-555555555555',
    'USDT',
    'Tether USD (Ethereum)',
    'STABLECOIN',
    'FIAT_BACKED',
    'USDT is a stablecoin pegged to the US Dollar, issued on Ethereum blockchain.',
    'https://assets.coingecko.com/coins/images/325/large/Tether.png',
    'https://tether.to',
    '{"coingecko_id": "tether", "peg_currency": "USD", "chain": "ethereum"}',
    NOW(),
    NOW()
);

-- USD Coin (USDC) - Ethereum
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a6666666-6666-6666-6666-666666666666',
    'USDC',
    'USD Coin (Ethereum)',
    'STABLECOIN',
    'FIAT_BACKED',
    'USDC is a fully collateralized US dollar stablecoin on Ethereum.',
    'https://assets.coingecko.com/coins/images/6319/large/USD_Coin_icon.png',
    'https://www.circle.com/usdc',
    '{"coingecko_id": "usd-coin", "peg_currency": "USD", "chain": "ethereum"}',
    NOW(),
    NOW()
);

-- USD Coin (USDC) - Polygon (demonstrates symbol collision across chains)
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a7777777-7777-7777-7777-777777777777',
    'USDC',
    'USD Coin (Polygon)',
    'STABLECOIN',
    'FIAT_BACKED',
    'USDC is a fully collateralized US dollar stablecoin on Polygon.',
    'https://assets.coingecko.com/coins/images/6319/large/USD_Coin_icon.png',
    'https://www.circle.com/usdc',
    '{"coingecko_id": "usd-coin", "peg_currency": "USD", "chain": "polygon"}',
    NOW(),
    NOW()
);

-- Bridged USDC (USDC.e) - Polygon
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a8888888-8888-8888-8888-888888888888',
    'USDC.e',
    'Bridged USD Coin (Polygon)',
    'STABLECOIN',
    'BRIDGED',
    'USDC.e is the bridged version of USDC on Polygon network.',
    'https://assets.coingecko.com/coins/images/6319/large/USD_Coin_icon.png',
    'https://www.circle.com/usdc',
    '{"coingecko_id": "usd-coin-polygon-pos-bridge", "peg_currency": "USD", "chain": "polygon", "bridge": "polygon_pos"}',
    NOW(),
    NOW()
);

-- Dai Stablecoin (DAI)
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'a9999999-9999-9999-9999-999999999999',
    'DAI',
    'Dai Stablecoin',
    'STABLECOIN',
    'CRYPTO_BACKED',
    'DAI is a decentralized stablecoin soft-pegged to the US Dollar backed by crypto collateral.',
    'https://assets.coingecko.com/coins/images/9956/large/Badge_Dai.png',
    'https://makerdao.com',
    '{"coingecko_id": "dai", "peg_currency": "USD", "protocol": "makerdao"}',
    NOW(),
    NOW()
);

-- Solana (SOL)
INSERT INTO assets (id, symbol, name, type, category, description, logo_url, website_url, metadata, created_at, updated_at)
VALUES (
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    'SOL',
    'Solana',
    'CRYPTOCURRENCY',
    'NATIVE',
    'Solana is a high-performance blockchain supporting builders around the world creating crypto apps.',
    'https://assets.coingecko.com/coins/images/4128/large/solana.png',
    'https://solana.com',
    '{"market_cap_rank": 5, "coingecko_id": "solana"}',
    NOW(),
    NOW()
);
