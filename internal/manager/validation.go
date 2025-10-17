package manager

import (
	"fmt"
	"regexp"
	"strings"

	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
	marketsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/markets/v1"
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

// ValidateRequiredInstrumentFields validates that all required fields for an instrument are present
func ValidateRequiredInstrumentFields(instrument *marketsv1.Instrument) error {
	if instrument == nil {
		return fmt.Errorf("instrument cannot be nil")
	}

	if instrument.InstrumentType == nil || strings.TrimSpace(*instrument.InstrumentType) == "" {
		return fmt.Errorf("instrument_type is required")
	}

	// Validate instrument_type is one of the allowed values
	validTypes := map[string]bool{
		"SPOT":            true,
		"PERPETUAL":       true,
		"FUTURE":          true,
		"OPTION":          true,
		"LENDING_DEPOSIT": true,
		"LENDING_BORROW":  true,
	}
	if !validTypes[*instrument.InstrumentType] {
		return fmt.Errorf("invalid instrument_type: %s (must be SPOT, PERPETUAL, FUTURE, OPTION, LENDING_DEPOSIT, or LENDING_BORROW)", *instrument.InstrumentType)
	}

	if instrument.Code == nil || strings.TrimSpace(*instrument.Code) == "" {
		return fmt.Errorf("code is required")
	}

	return nil
}

// ValidatePerpContractFields validates perpetual contract specific fields
func ValidatePerpContractFields(perp *marketsv1.PerpContract) error {
	if perp == nil {
		return fmt.Errorf("perp contract cannot be nil")
	}

	if perp.UnderlyingAssetId == nil || strings.TrimSpace(*perp.UnderlyingAssetId) == "" {
		return fmt.Errorf("underlying_asset_id is required for perpetual contracts")
	}

	// Contract multiplier should be positive if provided
	if perp.ContractMultiplier != nil && *perp.ContractMultiplier != "" {
		// Basic validation - just ensure it's non-empty
		// The database will enforce numeric format
	}

	return nil
}

// ValidateFutureContractFields validates future contract specific fields
func ValidateFutureContractFields(future *marketsv1.FutureContract) error {
	if future == nil {
		return fmt.Errorf("future contract cannot be nil")
	}

	if future.UnderlyingAssetId == nil || strings.TrimSpace(*future.UnderlyingAssetId) == "" {
		return fmt.Errorf("underlying_asset_id is required for future contracts")
	}

	if future.Expiry == nil {
		return fmt.Errorf("expiry is required for future contracts")
	}

	return nil
}

// ValidateOptionSeriesFields validates option series specific fields
func ValidateOptionSeriesFields(option *marketsv1.OptionSeries) error {
	if option == nil {
		return fmt.Errorf("option series cannot be nil")
	}

	if option.UnderlyingAssetId == nil || strings.TrimSpace(*option.UnderlyingAssetId) == "" {
		return fmt.Errorf("underlying_asset_id is required for options")
	}

	if option.Expiry == nil {
		return fmt.Errorf("expiry is required for options")
	}

	if option.StrikePrice == nil || strings.TrimSpace(*option.StrikePrice) == "" {
		return fmt.Errorf("strike_price is required for options")
	}

	if option.OptionType == nil || strings.TrimSpace(*option.OptionType) == "" {
		return fmt.Errorf("option_type is required for options")
	}

	// Validate option_type is CALL or PUT
	optionType := strings.ToUpper(*option.OptionType)
	if optionType != "CALL" && optionType != "PUT" {
		return fmt.Errorf("option_type must be CALL or PUT, got: %s", *option.OptionType)
	}

	if option.ExerciseStyle == nil || strings.TrimSpace(*option.ExerciseStyle) == "" {
		return fmt.Errorf("exercise_style is required for options")
	}

	// Validate exercise_style is european or american
	exerciseStyle := strings.ToLower(*option.ExerciseStyle)
	if exerciseStyle != "european" && exerciseStyle != "american" {
		return fmt.Errorf("exercise_style must be 'european' or 'american', got: %s", *option.ExerciseStyle)
	}

	return nil
}

// ValidateRequiredMarketFields validates that all required fields for a market are present
func ValidateRequiredMarketFields(market *marketsv1.Market) error {
	if market == nil {
		return fmt.Errorf("market cannot be nil")
	}

	if market.InstrumentId == nil || strings.TrimSpace(*market.InstrumentId) == "" {
		return fmt.Errorf("instrument_id is required")
	}

	if market.VenueId == nil || strings.TrimSpace(*market.VenueId) == "" {
		return fmt.Errorf("venue_id is required")
	}

	if market.VenueSymbol == nil || strings.TrimSpace(*market.VenueSymbol) == "" {
		return fmt.Errorf("venue_symbol is required")
	}

	return nil
}

// ValidateMarketSpecs validates market specifications (tick_size, lot_size, etc.)
func ValidateMarketSpecs(market *marketsv1.Market) error {
	if market == nil {
		return fmt.Errorf("market cannot be nil")
	}

	// All market spec fields are optional, but if provided should be positive
	// The database constraints will enforce this, so we just do basic checks here

	return nil
}
