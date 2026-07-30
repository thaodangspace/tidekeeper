//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/market/basket"
	"github.com/thaodangspace/tidekeepers-server/market/sector"
)

const (
	repositorySectorDefinitionID = "00000000-0000-0000-0000-000000000041"
	repositoryPolicyID           = "00000000-0000-0000-0000-000000000042"
	repositoryDailyTideID        = "00000000-0000-0000-0000-000000000043"
	repositoryWindowID           = "00000000-0000-0000-0000-000000000044"
	repositoryOpenOneID          = "00000000-0000-0000-0000-000000000045"
	repositoryCloseOneID         = "00000000-0000-0000-0000-000000000046"
	repositoryOpenTwoID          = "00000000-0000-0000-0000-000000000047"
	repositoryCloseTwoID         = "00000000-0000-0000-0000-000000000048"
)

func TestBasketMetricRepositoryIsIdempotentAndConflictSafe(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}
	admin, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	t.Cleanup(func() { admin.Close(ctx) })
	schema := "basket_metric_repository_" + randomHex(t)
	mustExec(t, ctx, admin, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, admin, "SET search_path TO "+schema)
	applyMigration(t, ctx, admin, "000001_foundation.up.sql")
	applyMigration(t, ctx, admin, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, admin, "000005_four_sector_calculation.up.sql")
	seedValidDraft(t, ctx, admin)
	seedRepositoryCalculationInputs(t, ctx, admin)

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse database URL: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)

	input := repositoryCalculationInput()
	metric, err := (basket.Calculator{}).Calculate(input)
	if err != nil {
		t.Fatalf("Calculate(): %v", err)
	}
	repository := basket.NewRepository(pool)
	service := basket.NewService(repository)
	calculated, firstID, idempotent, err := service.CalculateAndPersist(ctx, input)
	if err != nil || calculated.InputChecksum != metric.InputChecksum {
		t.Fatalf("CalculateAndPersist() error = %v", err)
	}
	if idempotent || firstID == "" {
		t.Fatalf("first CalculateAndPersist() = id:%q idempotent:%t", firstID, idempotent)
	}
	firstID, idempotent, err = repository.PersistMetric(ctx, input, metric)
	if err != nil || !idempotent || firstID == "" {
		t.Fatalf("idempotent PersistMetric() = id:%q idempotent:%t err:%v", firstID, idempotent, err)
	}
	secondID, idempotent, err := repository.PersistMetric(ctx, input, metric)
	if err != nil || !idempotent || secondID != firstID {
		t.Fatalf("second PersistMetric() = id:%q idempotent:%t err:%v", secondID, idempotent, err)
	}
	conflicting := metric
	conflicting.InputChecksum[0]++
	if _, _, err := repository.PersistMetric(ctx, input, conflicting); !errors.Is(err, basket.ErrMetricConflict) {
		t.Fatalf("conflicting PersistMetric() error = %v, want ErrMetricConflict", err)
	}
	var metrics, components int
	if err := admin.QueryRow(ctx, `SELECT (SELECT count(*) FROM basket_metrics), (SELECT count(*) FROM basket_metric_components)`).Scan(&metrics, &components); err != nil {
		t.Fatalf("count persisted evidence: %v", err)
	}
	if metrics != 1 || components != 2 {
		t.Fatalf("persisted rows = metrics:%d components:%d, want 1/2", metrics, components)
	}
}

