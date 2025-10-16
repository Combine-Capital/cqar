package bootstrap

import (
	assetsv1 "github.com/Combine-Capital/cqc/gen/go/cqc/assets/v1"
)

// AssetData represents validated asset data ready for creation
type AssetData struct {
	Symbol      string
	Name        string
	Type        assetsv1.AssetType
	Category    string
	Description string
	LogoURL     string
	Homepage    string
	CoinGeckoID string // For fetching deployments later
}

// ChainData represents chain information for seeding
type ChainData struct {
	ChainID       string
	Name          string
	Type          string // ChainType as string: "EVM", "NON_EVM", "UTXO"
	NativeAssetID string
	RPCURLs       []string
	ExplorerURL   string
}

// DeploymentData represents asset deployment information
type DeploymentData struct {
	AssetID         string
	ChainID         string
	ContractAddress string
	Decimals        int32
	IsCanonical     bool
}

// SeedResult tracks the results of seeding operations
type SeedResult struct {
	TotalProcessed int
	Succeeded      int
	Failed         int
	Skipped        int
	Errors         []SeedError
	SkippedReasons []SkipReason
}

// SeedError represents a specific error during seeding
type SeedError struct {
	Entity string // asset symbol or chain name
	Reason string
	Error  error
}

// SkipReason represents why an entity was skipped
type SkipReason struct {
	Entity string
	Reason string
}

// AddSuccess increments the success counter
func (r *SeedResult) AddSuccess() {
	r.TotalProcessed++
	r.Succeeded++
}

// AddFailure increments the failure counter and records the error
func (r *SeedResult) AddFailure(entity, reason string, err error) {
	r.TotalProcessed++
	r.Failed++
	r.Errors = append(r.Errors, SeedError{
		Entity: entity,
		Reason: reason,
		Error:  err,
	})
}

// AddSkipped increments the skipped counter and records the reason
func (r *SeedResult) AddSkipped(entity, reason string) {
	r.TotalProcessed++
	r.Skipped++
	r.SkippedReasons = append(r.SkippedReasons, SkipReason{
		Entity: entity,
		Reason: reason,
	})
}

// Summary returns a human-readable summary of the seeding results
func (r *SeedResult) Summary() string {
	return ""
}
