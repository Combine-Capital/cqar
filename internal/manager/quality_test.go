package manager

import (
	"context"
	"testing"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestQualityManager_RaiseQualityFlag_Validation(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	mgr := NewQualityManager(repo)

	// Setup: Create an asset
	asset := &assetsv1.Asset{
		AssetId:   strPtr("scamcoin"),
		Symbol:    strPtr("SCAM"),
		Name:      strPtr("Scam Coin"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_ERC20),
	}
	repo.assets["scamcoin"] = asset

	flagType := assetsv1.FlagType_FLAG_TYPE_SCAM
	severity := assetsv1.FlagSeverity_FLAG_SEVERITY_CRITICAL

	tests := []struct {
		name        string
		flag        *assetsv1.AssetQualityFlag
		wantErr     bool
		wantErrCode codes.Code
	}{
		{
			name: "valid quality flag",
			flag: &assetsv1.AssetQualityFlag{
				FlagId:   strPtr("flag1"),
				AssetId:  strPtr("scamcoin"),
				FlagType: &flagType,
				Severity: &severity,
				Source:   strPtr("manual_review"),
				Reason:   strPtr("Contract has hidden mint function"),
			},
			wantErr: false,
		},
		{
			name: "missing flag type",
			flag: &assetsv1.AssetQualityFlag{
				FlagId:   strPtr("flag2"),
				AssetId:  strPtr("scamcoin"),
				Severity: &severity,
				Source:   strPtr("manual_review"),
				Reason:   strPtr("Some reason"),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing severity",
			flag: &assetsv1.AssetQualityFlag{
				FlagId:   strPtr("flag3"),
				AssetId:  strPtr("scamcoin"),
				FlagType: &flagType,
				Source:   strPtr("manual_review"),
				Reason:   strPtr("Some reason"),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing source",
			flag: &assetsv1.AssetQualityFlag{
				FlagId:   strPtr("flag4"),
				AssetId:  strPtr("scamcoin"),
				FlagType: &flagType,
				Severity: &severity,
				Reason:   strPtr("Some reason"),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing reason",
			flag: &assetsv1.AssetQualityFlag{
				FlagId:   strPtr("flag5"),
				AssetId:  strPtr("scamcoin"),
				FlagType: &flagType,
				Severity: &severity,
				Source:   strPtr("manual_review"),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "missing asset_id",
			flag: &assetsv1.AssetQualityFlag{
				FlagId:   strPtr("flag6"),
				FlagType: &flagType,
				Severity: &severity,
				Source:   strPtr("manual_review"),
				Reason:   strPtr("Some reason"),
			},
			wantErr:     true,
			wantErrCode: codes.InvalidArgument,
		},
		{
			name: "nonexistent asset",
			flag: &assetsv1.AssetQualityFlag{
				FlagId:   strPtr("flag7"),
				AssetId:  strPtr("nonexistent"),
				FlagType: &flagType,
				Severity: &severity,
				Source:   strPtr("manual_review"),
				Reason:   strPtr("Some reason"),
			},
			wantErr:     true,
			wantErrCode: codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.RaiseQualityFlag(ctx, tt.flag)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestQualityManager_IsAssetTradeable_CRITICAL(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	mgr := NewQualityManager(repo)

	// Setup: Create an asset
	asset := &assetsv1.Asset{
		AssetId:   strPtr("btc"),
		Symbol:    strPtr("BTC"),
		Name:      strPtr("Bitcoin"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_NATIVE),
	}
	repo.assets["btc"] = asset

	// Test 1: Asset with no flags should be tradeable
	tradeable, err := mgr.IsAssetTradeable(ctx, "btc")
	require.NoError(t, err)
	assert.True(t, tradeable, "Asset with no flags should be tradeable")

	// Test 2: Add INFO severity flag - should still be tradeable
	infoFlagType := assetsv1.FlagType_FLAG_TYPE_UNVERIFIED
	infoSeverity := assetsv1.FlagSeverity_FLAG_SEVERITY_INFO
	infoFlag := &assetsv1.AssetQualityFlag{
		FlagId:   strPtr("info_flag"),
		AssetId:  strPtr("btc"),
		FlagType: &infoFlagType,
		Severity: &infoSeverity,
		Source:   strPtr("automated_check"),
		Reason:   strPtr("Contract not verified on Etherscan"),
	}
	err = mgr.RaiseQualityFlag(ctx, infoFlag)
	require.NoError(t, err)

	tradeable, err = mgr.IsAssetTradeable(ctx, "btc")
	require.NoError(t, err)
	assert.True(t, tradeable, "Asset with INFO flag should be tradeable")

	// Test 3: Add CRITICAL severity flag - should NOT be tradeable
	criticalFlagType := assetsv1.FlagType_FLAG_TYPE_EXPLOITED
	criticalSeverity := assetsv1.FlagSeverity_FLAG_SEVERITY_CRITICAL
	criticalFlag := &assetsv1.AssetQualityFlag{
		FlagId:   strPtr("critical_flag"),
		AssetId:  strPtr("btc"),
		FlagType: &criticalFlagType,
		Severity: &criticalSeverity,
		Source:   strPtr("security_team"),
		Reason:   strPtr("Contract has been exploited, funds at risk"),
	}
	err = mgr.RaiseQualityFlag(ctx, criticalFlag)
	require.NoError(t, err)

	tradeable, err = mgr.IsAssetTradeable(ctx, "btc")
	require.NoError(t, err)
	assert.False(t, tradeable, "Asset with CRITICAL flag should NOT be tradeable")

	// Test 4: Resolve the CRITICAL flag - should be tradeable again
	err = mgr.ResolveQualityFlag(ctx, "critical_flag", "admin", "Issue resolved")
	require.NoError(t, err)

	tradeable, err = mgr.IsAssetTradeable(ctx, "btc")
	require.NoError(t, err)
	assert.True(t, tradeable, "Asset with resolved CRITICAL flag should be tradeable")
}

func TestQualityManager_IsAssetTradeable_MultipleSeverities(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	mgr := NewQualityManager(repo)

	// Setup: Create an asset
	asset := &assetsv1.Asset{
		AssetId:   strPtr("shitcoin"),
		Symbol:    strPtr("SHIT"),
		Name:      strPtr("Shit Coin"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_ERC20),
	}
	repo.assets["shitcoin"] = asset

	// Add multiple non-CRITICAL flags
	flags := []struct {
		id       string
		flagType assetsv1.FlagType
		severity assetsv1.FlagSeverity
	}{
		{"flag1", assetsv1.FlagType_FLAG_TYPE_LOW_LIQUIDITY, assetsv1.FlagSeverity_FLAG_SEVERITY_LOW},
		{"flag2", assetsv1.FlagType_FLAG_TYPE_TAX_TOKEN, assetsv1.FlagSeverity_FLAG_SEVERITY_MEDIUM},
		{"flag3", assetsv1.FlagType_FLAG_TYPE_UNVERIFIED, assetsv1.FlagSeverity_FLAG_SEVERITY_HIGH},
	}

	for _, f := range flags {
		flag := &assetsv1.AssetQualityFlag{
			FlagId:   strPtr(f.id),
			AssetId:  strPtr("shitcoin"),
			FlagType: &f.flagType,
			Severity: &f.severity,
			Source:   strPtr("automated"),
			Reason:   strPtr("Automated detection"),
		}
		err := mgr.RaiseQualityFlag(ctx, flag)
		require.NoError(t, err)
	}

	// Even with LOW, MEDIUM, HIGH flags, asset should still be tradeable
	tradeable, err := mgr.IsAssetTradeable(ctx, "shitcoin")
	require.NoError(t, err)
	assert.True(t, tradeable, "Asset with non-CRITICAL flags should be tradeable")

	// Add one CRITICAL flag - now should not be tradeable
	criticalFlagType := assetsv1.FlagType_FLAG_TYPE_SCAM
	criticalSeverity := assetsv1.FlagSeverity_FLAG_SEVERITY_CRITICAL
	criticalFlag := &assetsv1.AssetQualityFlag{
		FlagId:   strPtr("critical"),
		AssetId:  strPtr("shitcoin"),
		FlagType: &criticalFlagType,
		Severity: &criticalSeverity,
		Source:   strPtr("security_team"),
		Reason:   strPtr("Confirmed scam"),
	}
	err = mgr.RaiseQualityFlag(ctx, criticalFlag)
	require.NoError(t, err)

	tradeable, err = mgr.IsAssetTradeable(ctx, "shitcoin")
	require.NoError(t, err)
	assert.False(t, tradeable, "Asset with at least one CRITICAL flag should NOT be tradeable")
}

func TestQualityManager_ResolveQualityFlag(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	mgr := NewQualityManager(repo)

	// Setup: Create asset and flag
	asset := &assetsv1.Asset{
		AssetId:   strPtr("usdt"),
		Symbol:    strPtr("USDT"),
		Name:      strPtr("Tether"),
		AssetType: assetTypePtr(assetsv1.AssetType_ASSET_TYPE_ERC20),
	}
	repo.assets["usdt"] = asset

	flagType := assetsv1.FlagType_FLAG_TYPE_PAUSED
	severity := assetsv1.FlagSeverity_FLAG_SEVERITY_HIGH
	flag := &assetsv1.AssetQualityFlag{
		FlagId:   strPtr("pause_flag"),
		AssetId:  strPtr("usdt"),
		FlagType: &flagType,
		Severity: &severity,
		Source:   strPtr("contract_monitor"),
		Reason:   strPtr("Contract paused by owner"),
	}
	err := mgr.RaiseQualityFlag(ctx, flag)
	require.NoError(t, err)

	// Test 1: Resolve the flag
	err = mgr.ResolveQualityFlag(ctx, "pause_flag", "admin", "Contract unpaused")
	require.NoError(t, err)

	// Verify flag is resolved
	resolvedFlag, err := mgr.GetQualityFlag(ctx, "pause_flag")
	require.NoError(t, err)
	assert.NotNil(t, resolvedFlag.ResolvedAt, "Flag should have resolved_at timestamp")
	assert.Equal(t, "admin", *resolvedFlag.ResolvedBy)
	assert.Equal(t, "Contract unpaused", *resolvedFlag.ResolutionNotes)

	// Test 2: Try to resolve already resolved flag - should fail
	err = mgr.ResolveQualityFlag(ctx, "pause_flag", "admin", "Double resolve")
	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.FailedPrecondition, st.Code())

	// Test 3: Try to resolve nonexistent flag
	err = mgr.ResolveQualityFlag(ctx, "nonexistent", "admin", "Notes")
	require.Error(t, err)
	st, ok = status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestQualityManager_ResolveQualityFlag_Validation(t *testing.T) {
	ctx := context.Background()
	repo := newMockRepository()
	mgr := NewQualityManager(repo)

	tests := []struct {
		name            string
		flagID          string
		resolvedBy      string
		resolutionNotes string
		wantErr         bool
		wantErrCode     codes.Code
	}{
		{
			name:            "missing flag_id",
			flagID:          "",
			resolvedBy:      "admin",
			resolutionNotes: "Notes",
			wantErr:         true,
			wantErrCode:     codes.InvalidArgument,
		},
		{
			name:            "missing resolved_by",
			flagID:          "flag1",
			resolvedBy:      "",
			resolutionNotes: "Notes",
			wantErr:         true,
			wantErrCode:     codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mgr.ResolveQualityFlag(ctx, tt.flagID, tt.resolvedBy, tt.resolutionNotes)
			if tt.wantErr {
				require.Error(t, err)
				st, ok := status.FromError(err)
				require.True(t, ok)
				assert.Equal(t, tt.wantErrCode, st.Code())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
