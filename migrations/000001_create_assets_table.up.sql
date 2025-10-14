-- Create assets table
-- Canonical assets with metadata, supporting multi-chain deployments
CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- CRYPTOCURRENCY, STABLECOIN, NFT, WRAPPED, SYNTHETIC, GOVERNANCE, MEME, LP_TOKEN
    category VARCHAR(100), -- DeFi, Exchange, Payment, Privacy, Gaming, etc.
    description TEXT,
    logo_url TEXT,
    website_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index on symbol for fast lookups by symbol name
CREATE INDEX idx_assets_symbol ON assets(symbol);

-- Index on type for filtering assets by category
CREATE INDEX idx_assets_type ON assets(type);

-- Index on created_at for chronological queries
CREATE INDEX idx_assets_created_at ON assets(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE assets IS 'Canonical asset registry with unique UUIDs per token across all chains';
COMMENT ON COLUMN assets.id IS 'Unique canonical asset identifier (UUID)';
COMMENT ON COLUMN assets.symbol IS 'Asset ticker symbol (may not be unique across chains)';
COMMENT ON COLUMN assets.name IS 'Human-readable asset name';
COMMENT ON COLUMN assets.type IS 'Asset type classification';
COMMENT ON COLUMN assets.category IS 'Business/functional category for grouping';
COMMENT ON COLUMN assets.description IS 'Detailed asset description or purpose';
COMMENT ON COLUMN assets.logo_url IS 'URL to asset logo/icon image';
COMMENT ON COLUMN assets.website_url IS 'Official website URL';
