-- Create asset_identifiers table
-- Maps canonical assets to external provider IDs (CoinGecko, CoinMarketCap, DeFiLlama, etc.)
CREATE TABLE IF NOT EXISTS asset_identifiers (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    source VARCHAR(50) NOT NULL, -- coingecko, coinmarketcap, defillama, messari, etc.
    external_id VARCHAR(255) NOT NULL, -- Provider-specific ID (e.g., "bitcoin" for CoinGecko)
    is_primary BOOLEAN NOT NULL DEFAULT false, -- Is this the primary/preferred ID for this source?
    metadata JSONB, -- Additional provider-specific metadata
    verified_at TIMESTAMPTZ, -- When this mapping was verified
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure unique asset+source combination
    CONSTRAINT unique_asset_source UNIQUE(asset_id, source),
    -- Ensure unique external_id per source (one CoinGecko ID = one asset)
    CONSTRAINT unique_asset_source_external_id UNIQUE(source, external_id),
    -- Validate source format (lowercase alphanumeric with underscores)
    CONSTRAINT chk_source_format CHECK (source ~ '^[a-z0-9_]+$'),
    -- Validate external_id not empty
    CONSTRAINT chk_external_id_not_empty CHECK (LENGTH(TRIM(external_id)) > 0)
);

-- Partial unique index: only one primary identifier per asset per source
CREATE UNIQUE INDEX unique_primary_per_asset_source ON asset_identifiers(asset_id, source) WHERE is_primary = true;

-- Index on asset_id for "what external IDs does this asset have?" queries
CREATE INDEX idx_asset_identifiers_asset_id ON asset_identifiers(asset_id);

-- Index on source for "all CoinGecko mappings" queries
CREATE INDEX idx_asset_identifiers_source ON asset_identifiers(source);

-- Index on external_id for reverse lookups from external systems
CREATE INDEX idx_asset_identifiers_external_id ON asset_identifiers(external_id);

-- Composite index for source+external_id lookups (most common query pattern)
CREATE INDEX idx_asset_identifiers_source_external ON asset_identifiers(source, external_id);

-- Index on is_primary for finding primary identifiers
CREATE INDEX idx_asset_identifiers_primary ON asset_identifiers(is_primary) WHERE is_primary = true;

-- GIN index on metadata JSONB for flexible queries
CREATE INDEX idx_asset_identifiers_metadata ON asset_identifiers USING GIN (metadata);

-- Index on created_at for chronological queries
CREATE INDEX idx_asset_identifiers_created_at ON asset_identifiers(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE asset_identifiers IS 'Maps canonical assets to external provider identifiers';
COMMENT ON COLUMN asset_identifiers.id IS 'Unique identifier mapping UUID';
COMMENT ON COLUMN asset_identifiers.asset_id IS 'Canonical asset being mapped';
COMMENT ON COLUMN asset_identifiers.source IS 'External provider (coingecko, coinmarketcap, defillama, etc.)';
COMMENT ON COLUMN asset_identifiers.external_id IS 'Provider-specific identifier (e.g., "bitcoin" for CoinGecko)';
COMMENT ON COLUMN asset_identifiers.is_primary IS 'Is this the primary/preferred identifier for this source?';
COMMENT ON COLUMN asset_identifiers.metadata IS 'Additional provider-specific metadata (JSONB)';
COMMENT ON COLUMN asset_identifiers.verified_at IS 'Timestamp when mapping was last verified';
