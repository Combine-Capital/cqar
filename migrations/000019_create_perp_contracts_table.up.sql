-- Create perp_contracts table (1:1 with instruments where type = 'PERPETUAL')
CREATE TABLE IF NOT EXISTS perp_contracts (
    instrument_id UUID PRIMARY KEY REFERENCES instruments(id) ON DELETE CASCADE,
    underlying_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE RESTRICT,
    is_inverse BOOLEAN NOT NULL DEFAULT FALSE,
    is_quanto BOOLEAN NOT NULL DEFAULT FALSE,
    contract_multiplier NUMERIC(40, 20) NOT NULL DEFAULT 1.0 CHECK (contract_multiplier > 0),
    extensions JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for underlying asset lookups
CREATE INDEX idx_perp_contracts_underlying ON perp_contracts(underlying_asset_id);
