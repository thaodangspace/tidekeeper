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

	// ErrActiveVoyageExists is returned when creating a Voyage while one is active.
	ErrActiveVoyageExists = errors.New("active voyage already exists")
	// ErrNoActiveVoyage is returned when the player has no current voyage pointer.
	ErrNoActiveVoyage = errors.New("no active voyage")
	// ErrVoyageNotFound is returned when a voyage public ID is unknown or unowned.
	ErrVoyageNotFound = errors.New("voyage not found")
	// ErrVoyageTerminal is returned when mutating a terminal voyage.
	ErrVoyageTerminal = errors.New("voyage is terminal")
	// ErrIdempotencyKeyReused is returned when a key is reused for a different request.
	ErrIdempotencyKeyReused = errors.New("idempotency key reused with different request")
	// ErrInvalidIdempotencyKey is returned for missing or malformed idempotency keys.
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	// ErrInitializationUnavailable is returned when launch content or daily tide is missing.
	ErrInitializationUnavailable = errors.New("voyage initialization unavailable")
	// ErrServiceUnavailable is returned for database connectivity failures.
	ErrServiceUnavailable = errors.New("service unavailable")
	// ErrInternal is returned for unexpected internal failures.
	ErrInternal = errors.New("internal error")
)
