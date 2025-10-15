-- Rollback alignment of symbols table with CQC protobuf v0.3.0

-- Drop indexes
DROP INDEX IF EXISTS idx_symbols_metadata;
DROP INDEX IF EXISTS idx_symbols_delisted_at;
DROP INDEX IF EXISTS idx_symbols_is_active;
DROP INDEX IF EXISTS idx_symbols_symbol;

-- Remove added columns
ALTER TABLE symbols DROP COLUMN IF EXISTS metadata;
ALTER TABLE symbols DROP COLUMN IF EXISTS delisted_at;
ALTER TABLE symbols DROP COLUMN IF EXISTS is_active;
ALTER TABLE symbols DROP COLUMN IF EXISTS contract_size;
ALTER TABLE symbols DROP COLUMN IF EXISTS min_notional;
ALTER TABLE symbols DROP COLUMN IF EXISTS symbol;
