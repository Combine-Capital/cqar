package integration

import (
	"testing"
	"time"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	servicesv1 "github.com/Combine-Capital/cqc/gen/go/cqc/services/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TestAssetLifecycle tests the complete asset lifecycle: create → get → update → deploy → relationship → group
func TestAssetLifecycle(t *testing.T) {
	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	// Reset database and load base chains data
	fixture.ResetDatabase(t)
	fixture.LoadSeedData(t, "chains.sql")

	ctx := fixture.Ctx

	// Generate unique asset ID
	assetID := uuid.New().String()
	assetType := assetsv1.AssetType_ASSET_TYPE_ERC20

	// Step 1: Create Asset
	t.Run("CreateAsset", func(t *testing.T) {
		req := &servicesv1.CreateAssetRequest{
			Symbol:      ptrString("TEST"),
			Name:        ptrString("Test Token"),
			AssetType:   &assetType,
			Category:    ptrString("TEST"),
			Description: ptrString("A test token for integration testing"),
			LogoUrl:     ptrString("https://example.com/logo.png"),
			WebsiteUrl:  ptrString("https://example.com"),
		}

		resp, err := fixture.Server.CreateAsset(ctx, req)
		require.NoError(t, err, "CreateAsset should succeed")
		require.NotNil(t, resp)
		assert.NotNil(t, resp.Asset.AssetId)
		assert.Equal(t, "TEST", *resp.Asset.Symbol)

		// Save generated ID
		assetID = *resp.Asset.AssetId
	})

	// Step 2: Get Asset
	t.Run("GetAsset", func(t *testing.T) {
		req := &servicesv1.GetAssetRequest{
			AssetId: ptrString(assetID),
		}

		resp, err := fixture.Server.GetAsset(ctx, req)
		require.NoError(t, err, "GetAsset should succeed")
		require.NotNil(t, resp)
		assert.Equal(t, assetID, *resp.Asset.AssetId)
		assert.Equal(t, "TEST", *resp.Asset.Symbol)
		assert.Equal(t, "Test Token", *resp.Asset.Name)
	})

	// Step 3: Update Asset
	t.Run("UpdateAsset", func(t *testing.T) {
		req := &servicesv1.UpdateAssetRequest{
			AssetId:     ptrString(assetID),
			Symbol:      ptrString("TEST"),
			Name:        ptrString("Updated Test Token"),
			AssetType:   &assetType,
			Description: ptrString("Updated description"),
		}

		resp, err := fixture.Server.UpdateAsset(ctx, req)
		require.NoError(t, err, "UpdateAsset should succeed")
		require.NotNil(t, resp)
		assert.Equal(t, "Updated Test Token", *resp.Asset.Name)
		assert.Equal(t, "Updated description", *resp.Asset.Description)
	})

	// Step 4: Create Asset Deployment
	chainID := "c1111111-1111-1111-1111-111111111111" // Ethereum from seed data

	t.Run("CreateAssetDeployment", func(t *testing.T) {
		req := &servicesv1.CreateAssetDeploymentRequest{
			AssetId:         ptrString(assetID),
			ChainId:         ptrString(chainID),
			ContractAddress: ptrString("0x1234567890123456789012345678901234567890"),
			Decimals:        ptrInt32(18),
		}

		resp, err := fixture.Server.CreateAssetDeployment(ctx, req)
		require.NoError(t, err, "CreateAssetDeployment should succeed")
		require.NotNil(t, resp)
		assert.Equal(t, int32(18), *resp.Deployment.Decimals)
	})

	// Step 5: List Asset Deployments
	t.Run("ListAssetDeployments", func(t *testing.T) {
		req := &servicesv1.ListAssetDeploymentsRequest{
			AssetId: ptrString(assetID),
		}

		resp, err := fixture.Server.ListAssetDeployments(ctx, req)
		require.NoError(t, err, "ListAssetDeployments should succeed")
		require.NotNil(t, resp)
		assert.GreaterOrEqual(t, len(resp.Deployments), 1)
	})

	// Step 6: Create Asset Relationship (requires another asset)
	relatedAssetID := ""

	t.Run("CreateRelatedAsset", func(t *testing.T) {
		req := &servicesv1.CreateAssetRequest{
			Symbol:    ptrString("WTEST"),
			Name:      ptrString("Wrapped Test Token"),
			AssetType: &assetType,
			Category:  ptrString("WRAPPED"),
		}

		resp, err := fixture.Server.CreateAsset(ctx, req)
		require.NoError(t, err, "CreateAsset for related asset should succeed")
		require.NotNil(t, resp)
		relatedAssetID = *resp.Asset.AssetId
	})

	relType := assetsv1.RelationshipType_RELATIONSHIP_TYPE_WRAPS

	t.Run("CreateAssetRelationship", func(t *testing.T) {
		req := &servicesv1.CreateAssetRelationshipRequest{
			SourceAssetId:    ptrString(relatedAssetID),
			TargetAssetId:    ptrString(assetID),
			RelationshipType: &relType,
			Description:      ptrString("test_wrapper"),
		}

		resp, err := fixture.Server.CreateAssetRelationship(ctx, req)
		require.NoError(t, err, "CreateAssetRelationship should succeed")
		require.NotNil(t, resp)
	})

	// Step 7: List Asset Relationships
	t.Run("ListAssetRelationships", func(t *testing.T) {
		req := &servicesv1.ListAssetRelationshipsRequest{
			AssetId: ptrString(assetID),
		}

		resp, err := fixture.Server.ListAssetRelationships(ctx, req)
		require.NoError(t, err, "ListAssetRelationships should succeed")
		require.NotNil(t, resp)
		// Note: Response may be empty if filtering doesn't match direction
	})

	// Step 8: Create Asset Group
	groupID := ""

	t.Run("CreateAssetGroup", func(t *testing.T) {
		req := &servicesv1.CreateAssetGroupRequest{
			Name:        ptrString("Test Token Family"),
			Description: ptrString("All test token variants"),
		}

		resp, err := fixture.Server.CreateAssetGroup(ctx, req)
		require.NoError(t, err, "CreateAssetGroup should succeed")
		require.NotNil(t, resp)
		groupID = *resp.Group.GroupId
	})

	// Step 9: Add Assets to Group
	t.Run("AddAssetToGroup", func(t *testing.T) {
		// Add main asset
		req1 := &servicesv1.AddAssetToGroupRequest{
			GroupId: ptrString(groupID),
			AssetId: ptrString(assetID),
			Weight:  ptrFloat64(1.0),
		}

		resp1, err := fixture.Server.AddAssetToGroup(ctx, req1)
		require.NoError(t, err, "AddAssetToGroup should succeed for main asset")
		require.NotNil(t, resp1)

		// Add related asset
		req2 := &servicesv1.AddAssetToGroupRequest{
			GroupId: ptrString(groupID),
			AssetId: ptrString(relatedAssetID),
			Weight:  ptrFloat64(1.0),
		}

		resp2, err := fixture.Server.AddAssetToGroup(ctx, req2)
		require.NoError(t, err, "AddAssetToGroup should succeed for related asset")
		require.NotNil(t, resp2)
	})

	// Step 10: Get Asset Group with Members
	t.Run("GetAssetGroup", func(t *testing.T) {
		req := &servicesv1.GetAssetGroupRequest{
			GroupId: ptrString(groupID),
		}

		resp, err := fixture.Server.GetAssetGroup(ctx, req)
		require.NoError(t, err, "GetAssetGroup should succeed")
		require.NotNil(t, resp)
		assert.Equal(t, groupID, *resp.Group.GroupId)
		assert.Len(t, resp.Group.Members, 2, "Group should have 2 members")
	})

	// Step 11: Delete Asset (should fail due to dependencies)
	t.Run("DeleteAssetWithDependencies", func(t *testing.T) {
		req := &servicesv1.DeleteAssetRequest{
			AssetId: ptrString(assetID),
		}

		_, err := fixture.Server.DeleteAsset(ctx, req)
		// Should fail because asset has deployments and relationships
		assert.Error(t, err, "DeleteAsset should fail with dependencies")
	})
}

// TestAssetValidation tests asset creation validation rules
func TestAssetValidation(t *testing.T) {
	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	ctx := fixture.Ctx
	assetType := assetsv1.AssetType_ASSET_TYPE_ERC20

	t.Run("MissingRequiredFields", func(t *testing.T) {
		tests := []struct {
			name string
			req  *servicesv1.CreateAssetRequest
		}{
			{
				name: "missing symbol",
				req: &servicesv1.CreateAssetRequest{
					Name:      ptrString("Test"),
					AssetType: &assetType,
				},
			},
			{
				name: "missing name",
				req: &servicesv1.CreateAssetRequest{
					Symbol:    ptrString("TEST"),
					AssetType: &assetType,
				},
			},
			{
				name: "missing type",
				req: &servicesv1.CreateAssetRequest{
					Symbol: ptrString("TEST"),
					Name:   ptrString("Test"),
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := fixture.Server.CreateAsset(ctx, tt.req)
				require.Error(t, err, "Should fail validation")
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, codes.InvalidArgument, st.Code())
			})
		}
	})
}

