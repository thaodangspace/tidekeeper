package sector

import (
	"testing"

	"github.com/thaodangspace/tidekeepers-server/market/precision"
)

func TestScoreCalculatorUsesVersionedRelativeRankBlend(t *testing.T) {
	t.Parallel()

	result, err := (ScoreCalculator{}).Score(ScoreInput{
		Definition:                  benchmarkInput(nil).Definition,
		KeeperNormalizedPerformance: 800_000,
		SectorBenchmark:             333_333,
		SectorPercentile:            precision.NormalizedScale,
	})
	if err != nil {
		t.Fatalf("Score(): %v", err)
	}
	if result.RelativePerformance != 466_667 {
		t.Errorf("relative performance = %d, want 466667", result.RelativePerformance)
	}
	if result.RelativeComponent != 933_334 {
		t.Errorf("relative component = %d, want 933334", result.RelativeComponent)
	}
	if result.RankComponent != precision.NormalizedScale {
		t.Errorf("rank component = %d, want %d", result.RankComponent, precision.NormalizedScale)
	}
	if result.NormalizedScore != 953_334 || result.ScorePoints != 95 {
		t.Errorf("score = %#v, want normalized 953334 points 95", result)
	}
}

func TestScoreCalculatorClampsAndRejectsInvalidPercentile(t *testing.T) {
	t.Parallel()

	result, err := (ScoreCalculator{}).Score(ScoreInput{
		Definition:                  benchmarkInput(nil).Definition,
		KeeperNormalizedPerformance: -2_000_000,
		SectorBenchmark:             1_000_000,
		SectorPercentile:            0,
	})
	if err != nil {
		t.Fatalf("Score(): %v", err)
	}
	if result.RelativeComponent != -precision.NormalizedScale || result.NormalizedScore != -precision.NormalizedScale || result.ScorePoints != -100 {
		t.Errorf("clamped score = %#v", result)
	}
	_, err = (ScoreCalculator{}).Score(ScoreInput{Definition: benchmarkInput(nil).Definition, SectorPercentile: precision.NormalizedScale + 1})
	if err == nil {
		t.Fatal("out-of-range percentile accepted")
	}
}
