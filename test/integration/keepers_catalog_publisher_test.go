//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

func TestKeeperCatalogPublisherIsIdempotentAndConflictSafe(t *testing.T) {
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

	schema := "keepers_catalog_publisher_" + randomHex(t)
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
	poolConfig.MinConns = 0
	poolConfig.MaxConns = 2
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("open publisher pool: %v", err)
	}
	t.Cleanup(pool.Close)

	publisher := keeper.NewPublisher(pool)
	first, err := publisher.Publish(ctx, keeper.CatalogV1())
	if err != nil {
		t.Fatalf("first Publish(): %v", err)
	}
	if first.Idempotent || first.DefinitionCount != 10 || first.MappingCount != 10 || first.ComponentCount != 48 || first.NodeCount != 30 {
		t.Fatalf("first Publish() result = %#v", first)
	}

	second, err := publisher.Publish(ctx, keeper.CatalogV1())
	if err != nil {
		t.Fatalf("second Publish(): %v", err)
	}
	if !second.Idempotent || second.DefinitionCount != 10 || second.MappingCount != 10 || second.ComponentCount != 48 || second.NodeCount != 30 {
		t.Fatalf("second Publish() result = %#v", second)
	}
	if second.Checksum != first.Checksum {
		t.Fatalf("idempotent checksum = %x, want %x", second.Checksum, first.Checksum)
	}

	conflicting := keeper.CatalogV1()
	conflicting.Definitions[0].Name = "Changed Crest Sovereign"
	if _, err := publisher.Publish(ctx, conflicting); !errors.Is(err, keeper.ErrReleaseConflict) {
		t.Fatalf("Publish(conflicting) error = %v, want ErrReleaseConflict", err)
	}

	var definitions, mappings, components, nodes int
	if err := admin.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM keeper_definition_versions),
			(SELECT count(*) FROM basket_mapping_versions),
			(SELECT count(*) FROM basket_mapping_components),
			(SELECT count(*) FROM keeper_definition_upgrade_nodes)
	`).Scan(&definitions, &mappings, &components, &nodes); err != nil {
		t.Fatalf("count persisted catalog rows: %v", err)
	}
	if definitions != 10 || mappings != 10 || components != 48 || nodes != 30 {
		t.Fatalf("persisted counts = definitions:%d mappings:%d components:%d nodes:%d", definitions, mappings, components, nodes)
	}

	v2, err := publisher.Publish(ctx, keeper.CatalogV2())
	if err != nil {
		t.Fatalf("Publish(CatalogV2): %v", err)
	}
	if v2.Idempotent || v2.DefinitionCount != 13 || v2.MappingCount != 13 || v2.ComponentCount != 59 || v2.NodeCount != 39 {
		t.Fatalf("CatalogV2 publish result = %#v", v2)
	}
	v2Retry, err := publisher.Publish(ctx, keeper.CatalogV2())
	if err != nil || !v2Retry.Idempotent || v2Retry.Checksum != v2.Checksum {
		t.Fatalf("CatalogV2 retry = %#v, err=%v", v2Retry, err)
	}
	if v2Retry.NodeCount != 39 {
		t.Fatalf("CatalogV2 retry node count = %d, want 39", v2Retry.NodeCount)
	}
}
