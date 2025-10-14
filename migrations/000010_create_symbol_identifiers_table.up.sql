-- Create symbol_identifiers table
-- Maps canonical symbols to external provider IDs (exchange-specific symbol IDs, data provider IDs)
CREATE TABLE IF NOT EXISTS symbol_identifiers (
    id UUID PRIMARY KEY,
    symbol_id UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    source VARCHAR(50) NOT NULL, -- coingecko, coinmarketcap, tradingview, etc.
    external_id VARCHAR(255) NOT NULL, -- Provider-specific symbol ID
    is_primary BOOLEAN NOT NULL DEFAULT false, -- Is this the primary/preferred ID for this source?
    metadata JSONB, -- Additional provider-specific metadata
    verified_at TIMESTAMPTZ, -- When this mapping was verified
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure unique symbol+source combination
    CONSTRAINT unique_symbol_source UNIQUE(symbol_id, source),
    -- Ensure unique external_id per source
    CONSTRAINT unique_symbol_source_external_id UNIQUE(source, external_id),
    -- Validate source format (lowercase alphanumeric with underscores)
    CONSTRAINT chk_source_format CHECK (source ~ '^[a-z0-9_]+$'),
    -- Validate external_id not empty
    CONSTRAINT chk_external_id_not_empty CHECK (LENGTH(TRIM(external_id)) > 0)
);

-- Partial unique index: only one primary identifier per symbol per source
CREATE UNIQUE INDEX unique_primary_per_symbol_source ON symbol_identifiers(symbol_id, source) WHERE is_primary = true;

-- Index on symbol_id for "what external IDs does this symbol have?" queries
CREATE INDEX idx_symbol_identifiers_symbol_id ON symbol_identifiers(symbol_id);

-- Index on source for "all CoinGecko symbol mappings" queries
CREATE INDEX idx_symbol_identifiers_source ON symbol_identifiers(source);

-- Index on external_id for reverse lookups from external systems
CREATE INDEX idx_symbol_identifiers_external_id ON symbol_identifiers(external_id);

-- Composite index for source+external_id lookups (most common query pattern)
CREATE INDEX idx_symbol_identifiers_source_external ON symbol_identifiers(source, external_id);

-- Index on is_primary for finding primary identifiers
CREATE INDEX idx_symbol_identifiers_primary ON symbol_identifiers(is_primary) WHERE is_primary = true;

-- GIN index on metadata JSONB for flexible queries
CREATE INDEX idx_symbol_identifiers_metadata ON symbol_identifiers USING GIN (metadata);

-- Index on created_at for chronological queries
CREATE INDEX idx_symbol_identifiers_created_at ON symbol_identifiers(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE symbol_identifiers IS 'Maps canonical symbols to external provider identifiers';
COMMENT ON COLUMN symbol_identifiers.id IS 'Unique identifier mapping UUID';
COMMENT ON COLUMN symbol_identifiers.symbol_id IS 'Canonical symbol being mapped';
COMMENT ON COLUMN symbol_identifiers.source IS 'External provider (coingecko, coinmarketcap, tradingview, etc.)';
COMMENT ON COLUMN symbol_identifiers.external_id IS 'Provider-specific symbol identifier';
COMMENT ON COLUMN symbol_identifiers.is_primary IS 'Is this the primary/preferred identifier for this source?';
COMMENT ON COLUMN symbol_identifiers.metadata IS 'Additional provider-specific metadata (JSONB)';
COMMENT ON COLUMN symbol_identifiers.verified_at IS 'Timestamp when mapping was last verified';
