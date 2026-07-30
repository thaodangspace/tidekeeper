package sector

import "testing"

func TestEvaluateReadinessRequiresExactlyFourReadyTargetSectors(t *testing.T) {
	t.Parallel()

	definitions := []Definition{
		readyDefinition(Crest), readyDefinition(Ember), readyDefinition(Current), readyDefinition(Harbor),
	}
	benchmarks := []Benchmark{
		{Sector: Crest, Status: StatusReady, EligibleBasketCount: 2},
		{Sector: Ember, Status: StatusReady, EligibleBasketCount: 2},
		{Sector: Current, Status: StatusReady, EligibleBasketCount: 2},
		{Sector: Harbor, Status: StatusReady, EligibleBasketCount: 2},
		{Sector: "FORGE", Status: StatusDataIncomplete},
	}
	result := EvaluateReadiness(definitions, benchmarks)
	if !result.Ready || len(result.Sectors) != 4 {
		t.Fatalf("readiness = %#v, want all targets ready", result)
	}

	benchmarks[2].Status = StatusDataIncomplete
	result = EvaluateReadiness(definitions, benchmarks)
	if result.Ready || result.Sectors[2].Ready {
		t.Fatalf("incomplete Current benchmark did not block readiness: %#v", result)
	}
}

func TestEvaluateReadinessBlocksMissingDefinitionOrBenchmark(t *testing.T) {
	t.Parallel()

	result := EvaluateReadiness([]Definition{readyDefinition(Crest)}, []Benchmark{{Sector: Crest, Status: StatusReady, EligibleBasketCount: 2}})
	if result.Ready || len(result.Sectors) != 4 {
		t.Fatalf("partial inputs unexpectedly ready: %#v", result)
	}
	for _, entry := range result.Sectors {
		if entry.Sector != Crest && entry.Ready {
			t.Errorf("missing %s unexpectedly ready", entry.Sector)
		}
	}
}

func readyDefinition(key Key) Definition {
	return Definition{ID: string(key) + "-v2", Sector: key, BenchmarkMethod: EqualWeight, MinimumEligibleBaskets: 2,
		RelativeScale: 500_000, RelativeBlendWeight: 700_000, RankBlendWeight: 300_000, ScoreCap: 1_000_000}
}
