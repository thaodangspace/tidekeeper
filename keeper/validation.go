package keeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
)

var (
	contentKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	symbolPattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
)

var validSectors = tokenSet("CREST", "HARBOR", "FORGE", "CURRENT", "ECHO", "VEIL", "BLOOM", "EMBER")
var validRoles = tokenSet("VANGUARD", "WARDEN", "COMPOUNDER", "SUPPORT", "ORACLE", "CONTRARIAN", "TRICKSTER", "GROWTH")
var validRarities = tokenSet("COMMON", "RARE", "EPIC")
var validBaseRisks = tokenSet("VERY_LOW", "LOW_TO_MEDIUM", "MEDIUM", "MEDIUM_TO_HIGH", "HIGH", "VERY_HIGH")

// Validate confirms that a candidate release can be safely persisted as content.
func Validate(release CatalogRelease) error {
	if release.Version <= 0 {
		return errors.New("release version must be positive")
	}

	calculationSectors, turbulencePolicies, err := validateCalculationPolicies(release.SectorDefinitions, release.TurbulencePolicies)
	if err != nil {
		return err
	}

	assets := make(map[string]MarketAsset, len(release.Assets))
	symbols := make(map[string]struct{}, len(release.Assets))
	for _, asset := range release.Assets {
		if !contentKeyPattern.MatchString(asset.Key) {
			return fmt.Errorf("asset %q has an invalid key", asset.Key)
		}
		if !symbolPattern.MatchString(asset.Symbol) {
			return fmt.Errorf("asset %q has an invalid symbol", asset.Key)
		}
		if _, exists := assets[asset.Key]; exists {
			return fmt.Errorf("duplicate asset key %q", asset.Key)
		}
		if _, exists := symbols[asset.Symbol]; exists {
			return fmt.Errorf("duplicate asset symbol %q", asset.Symbol)
		}
		assets[asset.Key] = asset
		symbols[asset.Symbol] = struct{}{}
	}

	baskets := make(map[string]BasketMapping, len(release.Baskets))
	for _, basket := range release.Baskets {
		if !contentKeyPattern.MatchString(basket.Key) {
			return fmt.Errorf("basket %q has an invalid key", basket.Key)
		}
		if _, exists := baskets[basket.Key]; exists {
			return fmt.Errorf("duplicate basket key %q", basket.Key)
		}
		if len(basket.Components) == 0 {
			return fmt.Errorf("basket %q has no components", basket.Key)
		}

		componentAssets := make(map[string]struct{}, len(basket.Components))
		var total int64
		for _, component := range basket.Components {
			if _, exists := assets[component.AssetKey]; !exists {
				return fmt.Errorf("basket %q references unknown asset %q", basket.Key, component.AssetKey)
			}
			if _, exists := componentAssets[component.AssetKey]; exists {
				return fmt.Errorf("basket %q repeats asset %q", basket.Key, component.AssetKey)
			}
			if component.Weight <= 0 || component.Weight > WeightScale {
				return fmt.Errorf("basket %q has invalid weight for asset %q", basket.Key, component.AssetKey)
			}
			if total > WeightScale-component.Weight {
				return fmt.Errorf("basket %q component weights overflow fixed scale", basket.Key)
			}
			componentAssets[component.AssetKey] = struct{}{}
			total += component.Weight
		}
		if total != WeightScale {
			return fmt.Errorf("basket %q component weights total %d, want %d", basket.Key, total, WeightScale)
		}
		baskets[basket.Key] = basket
	}

	definitions := make(map[string]struct{}, len(release.Definitions))
	for _, definition := range release.Definitions {
		if err := validateDefinition(definition, baskets); err != nil {
			return err
		}
		if _, exists := definitions[definition.Key]; exists {
			return fmt.Errorf("duplicate Keeper key %q", definition.Key)
		}
		definitions[definition.Key] = struct{}{}
	}
	if err := validateBasketCalculationConfiguration(baskets, release.Definitions, calculationSectors, turbulencePolicies); err != nil {
		return err
	}
	return nil
}

