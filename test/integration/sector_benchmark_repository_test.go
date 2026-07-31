//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/market/precision"
	"github.com/thaodangspace/tidekeepers-server/market/sector"
)

func TestSectorBenchmarkRepositoryIsIdempotentAndConflictSafe(t *testing.T) {
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
	schema := "sector_benchmark_repository_" + randomHex(t)
	mustExec(t, ctx, admin, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, admin, "SET search_path TO "+schema)
	applyMigration(t, ctx, admin, "000001_foundation.up.sql")
	applyMigration(t, ctx, admin, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, admin, "000005_four_sector_calculation.up.sql")
	applyMigration(t, ctx, admin, "000007_keeper_definition_upgrade_nodes.up.sql")
	seedValidDraft(t, ctx, admin)
	mustExec(t, ctx, admin, `INSERT INTO sector_definition_versions (id, sector_key, content_version, benchmark_method, minimum_eligible_baskets, relative_scale_units, relative_blend_weight_units, rank_blend_weight_units, score_cap_units) VALUES ($1, 'CREST', 1, 'EQUAL_WEIGHT', 2, 500000, 700000, 300000, 1000000)`, repositorySectorDefinitionID)
	mustExec(t, ctx, admin, `INSERT INTO daily_tides (id, day_key, sequence_number, phase, lock_at, settle_after, content_version) VALUES ($1, '2026-08-02', 3, 'DATA_PENDING', '2026-08-02T00:00:00Z', '2026-08-02T00:01:00Z', 1)`, repositoryDailyTideID)

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

	definition := sector.Definition{ID: repositorySectorDefinitionID, Sector: sector.Crest, BenchmarkMethod: sector.EqualWeight, MinimumEligibleBaskets: 2, RelativeScale: 500000, RelativeBlendWeight: 700000, RankBlendWeight: 300000, ScoreCap: precision.NormalizedScale}
	var benchmark sector.Benchmark
	repository := sector.NewRepository(pool)
	service := sector.NewService(repository)
	calculated, firstID, idempotent, err := service.CalculateAndPersist(ctx, sector.BenchmarkInput{DailyTideID: repositoryDailyTideID, Definition: definition, Baskets: nil})
	if err != nil || calculated.Status != sector.StatusDataIncomplete {
		t.Fatalf("CalculateAndPersist() = %#v, err=%v", calculated, err)
	}
	if idempotent || firstID == "" {
		t.Fatalf("first CalculateAndPersist() = id:%q idempotent:%t", firstID, idempotent)
	}
	benchmark = calculated
	firstID, idempotent, err = repository.PersistBenchmark(ctx, repositoryDailyTideID, definition, benchmark)
	if err != nil || !idempotent || firstID == "" {
		t.Fatalf("idempotent PersistBenchmark() = id:%q idempotent:%t err:%v", firstID, idempotent, err)
	}
	readiness, err := repository.LoadReadiness(ctx, repositoryDailyTideID, 1)
	if err != nil || readiness.Ready || len(readiness.Sectors) != 4 || readiness.Sectors[0].Sector != sector.Crest || readiness.Sectors[0].MinimumRequiredBaskets != 2 {
		t.Fatalf("LoadReadiness() = %#v, err=%v", readiness, err)
	}
	secondID, idempotent, err := repository.PersistBenchmark(ctx, repositoryDailyTideID, definition, benchmark)
	if err != nil || !idempotent || secondID != firstID {
		t.Fatalf("second PersistBenchmark() = id:%q idempotent:%t err:%v", secondID, idempotent, err)
	}
	benchmark.InputChecksum[0]++
	if _, _, err := repository.PersistBenchmark(ctx, repositoryDailyTideID, definition, benchmark); !errors.Is(err, sector.ErrBenchmarkConflict) {
		t.Fatalf("conflicting PersistBenchmark() error = %v, want ErrBenchmarkConflict", err)
	}
}
