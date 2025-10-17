-- Create unified identifiers table
-- Maps assets, instruments, and markets to external provider IDs
CREATE TABLE IF NOT EXISTS identifiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('ASSET', 'INSTRUMENT', 'MARKET')),
    asset_id UUID REFERENCES assets(id) ON DELETE CASCADE,
    instrument_id UUID REFERENCES instruments(id) ON DELETE CASCADE,
    market_id UUID REFERENCES markets(id) ON DELETE CASCADE,
    source VARCHAR(50) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    is_primary BOOLEAN NOT NULL DEFAULT false,
    metadata JSONB,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure exactly one of asset_id, instrument_id, or market_id is set
    CONSTRAINT chk_one_entity_id CHECK (
        (entity_type = 'ASSET' AND asset_id IS NOT NULL AND instrument_id IS NULL AND market_id IS NULL) OR
        (entity_type = 'INSTRUMENT' AND asset_id IS NULL AND instrument_id IS NOT NULL AND market_id IS NULL) OR
        (entity_type = 'MARKET' AND asset_id IS NULL AND instrument_id IS NULL AND market_id IS NOT NULL)
    ),
    -- Ensure unique external_id per source (one CoinGecko ID = one entity)
    CONSTRAINT unique_source_external_id UNIQUE(source, external_id),
    -- Validate source format (lowercase alphanumeric with underscores)
    CONSTRAINT chk_source_format CHECK (source ~ '^[a-z0-9_]+$'),
    -- Validate external_id not empty
    CONSTRAINT chk_external_id_not_empty CHECK (LENGTH(TRIM(external_id)) > 0)
);

-- Partial unique index: only one primary identifier per entity per source
CREATE UNIQUE INDEX unique_primary_per_asset_source ON identifiers(asset_id, source) 
    WHERE entity_type = 'ASSET' AND is_primary = true;
CREATE UNIQUE INDEX unique_primary_per_instrument_source ON identifiers(instrument_id, source) 
    WHERE entity_type = 'INSTRUMENT' AND is_primary = true;
CREATE UNIQUE INDEX unique_primary_per_market_source ON identifiers(market_id, source) 
    WHERE entity_type = 'MARKET' AND is_primary = true;

-- Indexes for entity lookups
CREATE INDEX idx_identifiers_entity_type ON identifiers(entity_type);
CREATE INDEX idx_identifiers_asset_id ON identifiers(asset_id) WHERE asset_id IS NOT NULL;
CREATE INDEX idx_identifiers_instrument_id ON identifiers(instrument_id) WHERE instrument_id IS NOT NULL;
CREATE INDEX idx_identifiers_market_id ON identifiers(market_id) WHERE market_id IS NOT NULL;

-- Index on source for "all CoinGecko mappings" queries
CREATE INDEX idx_identifiers_source ON identifiers(source);

-- Composite index for source+external_id lookups (most common query pattern)
CREATE INDEX idx_identifiers_source_external ON identifiers(source, external_id);

-- Index on is_primary for finding primary identifiers
CREATE INDEX idx_identifiers_primary ON identifiers(is_primary) WHERE is_primary = true;

-- GIN index on metadata JSONB for flexible queries
CREATE INDEX idx_identifiers_metadata ON identifiers USING GIN (metadata);

-- Comments for documentation
COMMENT ON TABLE identifiers IS 'Unified table mapping assets, instruments, and markets to external provider identifiers';
COMMENT ON COLUMN identifiers.entity_type IS 'Type of entity: ASSET, INSTRUMENT, or MARKET';
COMMENT ON COLUMN identifiers.asset_id IS 'Canonical asset being mapped (only for entity_type=ASSET)';
COMMENT ON COLUMN identifiers.instrument_id IS 'Instrument being mapped (only for entity_type=INSTRUMENT)';
COMMENT ON COLUMN identifiers.market_id IS 'Market being mapped (only for entity_type=MARKET)';
COMMENT ON COLUMN identifiers.source IS 'External provider (coingecko, coinmarketcap, defillama, tradingview, etc.)';
COMMENT ON COLUMN identifiers.external_id IS 'Provider-specific identifier';
COMMENT ON COLUMN identifiers.is_primary IS 'Is this the primary/preferred identifier for this source?';
