-- Rollback alignment of chains table with CQC protobuf v0.3.0

-- Drop indexes
DROP INDEX IF EXISTS idx_chains_metadata;
DROP INDEX IF EXISTS idx_chains_is_testnet;

-- Remove fields added in up migration
ALTER TABLE chains DROP COLUMN IF EXISTS metadata;
ALTER TABLE chains DROP COLUMN IF EXISTS is_testnet;
ALTER TABLE chains DROP COLUMN IF EXISTS network_id;
