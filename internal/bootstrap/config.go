package bootstrap

import (
	"fmt"
	"os"

	"github.com/spf13/viper"
)

// Config holds bootstrap-specific configuration for external API clients
// and chain configuration.
type Config struct {
	// Coinbase API configuration for Top 100 assets
	Coinbase CoinbaseConfig `mapstructure:"coinbase"`

	// CoinGecko API configuration for asset details and deployments
	CoinGecko CoinGeckoConfig `mapstructure:"coingecko"`

	// CQAR gRPC endpoint configuration
	CQAR CQARConfig `mapstructure:"cqar"`

	// Chains to process during bootstrap
	Chains []string `mapstructure:"chains"`
}

// CoinbaseConfig holds Coinbase API client configuration
type CoinbaseConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"` // Optional
}

// CoinGeckoConfig holds CoinGecko API client configuration
type CoinGeckoConfig struct {
	BaseURL            string `mapstructure:"base_url"`
	APIKey             string `mapstructure:"api_key"`               // Required
	RateLimitPerSecond int    `mapstructure:"rate_limit_per_second"` // Default: 10
}

// CQARConfig holds CQAR gRPC client configuration
type CQARConfig struct {
	GRPCEndpoint string `mapstructure:"grpc_endpoint"`
}

// LoadConfig reads bootstrap configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	v.SetConfigFile(configPath)

	// Read environment variables with CQAR_BOOTSTRAP prefix
	v.SetEnvPrefix("CQAR_BOOTSTRAP")
	v.AutomaticEnv()

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found: %s", configPath)
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Apply defaults
	applyDefaults(&cfg)

	return &cfg, nil
}

// applyDefaults sets default values for optional configuration
func applyDefaults(cfg *Config) {
	// Coinbase defaults
	if cfg.Coinbase.BaseURL == "" {
		cfg.Coinbase.BaseURL = "https://api.coinbase.com"
	}

	// CoinGecko defaults
	if cfg.CoinGecko.BaseURL == "" {
		cfg.CoinGecko.BaseURL = "https://api.coingecko.com/api/v3"
	}
	if cfg.CoinGecko.RateLimitPerSecond == 0 {
		cfg.CoinGecko.RateLimitPerSecond = 10 // Free tier limit
	}

	// CQAR defaults
	if cfg.CQAR.GRPCEndpoint == "" {
		cfg.CQAR.GRPCEndpoint = "localhost:9090"
	}

	// Default chains to process
	if len(cfg.Chains) == 0 {
		cfg.Chains = []string{
			"ethereum",
			"polygon",
			"binance-smart-chain",
			"solana",
			"bitcoin",
			"arbitrum-one",
			"optimistic-ethereum",
		}
	}
}

// Validate checks that required configuration is present
func (c *Config) Validate() error {
	// CoinGecko API key is optional (free tier works without it)
	// Check environment variable if not set in config
	if c.CoinGecko.APIKey == "" {
		if key := os.Getenv("COINGECKO_API_KEY"); key != "" {
			c.CoinGecko.APIKey = key
		}
		// Note: Empty API key is acceptable - free tier has lower rate limits
	}

	// Validate CQAR endpoint
	if c.CQAR.GRPCEndpoint == "" {
		return fmt.Errorf("CQAR gRPC endpoint is required")
	}

	// Validate rate limit
	if c.CoinGecko.RateLimitPerSecond < 1 || c.CoinGecko.RateLimitPerSecond > 100 {
		return fmt.Errorf("CoinGecko rate limit must be between 1-100 requests/second, got: %d", c.CoinGecko.RateLimitPerSecond)
	}

	// Validate at least one chain configured
	if len(c.Chains) == 0 {
		return fmt.Errorf("at least one chain must be configured")
	}

	return nil
}
