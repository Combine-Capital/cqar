package client

import (
	"context"
	"fmt"
	"time"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// CQARClient is a wrapper around the CQAR gRPC client for bootstrap operations
type CQARClient struct {
	conn   *grpc.ClientConn
	client servicesv1.AssetRegistryClient
}

// NewCQARClient creates a new CQAR gRPC client connection
func NewCQARClient(ctx context.Context, endpoint string) (*CQARClient, error) {
	log.Debug().Str("endpoint", endpoint).Msg("Connecting to CQAR gRPC service")

	// Create gRPC connection with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(), // Wait for connection
	)
	if err != nil {
		return nil, fmt.Errorf("dial CQAR gRPC: %w", err)
	}

	client := servicesv1.NewAssetRegistryClient(conn)

	log.Info().Str("endpoint", endpoint).Msg("Connected to CQAR gRPC service")

	return &CQARClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close closes the gRPC connection
func (c *CQARClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// derefString safely dereferences a string pointer, returning empty string if nil
func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}

// CreateAsset creates a new asset in CQAR
func (c *CQARClient) CreateAsset(ctx context.Context, req *servicesv1.CreateAssetRequest) (*assetsv1.Asset, error) {
	symbol := ""
	if req.Symbol != nil {
		symbol = *req.Symbol
	}
	name := ""
	if req.Name != nil {
		name = *req.Name
	}

	log.Debug().
		Str("symbol", symbol).
		Str("name", name).
		Msg("Creating asset via CQAR gRPC")

	resp, err := c.client.CreateAsset(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create asset: %w", err)
	}

	assetID := ""
	if resp.Asset.AssetId != nil {
		assetID = *resp.Asset.AssetId
	}
	assetSymbol := ""
	if resp.Asset.Symbol != nil {
		assetSymbol = *resp.Asset.Symbol
	}

	log.Info().
		Str("id", assetID).
		Str("symbol", assetSymbol).
		Msg("Asset created successfully")

	return resp.Asset, nil
}

// GetAsset retrieves an asset by ID
func (c *CQARClient) GetAsset(ctx context.Context, assetID string) (*assetsv1.Asset, error) {
	log.Debug().Str("asset_id", assetID).Msg("Getting asset via CQAR gRPC")

	resp, err := c.client.GetAsset(ctx, &servicesv1.GetAssetRequest{
		AssetId: &assetID,
	})
	if err != nil {
		return nil, fmt.Errorf("get asset: %w", err)
	}

	return resp.Asset, nil
}

// SearchAssets searches for assets by query string
func (c *CQARClient) SearchAssets(ctx context.Context, query string) ([]*assetsv1.Asset, error) {
	log.Debug().Str("query", query).Msg("Searching assets via CQAR gRPC")

	resp, err := c.client.SearchAssets(ctx, &servicesv1.SearchAssetsRequest{
		Query: &query,
	})
	if err != nil {
		return nil, fmt.Errorf("search assets: %w", err)
	}

	return resp.Assets, nil
}

// CreateChain creates a new chain in CQAR
func (c *CQARClient) CreateChain(ctx context.Context, req *servicesv1.CreateChainRequest) (*assetsv1.Chain, error) {
	log.Debug().
		Str("id", req.GetChainType()).
		Str("name", req.GetName()).
		Msg("Creating chain via CQAR gRPC")

	resp, err := c.client.CreateChain(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create chain: %w", err)
	}

	log.Info().
		Str("id", derefString(resp.Chain.ChainId)).
		Str("name", derefString(resp.Chain.ChainName)).
		Msg("Chain created successfully")

	return resp.Chain, nil
}

// GetChain retrieves a chain by ID
func (c *CQARClient) GetChain(ctx context.Context, chainID string) (*assetsv1.Chain, error) {
	log.Debug().Str("chain_id", chainID).Msg("Getting chain via CQAR gRPC")

	resp, err := c.client.GetChain(ctx, &servicesv1.GetChainRequest{
		ChainId: &chainID,
	})
	if err != nil {
		return nil, fmt.Errorf("get chain: %w", err)
	}

	return resp.Chain, nil
}

// ListChains lists all chains
func (c *CQARClient) ListChains(ctx context.Context) ([]*assetsv1.Chain, error) {
	log.Debug().Msg("Listing chains via CQAR gRPC")

	resp, err := c.client.ListChains(ctx, &servicesv1.ListChainsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list chains: %w", err)
	}

	return resp.Chains, nil
}

// CreateAssetDeployment creates a new asset deployment on a chain
func (c *CQARClient) CreateAssetDeployment(ctx context.Context, req *servicesv1.CreateAssetDeploymentRequest) (*assetsv1.AssetDeployment, error) {
	log.Debug().
		Str("asset_id", derefString(req.AssetId)).
		Str("chain_id", derefString(req.ChainId)).
		Str("contract_address", derefString(req.ContractAddress)).
		Msg("Creating asset deployment via CQAR gRPC")

	resp, err := c.client.CreateAssetDeployment(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create asset deployment: %w", err)
	}

	log.Info().
		Str("asset_id", derefString(resp.Deployment.AssetId)).
		Str("chain_id", derefString(resp.Deployment.ChainId)).
		Msg("Asset deployment created successfully")

	return resp.Deployment, nil
}

// ListAssetDeployments lists deployments for an asset
func (c *CQARClient) ListAssetDeployments(ctx context.Context, assetID string) ([]*assetsv1.AssetDeployment, error) {
	log.Debug().Str("asset_id", assetID).Msg("Listing asset deployments via CQAR gRPC")

	resp, err := c.client.ListAssetDeployments(ctx, &servicesv1.ListAssetDeploymentsRequest{
		AssetId: &assetID,
	})
	if err != nil {
		return nil, fmt.Errorf("list asset deployments: %w", err)
	}

	return resp.Deployments, nil
}

// CreateAssetIdentifier creates an external identifier mapping for an asset
func (c *CQARClient) CreateAssetIdentifier(ctx context.Context, req *servicesv1.CreateAssetIdentifierRequest) error {
	log.Debug().
		Str("asset_id", derefString(req.AssetId)).
		Str("source", req.Source.String()).
		Str("external_id", derefString(req.ExternalId)).
		Msg("Creating asset identifier via CQAR gRPC")

	_, err := c.client.CreateAssetIdentifier(ctx, req)
	if err != nil {
		return fmt.Errorf("create asset identifier: %w", err)
	}

	log.Info().
		Str("asset_id", derefString(req.AssetId)).
		Str("source", req.Source.String()).
		Msg("Asset identifier created successfully")

	return nil
}
