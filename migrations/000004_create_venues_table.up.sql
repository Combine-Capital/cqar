-- Create venues table
-- Trading venues (exchanges, DEXs, aggregators, bridges, lending platforms)
CREATE TABLE IF NOT EXISTS venues (
    id VARCHAR(100) PRIMARY KEY, -- binance, uniswap_v3_eth, dydx, curve, aave_v3, etc.
    name VARCHAR(255) NOT NULL,
    venue_type VARCHAR(50) NOT NULL, -- CEX, DEX, DEX_AGGREGATOR, BRIDGE, LENDING, DERIVATIVES
    chain_id VARCHAR(50) REFERENCES chains(id) ON DELETE RESTRICT, -- NULL for CEX, required for DEX/DeFi
    protocol_address VARCHAR(255), -- Smart contract address for on-chain venues
    website_url TEXT,
    api_endpoint TEXT, -- REST/WebSocket API endpoint for CEX
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Validate venue_id format (lowercase alphanumeric with underscores)
    CONSTRAINT chk_venue_id_format CHECK (id ~ '^[a-z0-9_]+$'),
    -- Validate venue_type enum
    CONSTRAINT chk_venue_type CHECK (venue_type IN ('CEX', 'DEX', 'DEX_AGGREGATOR', 'BRIDGE', 'LENDING', 'DERIVATIVES')),
    -- DEX venues must have a chain_id
    CONSTRAINT chk_dex_has_chain CHECK (
        (venue_type = 'CEX') OR 
        (chain_id IS NOT NULL)
    )
);

-- Index on venue_type for filtering venues by category
CREATE INDEX idx_venues_type ON venues(venue_type);

-- Index on chain_id for "which venues operate on Ethereum?" queries
CREATE INDEX idx_venues_chain_id ON venues(chain_id);

-- Index on is_active for filtering active venues
CREATE INDEX idx_venues_active ON venues(is_active) WHERE is_active = true;

-- Comments for documentation
COMMENT ON TABLE venues IS 'Trading venues and DeFi protocols where assets and symbols are available';
COMMENT ON COLUMN venues.id IS 'Unique venue identifier (lowercase, e.g., "binance", "uniswap_v3_eth")';
COMMENT ON COLUMN venues.name IS 'Human-readable venue name (e.g., "Binance", "Uniswap V3 Ethereum")';
COMMENT ON COLUMN venues.venue_type IS 'Venue category: CEX, DEX, DEX_AGGREGATOR, BRIDGE, LENDING, DERIVATIVES';
COMMENT ON COLUMN venues.chain_id IS 'Blockchain chain for on-chain venues (NULL for CEX)';
COMMENT ON COLUMN venues.protocol_address IS 'Smart contract address for on-chain protocols';
COMMENT ON COLUMN venues.website_url IS 'Official website URL';
COMMENT ON COLUMN venues.api_endpoint IS 'API endpoint for programmatic access (primarily CEX)';
COMMENT ON COLUMN venues.is_active IS 'Whether venue is currently operational';