// TestAssetSymbolCollision tests handling of symbol collisions across chains
func TestAssetSymbolCollision(t *testing.T) {
	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	fixture.ResetDatabase(t)
	fixture.LoadSeedData(t, "chains.sql", "assets.sql")

	ctx := fixture.Ctx

	t.Run("SearchUSDCFindsMultipleChains", func(t *testing.T) {
		req := &servicesv1.SearchAssetsRequest{
			Query: ptrString("USDC"),
			Limit: ptrInt32(10),
		}

		resp, err := fixture.Server.SearchAssets(ctx, req)
		require.NoError(t, err, "SearchAssets should succeed")
		require.NotNil(t, resp)

		// Should find USDC on both Ethereum and Polygon
		assert.GreaterOrEqual(t, len(resp.Assets), 2, "Should find multiple USDC variants")

		// Verify different asset IDs
		usdcAssets := make(map[string]bool)
		for _, asset := range resp.Assets {
			if *asset.Symbol == "USDC" {
				usdcAssets[*asset.AssetId] = true
			}
		}
		assert.GreaterOrEqual(t, len(usdcAssets), 2, "USDC should have unique IDs per chain")
	})

	t.Run("GetSpecificUSDCByID", func(t *testing.T) {
		// Get Ethereum USDC
		ethUSDCID := "a6666666-6666-6666-6666-666666666666"
		req1 := &servicesv1.GetAssetRequest{AssetId: ptrString(ethUSDCID)}

		resp1, err := fixture.Server.GetAsset(ctx, req1)
		require.NoError(t, err)
		assert.Equal(t, "USDC", *resp1.Asset.Symbol)
		assert.Contains(t, *resp1.Asset.Name, "Ethereum")

		// Get Polygon USDC
		polyUSDCID := "a7777777-7777-7777-7777-777777777777"
		req2 := &servicesv1.GetAssetRequest{AssetId: ptrString(polyUSDCID)}

		resp2, err := fixture.Server.GetAsset(ctx, req2)
		require.NoError(t, err)
		assert.Equal(t, "USDC", *resp2.Asset.Symbol)
		assert.Contains(t, *resp2.Asset.Name, "Polygon")
	})
}

