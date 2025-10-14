-- Drop venues table and all associated indexes
DROP INDEX IF EXISTS idx_venues_active;
DROP INDEX IF EXISTS idx_venues_chain_id;
DROP INDEX IF EXISTS idx_venues_type;
DROP TABLE IF EXISTS venues CASCADE;