func validateDefinition(definition Definition, baskets map[string]BasketMapping) error {
	if !contentKeyPattern.MatchString(definition.Key) {
		return fmt.Errorf("Keeper %q has an invalid key", definition.Key)
	}
	if definition.Name == "" || definition.CurrentName == "" || !contentKeyPattern.MatchString(definition.CurrentKey) {
		return fmt.Errorf("Keeper %q has invalid display Current data", definition.Key)
	}
	if !validSectors[definition.Sector] || !validRoles[definition.Role] || !validRarities[definition.Rarity] || !validBaseRisks[definition.BaseRisk] {
		return fmt.Errorf("Keeper %q has an invalid classification", definition.Key)
	}
	if definition.ExpectedTurbulenceBPS != nil && (*definition.ExpectedTurbulenceBPS <= 0 || *definition.ExpectedTurbulenceBPS > 1_000_000) {
		return fmt.Errorf("Keeper %q has invalid expected turbulence", definition.Key)
	}
	if _, exists := baskets[definition.BasketMappingKey]; !exists {
		return fmt.Errorf("Keeper %q references unknown basket %q", definition.Key, definition.BasketMappingKey)
	}
	if err := validatePassiveRule(definition.PassiveRuleKey, definition.PassiveRuleConfig); err != nil {
		return fmt.Errorf("Keeper %q: %w", definition.Key, err)
	}
	if err := validateUpgradeTree(definition.UpgradeTree); err != nil {
		return fmt.Errorf("Keeper %q: %w", definition.Key, err)
	}
	return nil
}

var passiveRuleParameters = map[string]map[string]struct{}{
	"CROWN_OF_THE_TIDE":  parameterSet("minimumOutperformingComponents", "globalRelativeScoreBonusBps", "broadDeclineDepthPenaltyReductionBps"),
	"UNBROKEN_PEG":       parameterSet("depthPenaltyReductionBps", "stabilityThreshold", "hullDamageReductionBps"),
	"FEE_ENGINE":         parameterSet("riskAdjustedScoreBonusBpsPerStack", "maxStacks", "removeStacksOnFleetExit", "shockPressurePenaltyReductionPerStackBps"),
	"FLOW_TRANSFER":      parameterSet("minimumAdditionalOutperformingSectors", "diversificationScoreBonusBps", "volumeTrendBonusBps"),
	"READ_THE_WAKE":      parameterSet("signalTypes", "selectionPolicy", "objectiveContributionBonusBps"),
	"FROM_THE_DEPTHS":    parameterSet("depthExceedsExpectedTurbulence", "minimumRecoveryFromLowBps", "resurfaceScoreMultiplierBps", "fallbackDepthPenaltyReductionBps"),
	"ACCRUED_TIDE":       parameterSet("skillEffectBonusBpsPerDay", "maxConsecutiveDays", "extraSuppliesProbabilityBps", "deterministicSeedScope"),
	"WILD_SURGE":         parameterSet("positiveNormalizedPerformanceThresholdMicros", "sectorRelativeScoreMultiplierBps", "negativeNormalizedPerformanceThresholdMicros", "pressurePenaltyIncreaseBps", "pressurePenaltyApplicationCap"),
	"DISTRIBUTED_ENGINE": parameterSet("requiresPositiveBasketReturn", "requiresPositiveVolumeTrend", "minimumPositiveComponents", "performanceCeilingBonusBps", "outlierDepthPenaltyReductionBps"),
	"SECOND_CURRENT":     parameterSet("requiresPositiveLayerOneBenchmark", "fleetBonusSectorRelativeScoreBps", "layerOneDeepDeclineDepthPenaltyIncreaseBps"),
}

func validatePassiveRule(key string, config json.RawMessage) error {
	expectedParameters, exists := passiveRuleParameters[key]
	if !exists {
		return fmt.Errorf("unknown passive rule key %q", key)
	}

	var envelope struct {
		SchemaVersion int                        `json:"schemaVersion"`
		Parameters    map[string]json.RawMessage `json:"parameters"`
	}
	if len(config) == 0 || !isJSONObject(config) || json.Unmarshal(config, &envelope) != nil {
		return errors.New("invalid passive rule configuration: must be a JSON object")
	}
	if envelope.SchemaVersion != 1 {
		return errors.New("invalid passive rule configuration: schemaVersion must be 1")
	}
	if envelope.Parameters == nil {
		return errors.New("invalid passive rule configuration: parameters must be a JSON object")
	}
	for parameter := range expectedParameters {
		if _, exists := envelope.Parameters[parameter]; !exists {
			return fmt.Errorf("invalid passive rule configuration: missing parameter %q", parameter)
		}
	}
	for parameter := range envelope.Parameters {
		if _, exists := expectedParameters[parameter]; !exists {
			return fmt.Errorf("invalid passive rule configuration: unknown parameter %q", parameter)
		}
	}
	return nil
}

