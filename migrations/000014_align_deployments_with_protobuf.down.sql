-- Rollback alignment of deployments table with CQC protobuf v0.3.0

-- Drop indexes
DROP INDEX IF EXISTS idx_deployments_deployed_at;
DROP INDEX IF EXISTS idx_deployments_metadata;

-- Restore old columns
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS deployer_address VARCHAR(255);
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS deployment_block BIGINT;
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS is_canonical BOOLEAN DEFAULT false;

-- Rename address back to contract_address
ALTER TABLE deployments RENAME COLUMN address TO contract_address;

-- Remove new columns
ALTER TABLE deployments DROP COLUMN IF EXISTS metadata;
ALTER TABLE deployments DROP COLUMN IF EXISTS deployed_at;
ALTER TABLE deployments DROP COLUMN IF EXISTS chain_name;