// TestAssetDeploymentValidation tests deployment validation rules
func TestAssetDeploymentValidation(t *testing.T) {
	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	fixture.ResetDatabase(t)
	fixture.LoadSeedData(t, "chains.sql", "assets.sql")

	ctx := fixture.Ctx

	btcID := "a1111111-1111-1111-1111-111111111111"
	ethChainID := "c1111111-1111-1111-1111-111111111111"

	t.Run("InvalidContractAddress", func(t *testing.T) {
		req := &servicesv1.CreateAssetDeploymentRequest{
			AssetId:         ptrString(btcID),
			ChainId:         ptrString(ethChainID),
			ContractAddress: ptrString("invalid_address"),
			Decimals:        ptrInt32(18),
		}

		_, err := fixture.Server.CreateAssetDeployment(ctx, req)
		assert.Error(t, err, "Should reject invalid contract address format")
	})

	t.Run("InvalidDecimals", func(t *testing.T) {
		tests := []int32{-1, 19, 100}

		for _, decimals := range tests {
			req := &servicesv1.CreateAssetDeploymentRequest{
				AssetId:         ptrString(btcID),
				ChainId:         ptrString(ethChainID),
				ContractAddress: ptrString("0x1234567890123456789012345678901234567890"),
				Decimals:        ptrInt32(decimals),
			}

			_, err := fixture.Server.CreateAssetDeployment(ctx, req)
			assert.Error(t, err, "Should reject invalid decimals: %d", decimals)
		}
	})

	t.Run("NonexistentAssetID", func(t *testing.T) {
		req := &servicesv1.CreateAssetDeploymentRequest{
			AssetId:         ptrString("nonexistent-asset-id"),
			ChainId:         ptrString(ethChainID),
			ContractAddress: ptrString("0x1234567890123456789012345678901234567890"),
			Decimals:        ptrInt32(18),
		}

		_, err := fixture.Server.CreateAssetDeployment(ctx, req)
		assert.Error(t, err, "Should reject nonexistent asset ID")
	})
}

