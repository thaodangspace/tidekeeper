package voyage

import "errors"

var (
	// ErrVoyageNotActive is returned when ordinary gameplay attempts to mutate a
	// terminal Voyage.
	ErrVoyageNotActive = errors.New("voyage is not active")
	// ErrZeroHullMutation is returned when a mutation has no requested change.
	ErrZeroHullMutation = errors.New("hull mutation delta must not be zero")
	// ErrHullMutationNoEffect is returned when bounds clamping would leave Hull
	// unchanged.
	ErrHullMutationNoEffect = errors.New("hull mutation has no effect")
)
