-- Align deployments table with CQC protobuf v0.3.0
-- Remove is_canonical, deployment_block, deployer_address (too specific)
-- Add chain_name, deployed_at, metadata fields
-- Rename contract_address to address

-- Add new fields from protobuf
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS chain_name VARCHAR(100);
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS deployed_at TIMESTAMPTZ;
ALTER TABLE deployments ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Rename contract_address to address (more generic for multi-chain)
ALTER TABLE deployments RENAME COLUMN contract_address TO address;

-- Drop fields removed from protobuf (no longer needed)
ALTER TABLE deployments DROP COLUMN IF EXISTS is_canonical;
ALTER TABLE deployments DROP COLUMN IF EXISTS deployment_block;
ALTER TABLE deployments DROP COLUMN IF EXISTS deployer_address;

-- Update comments
COMMENT ON COLUMN deployments.address IS 'Smart contract address or "native" for native blockchain tokens';
COMMENT ON COLUMN deployments.chain_name IS 'Human-readable chain name (e.g., "Ethereum", "Polygon")';
COMMENT ON COLUMN deployments.deployed_at IS 'Timestamp of when this deployment occurred';
COMMENT ON COLUMN deployments.metadata IS 'Additional deployment-specific metadata as JSONB';

-- Add GIN index on metadata JSONB
CREATE INDEX IF NOT EXISTS idx_deployments_metadata ON deployments USING GIN (metadata);

-- Add index on deployed_at for chronological queries
CREATE INDEX IF NOT EXISTS idx_deployments_deployed_at ON deployments(deployed_at DESC);
