// Package player defines player identity read models and application boundaries.
package player

import (
	"context"
	"time"
)

// ID is an internal player identifier used only for authorization and joins.
type ID string

// Me is the transport-independent authenticated player summary.
type Me struct {
	PublicID            string
	OnboardingCompleted bool
	Locale              string
	Timezone            string
	ActiveVoyageID      *string
}

// KeeperUnlock is a permanent archetype unlock paired with the exact immutable
// definition version that originally granted it.
type KeeperUnlock struct {
	DefinitionKey     string
	DefinitionVersion int64
	Name              string
	CurrentName       string
	Sector            string
	Role              string
	Rarity            string
	UnlockSource      string
	UnlockedAt        time.Time
}

// Reader retrieves player-owned identity and meta-progression state.
type Reader interface {
	GetMe(ctx context.Context, playerID ID) (Me, error)
	ListUnlocks(ctx context.Context, playerID ID) ([]KeeperUnlock, error)
}

// Service orchestrates player identity reads.
type Service struct {
	reader Reader
}

// NewService creates a player read service.
func NewService(reader Reader) *Service {
	return &Service{reader: reader}
}

// GetMe returns the authenticated player's public summary.
func (s *Service) GetMe(ctx context.Context, playerID ID) (Me, error) {
	return s.reader.GetMe(ctx, playerID)
}

// ListUnlocks returns the authenticated player's permanent archetype unlocks.
func (s *Service) ListUnlocks(ctx context.Context, playerID ID) ([]KeeperUnlock, error) {
	return s.reader.ListUnlocks(ctx, playerID)
}
