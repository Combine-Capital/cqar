-- Create lending_borrows table (1:1 with instruments where type = 'LENDING_BORROW')
CREATE TABLE IF NOT EXISTS lending_borrows (
    instrument_id UUID PRIMARY KEY REFERENCES instruments(id) ON DELETE CASCADE,
    underlying_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    extensions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for underlying asset lookups
CREATE INDEX idx_lending_borrows_underlying ON lending_borrows(underlying_asset_id);
