// Package precision provides the fixed-point arithmetic used by authoritative
// market calculations. Values are scaled integers; callers must not use
// binary floating point for settlement inputs or outputs.
package precision

import (
	"errors"
	"math/big"
)

const (
	// WeightScale matches the existing immutable Keeper catalog representation.
	WeightScale int64 = 100_000_000
	// ReturnScale stores component and basket returns to eight decimal places.
	ReturnScale int64 = 100_000_000
	// NormalizedScale stores normalized performance and percentile values.
	NormalizedScale int64 = 1_000_000
	// ScoreScale stores score components to four decimal places.
	ScoreScale int64 = 10_000
)

var (
	ErrDivisionByZero = errors.New("precision division by zero")
	ErrOverflow       = errors.New("precision overflow")
	ErrInvalidRange   = errors.New("precision invalid clamp range")
)

// Add returns left + right when the exact result fits in int64.
func Add(left, right int64) (int64, error) {
	return checkedBigInt(new(big.Int).Add(big.NewInt(left), big.NewInt(right)))
}

// Subtract returns left - right when the exact result fits in int64.
func Subtract(left, right int64) (int64, error) {
	return checkedBigInt(new(big.Int).Sub(big.NewInt(left), big.NewInt(right)))
}

// MultiplyDivide calculates left*right/divisor, rounding half away from zero.
// Intermediate multiplication is exact and cannot overflow int64.
func MultiplyDivide(left, right, divisor int64) (int64, error) {
	if divisor == 0 {
		return 0, ErrDivisionByZero
	}
	product := new(big.Int).Mul(big.NewInt(left), big.NewInt(right))
	return roundHalfAwayFromZero(product, big.NewInt(divisor))
}

// Divide calculates numerator/divisor, rounding half away from zero.
func Divide(numerator, divisor int64) (int64, error) {
	if divisor == 0 {
		return 0, ErrDivisionByZero
	}
	return roundHalfAwayFromZero(big.NewInt(numerator), big.NewInt(divisor))
}

// Clamp bounds value inclusively. The range must be ordered.
func Clamp(value, minimum, maximum int64) (int64, error) {
	if minimum > maximum {
		return 0, ErrInvalidRange
	}
	if value < minimum {
		return minimum, nil
	}
	if value > maximum {
		return maximum, nil
	}
	return value, nil
}

func roundHalfAwayFromZero(numerator, divisor *big.Int) (int64, error) {
	if divisor.Sign() == 0 {
		return 0, ErrDivisionByZero
	}

	negative := numerator.Sign() != 0 && (numerator.Sign() < 0) != (divisor.Sign() < 0)
	absoluteNumerator := new(big.Int).Abs(new(big.Int).Set(numerator))
	absoluteDivisor := new(big.Int).Abs(new(big.Int).Set(divisor))
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(absoluteNumerator, absoluteDivisor, remainder)

	if new(big.Int).Lsh(remainder, 1).Cmp(absoluteDivisor) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if negative {
		quotient.Neg(quotient)
	}
	return checkedBigInt(quotient)
}

func checkedBigInt(value *big.Int) (int64, error) {
	if !value.IsInt64() {
		return 0, ErrOverflow
	}
	return value.Int64(), nil
}
