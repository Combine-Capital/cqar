-- Create quality_flags table
-- Tracks quality and security issues with assets (scams, exploits, etc.)
CREATE TABLE IF NOT EXISTS quality_flags (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    flag_type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    source VARCHAR(100) NOT NULL, -- Who raised the flag (e.g., "manual", "automated_scanner", "community")
    reason TEXT NOT NULL, -- Explanation of the flag
    evidence_url TEXT, -- Link to evidence (report, transaction, announcement)
    raised_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ, -- NULL if still active, timestamp when resolved
    resolved_by VARCHAR(100), -- Who resolved the flag
    resolution_notes TEXT, -- Explanation of resolution
    -- Validate flag_type enum
    CONSTRAINT chk_flag_type CHECK (flag_type IN (
        'SCAM',           -- Known scam token
        'RUGPULL',        -- Rug pull detected
        'EXPLOITED',      -- Contract exploited/hacked
        'DEPRECATED',     -- Token deprecated/migrated
        'SUSPICIOUS',     -- Suspicious activity detected
        'UNLICENSED',     -- Operating without proper licenses
        'SANCTIONS',      -- Subject to sanctions
        'LOW_LIQUIDITY',  -- Dangerously low liquidity
        'HIGH_SLIPPAGE'   -- Excessive slippage detected
    )),
    -- Validate severity enum
    CONSTRAINT chk_severity CHECK (severity IN (
        'INFO',           -- Informational only
        'WARNING',        -- User caution advised
        'CRITICAL'        -- Trading should be blocked
    )),
    -- If resolved, must have resolver and resolution notes
    CONSTRAINT chk_resolution_complete CHECK (
        (resolved_at IS NULL AND resolved_by IS NULL AND resolution_notes IS NULL) OR
        (resolved_at IS NOT NULL AND resolved_by IS NOT NULL AND resolution_notes IS NOT NULL)
    ),
    -- Resolved_at must be after raised_at
    CONSTRAINT chk_resolution_after_raise CHECK (resolved_at IS NULL OR resolved_at >= raised_at)
);

-- Index on asset_id for "what flags does this asset have?" queries
CREATE INDEX idx_quality_flags_asset_id ON quality_flags(asset_id);

-- Index on flag_type for filtering by flag category
CREATE INDEX idx_quality_flags_type ON quality_flags(flag_type);

-- Index on severity for critical flag queries
CREATE INDEX idx_quality_flags_severity ON quality_flags(severity);

-- Critical index: active critical flags (most important query for trading)
CREATE INDEX idx_quality_flags_active_critical ON quality_flags(asset_id, severity) 
    WHERE resolved_at IS NULL AND severity = 'CRITICAL';

-- Index on active flags (unresolved)
CREATE INDEX idx_quality_flags_active ON quality_flags(asset_id) WHERE resolved_at IS NULL;

-- Index on raised_at for chronological queries
CREATE INDEX idx_quality_flags_raised_at ON quality_flags(raised_at DESC);

-- Index on resolved_at for tracking resolutions
CREATE INDEX idx_quality_flags_resolved_at ON quality_flags(resolved_at DESC) WHERE resolved_at IS NOT NULL;

-- Index on source for "which flags came from automated scanner?" queries
CREATE INDEX idx_quality_flags_source ON quality_flags(source);

-- Comments for documentation
COMMENT ON TABLE quality_flags IS 'Quality and security flags for assets (scams, exploits, deprecations)';
COMMENT ON COLUMN quality_flags.id IS 'Unique quality flag identifier (UUID)';
COMMENT ON COLUMN quality_flags.asset_id IS 'Asset this flag applies to';
COMMENT ON COLUMN quality_flags.flag_type IS 'Type of flag: SCAM, RUGPULL, EXPLOITED, DEPRECATED, SUSPICIOUS, etc.';
COMMENT ON COLUMN quality_flags.severity IS 'Flag severity: INFO, WARNING, CRITICAL';
COMMENT ON COLUMN quality_flags.source IS 'Source of the flag (manual, automated, community)';
COMMENT ON COLUMN quality_flags.reason IS 'Detailed explanation of why flag was raised';
COMMENT ON COLUMN quality_flags.evidence_url IS 'Link to supporting evidence (report, transaction hash, announcement)';
COMMENT ON COLUMN quality_flags.raised_at IS 'Timestamp when flag was raised';
COMMENT ON COLUMN quality_flags.resolved_at IS 'Timestamp when flag was resolved (NULL if active)';
COMMENT ON COLUMN quality_flags.resolved_by IS 'Who resolved the flag';
COMMENT ON COLUMN quality_flags.resolution_notes IS 'Explanation of how/why flag was resolved';
