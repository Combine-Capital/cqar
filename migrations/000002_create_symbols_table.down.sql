-- Drop symbols table and all associated indexes
DROP INDEX IF EXISTS idx_symbols_created_at;
DROP INDEX IF EXISTS idx_symbols_expiry;
DROP INDEX IF EXISTS idx_symbols_base_quote;
DROP INDEX IF EXISTS idx_symbols_type;
DROP INDEX IF EXISTS idx_symbols_quote_asset;
DROP INDEX IF EXISTS idx_symbols_base_asset;
DROP TABLE IF EXISTS symbols CASCADE;