func seedRepositoryCalculationInputs(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	mustExec(t, ctx, conn, `INSERT INTO sector_definition_versions (id, sector_key, content_version, benchmark_method, minimum_eligible_baskets, relative_scale_units, relative_blend_weight_units, rank_blend_weight_units, score_cap_units) VALUES ($1, 'CREST', 1, 'EQUAL_WEIGHT', 2, 500000, 700000, 300000, 1000000)`, repositorySectorDefinitionID)
	mustExec(t, ctx, conn, `INSERT INTO expected_turbulence_policies (id, policy_key, content_version, policy_type, static_value_units, floor_units, rounding_mode) VALUES ($1, 'crest_static', 1, 'STATIC_CONTENT_VALUE', 4000000, 250000, 'ROUND_HALF_AWAY_FROM_ZERO')`, repositoryPolicyID)
	mustExec(t, ctx, conn, `UPDATE basket_mapping_versions SET sector_definition_version_id = $1, minimum_covered_weight_units = 100000000, expected_turbulence_policy_id = $2, normalization_cap_units = 2000000, benchmark_eligible = true WHERE id = $3`, repositorySectorDefinitionID, repositoryPolicyID, mappingOneID)
	mustExec(t, ctx, conn, `INSERT INTO daily_tides (id, day_key, sequence_number, phase, lock_at, settle_after, content_version) VALUES ($1, '2026-07-31', 2, 'DATA_PENDING', '2026-07-31T00:00:00Z', '2026-07-31T00:01:00Z', 1)`, repositoryDailyTideID)
	mustExec(t, ctx, conn, `INSERT INTO market_calculation_windows (id, daily_tide_id, provider_key, data_window_start, data_window_end, required_granularity_seconds, observation_tolerance_seconds) VALUES ($1, $2, 'fixture', '2026-07-31T00:00:00Z', '2026-08-01T00:00:00Z', 60, 300)`, repositoryWindowID, repositoryDailyTideID)
	for _, observation := range []struct{ id, asset, at, raw string }{
		{repositoryOpenOneID, assetOneID, "2026-07-31T00:00:00Z", "100"}, {repositoryCloseOneID, assetOneID, "2026-08-01T00:00:00Z", "110"},
		{repositoryOpenTwoID, assetTwoID, "2026-07-31T00:00:00Z", "100"}, {repositoryCloseTwoID, assetTwoID, "2026-08-01T00:00:00Z", "100"},
	} {
		mustExec(t, ctx, conn, `INSERT INTO market_price_observations (id, calculation_window_id, market_asset_id, observed_at, raw_price, price, quality_status) VALUES ($1, $2, $3, $4, $5, $6::numeric, 'VALID')`, observation.id, repositoryWindowID, observation.asset, observation.at, observation.raw, observation.raw)
	}
}

func repositoryCalculationInput() basket.CalculationInput {
	start := time.Date(2026, time.July, 31, 0, 0, 0, 0, time.UTC)
	return basket.CalculationInput{
		Mapping:      basket.MappingVersion{ID: mappingOneID, Key: "crest_large_cap", Sector: sector.Crest, MinimumCoveredWeight: 100000000, NormalizationCap: 2000000, ExpectedTurbulence: basket.ExpectedTurbulencePolicy{ID: repositoryPolicyID, Type: basket.TurbulencePolicyStaticContentValue, StaticValue: 4000000, Floor: 250000}, Components: []basket.MappingComponent{{MarketAssetID: assetOneID, TargetWeight: 40000000, Enabled: true}, {MarketAssetID: assetTwoID, TargetWeight: 60000000, Enabled: true}}},
		Window:       basket.CalculationWindow{ID: repositoryWindowID, DailyTideID: repositoryDailyTideID, ProviderKey: "fixture", Start: start, End: start.Add(24 * time.Hour), Tolerance: 5 * time.Minute},
		Observations: []basket.Observation{{ID: repositoryOpenOneID, ProviderKey: "fixture", MarketAssetID: assetOneID, ObservedAt: start, PriceUnits: 100, Status: basket.ObservationStatusValid}, {ID: repositoryCloseOneID, ProviderKey: "fixture", MarketAssetID: assetOneID, ObservedAt: start.Add(24 * time.Hour), PriceUnits: 110, Status: basket.ObservationStatusValid}, {ID: repositoryOpenTwoID, ProviderKey: "fixture", MarketAssetID: assetTwoID, ObservedAt: start, PriceUnits: 100, Status: basket.ObservationStatusValid}, {ID: repositoryCloseTwoID, ProviderKey: "fixture", MarketAssetID: assetTwoID, ObservedAt: start.Add(24 * time.Hour), PriceUnits: 100, Status: basket.ObservationStatusValid}},
	}
}
