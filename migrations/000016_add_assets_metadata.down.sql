-- Rollback assets metadata field

DROP INDEX IF EXISTS idx_assets_metadata;
ALTER TABLE assets DROP COLUMN IF EXISTS metadata;
