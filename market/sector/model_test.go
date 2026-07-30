package sector

import (
	"errors"
	"testing"

	"github.com/thaodangspace/tidekeepers-server/market/precision"
)

func TestTargetKeysAreCanonicalAndLimitedToFourSectors(t *testing.T) {
	t.Parallel()

	want := []Key{Crest, Ember, Current, Harbor}
	got := TargetKeys()
	if len(got) != len(want) {
		t.Fatalf("TargetKeys() len = %d, want %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("TargetKeys()[%d] = %q, want %q", index, got[index], want[index])
		}
	}
	if err := ValidateCalculationTarget("FORGE"); !errors.Is(err, ErrUnsupportedCalculationSector) {
		t.Fatalf("ValidateCalculationTarget(FORGE) = %v, want ErrUnsupportedCalculationSector", err)
	}
}

func TestDefinitionValidation(t *testing.T) {
	t.Parallel()

	definition := Definition{
		ID:                     "sector-crest-v2",
		Sector:                 Crest,
		BenchmarkMethod:        EqualWeight,
		MinimumEligibleBaskets: 2,
		RelativeScale:          500_000,
		RelativeBlendWeight:    700_000,
		RankBlendWeight:        300_000,
		ScoreCap:               precision.ScoreScale * 100,
	}
	if err := definition.Validate(); err != nil {
		t.Fatalf("valid definition rejected: %v", err)
	}

	definition.RankBlendWeight = 299_999
	if err := definition.Validate(); err == nil {
		t.Fatal("invalid blend accepted")
	}
}
