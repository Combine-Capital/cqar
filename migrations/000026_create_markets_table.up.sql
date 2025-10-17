-- Create markets table (venue-specific listings of instruments)
CREATE TABLE IF NOT EXISTS markets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument_id UUID NOT NULL REFERENCES instruments(id) ON DELETE CASCADE,
    venue_id VARCHAR(100) NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
    venue_symbol VARCHAR(255) NOT NULL,
    settlement_asset_id UUID REFERENCES assets(id) ON DELETE RESTRICT,
    price_currency_asset_id UUID REFERENCES assets(id) ON DELETE RESTRICT,
    tick_size NUMERIC(40, 20) CHECK (tick_size IS NULL OR tick_size > 0),
    lot_size NUMERIC(40, 20) CHECK (lot_size IS NULL OR lot_size > 0),
    min_order_size NUMERIC(40, 20) CHECK (min_order_size IS NULL OR min_order_size > 0),
    max_order_size NUMERIC(40, 20) CHECK (max_order_size IS NULL OR max_order_size > 0),
    min_notional NUMERIC(40, 20) CHECK (min_notional IS NULL OR min_notional > 0),
    maker_fee NUMERIC(10, 8),
    taker_fee NUMERIC(10, 8),
    funding_interval_secs INTEGER CHECK (funding_interval_secs IS NULL OR funding_interval_secs > 0),
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'delisted')),
    listed_at TIMESTAMPTZ,
    delisted_at TIMESTAMPTZ,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(venue_id, venue_symbol),
    CHECK (delisted_at IS NULL OR delisted_at > listed_at)
);

-- Create indexes for lookups
CREATE INDEX idx_markets_instrument ON markets(instrument_id);
CREATE INDEX idx_markets_venue ON markets(venue_id);
CREATE INDEX idx_markets_venue_symbol ON markets(venue_id, venue_symbol);
CREATE INDEX idx_markets_settlement_asset ON markets(settlement_asset_id);
CREATE INDEX idx_markets_price_currency_asset ON markets(price_currency_asset_id);
CREATE INDEX idx_markets_status ON markets(status);
