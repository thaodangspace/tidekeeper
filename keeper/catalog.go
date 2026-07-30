// Package keeper defines versioned Keeper catalog content and its validation rules.
package keeper

import "encoding/json"

const (
	// WeightScale is the fixed decimal scale used for basket component weights.
	WeightScale int64 = 100_000_000
	// CalculationScale is the fixed scale used by normalized calculation policy values.
	CalculationScale int64 = 1_000_000
)

// CatalogRelease is a complete candidate version of the static Keeper catalog.
type CatalogRelease struct {
	Version            int64
	Assets             []MarketAsset
	SectorDefinitions  []SectorDefinition
	TurbulencePolicies []ExpectedTurbulencePolicy
	Baskets            []BasketMapping
	Definitions        []Definition
}

// SectorDefinition is an immutable policy for one of the four calculation Sectors.
type SectorDefinition struct {
	Sector                 string
	BenchmarkMethod        string
	MinimumEligibleBaskets int
	RelativeScale          int64
	RelativeBlendWeight    int64
	RankBlendWeight        int64
	ScoreCap               int64
}

// ExpectedTurbulencePolicy supplies the fixed baseline used to normalize a basket.
type ExpectedTurbulencePolicy struct {
	Key          string
	Type         string
	StaticValue  int64
	Floor        int64
	RoundingMode string
}

// MarketAsset identifies an externally-priced asset without coupling it to a provider.
type MarketAsset struct {
	Key    string
	Symbol string
}

// BasketMapping maps a fictional Current to weighted canonical market assets.
type BasketMapping struct {
	Key                         string
	Sector                      string
	MinimumCoveredWeight        int64
	ExpectedTurbulencePolicyKey string
	NormalizationCap            int64
	BenchmarkEligible           bool
	Components                  []BasketComponent
}

// BasketComponent is one fixed-scale weight in a basket.
type BasketComponent struct {
	AssetKey string
	Weight   int64
}

// Definition is one immutable Keeper definition in a catalog release.
type Definition struct {
	Key                   string
	Name                  string
	CurrentKey            string
	CurrentName           string
	Sector                string
	Role                  string
	Rarity                string
	BaseRisk              string
	ExpectedTurbulenceBPS *int
	PassiveRuleKey        string
	PassiveRuleConfig     json.RawMessage
	UpgradeTree           json.RawMessage
	BasketMappingKey      string
}
