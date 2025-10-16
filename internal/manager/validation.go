package manager

import (
	"fmt"
	"regexp"
	"strings"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
)

// Validation constants
const (
	MinDecimals = 0
	MaxDecimals = 18
)

// Ethereum address regex (0x followed by 40 hex characters)
var ethereumAddressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// Solana address regex (base58, typically 32-44 characters)
var solanaAddressRegex = regexp.MustCompile(`^[1-9A-HJ-NP-Za-km-z]{32,44}$`)

// ValidateRequiredAssetFields validates that all required fields for an asset are present
func ValidateRequiredAssetFields(asset *assetsv1.Asset) error {
	if asset == nil {
		return fmt.Errorf("asset cannot be nil")
	}

	if asset.Symbol == nil || strings.TrimSpace(*asset.Symbol) == "" {
		return fmt.Errorf("symbol is required")
	}

	if asset.Name == nil || strings.TrimSpace(*asset.Name) == "" {
		return fmt.Errorf("name is required")
	}

	if asset.AssetType == nil || *asset.AssetType == assetsv1.AssetType_ASSET_TYPE_UNSPECIFIED {
		return fmt.Errorf("asset_type is required")
	}

	return nil
}

// ValidateContractAddress validates contract address format based on chain type
func ValidateContractAddress(contractAddress string, chainType string) error {
	if contractAddress == "" {
		return fmt.Errorf("contract_address is required")
	}

	// Allow "native" for native blockchain tokens
	if contractAddress == "native" {
		return nil
	}

	// Normalize chain type
	chainType = strings.ToUpper(chainType)

	switch chainType {
	case "EVM", "ETHEREUM", "POLYGON", "ARBITRUM", "OPTIMISM", "AVALANCHE", "BSC", "BASE":
		if !ethereumAddressRegex.MatchString(contractAddress) {
			return fmt.Errorf("invalid EVM contract address format: %s (expected 0x followed by 40 hex characters)", contractAddress)
		}
	case "SOLANA", "SPL":
		if !solanaAddressRegex.MatchString(contractAddress) {
			return fmt.Errorf("invalid Solana contract address format: %s (expected base58 address)", contractAddress)
		}
	default:
		// For unknown chain types, just check that it's non-empty
		// More specific validation can be added as needed
		if strings.TrimSpace(contractAddress) == "" {
			return fmt.Errorf("contract_address cannot be empty")
		}
	}

	return nil
}

// ValidateDecimals validates that decimals are within acceptable range
func ValidateDecimals(decimals int32) error {
	if decimals < MinDecimals || decimals > MaxDecimals {
		return fmt.Errorf("decimals must be between %d and %d, got %d", MinDecimals, MaxDecimals, decimals)
	}
	return nil
}

// ValidateRelationshipType validates that relationship type is specified
func ValidateRelationshipType(relType assetsv1.RelationshipType) error {
	if relType == assetsv1.RelationshipType_RELATIONSHIP_TYPE_UNSPECIFIED {
		return fmt.Errorf("relationship_type is required")
	}
	return nil
}

// ValidateFlagSeverity validates that severity is specified
func ValidateFlagSeverity(severity assetsv1.FlagSeverity) error {
	if severity == assetsv1.FlagSeverity_FLAG_SEVERITY_UNSPECIFIED {
		return fmt.Errorf("severity is required")
	}
	return nil
}

// ValidateFlagType validates that flag type is specified
func ValidateFlagType(flagType assetsv1.FlagType) error {
	if flagType == assetsv1.FlagType_FLAG_TYPE_UNSPECIFIED {
		return fmt.Errorf("flag_type is required")
	}
	return nil
}

// ValidateSource validates that source is non-empty
func ValidateSource(source string) error {
	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("source is required")
	}
	return nil
}

// ValidateReason validates that reason is non-empty for quality flags
func ValidateReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}
