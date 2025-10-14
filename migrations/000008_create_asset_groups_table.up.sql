-- Create asset_groups table
-- Groups of related assets for portfolio aggregation (e.g., "all_eth_variants")
CREATE TABLE IF NOT EXISTS asset_groups (
    id UUID PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Validate name format (lowercase with underscores)
    CONSTRAINT chk_group_name_format CHECK (name ~ '^[a-z0-9_]+$')
);

-- Create group_members table
-- Many-to-many relationship between assets and groups with optional weights
CREATE TABLE IF NOT EXISTS group_members (
    id UUID PRIMARY KEY,
    group_id UUID NOT NULL REFERENCES asset_groups(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    weight DECIMAL(10, 6) NOT NULL DEFAULT 1.0, -- Weight for aggregation (e.g., stETH might have 0.98 weight vs ETH)
    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure asset can only be in a group once
    CONSTRAINT unique_group_asset UNIQUE(group_id, asset_id),
    -- Validate weight is positive
    CONSTRAINT chk_weight_positive CHECK (weight > 0)
);

-- Indexes for asset_groups
CREATE INDEX idx_asset_groups_name ON asset_groups(name);
CREATE INDEX idx_asset_groups_created_at ON asset_groups(created_at DESC);

-- Indexes for group_members
CREATE INDEX idx_group_members_group_id ON group_members(group_id);
CREATE INDEX idx_group_members_asset_id ON group_members(asset_id);
CREATE INDEX idx_group_members_added_at ON group_members(added_at DESC);

-- Comments for documentation
COMMENT ON TABLE asset_groups IS 'Named groups of related assets for portfolio aggregation';
COMMENT ON COLUMN asset_groups.id IS 'Unique group identifier (UUID)';
COMMENT ON COLUMN asset_groups.name IS 'Unique group name (lowercase_with_underscores, e.g., "all_eth_variants")';
COMMENT ON COLUMN asset_groups.description IS 'Human-readable description of the group purpose';

COMMENT ON TABLE group_members IS 'Membership records linking assets to groups with optional weights';
COMMENT ON COLUMN group_members.id IS 'Unique membership identifier (UUID)';
COMMENT ON COLUMN group_members.group_id IS 'Asset group this membership belongs to';
COMMENT ON COLUMN group_members.asset_id IS 'Asset that is a member of the group';
COMMENT ON COLUMN group_members.weight IS 'Weight for aggregation (default 1.0, can be adjusted for derivatives)';
COMMENT ON COLUMN group_members.added_at IS 'Timestamp when asset was added to group';