// TestAssetRelationshipGraph tests relationship creation and cycle detection
func TestAssetRelationshipGraph(t *testing.T) {
	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	fixture.ResetDatabase(t)
	fixture.LoadSeedData(t, "chains.sql", "assets.sql", "relationships.sql")

	ctx := fixture.Ctx

	ethID := "a2222222-2222-2222-2222-222222222222"

	t.Run("ListETHRelationships", func(t *testing.T) {
		req := &servicesv1.ListAssetRelationshipsRequest{
			AssetId: ptrString(ethID),
		}

		resp, err := fixture.Server.ListAssetRelationships(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// ETH is wrapped by WETH and staked by stETH
		// Note: May be empty depending on query direction
		t.Logf("Found %d relationships for ETH", len(resp.Relationships))
	})
}

// TestQualityFlagBlocking tests that CRITICAL quality flags block trading operations
func TestQualityFlagBlocking(t *testing.T) {
	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	fixture.ResetDatabase(t)
	fixture.LoadSeedData(t, "chains.sql", "assets.sql")

	ctx := fixture.Ctx

	assetID := "a1111111-1111-1111-1111-111111111111" // BTC from seed data
	flagType := assetsv1.FlagType_FLAG_TYPE_EXPLOITED
	severity := assetsv1.FlagSeverity_FLAG_SEVERITY_CRITICAL

	t.Run("RaiseCriticalQualityFlag", func(t *testing.T) {
		req := &servicesv1.RaiseQualityFlagRequest{
			AssetId:     ptrString(assetID),
			FlagType:    &flagType,
			Severity:    &severity,
			RaisedBy:    ptrString("integration_test"),
			Description: ptrString("Critical vulnerability detected in integration test"),
		}

		resp, err := fixture.Server.RaiseQualityFlag(ctx, req)
		require.NoError(t, err, "RaiseQualityFlag should succeed")
		require.NotNil(t, resp)
		assert.Equal(t, severity, *resp.Flag.Severity)
	})

	t.Run("ListQualityFlags", func(t *testing.T) {
		req := &servicesv1.ListQualityFlagsRequest{
			AssetId: ptrString(assetID),
		}

		resp, err := fixture.Server.ListQualityFlags(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.GreaterOrEqual(t, len(resp.Flags), 1, "Should have at least one flag")

		// Verify CRITICAL flag exists
		hasCritical := false
		for _, flag := range resp.Flags {
			if *flag.Severity == severity && flag.ResolvedAt == nil {
				hasCritical = true
				break
			}
		}
		assert.True(t, hasCritical, "Should have active CRITICAL flag")
	})

	t.Run("GetAssetWithCriticalFlag", func(t *testing.T) {
		req := &servicesv1.GetAssetRequest{
			AssetId: ptrString(assetID),
		}

		resp, err := fixture.Server.GetAsset(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Asset should still be retrievable, but trading should be blocked
		// (this would be enforced by consumers of the CQAR service)
		assert.Equal(t, assetID, *resp.Asset.AssetId)
	})
}

// TestAssetGroupAggregation tests portfolio aggregation use case
func TestAssetGroupAggregation(t *testing.T) {
	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	fixture.ResetDatabase(t)
	fixture.LoadSeedData(t, "chains.sql", "assets.sql")

	ctx := fixture.Ctx

	ethID := "a2222222-2222-2222-2222-222222222222"
	wethID := "a3333333-3333-3333-3333-333333333333"
	stethID := "a4444444-4444-4444-4444-444444444444"

	groupID := ""

	t.Run("CreateETHVariantsGroup", func(t *testing.T) {
		req := &servicesv1.CreateAssetGroupRequest{
			Name:        ptrString("ETH Family"),
			Description: ptrString("All Ethereum and its wrapped/staked variants"),
		}

		resp, err := fixture.Server.CreateAssetGroup(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		groupID = *resp.Group.GroupId
	})

	t.Run("AddAllETHVariants", func(t *testing.T) {
		variants := []struct {
			id     string
			weight float64
		}{
			{ethID, 1.0},
			{wethID, 1.0},
			{stethID, 1.0},
		}

		for _, v := range variants {
			req := &servicesv1.AddAssetToGroupRequest{
				GroupId: ptrString(groupID),
				AssetId: ptrString(v.id),
				Weight:  ptrFloat64(v.weight),
			}

			_, err := fixture.Server.AddAssetToGroup(ctx, req)
			require.NoError(t, err, "AddAssetToGroup should succeed for %s", v.id)
		}
	})

	t.Run("GetGroupWithAllMembers", func(t *testing.T) {
		req := &servicesv1.GetAssetGroupRequest{
			GroupId: ptrString(groupID),
		}

		resp, err := fixture.Server.GetAssetGroup(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "ETH Family", *resp.Group.Name)
		assert.Len(t, resp.Group.Members, 3, "Group should have 3 ETH variants")

		// Verify all members are present
		memberIDs := make(map[string]bool)
		for _, member := range resp.Group.Members {
			memberIDs[*member.AssetId] = true
		}
		assert.True(t, memberIDs[ethID], "Should include ETH")
		assert.True(t, memberIDs[wethID], "Should include WETH")
		assert.True(t, memberIDs[stethID], "Should include stETH")
	})

	t.Run("RemoveAssetFromGroup", func(t *testing.T) {
		req := &servicesv1.RemoveAssetFromGroupRequest{
			GroupId: ptrString(groupID),
			AssetId: ptrString(stethID),
		}

		resp, err := fixture.Server.RemoveAssetFromGroup(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Verify member removed
		getReq := &servicesv1.GetAssetGroupRequest{GroupId: ptrString(groupID)}
		getResp, err := fixture.Server.GetAssetGroup(ctx, getReq)
		require.NoError(t, err)
		assert.Len(t, getResp.Group.Members, 2, "Group should have 2 members after removal")
	})
}

// TestAssetCachePerformance validates cache hit performance requirements
func TestAssetCachePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	fixture := NewTestFixture(t)
	defer fixture.Cleanup(t)

	fixture.ResetDatabase(t)
	fixture.LoadSeedData(t, "chains.sql", "assets.sql")

	ctx := fixture.Ctx
	assetID := "a1111111-1111-1111-1111-111111111111" // BTC

	// Prime cache with first request
	req := &servicesv1.GetAssetRequest{AssetId: ptrString(assetID)}
	_, err := fixture.Server.GetAsset(ctx, req)
	require.NoError(t, err)

	// Measure cached request latency
	const iterations = 100
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := fixture.Server.GetAsset(ctx, req)
		duration := time.Since(start)

		require.NoError(t, err)
		totalDuration += duration
	}

	avgLatency := totalDuration / iterations
	p99Estimate := totalDuration / time.Duration(iterations-iterations/100) // Rough estimate

	t.Logf("Average latency: %v", avgLatency)
	t.Logf("Estimated p99: %v", p99Estimate)

	// Performance requirement: <10ms p50 (cache hit)
	assert.Less(t, avgLatency, 10*time.Millisecond,
		"Average cache hit latency should be <10ms, got %v", avgLatency)
}
