-- Align chains table with CQC protobuf v0.3.0
-- Remove rpc_urls (moved to separate service)
-- Add network_id, is_testnet, metadata fields

-- Add new fields from protobuf
ALTER TABLE chains ADD COLUMN IF NOT EXISTS network_id BIGINT;
ALTER TABLE chains ADD COLUMN IF NOT EXISTS is_testnet BOOLEAN DEFAULT false;
ALTER TABLE chains ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Add index on is_testnet for filtering
CREATE INDEX IF NOT EXISTS idx_chains_is_testnet ON chains(is_testnet);

-- Add GIN index on metadata JSONB for flexible queries
CREATE INDEX IF NOT EXISTS idx_chains_metadata ON chains USING GIN (metadata);

-- Comments for new fields
COMMENT ON COLUMN chains.network_id IS 'Network ID for transaction signing (e.g., 1 for Ethereum mainnet, 137 for Polygon) - primarily for EVM chains';
COMMENT ON COLUMN chains.is_testnet IS 'Whether this is a testnet (true) or mainnet (false)';
COMMENT ON COLUMN chains.metadata IS 'Additional chain-specific metadata as JSONB';
