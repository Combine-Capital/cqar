-- Create spot_instruments table (1:1 with instruments where type = 'SPOT')
CREATE TABLE IF NOT EXISTS spot_instruments (
    instrument_id UUID PRIMARY KEY REFERENCES instruments(id) ON DELETE CASCADE,
    base_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    quote_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    extensions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (base_asset_id != quote_asset_id)
);

-- Create indexes for lookups
CREATE INDEX idx_spot_instruments_base_asset ON spot_instruments(base_asset_id);
CREATE INDEX idx_spot_instruments_quote_asset ON spot_instruments(quote_asset_id);
CREATE INDEX idx_spot_instruments_pair ON spot_instruments(base_asset_id, quote_asset_id);
