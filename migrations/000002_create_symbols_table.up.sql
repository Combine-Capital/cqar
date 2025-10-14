-- Create symbols table
-- Trading pairs/markets with base/quote assets and market specifications
CREATE TABLE IF NOT EXISTS symbols (
    id UUID PRIMARY KEY,
    base_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    quote_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    settlement_asset_id UUID REFERENCES assets(id) ON DELETE RESTRICT,
    symbol_type VARCHAR(50) NOT NULL, -- SPOT, PERPETUAL, FUTURE, OPTION, MARGIN
    tick_size DECIMAL(30, 18) NOT NULL,
    lot_size DECIMAL(30, 18) NOT NULL,
    min_order_size DECIMAL(30, 18) NOT NULL,
    max_order_size DECIMAL(30, 18) NOT NULL,
    -- Option-specific fields (nullable for non-option symbols)
    strike_price DECIMAL(30, 18),
    expiry TIMESTAMPTZ,
    option_type VARCHAR(10), -- CALL, PUT
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure unique combination of base/quote/type/strike/expiry
    CONSTRAINT unique_symbol UNIQUE(base_asset_id, quote_asset_id, symbol_type, strike_price, expiry),
    -- Validate tick_size and lot_size are positive
    CONSTRAINT chk_tick_size_positive CHECK (tick_size > 0),
    CONSTRAINT chk_lot_size_positive CHECK (lot_size > 0),
    -- Validate min < max order size
    CONSTRAINT chk_order_size_range CHECK (min_order_size <= max_order_size),
    -- Validate option fields: if option_type is set, strike_price and expiry must be set
    CONSTRAINT chk_option_fields CHECK (
        (symbol_type != 'OPTION') OR 
        (strike_price IS NOT NULL AND expiry IS NOT NULL AND option_type IS NOT NULL)
    ),
    -- Validate option_type enum
    CONSTRAINT chk_option_type CHECK (option_type IS NULL OR option_type IN ('CALL', 'PUT'))
);

-- Index on base_asset_id for "which symbols have BTC as base?" queries
CREATE INDEX idx_symbols_base_asset ON symbols(base_asset_id);

-- Index on quote_asset_id for "which symbols quote in USDT?" queries
CREATE INDEX idx_symbols_quote_asset ON symbols(quote_asset_id);

-- Index on symbol_type for filtering by market type
CREATE INDEX idx_symbols_type ON symbols(symbol_type);

-- Composite index for base+quote lookups (common query pattern)
CREATE INDEX idx_symbols_base_quote ON symbols(base_asset_id, quote_asset_id);

-- Index on expiry for option symbols (find expiring options)
CREATE INDEX idx_symbols_expiry ON symbols(expiry) WHERE expiry IS NOT NULL;

-- Index on created_at for chronological queries
CREATE INDEX idx_symbols_created_at ON symbols(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE symbols IS 'Canonical trading pairs/markets with unique UUIDs and market specifications';
COMMENT ON COLUMN symbols.id IS 'Unique canonical symbol identifier (UUID)';
COMMENT ON COLUMN symbols.base_asset_id IS 'Base asset of the trading pair (e.g., BTC in BTC/USDT)';
COMMENT ON COLUMN symbols.quote_asset_id IS 'Quote asset of the trading pair (e.g., USDT in BTC/USDT)';
COMMENT ON COLUMN symbols.settlement_asset_id IS 'Settlement asset for derivatives (often same as quote)';
COMMENT ON COLUMN symbols.symbol_type IS 'Market type: SPOT, PERPETUAL, FUTURE, OPTION, MARGIN';
COMMENT ON COLUMN symbols.tick_size IS 'Minimum price increment';
COMMENT ON COLUMN symbols.lot_size IS 'Minimum quantity increment';
COMMENT ON COLUMN symbols.min_order_size IS 'Minimum order size in base asset units';
COMMENT ON COLUMN symbols.max_order_size IS 'Maximum order size in base asset units';
COMMENT ON COLUMN symbols.strike_price IS 'Strike price for option contracts (options only)';
COMMENT ON COLUMN symbols.expiry IS 'Expiry timestamp for derivatives (futures/options)';
COMMENT ON COLUMN symbols.option_type IS 'Option type: CALL or PUT (options only)';
