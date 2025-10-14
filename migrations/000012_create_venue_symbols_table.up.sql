-- Create venue_symbols table
-- Maps canonical symbols to venue-specific symbol strings with fees and status
CREATE TABLE IF NOT EXISTS venue_symbols (
    id UUID PRIMARY KEY,
    venue_id VARCHAR(100) NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
    symbol_id UUID NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    venue_symbol VARCHAR(100) NOT NULL, -- Venue-specific symbol string (e.g., "BTCUSDT" for Binance)
    maker_fee DECIMAL(10, 6), -- Maker fee as percentage (e.g., 0.001 for 0.1%)
    taker_fee DECIMAL(10, 6), -- Taker fee as percentage (e.g., 0.001 for 0.1%)
    min_notional DECIMAL(30, 18), -- Minimum order notional value (quote currency)
    is_active BOOLEAN NOT NULL DEFAULT true,
    listed_at TIMESTAMPTZ, -- When symbol was listed on venue
    delisted_at TIMESTAMPTZ, -- When symbol was delisted (NULL if still listed)
    metadata JSONB, -- Additional venue-specific metadata (order types, leverage, etc.)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure unique venue+symbol combination (one symbol appears once per venue)
    CONSTRAINT unique_venue_symbol UNIQUE(venue_id, symbol_id),
    -- Ensure venue_symbol is unique per venue (venue-specific string must be unique)
    CONSTRAINT unique_venue_symbol_string UNIQUE(venue_id, venue_symbol),
    -- Validate fees are between 0 and 100% (1.0)
    CONSTRAINT chk_maker_fee_range CHECK (maker_fee IS NULL OR (maker_fee >= 0 AND maker_fee <= 1.0)),
    CONSTRAINT chk_taker_fee_range CHECK (taker_fee IS NULL OR (taker_fee >= 0 AND taker_fee <= 1.0)),
    -- Validate min_notional is non-negative
    CONSTRAINT chk_min_notional_non_negative CHECK (min_notional IS NULL OR min_notional >= 0),
    -- Delisted_at must be after listed_at
    CONSTRAINT chk_delist_after_list CHECK (delisted_at IS NULL OR listed_at IS NULL OR delisted_at >= listed_at),
    -- Validate venue_symbol not empty
    CONSTRAINT chk_venue_symbol_not_empty CHECK (LENGTH(TRIM(venue_symbol)) > 0)
);

-- Index on venue_id for "which symbols are on Binance?" queries
CREATE INDEX idx_venue_symbols_venue_id ON venue_symbols(venue_id);

-- Index on symbol_id for "which venues list BTC/USDT?" queries
CREATE INDEX idx_venue_symbols_symbol_id ON venue_symbols(symbol_id);

-- Composite index for venue+symbol lookups (most common query pattern)
CREATE INDEX idx_venue_symbols_venue_symbol_id ON venue_symbols(venue_id, symbol_id);

-- Index on venue_symbol for reverse lookups (THE CRITICAL CQMD USE CASE)
-- This enables fast GetVenueSymbol(venue_id="binance", venue_symbol="BTCUSDT") queries
CREATE INDEX idx_venue_symbols_venue_string ON venue_symbols(venue_id, venue_symbol);

-- Index on is_active for filtering active symbols
CREATE INDEX idx_venue_symbols_active ON venue_symbols(venue_id, is_active) WHERE is_active = true;

-- GIN index on metadata JSONB for flexible queries
CREATE INDEX idx_venue_symbols_metadata ON venue_symbols USING GIN (metadata);

-- Index on listed_at for chronological queries
CREATE INDEX idx_venue_symbols_listed_at ON venue_symbols(listed_at DESC) WHERE listed_at IS NOT NULL;

-- Index on delisted_at for finding delisted symbols
CREATE INDEX idx_venue_symbols_delisted_at ON venue_symbols(delisted_at DESC) WHERE delisted_at IS NOT NULL;

-- Index on active listings (not delisted)
CREATE INDEX idx_venue_symbols_active_listings ON venue_symbols(venue_id, symbol_id) WHERE delisted_at IS NULL;

-- Comments for documentation
COMMENT ON TABLE venue_symbols IS 'Maps canonical symbols to venue-specific symbol strings with fees and status';
COMMENT ON COLUMN venue_symbols.id IS 'Unique venue-symbol mapping UUID';
COMMENT ON COLUMN venue_symbols.venue_id IS 'Venue where symbol is listed';
COMMENT ON COLUMN venue_symbols.symbol_id IS 'Canonical symbol UUID';
COMMENT ON COLUMN venue_symbols.venue_symbol IS 'Venue-specific symbol string (e.g., "BTCUSDT" for Binance)';
COMMENT ON COLUMN venue_symbols.maker_fee IS 'Maker fee as decimal percentage (e.g., 0.001 = 0.1%)';
COMMENT ON COLUMN venue_symbols.taker_fee IS 'Taker fee as decimal percentage (e.g., 0.001 = 0.1%)';
COMMENT ON COLUMN venue_symbols.min_notional IS 'Minimum order notional value in quote currency units';
COMMENT ON COLUMN venue_symbols.is_active IS 'Whether symbol is currently tradeable on venue';
COMMENT ON COLUMN venue_symbols.listed_at IS 'Timestamp when symbol was listed on venue';
COMMENT ON COLUMN venue_symbols.delisted_at IS 'Timestamp when symbol was delisted (NULL if still listed)';
COMMENT ON COLUMN venue_symbols.metadata IS 'Additional venue-specific metadata (JSONB): order types, leverage limits, etc.';
