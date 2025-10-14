package config

import (
	"time"

	cqiconfig "github.com/Combine-Capital/cqi/pkg/config"
)

// Config extends CQI base configuration with CQAR-specific cache TTL settings
type Config struct {
	cqiconfig.Config `mapstructure:",squash"` // Embed CQI config

	// CQAR-specific cache TTLs
	CacheTTL CacheTTLConfig `mapstructure:"cache_ttl"`
}

// CacheTTLConfig contains CQAR-specific cache TTL settings per entity type
type CacheTTLConfig struct {
	Asset       time.Duration `mapstructure:"asset"`        // Default: 60m
	Symbol      time.Duration `mapstructure:"symbol"`       // Default: 60m
	Venue       time.Duration `mapstructure:"venue"`        // Default: 60m
	VenueAsset  time.Duration `mapstructure:"venue_asset"`  // Default: 15m
	VenueSymbol time.Duration `mapstructure:"venue_symbol"` // Default: 15m
	QualityFlag time.Duration `mapstructure:"quality_flag"` // Default: 5m
}

// Load reads configuration from file and environment variables
// Uses CQI's config.Load function and applies CQAR-specific defaults
func Load(configPath, envPrefix string) (*Config, error) {
	// Load base CQI configuration
	baseCfg, err := cqiconfig.Load(configPath, envPrefix)
	if err != nil {
		return nil, err
	}

	// Wrap in CQAR config
	cfg := &Config{
		Config: *baseCfg,
	}

	// Apply CQAR-specific defaults
	applyDefaults(cfg)

	return cfg, nil
}

// MustLoad loads configuration and panics on error.
// This is useful in main() where configuration errors should be fatal.
func MustLoad(configPath, envPrefix string) *Config {
	cfg, err := Load(configPath, envPrefix)
	if err != nil {
		panic(err)
	}
	return cfg
}

// applyDefaults applies CQAR-specific default values
func applyDefaults(cfg *Config) {
	// Cache TTL defaults
	if cfg.CacheTTL.Asset == 0 {
		cfg.CacheTTL.Asset = 60 * time.Minute
	}
	if cfg.CacheTTL.Symbol == 0 {
		cfg.CacheTTL.Symbol = 60 * time.Minute
	}
	if cfg.CacheTTL.Venue == 0 {
		cfg.CacheTTL.Venue = 60 * time.Minute
	}
	if cfg.CacheTTL.VenueAsset == 0 {
		cfg.CacheTTL.VenueAsset = 15 * time.Minute
	}
	if cfg.CacheTTL.VenueSymbol == 0 {
		cfg.CacheTTL.VenueSymbol = 15 * time.Minute
	}
	if cfg.CacheTTL.QualityFlag == 0 {
		cfg.CacheTTL.QualityFlag = 5 * time.Minute
	}
}
