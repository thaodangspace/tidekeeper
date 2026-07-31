package voyage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

// VoyageKeeper is a Keeper instance owned by one Voyage, paired with its exact
// immutable definition version, authoritative upgrade node, and acquisition
// provenance. Level is derived from normalized node depth and never stored.
type VoyageKeeper struct {
	PublicID          string
	DefinitionKey     string
	DefinitionVersion int64
	Name              string
	CurrentName       string
	Sector            string
	Role              string
	Rarity            string
	UpgradeNodeKey    string
	Level             int
	AcquiredDay       int
	AcquiredSource    string
	AcquiredAt        time.Time
}

// VoyageKeepers is the current active Voyage's keeper inventory read model.
type VoyageKeepers struct {
	VoyageID string
	Keepers  []VoyageKeeper
}

// GetActiveVoyageKeepers returns the authenticated player's active Voyage
// inventory. Ownership and ACTIVE status are enforced in SQL, and the keeper
// list is filtered to the exact resolved Voyage so the response never mixes
// two Voyages. Returns ErrNoActiveVoyage when the player has no active Voyage.
func (s *Service) GetActiveVoyageKeepers(ctx context.Context, playerID string) (*VoyageKeepers, error) {
	pid, err := parseUUID(playerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	q := query.New(s.pool)
	voyageRow, err := q.GetActiveVoyageForPlayer(ctx, pid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoActiveVoyage
		}
		return nil, fmt.Errorf("%w: get active voyage: %w", ErrServiceUnavailable, err)
	}

	rows, err := q.ListVoyageKeepers(ctx, voyageRow.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: list voyage keepers: %w", ErrServiceUnavailable, err)
	}

	keepers := make([]VoyageKeeper, len(rows))
	for index, row := range rows {
		keepers[index] = VoyageKeeper{
			PublicID:          row.KeeperPublicID,
			DefinitionKey:     row.DefinitionKey,
			DefinitionVersion: row.DefinitionVersion,
			Name:              row.Name,
			CurrentName:       row.CurrentName,
			Sector:            row.SectorKey,
			Role:              row.RoleKey,
			Rarity:            row.RarityKey,
			UpgradeNodeKey:    row.UpgradeNodeKey,
			Level:             int(row.NodeDepth) + 1,
			AcquiredDay:       int(row.AcquiredDay),
			AcquiredSource:    row.AcquiredSource,
			AcquiredAt:        row.AcquiredAt.Time.UTC(),
		}
	}

	return &VoyageKeepers{
		VoyageID: voyageRow.PublicID,
		Keepers:  keepers,
	}, nil
}
