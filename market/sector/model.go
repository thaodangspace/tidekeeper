// Package sector defines pure four-sector market calculation types.
package sector

import (
	"errors"

	"github.com/thaodangspace/tidekeepers-server/market/precision"
)

// Key identifies a gameplay Sector.
type Key string

const (
	Crest   Key = "CREST"
	Ember   Key = "EMBER"
	Current Key = "CURRENT"
	Harbor  Key = "HARBOR"
)

var ErrUnsupportedCalculationSector = errors.New("unsupported calculation sector")

// TargetKeys returns the four MVP Sectors in their canonical order.
func TargetKeys() []Key {
	return []Key{Crest, Ember, Current, Harbor}
}

// IsCalculationTarget reports whether key is part of the four-sector MVP.
func (key Key) IsCalculationTarget() bool {
	switch key {
	case Crest, Ember, Current, Harbor:
		return true
	default:
		return false
	}
}

// ValidateCalculationTarget rejects sectors outside the first calculation release.
func ValidateCalculationTarget(key Key) error {
	if !key.IsCalculationTarget() {
		return ErrUnsupportedCalculationSector
	}
	return nil
}

// BenchmarkMethod controls how ready basket metrics contribute to a benchmark.
type BenchmarkMethod string

const EqualWeight BenchmarkMethod = "EQUAL_WEIGHT"

// Status is the durable state of one Sector benchmark.
type Status string

const (
	StatusReady          Status = "READY"
	StatusDataIncomplete Status = "DATA_INCOMPLETE"
)

// Definition is the immutable calculation policy for one Sector content version.
type Definition struct {
	ID                     string
	Sector                 Key
	BenchmarkMethod        BenchmarkMethod
	MinimumEligibleBaskets int
	RelativeScale          int64
	RelativeBlendWeight    int64
	RankBlendWeight        int64
	ScoreCap               int64
}

// Validate confirms that a definition has the phase-one invariant shape. More
// content-reference validation belongs to the catalog publication phase.
func (definition Definition) Validate() error {
	if err := ValidateCalculationTarget(definition.Sector); err != nil {
		return err
	}
	if definition.BenchmarkMethod != EqualWeight || definition.MinimumEligibleBaskets < 2 ||
		definition.RelativeScale <= 0 || definition.ScoreCap <= 0 ||
		definition.RelativeBlendWeight < 0 || definition.RankBlendWeight < 0 {
		return errors.New("invalid sector definition")
	}
	blend, err := precision.Add(definition.RelativeBlendWeight, definition.RankBlendWeight)
	if err != nil || blend != precision.NormalizedScale {
		return errors.New("invalid sector blend weights")
	}
	return nil
}
