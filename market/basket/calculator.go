package basket

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strconv"
	"time"

	"github.com/thaodangspace/tidekeepers-server/market/canonical"
	"github.com/thaodangspace/tidekeepers-server/market/precision"
)

var (
	ErrInvalidMapping = errors.New("invalid basket mapping")
	ErrInvalidWindow  = errors.New("invalid calculation window")
)

// Calculator deterministically derives a basket metric from immutable observations.
type Calculator struct{}

// Calculate derives component evidence and either a ready metric or a
// data-incomplete metric. Incomplete market data is a valid result, not an error.
func (Calculator) Calculate(input CalculationInput) (Metric, error) {
	if err := validateInput(input); err != nil {
		return Metric{}, err
	}

	components := append([]MappingComponent(nil), input.Mapping.Components...)
	sort.Slice(components, func(i, j int) bool { return components[i].MarketAssetID < components[j].MarketAssetID })
	metrics := make([]ComponentMetric, 0, len(components))
	validIndexes := make([]int, 0, len(components))
	var coveredWeight int64

	for _, component := range components {
		metric := ComponentMetric{MarketAssetID: component.MarketAssetID, TargetWeight: component.TargetWeight}
		if !component.Enabled {
			metric.Status = ComponentStatusDisabledByMapping
			metrics = append(metrics, metric)
			continue
		}
		open, openStatus := selectObservation(input.Observations, input.Window, component.MarketAssetID, input.Window.Start)
		close, closeStatus := selectObservation(input.Observations, input.Window, component.MarketAssetID, input.Window.End)
		if openStatus != ComponentStatusValid {
			metric.Status = openStatus
			metrics = append(metrics, metric)
			continue
		}
		if closeStatus != ComponentStatusValid {
			if closeStatus == ComponentStatusMissingOpen {
				closeStatus = ComponentStatusMissingClose
			}
			metric.Status = closeStatus
			metrics = append(metrics, metric)
			continue
		}
		priceDelta, err := precision.Subtract(close.PriceUnits, open.PriceUnits)
		if err != nil {
			return Metric{}, fmt.Errorf("calculate component price delta for %q: %w", component.MarketAssetID, err)
		}
		componentReturn, err := precision.MultiplyDivide(priceDelta, precision.ReturnScale, open.PriceUnits)
		if err != nil {
			return Metric{}, fmt.Errorf("calculate component return for %q: %w", component.MarketAssetID, err)
		}
		metric.Status = ComponentStatusValid
		metric.OpenObservationID = open.ID
		metric.CloseObservationID = close.ID
		metric.ComponentReturn = int64Value(componentReturn)
		metrics = append(metrics, metric)
		validIndexes = append(validIndexes, len(metrics)-1)
		coveredWeight, err = precision.Add(coveredWeight, component.TargetWeight)
		if err != nil {
			return Metric{}, fmt.Errorf("sum covered weight: %w", err)
		}
	}

	result := Metric{
		BasketMappingVersionID: input.Mapping.ID,
		Sector:                 input.Mapping.Sector,
		CoveredWeight:          coveredWeight,
		Components:             metrics,
	}
	if coveredWeight < input.Mapping.MinimumCoveredWeight {
		result.Status = MetricStatusDataIncomplete
		result.InputChecksum = checksum(input, result)
		return result, nil
	}
	if err := assignEffectiveWeights(result.Components, validIndexes, coveredWeight); err != nil {
		return Metric{}, err
	}
	var basketReturn int64
	for index := range result.Components {
		component := &result.Components[index]
		if component.Status != ComponentStatusValid {
			continue
		}
		contribution, err := precision.MultiplyDivide(*component.ComponentReturn, component.EffectiveWeight, precision.WeightScale)
		if err != nil {
			return Metric{}, fmt.Errorf("calculate contribution for %q: %w", component.MarketAssetID, err)
		}
		component.Contribution = int64Value(contribution)
		basketReturn, err = precision.Add(basketReturn, contribution)
		if err != nil {
			return Metric{}, fmt.Errorf("sum basket return: %w", err)
		}
	}
	effectiveTurbulence := input.Mapping.ExpectedTurbulence.StaticValue
	if effectiveTurbulence < input.Mapping.ExpectedTurbulence.Floor {
		effectiveTurbulence = input.Mapping.ExpectedTurbulence.Floor
	}
	normalized, err := precision.MultiplyDivide(basketReturn, precision.NormalizedScale, effectiveTurbulence)
	if err != nil {
		return Metric{}, fmt.Errorf("normalize basket return: %w", err)
	}
	normalized, err = precision.Clamp(normalized, -input.Mapping.NormalizationCap, input.Mapping.NormalizationCap)
	if err != nil {
		return Metric{}, fmt.Errorf("clamp normalized performance: %w", err)
	}
	result.Status = MetricStatusReady
	result.RawReturn = int64Value(basketReturn)
	result.ExpectedTurbulence = int64Value(effectiveTurbulence)
	result.NormalizedPerformance = int64Value(normalized)
	result.InputChecksum = checksum(input, result)
	return result, nil
}

