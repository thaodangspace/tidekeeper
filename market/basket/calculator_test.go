package basket

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/thaodangspace/tidekeepers-server/market/precision"
	"github.com/thaodangspace/tidekeepers-server/market/sector"
)

func TestCalculatorCalculatesReadyMetricWithEvidence(t *testing.T) {
	t.Parallel()

	metric, err := (Calculator{}).Calculate(calculationInput(80_000_000, fullObservations()))
	if err != nil {
		t.Fatalf("Calculate(): %v", err)
	}
	if metric.Status != MetricStatusReady || metric.CoveredWeight != precision.WeightScale {
		t.Fatalf("metric status/coverage = %s/%d", metric.Status, metric.CoveredWeight)
	}
	if got := dereference(metric.RawReturn); got != 2_400_000 {
		t.Errorf("raw return = %d, want 2400000", got)
	}
	if got := dereference(metric.NormalizedPerformance); got != 600_000 {
		t.Errorf("normalized performance = %d, want 600000", got)
	}
	wantContributions := []int64{2_000_000, 600_000, -200_000}
	for index, component := range metric.Components {
		if component.Status != ComponentStatusValid {
			t.Errorf("component %s status = %s, want VALID", component.MarketAssetID, component.Status)
		}
		if got := dereference(component.Contribution); got != wantContributions[index] {
			t.Errorf("component %s contribution = %d, want %d", component.MarketAssetID, got, wantContributions[index])
		}
	}
}

func TestCalculatorRenormalizesQualifyingPartialCoverageExactly(t *testing.T) {
	t.Parallel()

	observations := fullObservations()
	observations = observations[:4] // asset-c has neither observation.
	metric, err := (Calculator{}).Calculate(calculationInput(80_000_000, observations))
	if err != nil {
		t.Fatalf("Calculate(): %v", err)
	}
	if metric.Status != MetricStatusReady || metric.CoveredWeight != 80_000_000 {
		t.Fatalf("metric status/coverage = %s/%d", metric.Status, metric.CoveredWeight)
	}
	var effectiveWeight int64
	for _, component := range metric.Components {
		effectiveWeight += component.EffectiveWeight
	}
	if effectiveWeight != precision.WeightScale {
		t.Errorf("effective weights = %d, want %d", effectiveWeight, precision.WeightScale)
	}
	if got := dereference(metric.RawReturn); got != 3_250_000 {
		t.Errorf("raw return = %d, want 3250000", got)
	}
	if metric.Components[2].Status != ComponentStatusMissingOpen {
		t.Errorf("missing component status = %s, want MISSING_OPEN", metric.Components[2].Status)
	}
}

func TestCalculatorReturnsIncompleteWithoutFabricatedValues(t *testing.T) {
	t.Parallel()

	observations := fullObservations()[:4]
	metric, err := (Calculator{}).Calculate(calculationInput(85_000_000, observations))
	if err != nil {
		t.Fatalf("Calculate(): %v", err)
	}
	if metric.Status != MetricStatusDataIncomplete {
		t.Fatalf("status = %s, want DATA_INCOMPLETE", metric.Status)
	}
	if metric.RawReturn != nil || metric.NormalizedPerformance != nil || metric.ExpectedTurbulence != nil {
		t.Fatal("incomplete metric has fabricated aggregate values")
	}
}

func TestCalculatorIsInputOrderIndependentAndClassifiesInvalidData(t *testing.T) {
	t.Parallel()

	input := calculationInput(80_000_000, fullObservations())
	first, err := (Calculator{}).Calculate(input)
	if err != nil {
		t.Fatalf("first Calculate(): %v", err)
	}
	slices.Reverse(input.Observations)
	second, err := (Calculator{}).Calculate(input)
	if err != nil {
		t.Fatalf("second Calculate(): %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("reordered observations changed calculated metric")
	}

	invalid := calculationInput(80_000_000, fullObservations())
	invalid.Observations[0].PriceUnits = 0
	metric, err := (Calculator{}).Calculate(invalid)
	if err != nil {
		t.Fatalf("invalid observation Calculate(): %v", err)
	}
	if metric.Components[0].Status != ComponentStatusInvalidPrice {
		t.Errorf("invalid price status = %s, want INVALID_PRICE", metric.Components[0].Status)
	}
}

func calculationInput(minimumCoverage int64, observations []Observation) CalculationInput {
	start := time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC)
	return CalculationInput{
		Mapping: MappingVersion{
			ID: "mapping-crest-1", Key: "crest_core", Sector: sector.Crest,
			MinimumCoveredWeight: minimumCoverage, NormalizationCap: 2_000_000,
			ExpectedTurbulence: ExpectedTurbulencePolicy{ID: "crest-static-1", Type: TurbulencePolicyStaticContentValue, StaticValue: 4_000_000, Floor: 250_000},
			Components: []MappingComponent{
				{MarketAssetID: "asset-a", TargetWeight: 50_000_000, Enabled: true},
				{MarketAssetID: "asset-b", TargetWeight: 30_000_000, Enabled: true},
				{MarketAssetID: "asset-c", TargetWeight: 20_000_000, Enabled: true},
			},
		},
		Window:       CalculationWindow{DailyTideID: "tide-1", ProviderKey: "fixture", Start: start, End: start.Add(24 * time.Hour), Tolerance: 5 * time.Minute},
		Observations: observations,
	}
}

func fullObservations() []Observation {
	start := time.Date(2026, time.July, 30, 0, 0, 0, 0, time.UTC)
	return []Observation{
		{ID: "a-open", ProviderKey: "fixture", MarketAssetID: "asset-a", ObservedAt: start, PriceUnits: 100, Status: ObservationStatusValid},
		{ID: "a-close", ProviderKey: "fixture", MarketAssetID: "asset-a", ObservedAt: start.Add(24 * time.Hour), PriceUnits: 104, Status: ObservationStatusValid},
		{ID: "b-open", ProviderKey: "fixture", MarketAssetID: "asset-b", ObservedAt: start, PriceUnits: 100, Status: ObservationStatusValid},
		{ID: "b-close", ProviderKey: "fixture", MarketAssetID: "asset-b", ObservedAt: start.Add(24 * time.Hour), PriceUnits: 102, Status: ObservationStatusValid},
		{ID: "c-open", ProviderKey: "fixture", MarketAssetID: "asset-c", ObservedAt: start, PriceUnits: 100, Status: ObservationStatusValid},
		{ID: "c-close", ProviderKey: "fixture", MarketAssetID: "asset-c", ObservedAt: start.Add(24 * time.Hour), PriceUnits: 99, Status: ObservationStatusValid},
	}
}

func dereference(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
