package buildinfo

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	got := String()
	for _, want := range []string{"tidekeepers-api", "version=", "commit=", "built="} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, want it to contain %q", got, want)
		}
	}
}
