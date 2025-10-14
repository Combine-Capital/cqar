-- Create chains table
-- Blockchain networks that assets can be deployed on
CREATE TABLE IF NOT EXISTS chains (
    id VARCHAR(50) PRIMARY KEY, -- ethereum, polygon, arbitrum, base, optimism, solana, cosmos, etc.
    name VARCHAR(100) NOT NULL,
    chain_type VARCHAR(50) NOT NULL, -- EVM, COSMOS, SOLANA, BITCOIN, MOVE, etc.
    native_asset_id UUID REFERENCES assets(id) ON DELETE SET NULL,
    rpc_urls TEXT[], -- PostgreSQL array of RPC endpoint URLs
    explorer_url TEXT, -- Block explorer base URL
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Validate chain_id format (lowercase alphanumeric with underscores)
    CONSTRAINT chk_chain_id_format CHECK (id ~ '^[a-z0-9_]+$')
);

-- Index on chain_type for filtering by blockchain family
CREATE INDEX idx_chains_type ON chains(chain_type);

-- Index on native_asset_id for reverse lookups
CREATE INDEX idx_chains_native_asset ON chains(native_asset_id);

-- Comments for documentation
COMMENT ON TABLE chains IS 'Blockchain networks registry with metadata and RPC endpoints';
COMMENT ON COLUMN chains.id IS 'Unique chain identifier (lowercase, e.g., "ethereum", "polygon")';
COMMENT ON COLUMN chains.name IS 'Human-readable chain name (e.g., "Ethereum Mainnet")';
COMMENT ON COLUMN chains.chain_type IS 'Blockchain family/type (EVM, COSMOS, SOLANA, etc.)';
COMMENT ON COLUMN chains.native_asset_id IS 'Native gas token asset (e.g., ETH for Ethereum)';
COMMENT ON COLUMN chains.rpc_urls IS 'Array of RPC endpoint URLs for blockchain interaction';
COMMENT ON COLUMN chains.explorer_url IS 'Block explorer base URL (e.g., "https://etherscan.io")';