func validateInput(input CalculationInput) error {
	mapping := input.Mapping
	if mapping.ID == "" || mapping.Key == "" || !mapping.Sector.IsCalculationTarget() || mapping.MinimumCoveredWeight <= 0 ||
		mapping.MinimumCoveredWeight > precision.WeightScale || mapping.NormalizationCap <= 0 ||
		mapping.ExpectedTurbulence.Type != TurbulencePolicyStaticContentValue || mapping.ExpectedTurbulence.StaticValue <= 0 ||
		mapping.ExpectedTurbulence.Floor <= 0 || len(mapping.Components) == 0 {
		return ErrInvalidMapping
	}
	if input.Window.DailyTideID == "" || input.Window.ProviderKey == "" || !input.Window.Start.Before(input.Window.End) || input.Window.Tolerance < 0 {
		return ErrInvalidWindow
	}
	seen := make(map[string]struct{}, len(mapping.Components))
	var totalWeight int64
	for _, component := range mapping.Components {
		if component.MarketAssetID == "" || !ValidWeight(component.TargetWeight) {
			return ErrInvalidMapping
		}
		if _, exists := seen[component.MarketAssetID]; exists {
			return ErrInvalidMapping
		}
		seen[component.MarketAssetID] = struct{}{}
		var err error
		totalWeight, err = precision.Add(totalWeight, component.TargetWeight)
		if err != nil {
			return ErrInvalidMapping
		}
	}
	if totalWeight != precision.WeightScale {
		return ErrInvalidMapping
	}
	return nil
}

func selectObservation(observations []Observation, window CalculationWindow, assetID string, target time.Time) (*Observation, ComponentStatus) {
	var nearest *Observation
	var nearestDistance time.Duration
	seenMatchingAsset := false
	seenWithinTolerance := false
	var rejected ComponentStatus
	for index := range observations {
		observation := &observations[index]
		if observation.MarketAssetID != assetID || observation.ProviderKey != window.ProviderKey {
			continue
		}
		seenMatchingAsset = true
		distance := observation.ObservedAt.Sub(target)
		if distance < 0 {
			distance = -distance
		}
		if distance > window.Tolerance {
			continue
		}
		seenWithinTolerance = true
		if observation.Status != ObservationStatusValid || observation.PriceUnits <= 0 {
			rejected = observationComponentStatus(*observation)
			continue
		}
		if nearest == nil || distance < nearestDistance || (distance == nearestDistance && observation.ID < nearest.ID) {
			nearest = observation
			nearestDistance = distance
		}
	}
	if nearest != nil {
		return nearest, ComponentStatusValid
	}
	if !seenMatchingAsset {
		return nil, ComponentStatusMissingOpen
	}
	if !seenWithinTolerance {
		return nil, ComponentStatusOutsideTolerance
	}
	if rejected != "" {
		return nil, rejected
	}
	return nil, ComponentStatusMissingOpen
}

func observationComponentStatus(observation Observation) ComponentStatus {
	switch observation.Status {
	case ObservationStatusOutlierFlagged:
		return ComponentStatusOutlierFlagged
	case ObservationStatusProviderRejected:
		return ComponentStatusProviderRejected
	default:
		return ComponentStatusInvalidPrice
	}
}

func assignEffectiveWeights(metrics []ComponentMetric, validIndexes []int, coveredWeight int64) error {
	type remainder struct {
		index int
		value *big.Int
	}
	remainders := make([]remainder, 0, len(validIndexes))
	var assigned int64
	for _, index := range validIndexes {
		numerator := new(big.Int).Mul(big.NewInt(metrics[index].TargetWeight), big.NewInt(precision.WeightScale))
		quotient, remainderValue := new(big.Int), new(big.Int)
		quotient.QuoRem(numerator, big.NewInt(coveredWeight), remainderValue)
		if !quotient.IsInt64() {
			return precision.ErrOverflow
		}
		metrics[index].EffectiveWeight = quotient.Int64()
		var err error
		assigned, err = precision.Add(assigned, metrics[index].EffectiveWeight)
		if err != nil {
			return err
		}
		remainders = append(remainders, remainder{index: index, value: remainderValue})
	}
	remaining, err := precision.Subtract(precision.WeightScale, assigned)
	if err != nil {
		return err
	}
	sort.Slice(remainders, func(i, j int) bool {
		if compared := remainders[i].value.Cmp(remainders[j].value); compared != 0 {
			return compared > 0
		}
		return metrics[remainders[i].index].MarketAssetID < metrics[remainders[j].index].MarketAssetID
	})
	for index := int64(0); index < remaining; index++ {
		metrics[remainders[index].index].EffectiveWeight++
	}
	return nil
}

func checksum(input CalculationInput, metric Metric) [32]byte {
	parts := []string{
		input.Window.DailyTideID, input.Mapping.ID, string(input.Mapping.Sector),
		strconv.FormatInt(input.Mapping.MinimumCoveredWeight, 10), strconv.FormatInt(input.Mapping.NormalizationCap, 10),
		input.Mapping.ExpectedTurbulence.ID, string(input.Mapping.ExpectedTurbulence.Type),
		strconv.FormatInt(input.Mapping.ExpectedTurbulence.StaticValue, 10), strconv.FormatInt(input.Mapping.ExpectedTurbulence.Floor, 10),
		string(metric.Status), strconv.FormatInt(metric.CoveredWeight, 10),
	}
	for _, component := range metric.Components {
		parts = append(parts, component.MarketAssetID, string(component.Status), strconv.FormatInt(component.TargetWeight, 10),
			strconv.FormatInt(component.EffectiveWeight, 10), component.OpenObservationID, component.CloseObservationID)
		if component.ComponentReturn != nil {
			parts = append(parts, strconv.FormatInt(*component.ComponentReturn, 10))
		}
		if component.Contribution != nil {
			parts = append(parts, strconv.FormatInt(*component.Contribution, 10))
		}
	}
	return canonical.Checksum(parts...)
}

func int64Value(value int64) *int64 { return &value }
