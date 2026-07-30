package precision

import (
	"errors"
	"math"
	"testing"
)

func TestDivideRoundsHalfAwayFromZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		numerator int64
		divisor   int64
		want      int64
	}{
		{name: "positive half", numerator: 1, divisor: 2, want: 1},
		{name: "negative numerator half", numerator: -1, divisor: 2, want: -1},
		{name: "negative divisor half", numerator: 1, divisor: -2, want: -1},
		{name: "both negative half", numerator: -1, divisor: -2, want: 1},
		{name: "below half", numerator: 1, divisor: 3, want: 0},
		{name: "above half", numerator: 2, divisor: 3, want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Divide(test.numerator, test.divisor)
			if err != nil {
				t.Fatalf("Divide(%d, %d): %v", test.numerator, test.divisor, err)
			}
			if got != test.want {
				t.Errorf("Divide(%d, %d) = %d, want %d", test.numerator, test.divisor, got, test.want)
			}
		})
	}
}

func TestArithmeticRejectsInvalidAndOverflowingValues(t *testing.T) {
	t.Parallel()

	if _, err := Divide(1, 0); !errors.Is(err, ErrDivisionByZero) {
		t.Fatalf("Divide division error = %v, want ErrDivisionByZero", err)
	}
	if _, err := MultiplyDivide(math.MaxInt64, 2, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("MultiplyDivide overflow error = %v, want ErrOverflow", err)
	}
	if _, err := Add(math.MaxInt64, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("Add overflow error = %v, want ErrOverflow", err)
	}
	if _, err := Subtract(math.MinInt64, 1); !errors.Is(err, ErrOverflow) {
		t.Fatalf("Subtract overflow error = %v, want ErrOverflow", err)
	}
}

func TestMultiplyDivideAndClamp(t *testing.T) {
	t.Parallel()

	got, err := MultiplyDivide(25, 4, 3)
	if err != nil {
		t.Fatalf("MultiplyDivide: %v", err)
	}
	if got != 33 {
		t.Errorf("MultiplyDivide = %d, want 33", got)
	}

	for _, test := range []struct {
		value, minimum, maximum, want int64
	}{
		{value: -3, minimum: -2, maximum: 2, want: -2},
		{value: 1, minimum: -2, maximum: 2, want: 1},
		{value: 5, minimum: -2, maximum: 2, want: 2},
	} {
		got, err := Clamp(test.value, test.minimum, test.maximum)
		if err != nil {
			t.Fatalf("Clamp: %v", err)
		}
		if got != test.want {
			t.Errorf("Clamp(%d, %d, %d) = %d, want %d", test.value, test.minimum, test.maximum, got, test.want)
		}
	}
	if _, err := Clamp(0, 1, -1); !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("Clamp invalid range error = %v, want ErrInvalidRange", err)
	}
}
