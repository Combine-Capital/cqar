-- Drop assets table and all associated indexes
DROP INDEX IF EXISTS idx_assets_created_at;
DROP INDEX IF EXISTS idx_assets_type;
DROP INDEX IF EXISTS idx_assets_symbol;
DROP TABLE IF EXISTS assets CASCADE;
