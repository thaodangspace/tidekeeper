package player

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

// Repository reads player-owned identity state from PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a player reader backed by the application pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetMe returns the player identity summary for an authenticated internal ID.
func (r *Repository) GetMe(ctx context.Context, playerID ID) (Me, error) {
	databaseID, err := parseDatabaseID(playerID)
	if err != nil {
		return Me{}, err
	}

	row, err := query.New(r.pool).GetMeForPlayer(ctx, databaseID)
	if err != nil {
		return Me{}, fmt.Errorf("get player identity: %w", err)
	}

	return Me{
		PublicID:            row.PlayerPublicID,
		OnboardingCompleted: row.OnboardingCompleted,
		Locale:              row.Locale,
		Timezone:            row.Timezone,
		ActiveVoyageID:      row.CurrentVoyagePublicID,
	}, nil
}

// ListKeepers returns the player's inventory paired with immutable definition metadata.
func (r *Repository) ListKeepers(ctx context.Context, playerID ID) ([]Keeper, error) {
	databaseID, err := parseDatabaseID(playerID)
	if err != nil {
		return nil, err
	}

	rows, err := query.New(r.pool).ListKeeperInstancesForPlayer(ctx, databaseID)
	if err != nil {
		return nil, fmt.Errorf("list player keepers: %w", err)
	}

	keepers := make([]Keeper, len(rows))
	for index, row := range rows {
		keepers[index] = Keeper{
			PublicID:      row.KeeperPublicID,
			DefinitionKey: row.DefinitionKey,
			Name:          row.Name,
			CurrentName:   row.CurrentName,
			Sector:        row.SectorKey,
			Role:          row.RoleKey,
			Rarity:        row.RarityKey,
			Level:         int(row.Level),
		}
	}
	return keepers, nil
}

func parseDatabaseID(playerID ID) (pgtype.UUID, error) {
	var databaseID pgtype.UUID
	if err := databaseID.Scan(string(playerID)); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse player ID: %w", err)
	}
	return databaseID, nil
}
