-- Create lending_deposits table (1:1 with instruments where type = 'LENDING_DEPOSIT')
CREATE TABLE IF NOT EXISTS lending_deposits (
    instrument_id UUID PRIMARY KEY REFERENCES instruments(id) ON DELETE CASCADE,
    underlying_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    extensions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for underlying asset lookups
CREATE INDEX idx_lending_deposits_underlying ON lending_deposits(underlying_asset_id);
