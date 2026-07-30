package voyage

import (
	"math"
	"testing"
)

func TestNewHull(t *testing.T) {
	tests := []struct {
		name    string
		current int32
		maximum int32
		wantErr bool
	}{
		{name: "valid", current: 80, maximum: 100},
		{name: "zero current is valid", current: 0, maximum: 100},
		{name: "rejects non-positive maximum", current: 0, maximum: 0, wantErr: true},
		{name: "rejects negative current", current: -1, maximum: 100, wantErr: true},
		{name: "rejects current above maximum", current: 101, maximum: 100, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hull, err := NewHull(tt.current, tt.maximum)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewHull(%d, %d) error = %v, wantErr %t", tt.current, tt.maximum, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got := hull.Current(); got != tt.current {
				t.Errorf("Current() = %d, want %d", got, tt.current)
			}
			if got := hull.Maximum(); got != tt.maximum {
				t.Errorf("Maximum() = %d, want %d", got, tt.maximum)
			}
		})
	}
}

func TestHullApplyDelta(t *testing.T) {
	tests := []struct {
		name               string
		current            int32
		maximum            int32
		delta              int32
		wantAfter          int32
		wantEffectiveDelta int32
		wantCapped         bool
		wantDestroyed      bool
	}{
		{name: "ordinary damage", current: 80, maximum: 100, delta: -12, wantAfter: 68, wantEffectiveDelta: -12},
		{name: "lethal damage", current: 8, maximum: 100, delta: -12, wantAfter: 0, wantEffectiveDelta: -8, wantCapped: true, wantDestroyed: true},
		{name: "ordinary healing", current: 70, maximum: 100, delta: 10, wantAfter: 80, wantEffectiveDelta: 10},
		{name: "capped healing", current: 95, maximum: 100, delta: 10, wantAfter: 100, wantEffectiveDelta: 5, wantCapped: true},
		{name: "maximum positive delta cannot overflow", current: math.MaxInt32 - 10, maximum: math.MaxInt32, delta: math.MaxInt32, wantAfter: math.MaxInt32, wantEffectiveDelta: 10, wantCapped: true},
		{name: "minimum negative delta cannot overflow", current: 10, maximum: 100, delta: math.MinInt32, wantAfter: 0, wantEffectiveDelta: -10, wantCapped: true, wantDestroyed: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hull, err := NewHull(tt.current, tt.maximum)
			if err != nil {
				t.Fatalf("NewHull() error = %v", err)
			}

			next, change := hull.ApplyDelta(tt.delta)
			if got := next.Current(); got != tt.wantAfter {
				t.Errorf("next.Current() = %d, want %d", got, tt.wantAfter)
			}
			if got := next.Maximum(); got != tt.maximum {
				t.Errorf("next.Maximum() = %d, want %d", got, tt.maximum)
			}
			if got := change.Before; got != tt.current {
				t.Errorf("change.Before = %d, want %d", got, tt.current)
			}
			if got := change.RequestedDelta; got != tt.delta {
				t.Errorf("change.RequestedDelta = %d, want %d", got, tt.delta)
			}
			if got := change.EffectiveDelta; got != tt.wantEffectiveDelta {
				t.Errorf("change.EffectiveDelta = %d, want %d", got, tt.wantEffectiveDelta)
			}
			if got := change.After; got != tt.wantAfter {
				t.Errorf("change.After = %d, want %d", got, tt.wantAfter)
			}
			if got := change.Capped; got != tt.wantCapped {
				t.Errorf("change.Capped = %t, want %t", got, tt.wantCapped)
			}
			if got := change.Destroyed; got != tt.wantDestroyed {
				t.Errorf("change.Destroyed = %t, want %t", got, tt.wantDestroyed)
			}
		})
	}
}

func TestHullState(t *testing.T) {
	tests := []struct {
		name    string
		current int32
		want    HullState
	}{
		{name: "stable", current: 61, want: HullStateStable},
		{name: "damaged at upper boundary", current: 60, want: HullStateDamaged},
		{name: "damaged", current: 26, want: HullStateDamaged},
		{name: "critical at upper boundary", current: 25, want: HullStateCritical},
		{name: "critical", current: 1, want: HullStateCritical},
		{name: "destroyed", current: 0, want: HullStateDestroyed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hull, err := NewHull(tt.current, 100)
			if err != nil {
				t.Fatalf("NewHull() error = %v", err)
			}
			if got := hull.State(); got != tt.want {
				t.Errorf("State() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHullRatioBasisPoints(t *testing.T) {
	tests := []struct {
		name    string
		current int32
		maximum int32
		want    int32
	}{
		{name: "truncates fractional basis points", current: 1, maximum: 3, want: 3_333},
		{name: "maximum values do not overflow", current: math.MaxInt32, maximum: math.MaxInt32, want: 10_000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hull, err := NewHull(tt.current, tt.maximum)
			if err != nil {
				t.Fatalf("NewHull() error = %v", err)
			}
			if got := hull.RatioBasisPoints(); got != tt.want {
				t.Errorf("RatioBasisPoints() = %d, want %d", got, tt.want)
			}
		})
	}
}
