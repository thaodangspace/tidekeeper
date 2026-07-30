//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
)

const (
	sectorDefinitionID  = "00000000-0000-0000-0000-000000000031"
	turbulencePolicyID  = "00000000-0000-0000-0000-000000000032"
	dailyTideID         = "00000000-0000-0000-0000-000000000033"
	calculationWindowID = "00000000-0000-0000-0000-000000000034"
	basketMetricID      = "00000000-0000-0000-0000-000000000035"
	sectorBenchmarkID   = "00000000-0000-0000-0000-000000000036"
)

func TestFourSectorCalculationSchema(t *testing.T) {
	ctx := context.Background()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	t.Cleanup(func() { conn.Close(ctx) })

	schema := "four_sector_calculation_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, conn, "SET search_path TO "+schema)
	applyMigration(t, ctx, conn, "000001_foundation.up.sql")
	applyMigration(t, ctx, conn, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, conn, "000005_four_sector_calculation.up.sql")
	seedValidDraft(t, ctx, conn)

	var sectors int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM sectors").Scan(&sectors); err != nil {
		t.Fatalf("count target sectors: %v", err)
	}
	if sectors != 4 {
		t.Fatalf("target sector count = %d, want 4", sectors)
	}

	expectExecError(t, ctx, conn, `
		INSERT INTO sector_definition_versions (
			id, sector_key, content_version, benchmark_method, minimum_eligible_baskets,
			relative_scale_units, relative_blend_weight_units, rank_blend_weight_units, score_cap_units
		) VALUES ($1, 'FORGE', 1, 'EQUAL_WEIGHT', 2, 500000, 700000, 300000, 1000000)`,
		"00000000-0000-0000-0000-000000000030",
	)

	mustExec(t, ctx, conn, `
		INSERT INTO sector_definition_versions (
			id, sector_key, content_version, benchmark_method, minimum_eligible_baskets,
			relative_scale_units, relative_blend_weight_units, rank_blend_weight_units, score_cap_units
		) VALUES ($1, 'CREST', 1, 'EQUAL_WEIGHT', 2, 500000, 700000, 300000, 1000000)`,
		sectorDefinitionID,
	)
	mustExec(t, ctx, conn, `
		INSERT INTO expected_turbulence_policies (
			id, policy_key, content_version, policy_type, static_value_units, floor_units, rounding_mode
		) VALUES ($1, 'crest_static', 1, 'STATIC_CONTENT_VALUE', 4000000, 250000, 'ROUND_HALF_AWAY_FROM_ZERO')`,
		turbulencePolicyID,
	)
	mustExec(t, ctx, conn, `
		UPDATE basket_mapping_versions
		SET sector_definition_version_id = $1,
			minimum_covered_weight_units = 80000000,
			expected_turbulence_policy_id = $2,
			normalization_cap_units = 2000000,
			benchmark_eligible = true
		WHERE id = $3`,
		sectorDefinitionID, turbulencePolicyID, mappingOneID,
	)
	mustExec(t, ctx, conn, `
		INSERT INTO daily_tides (
			id, day_key, sequence_number, phase, lock_at, settle_after, content_version
		) VALUES ($1, '2026-07-30', 1, 'DATA_PENDING',
			'2026-07-30T00:00:00Z', '2026-07-30T00:01:00Z', 1)`,
		dailyTideID,
	)
	mustExec(t, ctx, conn, `
		INSERT INTO market_calculation_windows (
			id, daily_tide_id, provider_key, data_window_start, data_window_end,
			required_granularity_seconds, observation_tolerance_seconds
		) VALUES ($1, $2, 'fixture_provider', '2026-07-30T00:00:00Z',
			'2026-07-31T00:00:00Z', 60, 300)`,
		calculationWindowID, dailyTideID,
	)
	mustExec(t, ctx, conn, `
		INSERT INTO basket_metrics (
			id, daily_tide_id, basket_mapping_version_id, calculation_window_id, status,
			covered_weight_units, raw_return_units, expected_turbulence_units,
			normalized_performance_units, input_checksum
		) VALUES ($1, $2, $3, $4, 'READY', 100000000, 4000000, 4000000, 1000000, $5)`,
		basketMetricID, dailyTideID, mappingOneID, calculationWindowID, bytes.Repeat([]byte{1}, 32),
	)

	expectExecError(t, ctx, conn, `
		INSERT INTO basket_metrics (
			id, daily_tide_id, basket_mapping_version_id, calculation_window_id, status,
			covered_weight_units, raw_return_units, expected_turbulence_units,
			normalized_performance_units, input_checksum
		) VALUES ($1, $2, $3, $4, 'READY', 100000000, 4000000, 4000000, 1000000, $5)`,
		"00000000-0000-0000-0000-000000000037", dailyTideID, mappingOneID, calculationWindowID, bytes.Repeat([]byte{2}, 32),
	)
	expectExecError(t, ctx, conn, `
		INSERT INTO basket_metrics (
			id, daily_tide_id, basket_mapping_version_id, calculation_window_id, status,
			covered_weight_units, input_checksum
		) VALUES ($1, $2, $3, $4, 'UNKNOWN', 0, $5)`,
		"00000000-0000-0000-0000-000000000038", dailyTideID, mappingOneID, calculationWindowID, bytes.Repeat([]byte{3}, 32),
	)

	mustExec(t, ctx, conn, `
		INSERT INTO sector_benchmarks (
			id, daily_tide_id, sector_definition_version_id, status, calculation_method,
			eligible_basket_count, benchmark_units, minimum_required_baskets, input_checksum
		) VALUES ($1, $2, $3, 'READY', 'EQUAL_WEIGHT', 2, 500000, 2, $4)`,
		sectorBenchmarkID, dailyTideID, sectorDefinitionID, bytes.Repeat([]byte{4}, 32),
	)
	expectExecError(t, ctx, conn, `
		INSERT INTO sector_benchmarks (
			id, daily_tide_id, sector_definition_version_id, status, calculation_method,
			eligible_basket_count, benchmark_units, minimum_required_baskets, input_checksum
		) VALUES ($1, $2, $3, 'READY', 'EQUAL_WEIGHT', 2, 500000, 2, $4)`,
		"00000000-0000-0000-0000-000000000039", dailyTideID, sectorDefinitionID, bytes.Repeat([]byte{5}, 32),
	)

	applyMigration(t, ctx, conn, "000005_four_sector_calculation.down.sql")
	var tableExists bool
	if err := conn.QueryRow(ctx, "SELECT to_regclass(current_schema() || '.sector_benchmarks') IS NOT NULL").Scan(&tableExists); err != nil {
		t.Fatalf("check calculation migration rollback: %v", err)
	}
	if tableExists {
		t.Fatal("calculation tables remain after down migration")
	}
}
