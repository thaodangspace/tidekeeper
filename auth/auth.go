// Package auth defines opaque-session validation and authenticated principals.
package auth

import (
	"context"
	"errors"
)

var (
	// ErrSessionExpired is returned for every presented but unusable credential.
	ErrSessionExpired = errors.New("session expired")
)

// Principal contains only identifiers required for request authorization.
type Principal struct {
	AccountID string
	PlayerID  string
}

// Authenticator resolves a session digest to an authenticated principal.
type Authenticator interface {
	Authenticate(ctx context.Context, digest Digest) (Principal, error)
}
