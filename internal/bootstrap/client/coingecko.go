package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// CoinGeckoClient is a client for the CoinGecko API with rate limiting
type CoinGeckoClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// NewCoinGeckoClient creates a new CoinGecko API client with rate limiting
func NewCoinGeckoClient(baseURL, apiKey string, rateLimit int) *CoinGeckoClient {
	return &CoinGeckoClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: rate.NewLimiter(rate.Limit(rateLimit), 1), // requests per second
	}
}

// AssetDetails represents detailed asset information from CoinGecko
type AssetDetails struct {
	ID          string            `json:"id"`
	Symbol      string            `json:"symbol"`
	Name        string            `json:"name"`
	Description map[string]string `json:"description"`
	Image       AssetImage        `json:"image"`
	Links       AssetLinks        `json:"links"`
	Platforms   map[string]string `json:"platforms"` // chain -> contract_address
	MarketData  AssetMarketData   `json:"market_data"`
}

// AssetImage contains asset logo URLs
type AssetImage struct {
	Thumb string `json:"thumb"`
	Small string `json:"small"`
	Large string `json:"large"`
}

// AssetLinks contains asset-related links
type AssetLinks struct {
	Homepage   []string `json:"homepage"`
	Blockchain []string `json:"blockchain_site"` // Changed to array to match API response
}

// AssetMarketData contains market-related data
type AssetMarketData struct {
	MarketCapRank int `json:"market_cap_rank"`
}

// PlatformInfo represents contract deployment information per chain
type PlatformInfo struct {
	ChainID         string
	ContractAddress string
	Decimals        int32
}

// GetAssetByID fetches detailed asset information by CoinGecko ID
func (c *CoinGeckoClient) GetAssetByID(ctx context.Context, coinGeckoID string) (*AssetDetails, error) {
	// Apply rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	log.Debug().Str("coingecko_id", coinGeckoID).Msg("Fetching CoinGecko asset details")

	url := fmt.Sprintf("%s/coins/%s", c.baseURL, coinGeckoID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add API key header if provided
	if c.apiKey != "" {
		req.Header.Set("x-cg-pro-api-key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limit exceeded (HTTP 429)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var details AssetDetails
	if err := json.NewDecoder(resp.Body).Decode(&details); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &details, nil
}

// GetAssetBySymbol searches for an asset by symbol and returns detailed information
func (c *CoinGeckoClient) GetAssetBySymbol(ctx context.Context, symbol string) (*AssetDetails, error) {
	// First, search for the asset by symbol to get the CoinGecko ID
	coinGeckoID, err := c.searchAssetID(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("search asset by symbol: %w", err)
	}

	// Then fetch full details
	return c.GetAssetByID(ctx, coinGeckoID)
}

// searchAssetID searches for an asset ID by symbol
func (c *CoinGeckoClient) searchAssetID(ctx context.Context, symbol string) (string, error) {
	// Apply rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limiter: %w", err)
	}

	log.Debug().Str("symbol", symbol).Msg("Searching CoinGecko for asset by symbol")

	url := fmt.Sprintf("%s/search?query=%s", c.baseURL, strings.ToLower(symbol))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("x-cg-pro-api-key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var searchResult struct {
		Coins []struct {
			ID     string `json:"id"`
			Symbol string `json:"symbol"`
			Name   string `json:"name"`
		} `json:"coins"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	// Find exact symbol match (case-insensitive)
	symbolLower := strings.ToLower(symbol)
	for _, coin := range searchResult.Coins {
		if strings.ToLower(coin.Symbol) == symbolLower {
			return coin.ID, nil
		}
	}

	return "", fmt.Errorf("asset not found for symbol: %s", symbol)
}

// GetPlatformInfo extracts platform deployment information from asset details
func GetPlatformInfo(details *AssetDetails) []PlatformInfo {
	platforms := make([]PlatformInfo, 0, len(details.Platforms))

	for chainID, contractAddress := range details.Platforms {
		// Skip empty contract addresses
		if contractAddress == "" {
			continue
		}

		// Some CoinGecko responses include platform prefix (e.g., "ethereum:0x...")
		// Strip the prefix if present
		if idx := strings.Index(contractAddress, ":"); idx != -1 {
			contractAddress = contractAddress[idx+1:]
		}

		platforms = append(platforms, PlatformInfo{
			ChainID:         chainID,
			ContractAddress: contractAddress,
			Decimals:        18, // Default to 18, actual value must be fetched from contract
		})
	}

	return platforms
}

// GetTop100Assets fetches CoinGecko's top 100 cryptocurrencies by market cap
func (c *CoinGeckoClient) GetTop100Assets(ctx context.Context) ([]AssetDetails, error) {
	// Apply rate limiting
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	log.Debug().Msg("Fetching CoinGecko Top 100 assets")

	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1&sparkline=false", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("x-cg-pro-api-key", c.apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("rate limit exceeded (HTTP 429)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var markets []struct {
		ID            string `json:"id"`
		Symbol        string `json:"symbol"`
		Name          string `json:"name"`
		Image         string `json:"image"`
		MarketCapRank int    `json:"market_cap_rank"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&markets); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Convert to AssetDetails
	assets := make([]AssetDetails, 0, len(markets))
	for _, market := range markets {
		assets = append(assets, AssetDetails{
			ID:     market.ID,
			Symbol: market.Symbol,
			Name:   market.Name,
			Image: AssetImage{
				Large: market.Image,
			},
			MarketData: AssetMarketData{
				MarketCapRank: market.MarketCapRank,
			},
		})
	}

	log.Info().Int("count", len(assets)).Msg("Fetched CoinGecko Top 100 assets")

	return assets, nil
}
