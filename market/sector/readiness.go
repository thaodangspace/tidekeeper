package sector

// ReadinessEntry reports one target Sector's benchmark state for a Daily Tide.
type ReadinessEntry struct {
	Sector                 Key
	Status                 Status
	EligibleBasketCount    int
	MinimumRequiredBaskets int
	Ready                  bool
}

// Readiness is the deterministic settlement gate for the four-sector release.
type Readiness struct {
	Ready   bool
	Sectors []ReadinessEntry
}

// EvaluateReadiness requires exactly one target definition and benchmark result
// for Crest, Ember, Current, and Harbor. Other catalog sectors are irrelevant.
func EvaluateReadiness(definitions []Definition, benchmarks []Benchmark) Readiness {
	definitionBySector := make(map[Key]Definition, len(definitions))
	for _, definition := range definitions {
		if definition.Sector.IsCalculationTarget() {
			definitionBySector[definition.Sector] = definition
		}
	}
	benchmarkBySector := make(map[Key]Benchmark, len(benchmarks))
	for _, benchmark := range benchmarks {
		if benchmark.Sector.IsCalculationTarget() {
			benchmarkBySector[benchmark.Sector] = benchmark
		}
	}

	result := Readiness{Ready: true, Sectors: make([]ReadinessEntry, 0, len(TargetKeys()))}
	for _, sectorKey := range TargetKeys() {
		definition, hasDefinition := definitionBySector[sectorKey]
		benchmark, hasBenchmark := benchmarkBySector[sectorKey]
		entry := ReadinessEntry{Sector: sectorKey}
		if hasDefinition {
			entry.MinimumRequiredBaskets = definition.MinimumEligibleBaskets
		}
		if hasBenchmark {
			entry.Status = benchmark.Status
			entry.EligibleBasketCount = benchmark.EligibleBasketCount
		}
		entry.Ready = hasDefinition && hasBenchmark && benchmark.Status == StatusReady &&
			benchmark.EligibleBasketCount >= definition.MinimumEligibleBaskets
		if !entry.Ready {
			result.Ready = false
		}
		result.Sectors = append(result.Sectors, entry)
	}
	return result
}
