package daily

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

// Repository reads daily context state from PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository constructs a daily context reader backed by the application pool.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// GetCurrentDailyContext returns the current voyage daily context for the given player.
func (r *Repository) GetCurrentDailyContext(ctx context.Context, playerID ID) (DailyContext, error) {
	databaseID, err := parseDatabaseID(playerID)
	if err != nil {
		return DailyContext{}, fmt.Errorf("parse player ID: %w", err)
	}

	row, err := query.New(r.pool).GetCurrentDailyContextForPlayer(ctx, databaseID)
	if err != nil {
		return DailyContext{}, classifyDBError(err)
	}

	if !row.CurrentVoyageID.Valid {
		return DailyContext{}, ErrNoActiveVoyage
	}

	if row.VoyagePublicID == nil {
		return DailyContext{}, ErrVoyageNotFound
	}

	if !row.PlayerDailyStateID.Valid {
		return DailyContext{}, fmt.Errorf("%w: missing daily state", ErrInternal)
	}

	if row.ProjectionSchemaVersion == nil {
		return DailyContext{}, fmt.Errorf("%w: missing projection schema version", ErrInternal)
	}

	proj, err := decodeAndValidateSchemaV1Projection(*row.ProjectionSchemaVersion, row.Projection)
	if err != nil {
		return DailyContext{}, fmt.Errorf("%w: %w", ErrInternal, err)
	}

	modifier, objective, signals, lineup, inventory, shop, strategies := mapV1ToDaily(proj)

	var selectedStrategyID *string
	if proj.SelectedStrategyID != nil && *proj.SelectedStrategyID != "" {
		id := *proj.SelectedStrategyID
		selectedStrategyID = &id
	}
	pendingRewardCount := proj.PendingRewardCount

	var dayNumber int32
	if row.CurrentDayNumber != nil {
		dayNumber = int32(*row.CurrentDayNumber)
	}

	var fundHealth, maxFundHealth, capital int32
	if row.FundHealth != nil {
		fundHealth = *row.FundHealth
	}
	if row.MaxFundHealth != nil {
		maxFundHealth = *row.MaxFundHealth
	}
	if row.Capital != nil {
		capital = *row.Capital
	}

	score := safeScore(row.Score)

	var playerStateVersion int64
	if row.PlayerStateVersion != nil {
		playerStateVersion = *row.PlayerStateVersion
	}

	return DailyContext{
		ServerNow: row.ServerNow.Time.UTC(),
		Voyage: VoyageSummary{
			PublicID:      *row.VoyagePublicID,
			Status:        *row.VoyageStatus,
			DayNumber:     dayNumber,
			FundHealth:    fundHealth,
			MaxFundHealth: maxFundHealth,
			Capital:       capital,
			Score:         score,
		},
		Daily: DailyDetail{
			Phase:              *row.PlayerPhase,
			LockAt:             row.LockAt.Time.UTC(),
			SettleAfter:        row.SettleAfter.Time.UTC(),
			Version:            playerStateVersion,
			Modifier:           modifier,
			Objective:          objective,
			Signals:            signals,
			Lineup:             lineup,
			Inventory:          inventory,
			Shop:               shop,
			Strategies:         strategies,
			SelectedStrategyID: selectedStrategyID,
			PendingRewardCount: pendingRewardCount,
		},
	}, nil
}

func parseDatabaseID(playerID ID) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(string(playerID)); err != nil {
		return pgtype.UUID{}, fmt.Errorf("parse player ID: %w", err)
	}
	return id, nil
}

func classifyDBError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%w: %w", ErrServiceUnavailable, err)
	}
	return fmt.Errorf("%w: %w", ErrInternal, err)
}

func safeScore(score any) string {
	switch v := score.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return "0.0000"
	}
}
