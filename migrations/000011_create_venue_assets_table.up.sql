-- Create venue_assets table
-- Maps which assets are available on which venues with availability flags and fees
CREATE TABLE IF NOT EXISTS venue_assets (
    id UUID PRIMARY KEY,
    venue_id VARCHAR(100) NOT NULL REFERENCES venues(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    venue_symbol VARCHAR(50), -- Venue-specific asset symbol (e.g., "WBTC" on Binance might be "BTCB")
    deposit_enabled BOOLEAN NOT NULL DEFAULT true,
    withdraw_enabled BOOLEAN NOT NULL DEFAULT true,
    trading_enabled BOOLEAN NOT NULL DEFAULT true,
    withdraw_fee DECIMAL(30, 18), -- Withdrawal fee in asset units
    min_withdraw_amount DECIMAL(30, 18), -- Minimum withdrawal amount
    min_deposit_amount DECIMAL(30, 18), -- Minimum deposit amount
    network_confirmations SMALLINT, -- Required confirmations for deposits
    metadata JSONB, -- Additional venue-specific metadata
    listed_at TIMESTAMPTZ, -- When asset was listed on venue
    delisted_at TIMESTAMPTZ, -- When asset was delisted (NULL if still listed)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Ensure unique venue+asset combination (one asset appears once per venue)
    CONSTRAINT unique_venue_asset UNIQUE(venue_id, asset_id),
    -- If venue_symbol is provided, ensure it's unique per venue
    CONSTRAINT unique_venue_symbol_per_venue UNIQUE(venue_id, venue_symbol),
    -- Validate fees and amounts are non-negative
    CONSTRAINT chk_withdraw_fee_non_negative CHECK (withdraw_fee IS NULL OR withdraw_fee >= 0),
    CONSTRAINT chk_min_withdraw_non_negative CHECK (min_withdraw_amount IS NULL OR min_withdraw_amount >= 0),
    CONSTRAINT chk_min_deposit_non_negative CHECK (min_deposit_amount IS NULL OR min_deposit_amount >= 0),
    -- Validate network_confirmations is positive if set
    CONSTRAINT chk_confirmations_positive CHECK (network_confirmations IS NULL OR network_confirmations > 0),
    -- Delisted_at must be after listed_at
    CONSTRAINT chk_delist_after_list CHECK (delisted_at IS NULL OR listed_at IS NULL OR delisted_at >= listed_at)
);

-- Index on venue_id for "which assets are available on Binance?" queries
CREATE INDEX idx_venue_assets_venue_id ON venue_assets(venue_id);

-- Index on asset_id for "which venues trade BTC?" queries
CREATE INDEX idx_venue_assets_asset_id ON venue_assets(asset_id);

-- Composite index for venue+asset lookups (most common query pattern)
CREATE INDEX idx_venue_assets_venue_asset ON venue_assets(venue_id, asset_id);

-- Index on venue_symbol for reverse lookups
CREATE INDEX idx_venue_assets_venue_symbol ON venue_assets(venue_id, venue_symbol) WHERE venue_symbol IS NOT NULL;

-- Index on trading_enabled for filtering tradeable assets
CREATE INDEX idx_venue_assets_trading_enabled ON venue_assets(venue_id, trading_enabled) WHERE trading_enabled = true;

-- Index on deposit_enabled for filtering depositable assets
CREATE INDEX idx_venue_assets_deposit_enabled ON venue_assets(venue_id, deposit_enabled) WHERE deposit_enabled = true;

-- Index on withdraw_enabled for filtering withdrawable assets
CREATE INDEX idx_venue_assets_withdraw_enabled ON venue_assets(venue_id, withdraw_enabled) WHERE withdraw_enabled = true;

-- GIN index on metadata JSONB for flexible queries
CREATE INDEX idx_venue_assets_metadata ON venue_assets USING GIN (metadata);

-- Index on listed_at for chronological queries
CREATE INDEX idx_venue_assets_listed_at ON venue_assets(listed_at DESC) WHERE listed_at IS NOT NULL;

-- Index on delisted_at for finding delisted assets
CREATE INDEX idx_venue_assets_delisted_at ON venue_assets(delisted_at DESC) WHERE delisted_at IS NOT NULL;

-- Index on active listings (not delisted)
CREATE INDEX idx_venue_assets_active ON venue_assets(venue_id, asset_id) WHERE delisted_at IS NULL;

-- Comments for documentation
COMMENT ON TABLE venue_assets IS 'Maps asset availability on venues with deposit/withdraw/trading flags and fees';
COMMENT ON COLUMN venue_assets.id IS 'Unique venue-asset mapping UUID';
COMMENT ON COLUMN venue_assets.venue_id IS 'Venue where asset is available';
COMMENT ON COLUMN venue_assets.asset_id IS 'Asset available on the venue';
COMMENT ON COLUMN venue_assets.venue_symbol IS 'Venue-specific asset symbol (may differ from canonical symbol)';
COMMENT ON COLUMN venue_assets.deposit_enabled IS 'Whether deposits are enabled for this asset';
COMMENT ON COLUMN venue_assets.withdraw_enabled IS 'Whether withdrawals are enabled for this asset';
COMMENT ON COLUMN venue_assets.trading_enabled IS 'Whether trading is enabled for this asset';
COMMENT ON COLUMN venue_assets.withdraw_fee IS 'Withdrawal fee in asset units';
COMMENT ON COLUMN venue_assets.min_withdraw_amount IS 'Minimum withdrawal amount in asset units';
COMMENT ON COLUMN venue_assets.min_deposit_amount IS 'Minimum deposit amount in asset units';
COMMENT ON COLUMN venue_assets.network_confirmations IS 'Required network confirmations for deposits';
COMMENT ON COLUMN venue_assets.metadata IS 'Additional venue-specific metadata (JSONB)';
COMMENT ON COLUMN venue_assets.listed_at IS 'Timestamp when asset was listed on venue';
COMMENT ON COLUMN venue_assets.delisted_at IS 'Timestamp when asset was delisted (NULL if still listed)';
