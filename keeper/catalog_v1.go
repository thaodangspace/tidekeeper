package keeper

import (
	"encoding/json"
	"fmt"
)

// CatalogV1 returns the approved MVP roster as seed input. PostgreSQL becomes
// the authoritative runtime source after this release is published.
func CatalogV1() CatalogRelease {
	return CatalogRelease{
		Version: 1,
		Assets: []MarketAsset{
			asset("aave", "AAVE"), asset("api3", "API3"), asset("arbitrum", "ARB"), asset("arrr", "ARRR"), asset("bitcoin", "BTC"),
			asset("binance_coin", "BNB"), asset("bonk", "BONK"), asset("cake", "CAKE"), asset("compound", "COMP"), asset("crv", "CRV"),
			asset("decred", "DCR"), asset("dai", "DAI"), asset("dogecoin", "DOGE"), asset("ethereum", "ETH"), asset("ethena_usde", "USDe"),
			asset("filecoin", "FIL"), asset("grass", "GRASS"), asset("graph", "GRT"), asset("hyperliquid", "HYPE"), asset("jupiter", "JUP"),
			asset("kamino", "KMNO"), asset("leo_token", "LEO"), asset("chainlink", "LINK"), asset("mantle", "MNT"), asset("morpho", "MORPHO"),
			asset("monero", "XMR"), asset("okb", "OKB"), asset("polygon", "POL"), asset("pepe", "PEPE"), asset("pump", "PUMP"),
			asset("pyth", "PYTH"), asset("redstone", "RED"), asset("render", "RENDER"), asset("shiba_inu", "SHIB"), asset("solana", "SOL"),
			asset("syrup", "SYRUP"), asset("bittensor", "TAO"), asset("tellor", "TRB"), asset("tether", "USDT"), asset("usdc", "USDC"),
			asset("usds", "USDS"), asset("uniswap", "UNI"), asset("crypto_com", "CRO"), asset("verge", "XVG"), asset("whitebit", "WBT"),
			asset("xrp", "XRP"), asset("zcash", "ZEC"),
		},
		Baskets: []BasketMapping{
			basket("crest_large_cap", component("bitcoin", 35), component("ethereum", 25), component("binance_coin", 15), component("solana", 15), component("xrp", 10)),
			basket("harbor_stability", component("tether", 30), component("usdc", 30), component("usds", 15), component("dai", 15), component("ethena_usde", 10)),
			basket("forge_commerce", component("binance_coin", 30), component("whitebit", 20), component("leo_token", 20), component("crypto_com", 15), component("okb", 15)),
			basket("exchange_flow", component("hyperliquid", 30), component("uniswap", 25), component("jupiter", 15), component("cake", 15), component("crv", 15)),
			basket("oracle_signal", component("chainlink", 45), component("pyth", 20), component("redstone", 15), component("tellor", 10), component("api3", 10)),
			basket("veiled_reversal", component("zcash", 30), component("monero", 30), component("decred", 15), component("arrr", 15), component("verge", 10)),
			basket("lending_growth", component("aave", 30), component("morpho", 25), component("syrup", 15), component("compound", 15), component("kamino", 15)),
			basket("wild_momentum", component("dogecoin", 30), component("shiba_inu", 20), component("pepe", 20), component("pump", 15), component("bonk", 15)),
			basket("engine_infrastructure", component("bittensor", 30), component("render", 25), component("filecoin", 20), component("grass", 15), component("graph", 10)),
			basket("second_current", component("mantle", 40), component("polygon", 35), component("arbitrum", 25)),
		},
		Definitions: []Definition{
			definition("crest_sovereign", "Crest Sovereign", "sovereign_current", "Sovereign Current", "CREST", "VANGUARD", "COMMON", "LOW_TO_MEDIUM", "CROWN_OF_THE_TIDE", `{"schemaVersion":1,"parameters":{"minimumOutperformingComponents":3,"globalRelativeScoreBonusBps":2000,"broadDeclineDepthPenaltyReductionBps":1500}}`, "crest_large_cap", upgradeTree("deep_crown", "Deep Crown", `{"broadDeclineDepthPenaltyReductionBps":2500}`, "rising_throne", "Rising Throne", `{"sectorRelativeScoreBonusBps":null}`)),
			definition("harbor_warden", "Harbor Warden", "harbor_current", "Harbor Current", "HARBOR", "WARDEN", "COMMON", "VERY_LOW", "UNBROKEN_PEG", `{"schemaVersion":1,"parameters":{"depthPenaltyReductionBps":3500,"stabilityThreshold":null,"hullDamageReductionBps":null}}`, "harbor_stability", upgradeTree("deep_shelter", "Deep Shelter", `{"depthPenaltyReductionBps":4500}`, "safe_yield", "Safe Yield", `{"riskAdjustedScoreBonusBps":null}`)),
			definition("iron_broker", "Iron Broker", "commerce_current", "Commerce Current", "FORGE", "COMPOUNDER", "RARE", "MEDIUM", "FEE_ENGINE", `{"schemaVersion":1,"parameters":{"riskAdjustedScoreBonusBpsPerStack":500,"maxStacks":4,"removeStacksOnFleetExit":true,"shockPressurePenaltyReductionPerStackBps":null}}`, "forge_commerce", upgradeTree("deep_ledger", "Deep Ledger", `{"maxStacks":6}`, "liquid_reserve", "Liquid Reserve", `{"shockPressurePenaltyReductionPerStackBps":null}`)),
			definition("current_weaver", "Current Weaver", "exchange_current", "Exchange Current", "CURRENT", "SUPPORT", "COMMON", "MEDIUM_TO_HIGH", "FLOW_TRANSFER", `{"schemaVersion":1,"parameters":{"minimumAdditionalOutperformingSectors":1,"diversificationScoreBonusBps":1500,"volumeTrendBonusBps":null}}`, "exchange_flow", upgradeTree("cross_current", "Cross Current", `{"exchangeCurrentRequired":false,"minimumOutperformingSectors":2}`, "deep_liquidity", "Deep Liquidity", `{"volumeTrendBonusBps":null}`)),
			definition("echo_seer", "Echo Seer", "oracle_current", "Oracle Current", "ECHO", "ORACLE", "RARE", "MEDIUM", "READ_THE_WAKE", `{"schemaVersion":1,"parameters":{"signalTypes":["SECTOR_BREADTH","CORRELATION","VOLUME_TREND","TURBULENCE","DEPTH_RISK","RECOVERY_STRENGTH"],"selectionPolicy":null,"objectiveContributionBonusBps":1000}}`, "oracle_signal", upgradeTree("clear_echo", "Clear Echo", `{"showSignalConfidence":true}`, "stored_wake", "Stored Wake", `{"retainPriorDaySignal":true}`)),
			definition("veil_reversalist", "Veil Reversalist", "veiled_current", "Veiled Current", "VEIL", "CONTRARIAN", "RARE", "HIGH", "FROM_THE_DEPTHS", `{"schemaVersion":1,"parameters":{"depthExceedsExpectedTurbulence":true,"minimumRecoveryFromLowBps":5000,"resurfaceScoreMultiplierBps":20000,"fallbackDepthPenaltyReductionBps":null}}`, "veiled_reversal", upgradeTree("deeper_return", "Deeper Return", `{"minimumRecoveryFromLowBps":4000}`, "hidden_exit", "Hidden Exit", `{"fallbackDepthPenaltyReductionBps":null}`)),
			definition("bloom_creditor", "Bloom Creditor", "lending_current", "Lending Current", "BLOOM", "COMPOUNDER", "RARE", "MEDIUM", "ACCRUED_TIDE", `{"schemaVersion":1,"parameters":{"skillEffectBonusBpsPerDay":400,"maxConsecutiveDays":5,"extraSuppliesProbabilityBps":null,"deterministicSeedScope":"VOYAGE"}}`, "lending_growth", upgradeTree("deep_roots", "Deep Roots", `{"retainedStacksBps":5000}`, "shared_yield", "Shared Yield", `{"adjacentKeeperSkillEffectBps":null}`)),
			definition("ember_trickster", "Ember Trickster", "wild_current", "Wild Current", "EMBER", "TRICKSTER", "COMMON", "VERY_HIGH", "WILD_SURGE", `{"schemaVersion":1,"parameters":{"positiveNormalizedPerformanceThresholdMicros":10000,"sectorRelativeScoreMultiplierBps":15000,"negativeNormalizedPerformanceThresholdMicros":-10000,"pressurePenaltyIncreaseBps":3000,"pressurePenaltyApplicationCap":null}}`, "wild_momentum", upgradeTree("burning_tail", "Burning Tail", `{"sectorRelativeScoreMultiplierBps":17500}`, "last_laugh", "Last Laugh", `{"pressurePenaltyApplicationCap":1}`)),
			definition("forge_architect", "Forge Architect", "engine_current", "Engine Current", "FORGE", "GROWTH", "EPIC", "HIGH", "DISTRIBUTED_ENGINE", `{"schemaVersion":1,"parameters":{"requiresPositiveBasketReturn":true,"requiresPositiveVolumeTrend":true,"minimumPositiveComponents":3,"performanceCeilingBonusBps":3000,"outlierDepthPenaltyReductionBps":null}}`, "engine_infrastructure", upgradeTree("more_nodes", "More Nodes", `{"minimumPositiveComponents":3,"requiresNewBasketVersion":true}`, "redundant_engine", "Redundant Engine", `{"outlierDepthPenaltyReductionBps":null}`)),
			definition("crest_pathfinder", "Crest Pathfinder", "second_current", "Second Current", "CREST", "GROWTH", "RARE", "MEDIUM_TO_HIGH", "SECOND_CURRENT", `{"schemaVersion":1,"parameters":{"requiresPositiveLayerOneBenchmark":true,"fleetBonusSectorRelativeScoreBps":2500,"layerOneDeepDeclineDepthPenaltyIncreaseBps":2000}}`, "second_current", upgradeTree("higher_path", "Higher Path", `{"fleetBonusSectorRelativeScoreBps":3500}`, "safety_route", "Safety Route", `{"layerOneDeepDeclineDepthPenaltyIncreaseBps":1000}`)),
		},
	}
}

