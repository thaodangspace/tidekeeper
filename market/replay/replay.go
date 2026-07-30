// Package replay verifies deterministic market-calculation outputs without mutating state.
package replay

import (
	"errors"
	"reflect"

	"github.com/thaodangspace/tidekeepers-server/market/basket"
	"github.com/thaodangspace/tidekeepers-server/market/sector"
)

var ErrMismatch = errors.New("calculation replay mismatch")

// VerifyBasket recalculates and compares a persisted basket metric exactly.
func VerifyBasket(input basket.CalculationInput, expected basket.Metric) error {
	actual, err := (basket.Calculator{}).Calculate(input)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, expected) {
		return ErrMismatch
	}
	return nil
}

// VerifyBenchmark recalculates and compares a persisted Sector benchmark exactly.
func VerifyBenchmark(input sector.BenchmarkInput, expected sector.Benchmark) error {
	actual, err := (sector.BenchmarkCalculator{}).Calculate(input)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(actual, expected) {
		return ErrMismatch
	}
	return nil
}
