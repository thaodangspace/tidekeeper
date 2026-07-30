package keeper

import (
	"slices"
	"strings"
	"testing"
)

func TestCalculationContentValidation(t *testing.T) {
	t.Parallel()

	release := validCalculationRelease()
	if err := Validate(release); err != nil {
		t.Fatalf("Validate(valid calculation release): %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*CatalogRelease)
		want   string
	}{
		{
			name: "insufficient benchmark baskets",
			mutate: func(release *CatalogRelease) {
				for index := range release.Baskets {
					if release.Baskets[index].Sector == "HARBOR" {
						release.Baskets[index].BenchmarkEligible = false
						return
					}
				}
			},
			want: "eligible baskets",
		},
		{
			name: "unknown policy",
			mutate: func(release *CatalogRelease) {
				release.Baskets[0].ExpectedTurbulencePolicyKey = "missing_policy"
			},
			want: "unknown turbulence policy",
		},
		{
			name: "keeper basket sector mismatch",
			mutate: func(release *CatalogRelease) {
				release.Definitions[0].Sector = "EMBER"
			},
			want: "does not match basket sector",
		},
		{
			name: "invalid blend",
			mutate: func(release *CatalogRelease) {
				release.SectorDefinitions[0].RankBlendWeight--
			},
			want: "invalid calculation definition",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := validCalculationRelease()
			test.mutate(&candidate)
			if err := Validate(candidate); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, test.want)
			}
		})
	}
}

func TestCalculationContentChecksumIsOrderIndependent(t *testing.T) {
	t.Parallel()

	release := validCalculationRelease()
	want, err := Checksum(release)
	if err != nil {
		t.Fatalf("Checksum(): %v", err)
	}
	slices.Reverse(release.SectorDefinitions)
	slices.Reverse(release.TurbulencePolicies)
	got, err := Checksum(release)
	if err != nil {
		t.Fatalf("Checksum(reordered): %v", err)
	}
	if got != want {
		t.Fatal("checksum changed when calculation content order changed")
	}
}

func validCalculationRelease() CatalogRelease {
	release := CatalogV1()
	release.Version = 2
	release.SectorDefinitions = []SectorDefinition{
		{Sector: "CREST", BenchmarkMethod: "EQUAL_WEIGHT", MinimumEligibleBaskets: 2, RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: 1_000_000},
		{Sector: "EMBER", BenchmarkMethod: "EQUAL_WEIGHT", MinimumEligibleBaskets: 2, RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: 1_000_000},
		{Sector: "CURRENT", BenchmarkMethod: "EQUAL_WEIGHT", MinimumEligibleBaskets: 2, RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: 1_000_000},
		{Sector: "HARBOR", BenchmarkMethod: "EQUAL_WEIGHT", MinimumEligibleBaskets: 2, RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: 1_000_000},
	}
	release.TurbulencePolicies = []ExpectedTurbulencePolicy{
		{Key: "crest_static", Type: "STATIC_CONTENT_VALUE", StaticValue: 4_000_000, Floor: 250_000, RoundingMode: "ROUND_HALF_AWAY_FROM_ZERO"},
		{Key: "ember_static", Type: "STATIC_CONTENT_VALUE", StaticValue: 15_000_000, Floor: 1_000_000, RoundingMode: "ROUND_HALF_AWAY_FROM_ZERO"},
		{Key: "current_static", Type: "STATIC_CONTENT_VALUE", StaticValue: 8_000_000, Floor: 500_000, RoundingMode: "ROUND_HALF_AWAY_FROM_ZERO"},
		{Key: "harbor_static", Type: "STATIC_CONTENT_VALUE", StaticValue: 500_000, Floor: 250_000, RoundingMode: "ROUND_HALF_AWAY_FROM_ZERO"},
	}
	sectorByBasket := map[string]string{
		"crest_large_cap": "CREST", "harbor_stability": "HARBOR", "forge_commerce": "HARBOR",
		"exchange_flow": "CURRENT", "oracle_signal": "CURRENT", "veiled_reversal": "EMBER",
		"lending_growth": "EMBER", "wild_momentum": "EMBER", "engine_infrastructure": "CREST", "second_current": "CURRENT",
	}
	policyBySector := map[string]string{"CREST": "crest_static", "EMBER": "ember_static", "CURRENT": "current_static", "HARBOR": "harbor_static"}
	for index := range release.Baskets {
		basket := &release.Baskets[index]
		basket.Sector = sectorByBasket[basket.Key]
		basket.MinimumCoveredWeight = 80_000_000
		basket.ExpectedTurbulencePolicyKey = policyBySector[basket.Sector]
		basket.NormalizationCap = 2_000_000
		basket.BenchmarkEligible = true
	}
	for index := range release.Definitions {
		release.Definitions[index].Sector = sectorByBasket[release.Definitions[index].BasketMappingKey]
	}
	return release
}