func asset(key, symbol string) MarketAsset {
	return MarketAsset{Key: key, Symbol: symbol}
}

func basket(key string, components ...BasketComponent) BasketMapping {
	return BasketMapping{Key: key, Components: components}
}

func component(assetKey string, percentage int64) BasketComponent {
	return BasketComponent{AssetKey: assetKey, Weight: percentage * 1_000_000}
}

func definition(key, name, currentKey, currentName, sector, role, rarity, baseRisk, ruleKey, ruleConfig, basketKey string, upgrades json.RawMessage) Definition {
	return Definition{
		Key:               key,
		Name:              name,
		CurrentKey:        currentKey,
		CurrentName:       currentName,
		Sector:            sector,
		Role:              role,
		Rarity:            rarity,
		BaseRisk:          baseRisk,
		PassiveRuleKey:    ruleKey,
		PassiveRuleConfig: mustRawJSON(ruleConfig),
		UpgradeTree:       upgrades,
		BasketMappingKey:  basketKey,
	}
}

func upgradeTree(firstKey, firstName, firstEffects, secondKey, secondName, secondEffects string) json.RawMessage {
	value := struct {
		RootNodeKey string `json:"rootNodeKey"`
		Nodes       []struct {
			Key     string          `json:"key"`
			Name    string          `json:"name"`
			Next    []string        `json:"next"`
			Effects json.RawMessage `json:"effects"`
		} `json:"nodes"`
	}{
		RootNodeKey: "base",
		Nodes: []struct {
			Key     string          `json:"key"`
			Name    string          `json:"name"`
			Next    []string        `json:"next"`
			Effects json.RawMessage `json:"effects"`
		}{
			{Key: "base", Name: "Base", Next: []string{firstKey, secondKey}, Effects: mustRawJSON(`{}`)},
			{Key: firstKey, Name: firstName, Next: []string{}, Effects: mustRawJSON(firstEffects)},
			{Key: secondKey, Name: secondName, Next: []string{}, Effects: mustRawJSON(secondEffects)},
		},
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("marshal catalog upgrade tree: %v", err))
	}
	return encoded
}

func mustRawJSON(value string) json.RawMessage {
	if !json.Valid([]byte(value)) {
		panic(fmt.Sprintf("invalid static catalog JSON: %s", value))
	}
	return json.RawMessage(value)
}
