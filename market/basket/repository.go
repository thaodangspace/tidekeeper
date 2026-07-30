package basket

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

var ErrMetricConflict = errors.New("basket metric scope has different persisted input")

// Repository persists immutable basket calculation evidence.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// PersistMetric atomically persists a calculated metric and all component evidence.
// Retrying the same scope/checksum returns the existing metric ID without new rows.
func (repository *Repository) PersistMetric(ctx context.Context, input CalculationInput, metric Metric) (string, bool, error) {
	if repository == nil || repository.pool == nil || input.Window.ID == "" {
		return "", false, errors.New("basket metric persistence requires a repository and calculation window ID")
	}
	dailyTideID, err := parseUUID(input.Window.DailyTideID)
	if err != nil {
		return "", false, fmt.Errorf("daily tide ID: %w", err)
	}
	mappingID, err := parseUUID(metric.BasketMappingVersionID)
	if err != nil {
		return "", false, fmt.Errorf("basket mapping version ID: %w", err)
	}
	windowID, err := parseUUID(input.Window.ID)
	if err != nil {
		return "", false, fmt.Errorf("calculation window ID: %w", err)
	}

	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", false, fmt.Errorf("begin basket metric persistence: %w", err)
	}
	defer tx.Rollback(ctx)
	scope := "calculate_basket_metrics:" + input.Window.DailyTideID + ":" + metric.BasketMappingVersionID
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", scope); err != nil {
		return "", false, fmt.Errorf("lock basket metric scope: %w", err)
	}
	queries := query.New(tx)
	existing, err := queries.GetBasketMetricByTideAndMapping(ctx, query.GetBasketMetricByTideAndMappingParams{
		DailyTideID: dailyTideID, BasketMappingVersionID: mappingID,
	})
	if err == nil {
		if !bytes.Equal(existing.InputChecksum, metric.InputChecksum[:]) {
			return "", false, ErrMetricConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return "", false, fmt.Errorf("commit idempotent basket metric persistence: %w", err)
		}
		return uuidString(existing.ID), true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", false, fmt.Errorf("load existing basket metric: %w", err)
	}

	metricID := uuid.New()
	if err := queries.InsertBasketMetric(ctx, query.InsertBasketMetricParams{
		ID: uuidPG(metricID), DailyTideID: dailyTideID, BasketMappingVersionID: mappingID, CalculationWindowID: windowID,
		Status: string(metric.Status), CoveredWeightUnits: metric.CoveredWeight,
		RawReturnUnits: metric.RawReturn, ExpectedTurbulenceUnits: metric.ExpectedTurbulence,
		NormalizedPerformanceUnits: metric.NormalizedPerformance, InputChecksum: metric.InputChecksum[:],
	}); err != nil {
		return "", false, fmt.Errorf("insert basket metric: %w", err)
	}
	for _, component := range metric.Components {
		assetID, err := parseUUID(component.MarketAssetID)
		if err != nil {
			return "", false, fmt.Errorf("component asset ID: %w", err)
		}
		qualityFlags := component.QualityFlags
		if qualityFlags == nil {
			qualityFlags = []string{}
		}
		flags, err := json.Marshal(qualityFlags)
		if err != nil {
			return "", false, fmt.Errorf("marshal component quality flags: %w", err)
		}
		openObservationID, err := nullableUUID(component.OpenObservationID)
		if err != nil {
			return "", false, fmt.Errorf("open observation ID: %w", err)
		}
		closeObservationID, err := nullableUUID(component.CloseObservationID)
		if err != nil {
			return "", false, fmt.Errorf("close observation ID: %w", err)
		}
		if err := queries.InsertBasketMetricComponent(ctx, query.InsertBasketMetricComponentParams{
			ID: uuidPG(uuid.New()), BasketMetricID: uuidPG(metricID), MarketAssetID: assetID, Status: string(component.Status),
			TargetWeightUnits: component.TargetWeight, EffectiveWeightUnits: int64Pointer(component.EffectiveWeight, component.Status == ComponentStatusValid),
			OpenObservationID: openObservationID, CloseObservationID: closeObservationID,
			ComponentReturnUnits: component.ComponentReturn, ContributionUnits: component.Contribution, QualityFlags: flags,
		}); err != nil {
			return "", false, fmt.Errorf("insert basket metric component %q: %w", component.MarketAssetID, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return "", false, fmt.Errorf("commit basket metric persistence: %w", err)
	}
	return metricID.String(), false, nil
}

func parseUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return uuidPG(parsed), nil
}

func nullableUUID(value string) (pgtype.UUID, error) {
	if value == "" {
		return pgtype.UUID{}, nil
	}
	return parseUUID(value)
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

func int64Pointer(value int64, valid bool) *int64 {
	if !valid {
		return nil
	}
	return &value
}
