package keeper

// CatalogV2 extends the immutable v1 roster with the four-sector calculation
// configuration and one additional benchmark-eligible Keeper basket for Ember,
// Current, and Harbor. It deliberately reuses the v1 asset universe so the
// release introduces no provider-identity dependency.
func CatalogV2() CatalogRelease {
	release := CatalogV1()
	release.Version = 2
	release.SectorDefinitions = []SectorDefinition{
		{Sector: "CREST", BenchmarkMethod: "EQUAL_WEIGHT", MinimumEligibleBaskets: 2, RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: CalculationScale},
		{Sector: "EMBER", BenchmarkMethod: "EQUAL_WEIGHT", MinimumEligibleBaskets: 2, RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: CalculationScale},
		{Sector: "CURRENT", BenchmarkMethod: "EQUAL_WEIGHT", MinimumEligibleBaskets: 2, RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: CalculationScale},
		{Sector: "HARBOR", BenchmarkMethod: "EQUAL_WEIGHT", MinimumEligibleBaskets: 2, RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: CalculationScale},
	}
	release.TurbulencePolicies = []ExpectedTurbulencePolicy{
		{Key: "crest_static", Type: "STATIC_CONTENT_VALUE", StaticValue: 4_000_000, Floor: 250_000, RoundingMode: "ROUND_HALF_AWAY_FROM_ZERO"},
		{Key: "ember_static", Type: "STATIC_CONTENT_VALUE", StaticValue: 15_000_000, Floor: 1_000_000, RoundingMode: "ROUND_HALF_AWAY_FROM_ZERO"},
		{Key: "current_static", Type: "STATIC_CONTENT_VALUE", StaticValue: 8_000_000, Floor: 500_000, RoundingMode: "ROUND_HALF_AWAY_FROM_ZERO"},
		{Key: "harbor_static", Type: "STATIC_CONTENT_VALUE", StaticValue: 500_000, Floor: 250_000, RoundingMode: "ROUND_HALF_AWAY_FROM_ZERO"},
	}

	release.Baskets = append(release.Baskets,
		basket("ember_breakers", component("dogecoin", 35), component("shiba_inu", 25), component("pepe", 20), component("bonk", 20)),
		basket("current_networks", component("mantle", 35), component("polygon", 35), component("arbitrum", 30)),
		basket("harbor_reserve", component("tether", 40), component("usdc", 35), component("dai", 15), component("usds", 10)),
	)
	calculationConfig := map[string]struct {
		sector   string
		coverage int64
		policy   string
	}{
		"crest_large_cap":  {sector: "CREST", coverage: 80_000_000, policy: "crest_static"},
		"second_current":   {sector: "CREST", coverage: 80_000_000, policy: "crest_static"},
		"wild_momentum":    {sector: "EMBER", coverage: 85_000_000, policy: "ember_static"},
		"ember_breakers":   {sector: "EMBER", coverage: 85_000_000, policy: "ember_static"},
		"exchange_flow":    {sector: "CURRENT", coverage: 80_000_000, policy: "current_static"},
		"current_networks": {sector: "CURRENT", coverage: 80_000_000, policy: "current_static"},
		"harbor_stability": {sector: "HARBOR", coverage: 90_000_000, policy: "harbor_static"},
		"harbor_reserve":   {sector: "HARBOR", coverage: 90_000_000, policy: "harbor_static"},
	}
	for index := range release.Baskets {
		config, exists := calculationConfig[release.Baskets[index].Key]
		if !exists {
			continue
		}
		release.Baskets[index].Sector = config.sector
		release.Baskets[index].MinimumCoveredWeight = config.coverage
		release.Baskets[index].ExpectedTurbulencePolicyKey = config.policy
		release.Baskets[index].NormalizationCap = 2_000_000
		release.Baskets[index].BenchmarkEligible = true
	}

	release.Definitions = append(release.Definitions,
		definition("ember_runner", "Ember Runner", "breaker_current", "Breaker Current", "EMBER", "TRICKSTER", "RARE", "VERY_HIGH", "WILD_SURGE", `{"schemaVersion":1,"parameters":{"positiveNormalizedPerformanceThresholdMicros":10000,"sectorRelativeScoreMultiplierBps":15000,"negativeNormalizedPerformanceThresholdMicros":-10000,"pressurePenaltyIncreaseBps":3000,"pressurePenaltyApplicationCap":null}}`, "ember_breakers", upgradeTree("burning_tail", "Burning Tail", `{"sectorRelativeScoreMultiplierBps":17500}`, "last_laugh", "Last Laugh", `{"pressurePenaltyApplicationCap":1}`)),
		definition("current_architect", "Current Architect", "network_current", "Network Current", "CURRENT", "SUPPORT", "RARE", "MEDIUM", "FLOW_TRANSFER", `{"schemaVersion":1,"parameters":{"minimumAdditionalOutperformingSectors":1,"diversificationScoreBonusBps":1500,"volumeTrendBonusBps":null}}`, "current_networks", upgradeTree("cross_current", "Cross Current", `{"exchangeCurrentRequired":false,"minimumOutperformingSectors":2}`, "deep_liquidity", "Deep Liquidity", `{"volumeTrendBonusBps":null}`)),
		definition("harbor_sentinel", "Harbor Sentinel", "reserve_current", "Reserve Current", "HARBOR", "WARDEN", "RARE", "VERY_LOW", "UNBROKEN_PEG", `{"schemaVersion":1,"parameters":{"depthPenaltyReductionBps":3500,"stabilityThreshold":null,"hullDamageReductionBps":null}}`, "harbor_reserve", upgradeTree("deep_shelter", "Deep Shelter", `{"depthPenaltyReductionBps":4500}`, "safe_yield", "Safe Yield", `{"riskAdjustedScoreBonusBps":null}`)),
	)
	return release
}
