package voyage

import "fmt"

// Hull is a Voyage's bounded survival resource.
//
// Its fields are private so callers must use NewHull to construct valid values.
type Hull struct {
	current int32
	maximum int32
}

// NewHull constructs a Hull with a valid current value and maximum capacity.
func NewHull(current, maximum int32) (Hull, error) {
	if maximum <= 0 {
		return Hull{}, fmt.Errorf("maximum hull must be greater than zero")
	}
	if current < 0 || current > maximum {
		return Hull{}, fmt.Errorf("current hull must be between zero and maximum")
	}

	return Hull{current: current, maximum: maximum}, nil
}

// Current returns the current Hull balance.
func (h Hull) Current() int32 {
	return h.current
}

// Maximum returns the current Hull capacity.
func (h Hull) Maximum() int32 {
	return h.maximum
}

// IsDestroyed reports whether the Voyage has no remaining Hull.
func (h Hull) IsDestroyed() bool {
	return h.current == 0
}

// RatioBasisPoints returns current Hull as a percentage of maximum Hull in basis
// points. Hull values must be constructed with NewHull before calling this method.
func (h Hull) RatioBasisPoints() int32 {
	return int32(int64(h.current) * 10_000 / int64(h.maximum))
}

// HullState is a display classification derived from the current Hull ratio.
type HullState string

const (
	HullStateStable    HullState = "STABLE"
	HullStateDamaged   HullState = "DAMAGED"
	HullStateCritical  HullState = "CRITICAL"
	HullStateDestroyed HullState = "DESTROYED"
)

// State returns the derived display state for the Hull balance.
func (h Hull) State() HullState {
	if h.IsDestroyed() {
		return HullStateDestroyed
	}

	switch ratio := h.RatioBasisPoints(); {
	case ratio <= 2_500:
		return HullStateCritical
	case ratio <= 6_000:
		return HullStateDamaged
	default:
		return HullStateStable
	}
}

// HullChange describes a bounded Hull mutation.
type HullChange struct {
	Before         int32
	RequestedDelta int32
	EffectiveDelta int32
	After          int32
	Capped         bool
	Destroyed      bool
}

// ApplyDelta applies delta while clamping the resulting balance to [0, Maximum].
// It uses int64 for the intermediate addition to prevent integer overflow.
func (h Hull) ApplyDelta(delta int32) (Hull, HullChange) {
	before := h.current
	requestedAfter := int64(before) + int64(delta)

	after := requestedAfter
	if after < 0 {
		after = 0
	}
	if after > int64(h.maximum) {
		after = int64(h.maximum)
	}

	next := Hull{
		current: int32(after),
		maximum: h.maximum,
	}
	effectiveDelta := next.current - before

	return next, HullChange{
		Before:         before,
		RequestedDelta: delta,
		EffectiveDelta: effectiveDelta,
		After:          next.current,
		Capped:         effectiveDelta != delta,
		Destroyed:      next.IsDestroyed(),
	}
}
