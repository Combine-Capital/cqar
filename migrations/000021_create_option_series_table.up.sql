-- Create option_series table (1:1 with instruments where type = 'OPTION')
CREATE TABLE IF NOT EXISTS option_series (
    instrument_id UUID PRIMARY KEY REFERENCES instruments(id) ON DELETE CASCADE,
    underlying_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    expiry TIMESTAMPTZ NOT NULL,
    strike_price NUMERIC(40, 20) NOT NULL CHECK (strike_price > 0),
    option_type VARCHAR(10) NOT NULL CHECK (option_type IN ('CALL', 'PUT')),
    exercise_style VARCHAR(20) NOT NULL CHECK (exercise_style IN ('european', 'american')),
    contract_multiplier NUMERIC(40, 20) NOT NULL DEFAULT 1.0 CHECK (contract_multiplier > 0),
    extensions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for lookups
CREATE INDEX idx_option_series_underlying ON option_series(underlying_asset_id);
CREATE INDEX idx_option_series_expiry ON option_series(expiry);
CREATE INDEX idx_option_series_strike ON option_series(strike_price);
CREATE INDEX idx_option_series_type ON option_series(option_type);
CREATE INDEX idx_option_series_composite ON option_series(underlying_asset_id, expiry, strike_price, option_type);
