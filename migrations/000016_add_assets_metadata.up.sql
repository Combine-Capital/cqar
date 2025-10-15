-- Align assets table with CQC protobuf v0.3.0
-- Add metadata JSONB field

-- Add metadata field
ALTER TABLE assets ADD COLUMN IF NOT EXISTS metadata JSONB;

-- Add GIN index on metadata for flexible queries
CREATE INDEX IF NOT EXISTS idx_assets_metadata ON assets USING GIN (metadata);

-- Add comment
COMMENT ON COLUMN assets.metadata IS 'Additional asset-specific metadata as structured JSONB data';
