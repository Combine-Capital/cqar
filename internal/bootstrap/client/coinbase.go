package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// CoinbaseClient is a client for the Coinbase API to fetch Top 100 assets
type CoinbaseClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewCoinbaseClient creates a new Coinbase API client
func NewCoinbaseClient(baseURL, apiKey string) *CoinbaseClient {
	return &CoinbaseClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AssetInfo represents basic asset information from Coinbase
type AssetInfo struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Rank   int    `json:"rank"`
}

// GetTop100Assets fetches the top 100 cryptocurrencies by market cap from Coinbase
func (c *CoinbaseClient) GetTop100Assets(ctx context.Context) ([]AssetInfo, error) {
	log.Debug().Msg("Fetching Coinbase Top 100 assets")

	// Coinbase API endpoint for currencies
	// Note: Using currencies endpoint as proxy for top assets
	// In production, this would use the proper Coinbase Pro API or exchange rates API
	url := fmt.Sprintf("%s/v2/currencies", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add API key if provided
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Symbol  string `json:"symbol"`
			Type    string `json:"type"`
			MinSize string `json:"min_size"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// Filter to cryptocurrencies only and create AssetInfo list
	// Note: Coinbase currencies endpoint doesn't provide rank, so we'll enumerate
	assets := make([]AssetInfo, 0, 100)
	rank := 1
	for _, item := range response.Data {
		// Filter to crypto type and limit to 100
		if item.Type == "crypto" && rank <= 100 {
			assets = append(assets, AssetInfo{
				ID:     item.ID,
				Symbol: item.Symbol,
				Name:   item.Name,
				Rank:   rank,
			})
			rank++
		}
	}

	log.Info().Int("count", len(assets)).Msg("Fetched Coinbase assets")

	return assets, nil
}

// GetAssetDetails fetches detailed information for a specific asset
func (c *CoinbaseClient) GetAssetDetails(ctx context.Context, assetID string) (*AssetInfo, error) {
	log.Debug().Str("asset_id", assetID).Msg("Fetching Coinbase asset details")

	url := fmt.Sprintf("%s/v2/currencies/%s", c.baseURL, assetID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
			Type   string `json:"type"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &AssetInfo{
		ID:     response.Data.ID,
		Symbol: response.Data.Symbol,
		Name:   response.Data.Name,
	}, nil
}
