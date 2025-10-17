package integration

import (
	"testing"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSmokeTest verifies basic service functionality with the new Instrument/Market architecture
func TestSmokeTest(t *testing.T) {
	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	// Reset database (no seed data needed)
	fixture.ResetDatabase(t)

	ctx := fixture.Ctx

	t.Run("CreateAsset", func(t *testing.T) {
		assetType := assetsv1.AssetType_ASSET_TYPE_ERC20

		req := &servicesv1.CreateAssetRequest{
			Symbol:      ptrString("BTC"),
			Name:        ptrString("Bitcoin"),
			AssetType:   &assetType,
			Category:    ptrString("CRYPTO"),
			Description: ptrString("The first cryptocurrency"),
		}

		resp, err := fixture.Server.CreateAsset(ctx, req)
		require.NoError(t, err, "CreateAsset should succeed")
		require.NotNil(t, resp)
		require.NotNil(t, resp.Asset)
		assert.NotNil(t, resp.Asset.AssetId)
		assert.Equal(t, "BTC", *resp.Asset.Symbol)
		assert.Equal(t, "Bitcoin", *resp.Asset.Name)
	})

	t.Run("GetAsset", func(t *testing.T) {
		// First create an asset
		assetType := assetsv1.AssetType_ASSET_TYPE_ERC20
		createReq := &servicesv1.CreateAssetRequest{
			Symbol:    ptrString("ETH"),
			Name:      ptrString("Ethereum"),
			AssetType: &assetType,
			Category:  ptrString("CRYPTO"),
		}

		createResp, err := fixture.Server.CreateAsset(ctx, createReq)
		require.NoError(t, err)
		require.NotNil(t, createResp.Asset.AssetId)

		// Now retrieve it
		getReq := &servicesv1.GetAssetRequest{
			AssetId: createResp.Asset.AssetId,
		}

		getResp, err := fixture.Server.GetAsset(ctx, getReq)
		require.NoError(t, err, "GetAsset should succeed")
		require.NotNil(t, getResp)
		require.NotNil(t, getResp.Asset)
		assert.Equal(t, *createResp.Asset.AssetId, *getResp.Asset.AssetId)
		assert.Equal(t, "ETH", *getResp.Asset.Symbol)
	})

	t.Run("ServiceHealthCheck", func(t *testing.T) {
		// If we got here, the service is running and connected successfully
		assert.NotNil(t, fixture.Server, "gRPC client should be initialized")
		assert.NotNil(t, fixture.Repository, "Repository should be initialized")
		assert.NotNil(t, fixture.AssetManager, "AssetManager should be initialized")
		assert.NotNil(t, fixture.InstrumentManager, "InstrumentManager should be initialized")
		assert.NotNil(t, fixture.MarketManager, "MarketManager should be initialized")
		assert.NotNil(t, fixture.VenueManager, "VenueManager should be initialized")
	})
}
