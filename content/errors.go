// Package content defines the deterministic gameplay-content domain that owns
// complete-release validation, canonical checksumming, and publication for the
// Keeper catalog plus the versioned Strategy, Relic, Synergy, Daily Modifier,
// Daily Objective, and game-rule-set kinds.
package content

import "fmt"

// ValidationError identifies one validation failure in a candidate gameplay
// release. Kind and Key are populated when the failure belongs to a specific
// definition; Key is empty for release-wide failures.
type ValidationError struct {
	Kind   Kind
	Key    string
	Detail string
}

func (e *ValidationError) Error() string {
	if e.Key == "" {
		return fmt.Sprintf("%s content: %s", e.Kind, e.Detail)
	}
	return fmt.Sprintf("%s content %q: %s", e.Kind, e.Key, e.Detail)
}

// validationErrorf builds a structured validation failure.
func validationErrorf(kind Kind, key, format string, args ...any) error {
	return &ValidationError{Kind: kind, Key: key, Detail: fmt.Sprintf(format, args...)}
}
