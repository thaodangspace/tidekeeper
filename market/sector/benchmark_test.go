package sector

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/thaodangspace/tidekeepers-server/market/precision"
)

func TestBenchmarkCalculatorComputesEqualWeightRankedBenchmark(t *testing.T) {
	t.Parallel()

	result, err := (BenchmarkCalculator{}).Calculate(benchmarkInput([]BenchmarkBasket{
		readyBasket("metric-c", "mapping-c", 800_000),
		readyBasket("metric-a", "mapping-a", -300_000),
		readyBasket("metric-b", "mapping-b", 500_000),
	}))
	if err != nil {
		t.Fatalf("Calculate(): %v", err)
	}
	if result.Status != StatusReady || result.EligibleBasketCount != 3 {
		t.Fatalf("benchmark status/count = %s/%d", result.Status, result.EligibleBasketCount)
	}
	if got := value(result.Value); got != 333_333 {
		t.Errorf("benchmark value = %d, want 333333", got)
	}
	if got := result.Members[0].Percentile; got != 0 {
		t.Errorf("mapping-a percentile = %d, want 0", got)
	}
	if got := result.Members[1].Percentile; got != 500_000 {
		t.Errorf("mapping-b percentile = %d, want 500000", got)
	}
	if got := result.Members[2].Percentile; got != precision.NormalizedScale {
		t.Errorf("mapping-c percentile = %d, want %d", got, precision.NormalizedScale)
	}
}

func TestBenchmarkCalculatorHandlesTiesAndIncompleteInput(t *testing.T) {
	t.Parallel()

	result, err := (BenchmarkCalculator{}).Calculate(benchmarkInput([]BenchmarkBasket{
		readyBasket("metric-a", "mapping-a", -200_000),
		readyBasket("metric-b", "mapping-b", 300_000),
		readyBasket("metric-c", "mapping-c", 300_000),
		readyBasket("metric-d", "mapping-d", 800_000),
	}))
	if err != nil {
		t.Fatalf("Calculate(): %v", err)
	}
	if result.Members[1].Percentile != 500_000 || result.Members[2].Percentile != 500_000 {
		t.Fatalf("tied percentiles = %d/%d, want 500000/500000", result.Members[1].Percentile, result.Members[2].Percentile)
	}

	incomplete, err := (BenchmarkCalculator{}).Calculate(benchmarkInput([]BenchmarkBasket{readyBasket("metric-a", "mapping-a", 10)}))
	if err != nil {
		t.Fatalf("incomplete Calculate(): %v", err)
	}
	if incomplete.Status != StatusDataIncomplete || incomplete.Value != nil {
		t.Fatalf("incomplete result = %#v", incomplete)
	}
}

func TestBenchmarkCalculatorRejectsDuplicateAndIsOrderIndependent(t *testing.T) {
	t.Parallel()

	baskets := []BenchmarkBasket{
		readyBasket("metric-a", "mapping-a", -100),
		readyBasket("metric-b", "mapping-b", 100),
	}
	first, err := (BenchmarkCalculator{}).Calculate(benchmarkInput(baskets))
	if err != nil {
		t.Fatalf("first Calculate(): %v", err)
	}
	slices.Reverse(baskets)
	second, err := (BenchmarkCalculator{}).Calculate(benchmarkInput(baskets))
	if err != nil {
		t.Fatalf("second Calculate(): %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("input order changed benchmark output")
	}

	_, err = (BenchmarkCalculator{}).Calculate(benchmarkInput([]BenchmarkBasket{
		readyBasket("metric-a", "mapping-a", 1), readyBasket("metric-b", "mapping-a", 2),
	}))
	if !errors.Is(err, ErrDuplicateBasketMapping) {
		t.Fatalf("duplicate error = %v, want ErrDuplicateBasketMapping", err)
	}
}

func benchmarkInput(baskets []BenchmarkBasket) BenchmarkInput {
	return BenchmarkInput{
		DailyTideID: "tide-1",
		Definition: Definition{
			ID: "crest-v2", Sector: Crest, BenchmarkMethod: EqualWeight, MinimumEligibleBaskets: 2,
			RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: precision.NormalizedScale,
		},
		Baskets: baskets,
	}
}

func readyBasket(metricID, mappingID string, normalized int64) BenchmarkBasket {
	return BenchmarkBasket{
		BasketMetricID: metricID, BasketMappingVersionID: mappingID, Sector: Crest,
		BenchmarkEligible: true, MetricStatus: "READY", NormalizedPerformance: &normalized,
	}
}

func value(pointer *int64) int64 {
	if pointer == nil {
		return 0
	}
	return *pointer
}
