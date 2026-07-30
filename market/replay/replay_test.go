package replay

import (
	"errors"
	"testing"
	"time"

	"github.com/thaodangspace/tidekeepers-server/market/basket"
	"github.com/thaodangspace/tidekeepers-server/market/sector"
)

func TestVerifyBasketAndBenchmark(t *testing.T) {
	t.Parallel()

	basketInput := basket.CalculationInput{
		Mapping: basket.MappingVersion{ID: "mapping", Key: "crest", Sector: sector.Crest, MinimumCoveredWeight: 100_000_000, NormalizationCap: 2_000_000, ExpectedTurbulence: basket.ExpectedTurbulencePolicy{Type: basket.TurbulencePolicyStaticContentValue, StaticValue: 1, Floor: 1}, Components: []basket.MappingComponent{{MarketAssetID: "a", TargetWeight: 50_000_000, Enabled: true}, {MarketAssetID: "b", TargetWeight: 50_000_000, Enabled: true}}},
		Window:  basket.CalculationWindow{DailyTideID: "tide", ProviderKey: "fixture", Start: time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC), End: time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)},
	}
	basketExpected, err := (basket.Calculator{}).Calculate(basketInput)
	if err != nil {
		t.Fatalf("Calculate basket: %v", err)
	}
	if err := VerifyBasket(basketInput, basketExpected); err != nil {
		t.Fatalf("VerifyBasket: %v", err)
	}
	basketExpected.Status = basket.MetricStatusReady
	if err := VerifyBasket(basketInput, basketExpected); !errors.Is(err, ErrMismatch) {
		t.Fatalf("VerifyBasket mismatch = %v", err)
	}

	first, second := int64(-100), int64(100)
	benchmarkInput := sector.BenchmarkInput{DailyTideID: "tide", Definition: sector.Definition{ID: "crest-v2", Sector: sector.Crest, BenchmarkMethod: sector.EqualWeight, MinimumEligibleBaskets: 2, RelativeScale: 500000, RelativeBlendWeight: 700000, RankBlendWeight: 300000, ScoreCap: 1000000}, Baskets: []sector.BenchmarkBasket{{BasketMetricID: "metric-a", BasketMappingVersionID: "map-a", Sector: sector.Crest, BenchmarkEligible: true, MetricStatus: "READY", NormalizedPerformance: &first}, {BasketMetricID: "metric-b", BasketMappingVersionID: "map-b", Sector: sector.Crest, BenchmarkEligible: true, MetricStatus: "READY", NormalizedPerformance: &second}}}
	benchmarkExpected, err := (sector.BenchmarkCalculator{}).Calculate(benchmarkInput)
	if err != nil {
		t.Fatalf("Calculate benchmark: %v", err)
	}
	if err := VerifyBenchmark(benchmarkInput, benchmarkExpected); err != nil {
		t.Fatalf("VerifyBenchmark: %v", err)
	}
	benchmarkExpected.EligibleBasketCount++
	if err := VerifyBenchmark(benchmarkInput, benchmarkExpected); !errors.Is(err, ErrMismatch) {
		t.Fatalf("VerifyBenchmark mismatch = %v", err)
	}
}
