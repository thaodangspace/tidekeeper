package basket

import (
	"testing"

	"github.com/thaodangspace/tidekeepers-server/market/precision"
)

func TestValidWeightUsesCatalogPrecision(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		weight int64
		want   bool
	}{
		{weight: -1, want: false},
		{weight: 0, want: false},
		{weight: 1, want: true},
		{weight: precision.WeightScale, want: true},
		{weight: precision.WeightScale + 1, want: false},
	} {
		if got := ValidWeight(test.weight); got != test.want {
			t.Errorf("ValidWeight(%d) = %t, want %t", test.weight, got, test.want)
		}
	}
}

func TestComponentStatusesAreStableDistinctTokens(t *testing.T) {
	t.Parallel()

	statuses := []ComponentStatus{
		ComponentStatusValid,
		ComponentStatusMissingOpen,
		ComponentStatusMissingClose,
		ComponentStatusOutsideTolerance,
		ComponentStatusInvalidPrice,
		ComponentStatusOutlierFlagged,
		ComponentStatusProviderRejected,
		ComponentStatusDisabledByMapping,
	}
	seen := make(map[ComponentStatus]struct{}, len(statuses))
	for _, status := range statuses {
		if status == "" {
			t.Fatal("empty component status")
		}
		if _, exists := seen[status]; exists {
			t.Fatalf("duplicate component status %q", status)
		}
		seen[status] = struct{}{}
	}
}
