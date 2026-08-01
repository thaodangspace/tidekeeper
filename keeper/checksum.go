package keeper

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

// Checksum returns a stable SHA-256 checksum for a validated catalog release.
func Checksum(release CatalogRelease) ([sha256.Size]byte, error) {
	if err := Validate(release); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("validate catalog before checksumming: %w", err)
	}
	encoded, err := canonicalReleaseJSON(release)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(encoded), nil
}

// CanonicalJSON returns the canonical schema-1 serialization of a catalog
// release without validating it. Callers must validate the release before
// relying on the output; the serialization is byte-identical to the payload
// hashed by Checksum so schema-1 meaning is preserved for aggregate schema-2
// checksums.
func CanonicalJSON(release CatalogRelease) ([]byte, error) {
	return canonicalReleaseJSON(release)
}

func canonicalReleaseJSON(release CatalogRelease) ([]byte, error) {
	type canonicalComponent struct {
		AssetKey string `json:"assetKey"`
		Weight   int64  `json:"weight"`
	}
	type canonicalBasket struct {
		Key                         string               `json:"key"`
		Sector                      string               `json:"sector"`
		MinimumCoveredWeight        int64                `json:"minimumCoveredWeight"`
		ExpectedTurbulencePolicyKey string               `json:"expectedTurbulencePolicyKey"`
		NormalizationCap            int64                `json:"normalizationCap"`
		BenchmarkEligible           bool                 `json:"benchmarkEligible"`
		Components                  []canonicalComponent `json:"components"`
	}
	type canonicalDefinition struct {
		Key                   string          `json:"key"`
		Name                  string          `json:"name"`
		CurrentKey            string          `json:"currentKey"`
		CurrentName           string          `json:"currentName"`
		Sector                string          `json:"sector"`
		Role                  string          `json:"role"`
		Rarity                string          `json:"rarity"`
		BaseRisk              string          `json:"baseRisk"`
		ExpectedTurbulenceBPS *int            `json:"expectedTurbulenceBps"`
		PassiveRuleKey        string          `json:"passiveRuleKey"`
		PassiveRuleConfig     json.RawMessage `json:"passiveRuleConfig"`
		UpgradeTree           json.RawMessage `json:"upgradeTree"`
		BasketMappingKey      string          `json:"basketMappingKey"`
	}
	canonical := struct {
		Version            int64                      `json:"version"`
		Assets             []MarketAsset              `json:"assets"`
		SectorDefinitions  []SectorDefinition         `json:"sectorDefinitions"`
		TurbulencePolicies []ExpectedTurbulencePolicy `json:"turbulencePolicies"`
		Baskets            []canonicalBasket          `json:"baskets"`
		Definitions        []canonicalDefinition      `json:"definitions"`
	}{
		Version: release.Version,
	}

	canonical.Assets = append([]MarketAsset(nil), release.Assets...)
	sort.Slice(canonical.Assets, func(i, j int) bool { return canonical.Assets[i].Key < canonical.Assets[j].Key })

	canonical.SectorDefinitions = append([]SectorDefinition(nil), release.SectorDefinitions...)
	sort.Slice(canonical.SectorDefinitions, func(i, j int) bool {
		return canonical.SectorDefinitions[i].Sector < canonical.SectorDefinitions[j].Sector
	})
	canonical.TurbulencePolicies = append([]ExpectedTurbulencePolicy(nil), release.TurbulencePolicies...)
	sort.Slice(canonical.TurbulencePolicies, func(i, j int) bool { return canonical.TurbulencePolicies[i].Key < canonical.TurbulencePolicies[j].Key })

	for _, basket := range release.Baskets {
		entry := canonicalBasket{
			Key: basket.Key, Sector: basket.Sector, MinimumCoveredWeight: basket.MinimumCoveredWeight,
			ExpectedTurbulencePolicyKey: basket.ExpectedTurbulencePolicyKey,
			NormalizationCap:            basket.NormalizationCap, BenchmarkEligible: basket.BenchmarkEligible,
			Components: make([]canonicalComponent, 0, len(basket.Components)),
		}
		for _, component := range basket.Components {
			entry.Components = append(entry.Components, canonicalComponent{AssetKey: component.AssetKey, Weight: component.Weight})
		}
		sort.Slice(entry.Components, func(i, j int) bool { return entry.Components[i].AssetKey < entry.Components[j].AssetKey })
		canonical.Baskets = append(canonical.Baskets, entry)
	}
	sort.Slice(canonical.Baskets, func(i, j int) bool { return canonical.Baskets[i].Key < canonical.Baskets[j].Key })

	for _, definition := range release.Definitions {
		passiveConfig, err := canonicalJSONObject(definition.PassiveRuleConfig)
		if err != nil {
			return nil, fmt.Errorf("canonicalize Keeper %q passive config: %w", definition.Key, err)
		}
		upgradeTree, err := canonicalJSONObject(definition.UpgradeTree)
		if err != nil {
			return nil, fmt.Errorf("canonicalize Keeper %q upgrade tree: %w", definition.Key, err)
		}
		canonical.Definitions = append(canonical.Definitions, canonicalDefinition{
			Key:                   definition.Key,
			Name:                  definition.Name,
			CurrentKey:            definition.CurrentKey,
			CurrentName:           definition.CurrentName,
			Sector:                definition.Sector,
			Role:                  definition.Role,
			Rarity:                definition.Rarity,
			BaseRisk:              definition.BaseRisk,
			ExpectedTurbulenceBPS: definition.ExpectedTurbulenceBPS,
			PassiveRuleKey:        definition.PassiveRuleKey,
			PassiveRuleConfig:     passiveConfig,
			UpgradeTree:           upgradeTree,
			BasketMappingKey:      definition.BasketMappingKey,
		})
	}
	sort.Slice(canonical.Definitions, func(i, j int) bool { return canonical.Definitions[i].Key < canonical.Definitions[j].Key })

	encoded, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical catalog: %w", err)
	}
	return encoded, nil
}

func canonicalJSONObject(raw json.RawMessage) (json.RawMessage, error) {
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(decoded)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
