-- Create deployments table
-- Tracks multi-chain deployments of assets (e.g., USDC on Ethereum, Polygon, Arbitrum)
CREATE TABLE IF NOT EXISTS deployments (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    chain_id VARCHAR(50) NOT NULL REFERENCES chains(id) ON DELETE RESTRICT,
    contract_address VARCHAR(255) NOT NULL,
    decimals SMALLINT NOT NULL,
    is_canonical BOOLEAN NOT NULL DEFAULT false,
    deployment_block BIGINT, -- Block number when contract was deployed
    deployer_address VARCHAR(255), -- Address that deployed the contract
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure unique asset+chain combination
    CONSTRAINT unique_asset_chain_deployment UNIQUE(asset_id, chain_id),
    -- Validate decimals range (0-18 is typical for ERC20, but allow up to 77 for edge cases)
    CONSTRAINT chk_decimals_range CHECK (decimals >= 0 AND decimals <= 77),
    -- Validate contract_address format (non-empty)
    CONSTRAINT chk_contract_address_not_empty CHECK (LENGTH(TRIM(contract_address)) > 0)
);

-- Partial unique index: only one canonical deployment per asset
CREATE UNIQUE INDEX unique_canonical_asset ON deployments(asset_id) WHERE is_canonical = true;

-- Index on asset_id for "which chains has this asset deployed on?" queries
CREATE INDEX idx_deployments_asset_id ON deployments(asset_id);

-- Index on chain_id for "which assets are deployed on this chain?" queries
CREATE INDEX idx_deployments_chain_id ON deployments(chain_id);

-- Index on is_canonical for finding canonical deployments
CREATE INDEX idx_deployments_canonical ON deployments(is_canonical) WHERE is_canonical = true;

-- Composite index for asset+chain lookups (most common query pattern)
CREATE INDEX idx_deployments_asset_chain ON deployments(asset_id, chain_id);

-- Index on contract_address for reverse lookups from on-chain data
CREATE INDEX idx_deployments_contract_address ON deployments(contract_address);

-- Index on created_at for chronological queries
CREATE INDEX idx_deployments_created_at ON deployments(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE deployments IS 'Multi-chain asset deployments with contract addresses and metadata';
COMMENT ON COLUMN deployments.id IS 'Unique deployment identifier (UUID)';
COMMENT ON COLUMN deployments.asset_id IS 'Canonical asset this deployment represents';
COMMENT ON COLUMN deployments.chain_id IS 'Blockchain where asset is deployed';
COMMENT ON COLUMN deployments.contract_address IS 'Smart contract address (or native token identifier)';
COMMENT ON COLUMN deployments.decimals IS 'Token decimals (0-77, typically 0-18)';
COMMENT ON COLUMN deployments.is_canonical IS 'Primary/official deployment for this asset';
COMMENT ON COLUMN deployments.deployment_block IS 'Block number when contract was deployed';
COMMENT ON COLUMN deployments.deployer_address IS 'Address that deployed the contract';
