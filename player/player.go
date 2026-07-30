// Package player defines player identity read models and application boundaries.
package player

import "context"

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

// Keeper is a player-owned instance paired with its immutable definition metadata.
type Keeper struct {
	PublicID      string
	DefinitionKey string
	Name          string
	CurrentName   string
	Sector        string
	Role          string
	Rarity        string
	Level         int
}

// Reader retrieves player-owned identity and inventory state.
type Reader interface {
	GetMe(ctx context.Context, playerID ID) (Me, error)
	ListKeepers(ctx context.Context, playerID ID) ([]Keeper, error)
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

// ListKeepers returns the authenticated player's permanently owned Keeper instances.
func (s *Service) ListKeepers(ctx context.Context, playerID ID) ([]Keeper, error) {
	return s.reader.ListKeepers(ctx, playerID)
}
