-- Create instruments table (base for all tradeable products)
CREATE TABLE IF NOT EXISTS instruments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    instrument_type VARCHAR(50) NOT NULL CHECK (instrument_type IN ('SPOT', 'PERPETUAL', 'FUTURE', 'OPTION', 'LENDING_DEPOSIT', 'LENDING_BORROW')),
    code VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(code)
);

-- Create index on instrument_type for filtering
CREATE INDEX idx_instruments_type ON instruments(instrument_type);
CREATE INDEX idx_instruments_code ON instruments(code);