func parameterSet(parameters ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(parameters))
	for _, parameter := range parameters {
		set[parameter] = struct{}{}
	}
	return set
}

func validateUpgradeTree(tree json.RawMessage) error {
	var parsed struct {
		RootNodeKey string `json:"rootNodeKey"`
		Nodes       []struct {
			Key     string          `json:"key"`
			Name    string          `json:"name"`
			Next    []string        `json:"next"`
			Effects json.RawMessage `json:"effects"`
		} `json:"nodes"`
	}
	if len(tree) == 0 || !isJSONObject(tree) || json.Unmarshal(tree, &parsed) != nil {
		return errors.New("upgrade tree must be a JSON object")
	}
	if !contentKeyPattern.MatchString(parsed.RootNodeKey) || len(parsed.Nodes) == 0 {
		return errors.New("upgrade tree requires a valid rootNodeKey and nodes")
	}

	nextByKey := make(map[string][]string, len(parsed.Nodes))
	inDegree := make(map[string]int, len(parsed.Nodes))
	for _, node := range parsed.Nodes {
		if !contentKeyPattern.MatchString(node.Key) {
			return fmt.Errorf("upgrade node %q has an invalid key", node.Key)
		}
		if node.Name == "" || !isJSONObject(node.Effects) {
			return fmt.Errorf("upgrade node %q must have a name and object effects", node.Key)
		}
		if _, exists := nextByKey[node.Key]; exists {
			return fmt.Errorf("duplicate upgrade node key %q", node.Key)
		}
		nextByKey[node.Key] = node.Next
		inDegree[node.Key] = 0
	}
	if _, exists := nextByKey[parsed.RootNodeKey]; !exists {
		return fmt.Errorf("upgrade root %q is not a node", parsed.RootNodeKey)
	}
	for nodeKey, next := range nextByKey {
		for _, childKey := range next {
			if _, exists := nextByKey[childKey]; !exists {
				return fmt.Errorf("upgrade node %q references unknown node %q", nodeKey, childKey)
			}
			inDegree[childKey]++
		}
	}
	if inDegree[parsed.RootNodeKey] != 0 {
		return errors.New("upgrade root has an incoming edge")
	}
	for nodeKey, degree := range inDegree {
		if nodeKey != parsed.RootNodeKey && degree != 1 {
			return fmt.Errorf("upgrade node %q must have exactly one parent", nodeKey)
		}
	}

	visiting := make(map[string]bool, len(nextByKey))
	visited := make(map[string]bool, len(nextByKey))
	var visit func(string) error
	visit = func(nodeKey string) error {
		if visiting[nodeKey] {
			return fmt.Errorf("upgrade tree contains cycle at %q", nodeKey)
		}
		if visited[nodeKey] {
			return nil
		}
		visiting[nodeKey] = true
		for _, childKey := range nextByKey[nodeKey] {
			if err := visit(childKey); err != nil {
				return err
			}
		}
		visiting[nodeKey] = false
		visited[nodeKey] = true
		return nil
	}
	if err := visit(parsed.RootNodeKey); err != nil {
		return err
	}
	if len(visited) != len(nextByKey) {
		keys := make([]string, 0, len(nextByKey)-len(visited))
		for key := range nextByKey {
			if !visited[key] {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		return fmt.Errorf("upgrade tree is disconnected at %q", keys[0])
	}
	return nil
}

var calculationSectorKeys = tokenSet("CREST", "EMBER", "CURRENT", "HARBOR")

func validateCalculationPolicies(definitions []SectorDefinition, policies []ExpectedTurbulencePolicy) (map[string]SectorDefinition, map[string]ExpectedTurbulencePolicy, error) {
	sectorDefinitions := make(map[string]SectorDefinition, len(definitions))
	turbulencePolicies := make(map[string]ExpectedTurbulencePolicy, len(policies))
	if len(definitions) == 0 && len(policies) == 0 {
		return sectorDefinitions, turbulencePolicies, nil
	}
	if len(definitions) != len(calculationSectorKeys) || len(policies) == 0 {
		return nil, nil, errors.New("calculation content requires four sector definitions and turbulence policies")
	}
	for _, definition := range definitions {
		if !calculationSectorKeys[definition.Sector] {
			return nil, nil, fmt.Errorf("unsupported calculation sector %q", definition.Sector)
		}
		if _, exists := sectorDefinitions[definition.Sector]; exists {
			return nil, nil, fmt.Errorf("duplicate sector definition %q", definition.Sector)
		}
		if definition.BenchmarkMethod != "EQUAL_WEIGHT" || definition.MinimumEligibleBaskets < 2 || definition.RelativeScale <= 0 || definition.ScoreCap <= 0 ||
			definition.RelativeBlendWeight < 0 || definition.RankBlendWeight < 0 || definition.RelativeBlendWeight+definition.RankBlendWeight != CalculationScale {
			return nil, nil, fmt.Errorf("invalid calculation definition for sector %q", definition.Sector)
		}
		sectorDefinitions[definition.Sector] = definition
	}
	for _, policy := range policies {
		if !contentKeyPattern.MatchString(policy.Key) {
			return nil, nil, fmt.Errorf("turbulence policy %q has an invalid key", policy.Key)
		}
		if _, exists := turbulencePolicies[policy.Key]; exists {
			return nil, nil, fmt.Errorf("duplicate turbulence policy %q", policy.Key)
		}
		if policy.Type != "STATIC_CONTENT_VALUE" || policy.StaticValue <= 0 || policy.Floor <= 0 || policy.Floor > policy.StaticValue || policy.RoundingMode != "ROUND_HALF_AWAY_FROM_ZERO" {
			return nil, nil, fmt.Errorf("invalid turbulence policy %q", policy.Key)
		}
		turbulencePolicies[policy.Key] = policy
	}
	return sectorDefinitions, turbulencePolicies, nil
}

func validateBasketCalculationConfiguration(baskets map[string]BasketMapping, definitions []Definition, sectors map[string]SectorDefinition, policies map[string]ExpectedTurbulencePolicy) error {
	if len(sectors) == 0 {
		for _, basket := range baskets {
			if basket.Sector != "" || basket.MinimumCoveredWeight != 0 || basket.ExpectedTurbulencePolicyKey != "" || basket.NormalizationCap != 0 || basket.BenchmarkEligible {
				return fmt.Errorf("basket %q has calculation configuration without sector definitions", basket.Key)
			}
		}
		return nil
	}

	eligibleCounts := make(map[string]int, len(sectors))
	for _, basket := range baskets {
		configured := basket.Sector != "" || basket.MinimumCoveredWeight != 0 || basket.ExpectedTurbulencePolicyKey != "" || basket.NormalizationCap != 0 || basket.BenchmarkEligible
		if !configured {
			continue
		}
		if _, exists := sectors[basket.Sector]; !exists || basket.MinimumCoveredWeight <= 0 || basket.MinimumCoveredWeight > WeightScale ||
			!contentKeyPattern.MatchString(basket.ExpectedTurbulencePolicyKey) || basket.NormalizationCap <= 0 {
			return fmt.Errorf("basket %q has invalid calculation configuration", basket.Key)
		}
		if _, exists := policies[basket.ExpectedTurbulencePolicyKey]; !exists {
			return fmt.Errorf("basket %q references unknown turbulence policy %q", basket.Key, basket.ExpectedTurbulencePolicyKey)
		}
		if basket.BenchmarkEligible {
			eligibleCounts[basket.Sector]++
		}
	}
	for sectorKey, definition := range sectors {
		if eligibleCounts[sectorKey] < definition.MinimumEligibleBaskets {
			return fmt.Errorf("sector %q has %d eligible baskets, want at least %d", sectorKey, eligibleCounts[sectorKey], definition.MinimumEligibleBaskets)
		}
	}
	for _, definition := range definitions {
		basket := baskets[definition.BasketMappingKey]
		if basket.Sector != "" && basket.Sector != definition.Sector {
			return fmt.Errorf("Keeper %q sector %q does not match basket sector %q", definition.Key, definition.Sector, basket.Sector)
		}
	}
	return nil
}

func isJSONObject(raw json.RawMessage) bool {
	var value any
	return json.Unmarshal(raw, &value) == nil && value != nil && jsonType(value) == "object"
}

func jsonType(value any) string {
	if _, ok := value.(map[string]any); ok {
		return "object"
	}
	return "other"
}

func tokenSet(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
