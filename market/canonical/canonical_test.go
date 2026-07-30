package canonical

import (
	"reflect"
	"testing"
)

func TestSortedStringsReturnsIndependentCanonicalOrder(t *testing.T) {
	t.Parallel()

	original := []string{"harbor", "crest", "ember"}
	got := SortedStrings(original)
	want := []string{"crest", "ember", "harbor"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("SortedStrings() = %v, want %v", got, want)
	}
	if original[0] != "harbor" {
		t.Errorf("SortedStrings mutated input: %v", original)
	}
}

func TestChecksumUsesUnambiguousPartBoundaries(t *testing.T) {
	t.Parallel()

	left := Checksum("ab", "c")
	right := Checksum("a", "bc")
	if left == right {
		t.Fatal("different part boundaries produced the same checksum")
	}
	if got := Checksum("crest", "ember"); got != Checksum("crest", "ember") {
		t.Fatal("identical inputs produced different checksums")
	}
}
