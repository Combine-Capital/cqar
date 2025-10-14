-- Drop chains table and all associated indexes
DROP INDEX IF EXISTS idx_chains_native_asset;
DROP INDEX IF EXISTS idx_chains_type;
DROP TABLE IF EXISTS chains CASCADE;
