-- Align symbols table with CQC protobuf v0.3.0
-- Add missing fields: symbol (human-readable), min_notional, contract_size, is_active, delisted_at, metadata

-- Add human-readable symbol representation (e.g., "BTC/USDT", "ETH-PERP")
ALTER TABLE symbols ADD COLUMN IF NOT EXISTS symbol VARCHAR(100);

-- Add min_notional field (minimum notional value: price * quantity)
ALTER TABLE symbols ADD COLUMN IF NOT EXISTS min_notional DECIMAL(30, 18);

-- Add contract_size field (contract multiplier for futures/options)
ALTER TABLE symbols ADD COLUMN IF NOT EXISTS contract_size DECIMAL(30, 18);

-- Add is_active flag (whether symbol is currently active and tradeable)
ALTER TABLE symbols ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Add delisted_at timestamp (when symbol was deactivated)
ALTER TABLE symbols ADD COLUMN IF NOT EXISTS delisted_at TIMESTAMPTZ;

-- Add metadata JSONB field
ALTER TABLE symbols ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_symbols_symbol ON symbols(symbol);
CREATE INDEX IF NOT EXISTS idx_symbols_is_active ON symbols(is_active);
CREATE INDEX IF NOT EXISTS idx_symbols_delisted_at ON symbols(delisted_at) WHERE delisted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_symbols_metadata ON symbols USING GIN (metadata);

-- Add comments
COMMENT ON COLUMN symbols.symbol IS 'Human-readable symbol representation (e.g., "BTC/USDT", "ETH-PERP", "BTC-25OCT24-30000-C")';
COMMENT ON COLUMN symbols.min_notional IS 'Minimum notional value (price * quantity) in quote currency';
COMMENT ON COLUMN symbols.contract_size IS 'Contract multiplier for futures/options (NULL for spot markets)';
COMMENT ON COLUMN symbols.is_active IS 'Whether this symbol is currently active and tradeable';
COMMENT ON COLUMN symbols.delisted_at IS 'Timestamp when this symbol was deactivated/delisted (NULL if still active)';
COMMENT ON COLUMN symbols.metadata IS 'Additional symbol-specific metadata as JSONB';
