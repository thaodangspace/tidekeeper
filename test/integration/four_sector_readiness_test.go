//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/keeper"
	"github.com/thaodangspace/tidekeepers-server/market/sector"
)

func TestFourSectorReadinessRequiresEveryTargetBenchmark(t *testing.T) {
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
	schema := "four_sector_readiness_" + randomHex(t)
	mustExec(t, ctx, admin, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, admin, "SET search_path TO "+schema)
	applyMigration(t, ctx, admin, "000001_foundation.up.sql")
	applyMigration(t, ctx, admin, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, admin, "000005_four_sector_calculation.up.sql")
	applyMigration(t, ctx, admin, "000007_keeper_definition_upgrade_nodes.up.sql")
	applyMigration(t, ctx, admin, "000009_gameplay_content_foundation.up.sql")

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
	if _, err := keeper.NewPublisher(pool).Publish(ctx, keeper.CatalogV2()); err != nil {
		t.Fatalf("publish CatalogV2: %v", err)
	}

	tideID := uuid.New().String()
	mustExec(t, ctx, admin, `INSERT INTO daily_tides (id, day_key, sequence_number, phase, lock_at, settle_after, content_version) VALUES ($1, '2026-08-03', 4, 'DATA_PENDING', '2026-08-03T00:00:00Z', '2026-08-03T00:01:00Z', 2)`, tideID)
	rows, err := admin.Query(ctx, `SELECT id, sector_key FROM sector_definition_versions WHERE content_version = 2 ORDER BY sector_key`)
	if err != nil {
		t.Fatalf("load sector definitions: %v", err)
	}
	defer rows.Close()
	definitionIDs := make([]string, 0, 4)
	for rows.Next() {
		var definitionID, sectorKey string
		if err := rows.Scan(&definitionID, &sectorKey); err != nil {
			t.Fatalf("scan sector definition: %v", err)
		}
		definitionIDs = append(definitionIDs, definitionID)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sector definitions: %v", err)
	}
	rows.Close()
	if len(definitionIDs) != 4 {
		t.Fatalf("target definitions = %d, want 4", len(definitionIDs))
	}
	for _, definitionID := range definitionIDs {
		mustExec(t, ctx, admin, `INSERT INTO sector_benchmarks (id, daily_tide_id, sector_definition_version_id, status, calculation_method, eligible_basket_count, benchmark_units, minimum_required_baskets, input_checksum) VALUES ($1, $2, $3, 'READY', 'EQUAL_WEIGHT', 2, 0, 2, $4)`, uuid.New().String(), tideID, definitionID, make([]byte, 32))
	}

	repository := sector.NewRepository(pool)
	readiness, err := repository.LoadReadiness(ctx, tideID, 2)
	if err != nil || !readiness.Ready || len(readiness.Sectors) != 4 {
		t.Fatalf("all-target readiness = %#v, err=%v", readiness, err)
	}
	mustExec(t, ctx, admin, `UPDATE sector_benchmarks SET status = 'DATA_INCOMPLETE' WHERE daily_tide_id = $1 AND sector_definition_version_id = (SELECT id FROM sector_definition_versions WHERE content_version = 2 AND sector_key = 'CURRENT')`, tideID)
	readiness, err = repository.LoadReadiness(ctx, tideID, 2)
	if err != nil || readiness.Ready {
		t.Fatalf("incomplete Current did not block readiness: %#v, err=%v", readiness, err)
	}
}
