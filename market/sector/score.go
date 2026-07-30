package sector

import (
	"fmt"

	"github.com/thaodangspace/tidekeepers-server/market/precision"
)

// ScoreInput is the immutable data needed to score one Keeper definition against a Sector benchmark.
type ScoreInput struct {
	Definition                  Definition
	KeeperNormalizedPerformance int64
	SectorBenchmark             int64
	SectorPercentile            int64
}

// ScoreOutput contains every persisted component of the Sector score curve.
type ScoreOutput struct {
	RelativePerformance int64
	RelativeComponent   int64
	RankComponent       int64
	NormalizedScore     int64
	ScorePoints         int64
}

// ScoreCalculator applies the versioned relative/rank score curve.
type ScoreCalculator struct{}

func (ScoreCalculator) Score(input ScoreInput) (ScoreOutput, error) {
	if err := input.Definition.Validate(); err != nil {
		return ScoreOutput{}, err
	}
	if input.SectorPercentile < 0 || input.SectorPercentile > precision.NormalizedScale {
		return ScoreOutput{}, fmt.Errorf("sector percentile out of bounds")
	}
	relative, err := precision.Subtract(input.KeeperNormalizedPerformance, input.SectorBenchmark)
	if err != nil {
		return ScoreOutput{}, err
	}
	relativeComponent, err := precision.MultiplyDivide(relative, precision.NormalizedScale, input.Definition.RelativeScale)
	if err != nil {
		return ScoreOutput{}, err
	}
	relativeComponent, err = precision.Clamp(relativeComponent, -precision.NormalizedScale, precision.NormalizedScale)
	if err != nil {
		return ScoreOutput{}, err
	}
	doubledPercentile, err := precision.MultiplyDivide(input.SectorPercentile, 2, 1)
	if err != nil {
		return ScoreOutput{}, err
	}
	rankComponent, err := precision.Subtract(doubledPercentile, precision.NormalizedScale)
	if err != nil {
		return ScoreOutput{}, err
	}
	relativeContribution, err := precision.MultiplyDivide(relativeComponent, input.Definition.RelativeBlendWeight, precision.NormalizedScale)
	if err != nil {
		return ScoreOutput{}, err
	}
	rankContribution, err := precision.MultiplyDivide(rankComponent, input.Definition.RankBlendWeight, precision.NormalizedScale)
	if err != nil {
		return ScoreOutput{}, err
	}
	normalized, err := precision.Add(relativeContribution, rankContribution)
	if err != nil {
		return ScoreOutput{}, err
	}
	normalized, err = precision.Clamp(normalized, -input.Definition.ScoreCap, input.Definition.ScoreCap)
	if err != nil {
		return ScoreOutput{}, err
	}
	points, err := precision.MultiplyDivide(normalized, 100, precision.NormalizedScale)
	if err != nil {
		return ScoreOutput{}, err
	}
	return ScoreOutput{
		RelativePerformance: relative,
		RelativeComponent:   relativeComponent,
		RankComponent:       rankComponent,
		NormalizedScore:     normalized,
		ScorePoints:         points,
	}, nil
}
