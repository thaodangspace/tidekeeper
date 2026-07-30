package sector

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

var (
	ErrBenchmarkConflict   = errors.New("sector benchmark scope has different persisted input")
	ErrKeeperScoreConflict = errors.New("keeper sector score scope has different persisted input")
)

// Repository persists immutable Sector benchmark evidence and Keeper analysis results.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// PersistBenchmark atomically stores a benchmark and its members. A retry with
// the same scope/checksum returns the existing benchmark ID.
func (repository *Repository) PersistBenchmark(ctx context.Context, dailyTideID string, definition Definition, benchmark Benchmark) (string, bool, error) {
	if repository == nil || repository.pool == nil {
		return "", false, errors.New("sector benchmark persistence requires a repository")
	}
	tideID, err := parseUUID(dailyTideID)
	if err != nil {
		return "", false, fmt.Errorf("daily tide ID: %w", err)
	}
	if err := definition.Validate(); err != nil {
		return "", false, err
	}
	definitionUUID, err := parseUUID(definition.ID)
	if err != nil {
		return "", false, fmt.Errorf("sector definition ID: %w", err)
	}
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", false, fmt.Errorf("begin benchmark persistence: %w", err)
	}
	defer tx.Rollback(ctx)
	scope := "calculate_sector_benchmark:" + dailyTideID + ":" + definition.ID
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope); err != nil {
		return "", false, fmt.Errorf("lock benchmark scope: %w", err)
	}
	queries := query.New(tx)
	existing, err := queries.GetSectorBenchmarkByTideAndDefinition(ctx, query.GetSectorBenchmarkByTideAndDefinitionParams{DailyTideID: tideID, SectorDefinitionVersionID: definitionUUID})
	if err == nil {
		if !bytes.Equal(existing.InputChecksum, benchmark.InputChecksum[:]) {
			return "", false, ErrBenchmarkConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return "", false, fmt.Errorf("commit idempotent benchmark persistence: %w", err)
		}
		return uuidString(existing.ID), true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, fmt.Errorf("load existing benchmark: %w", err)
	}
	benchmarkID := uuid.New()
	if err := queries.InsertSectorBenchmark(ctx, query.InsertSectorBenchmarkParams{
		ID: uuidPG(benchmarkID), DailyTideID: tideID, SectorDefinitionVersionID: definitionUUID,
		Status: string(benchmark.Status), CalculationMethod: string(EqualWeight), EligibleBasketCount: int32(benchmark.EligibleBasketCount),
		BenchmarkUnits: benchmark.Value, MinimumRequiredBaskets: int32(definition.MinimumEligibleBaskets), InputChecksum: benchmark.InputChecksum[:],
	}); err != nil {
		return "", false, fmt.Errorf("insert sector benchmark: %w", err)
	}
	for _, member := range benchmark.Members {
		metricID, err := parseUUID(member.BasketMetricID)
		if err != nil {
			return "", false, fmt.Errorf("basket metric ID: %w", err)
		}
		if err := queries.InsertSectorBenchmarkMember(ctx, query.InsertSectorBenchmarkMemberParams{
			ID: uuidPG(uuid.New()), SectorBenchmarkID: uuidPG(benchmarkID), BasketMetricID: metricID,
			BenchmarkWeightUnits: member.BenchmarkWeight, NormalizedPerformanceUnits: member.NormalizedPerformance,
			ContributionUnits: member.Contribution, RankIndexUnits: member.RankIndex, RankPercentileUnits: member.Percentile,
		}); err != nil {
			return "", false, fmt.Errorf("insert sector benchmark member %q: %w", member.BasketMetricID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("commit benchmark persistence: %w", err)
	}
	return benchmarkID.String(), false, nil
}

// PersistKeeperScore stores one definition-level analysis result. Player-specific
// settlement later projects this immutable analysis onto locked Keeper instances.
func (repository *Repository) PersistKeeperScore(ctx context.Context, dailyTideID, keeperDefinitionID, basketMetricID, sectorBenchmarkID string, percentile int64, score ScoreOutput) (string, error) {
	if repository == nil || repository.pool == nil {
		return "", errors.New("keeper score persistence requires a repository")
	}
	tideID, err := parseUUID(dailyTideID)
	if err != nil {
		return "", err
	}
	keeperID, err := parseUUID(keeperDefinitionID)
	if err != nil {
		return "", err
	}
	metricID, err := parseUUID(basketMetricID)
	if err != nil {
		return "", err
	}
	benchmarkID, err := parseUUID(sectorBenchmarkID)
	if err != nil {
		return "", err
	}
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin keeper score persistence: %w", err)
	}
	defer tx.Rollback(ctx)
	scope := "calculate_keeper_sector_score:" + dailyTideID + ":" + keeperDefinitionID
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope); err != nil {
		return "", fmt.Errorf("lock keeper score scope: %w", err)
	}
	queries := query.New(tx)
	existing, err := queries.GetKeeperSectorResultByTideAndDefinition(ctx, query.GetKeeperSectorResultByTideAndDefinitionParams{DailyTideID: tideID, KeeperDefinitionVersionID: keeperID})
	if err == nil {
		if existing.BasketMetricID != metricID || existing.SectorBenchmarkID != benchmarkID || existing.SectorRelativeUnits != score.RelativePerformance || existing.SectorPercentileUnits != percentile || existing.RelativeComponentUnits != score.RelativeComponent || existing.RankComponentUnits != score.RankComponent || existing.SectorScoreNormalizedUnits != score.NormalizedScore || existing.SectorScorePoints != score.ScorePoints {
			return "", ErrKeeperScoreConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return "", fmt.Errorf("commit idempotent keeper score persistence: %w", err)
		}
		return uuidString(existing.ID), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("load existing keeper sector result: %w", err)
	}
	resultID := uuid.New()
	if err := queries.InsertKeeperSectorResult(ctx, query.InsertKeeperSectorResultParams{
		ID: uuidPG(resultID), DailyTideID: tideID, KeeperDefinitionVersionID: keeperID, BasketMetricID: metricID, SectorBenchmarkID: benchmarkID,
		SectorRelativeUnits: score.RelativePerformance, SectorPercentileUnits: percentile, RelativeComponentUnits: score.RelativeComponent,
		RankComponentUnits: score.RankComponent, SectorScoreNormalizedUnits: score.NormalizedScore, SectorScorePoints: score.ScorePoints,
	}); err != nil {
		return "", fmt.Errorf("insert keeper sector result: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit keeper sector result: %w", err)
	}
	return resultID.String(), nil
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return uuidPG(parsed), nil
}

func uuidPG(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: value, Valid: true}
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}
