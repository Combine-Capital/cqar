-- Create future_contracts table (1:1 with instruments where type = 'FUTURE')
CREATE TABLE IF NOT EXISTS future_contracts (
    instrument_id UUID PRIMARY KEY REFERENCES instruments(id) ON DELETE CASCADE,
    underlying_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    expiry TIMESTAMPTZ NOT NULL,
    is_inverse BOOLEAN NOT NULL DEFAULT FALSE,
    is_quanto BOOLEAN NOT NULL DEFAULT FALSE,
    contract_multiplier NUMERIC(40, 20) NOT NULL DEFAULT 1.0 CHECK (contract_multiplier > 0),
    extensions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for underlying asset and expiry lookups
CREATE INDEX idx_future_contracts_underlying ON future_contracts(underlying_asset_id);
CREATE INDEX idx_future_contracts_expiry ON future_contracts(expiry);
CREATE INDEX idx_future_contracts_underlying_expiry ON future_contracts(underlying_asset_id, expiry);
