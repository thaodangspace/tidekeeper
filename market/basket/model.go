// Package basket defines pure basket-calculation inputs, evidence, and output
// statuses. It deliberately has no database, network, or clock dependency.
package basket

import (
	"time"

	"github.com/thaodangspace/tidekeepers-server/market/precision"
	"github.com/thaodangspace/tidekeepers-server/market/sector"
)

// MetricStatus is the durable state of one basket calculation.
type MetricStatus string

const (
	MetricStatusReady          MetricStatus = "READY"
	MetricStatusDataIncomplete MetricStatus = "DATA_INCOMPLETE"
)

// ComponentStatus records the outcome of resolving one mapped market asset.
type ComponentStatus string

const (
	ComponentStatusValid             ComponentStatus = "VALID"
	ComponentStatusMissingOpen       ComponentStatus = "MISSING_OPEN"
	ComponentStatusMissingClose      ComponentStatus = "MISSING_CLOSE"
	ComponentStatusOutsideTolerance  ComponentStatus = "OUTSIDE_TOLERANCE"
	ComponentStatusInvalidPrice      ComponentStatus = "INVALID_PRICE"
	ComponentStatusOutlierFlagged    ComponentStatus = "OUTLIER_FLAGGED"
	ComponentStatusProviderRejected  ComponentStatus = "PROVIDER_REJECTED"
	ComponentStatusDisabledByMapping ComponentStatus = "DISABLED_BY_MAPPING"
)

// TurbulencePolicyType identifies how expected turbulence was published.
type TurbulencePolicyType string

const TurbulencePolicyStaticContentValue TurbulencePolicyType = "STATIC_CONTENT_VALUE"

// ExpectedTurbulencePolicy is an immutable, already-scaled policy value.
type ExpectedTurbulencePolicy struct {
	ID          string
	Type        TurbulencePolicyType
	StaticValue int64
	Floor       int64
}

// MappingComponent identifies one weighted asset in a mapping version.
type MappingComponent struct {
	MarketAssetID string
	TargetWeight  int64
	Enabled       bool
	Sequence      int
}

// MappingVersion is the immutable content input to a basket calculation.
type MappingVersion struct {
	ID                   string
	Key                  string
	Sector               sector.Key
	MinimumCoveredWeight int64
	NormalizationCap     int64
	BenchmarkEligible    bool
	ExpectedTurbulence   ExpectedTurbulencePolicy
	Components           []MappingComponent
}

// ComponentMetric is persisted evidence for one mapping component.
// ObservationStatus records provider-side validation performed before a price is used.
type ObservationStatus string

const (
	ObservationStatusValid            ObservationStatus = "VALID"
	ObservationStatusInvalidPrice     ObservationStatus = "INVALID_PRICE"
	ObservationStatusOutlierFlagged   ObservationStatus = "OUTLIER_FLAGGED"
	ObservationStatusProviderRejected ObservationStatus = "PROVIDER_REJECTED"
)

// Observation is one immutable provider observation. PriceUnits use one caller-defined
// scale consistently for every observation in a calculation window.
type Observation struct {
	ID            string
	ProviderKey   string
	MarketAssetID string
	ObservedAt    time.Time
	PriceUnits    int64
	Status        ObservationStatus
}

// CalculationWindow is the locked shared market window for one Daily Tide.
type CalculationWindow struct {
	ID          string
	DailyTideID string
	ProviderKey string
	Start       time.Time
	End         time.Time
	Tolerance   time.Duration
}

// CalculationInput contains all immutable inputs required by the pure calculator.
type CalculationInput struct {
	Mapping      MappingVersion
	Window       CalculationWindow
	Observations []Observation
}

// ComponentMetric is persisted evidence for one mapping component.
type ComponentMetric struct {
	MarketAssetID      string
	Status             ComponentStatus
	TargetWeight       int64
	EffectiveWeight    int64
	OpenObservationID  string
	CloseObservationID string
	ComponentReturn    *int64
	Contribution       *int64
	QualityFlags       []string
}

// Metric is the aggregate output persisted for a mapping and Daily Tide.
type Metric struct {
	BasketMappingVersionID string
	Sector                 sector.Key
	Status                 MetricStatus
	CoveredWeight          int64
	RawReturn              *int64
	ExpectedTurbulence     *int64
	NormalizedPerformance  *int64
	Components             []ComponentMetric
	InputChecksum          [32]byte
}

// ValidWeight confirms a mapping weight uses the shared catalog scale.
func ValidWeight(value int64) bool {
	return value > 0 && value <= precision.WeightScale
}
