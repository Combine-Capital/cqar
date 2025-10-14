-- Create relationships table
-- Tracks relationships between assets (wraps, stakes, bridges, derivatives)
CREATE TABLE IF NOT EXISTS relationships (
    id UUID PRIMARY KEY,
    from_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    to_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) NOT NULL,
    conversion_rate DECIMAL(30, 18), -- Optional: exchange rate between assets (e.g., stETH:ETH ratio)
    protocol VARCHAR(100), -- Protocol facilitating the relationship (e.g., "Lido", "Uniswap", "Stargate")
    description TEXT, -- Human-readable explanation of the relationship
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure unique relationship between same assets in same direction for same type
    CONSTRAINT unique_asset_relationship UNIQUE(from_asset_id, to_asset_id, relationship_type),
    -- Validate relationship_type enum
    CONSTRAINT chk_relationship_type CHECK (relationship_type IN (
        'WRAPS',        -- WETH wraps ETH, WBTC wraps BTC
        'STAKES',       -- stETH stakes ETH, rETH stakes ETH
        'BRIDGES',      -- USDC.e bridges USDC, axlUSDC bridges USDC
        'DERIVES',      -- Derivative relationship (e.g., futures from spot)
        'LP_TOKEN',     -- LP token represents pool of assets
        'SYNTHETIC',    -- Synthetic asset tracking another
        'REBASES'       -- Rebasing token relationship
    )),
    -- Prevent self-referential relationships
    CONSTRAINT chk_no_self_reference CHECK (from_asset_id != to_asset_id),
    -- Validate conversion_rate is positive if set
    CONSTRAINT chk_conversion_rate_positive CHECK (conversion_rate IS NULL OR conversion_rate > 0)
);

-- Index on from_asset_id for "what does this asset wrap/stake/bridge?" queries
CREATE INDEX idx_relationships_from_asset ON relationships(from_asset_id);

-- Index on to_asset_id for "what wraps/stakes/bridges to this asset?" queries
CREATE INDEX idx_relationships_to_asset ON relationships(to_asset_id);

-- Index on relationship_type for filtering by relationship category
CREATE INDEX idx_relationships_type ON relationships(relationship_type);

-- Composite index for bidirectional lookups (find all relationships between two assets)
CREATE INDEX idx_relationships_bidirectional ON relationships(from_asset_id, to_asset_id);

-- Index on protocol for "which relationships use Lido?" queries
CREATE INDEX idx_relationships_protocol ON relationships(protocol) WHERE protocol IS NOT NULL;

-- Index on created_at for chronological queries
CREATE INDEX idx_relationships_created_at ON relationships(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE relationships IS 'Asset relationships for wrapping, staking, bridging, and derivative tracking';
COMMENT ON COLUMN relationships.id IS 'Unique relationship identifier (UUID)';
COMMENT ON COLUMN relationships.from_asset_id IS 'Source asset in the relationship';
COMMENT ON COLUMN relationships.to_asset_id IS 'Target asset in the relationship';
COMMENT ON COLUMN relationships.relationship_type IS 'Type of relationship: WRAPS, STAKES, BRIDGES, DERIVES, LP_TOKEN, SYNTHETIC, REBASES';
COMMENT ON COLUMN relationships.conversion_rate IS 'Optional exchange rate between assets (e.g., 0.98 for stETH:ETH)';
COMMENT ON COLUMN relationships.protocol IS 'Protocol facilitating the relationship (e.g., "Lido", "Stargate")';
COMMENT ON COLUMN relationships.description IS 'Human-readable explanation of the relationship';
