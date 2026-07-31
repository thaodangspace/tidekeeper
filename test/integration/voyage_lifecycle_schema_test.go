//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestVoyageLifecycleMigrationRoundTrip(t *testing.T) {
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

	schema := "voyage_lifecycle_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, conn, "SET search_path TO "+schema)

	applyMigration(t, ctx, conn, "000001_foundation.up.sql")
	applyMigration(t, ctx, conn, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, conn, "000003_auth_credentials.up.sql")
	applyMigration(t, ctx, conn, "000004_player_keeper_instances.up.sql")
	applyMigration(t, ctx, conn, "000006_voyage_lifecycle.up.sql")
	applyMigration(t, ctx, conn, "000007_keeper_definition_upgrade_nodes.up.sql")

	var releaseStatus string
	var releasePublishedAt pgtype.Timestamptz
	if err := conn.QueryRow(ctx, "SELECT status, published_at FROM content_releases WHERE version = 1").Scan(&releaseStatus, &releasePublishedAt); err != nil {
		t.Fatalf("load seeded content release: %v", err)
	}
	if releaseStatus != "PUBLISHED" {
		t.Errorf("seeded content release status = %q, want PUBLISHED", releaseStatus)
	}
	if !releasePublishedAt.Valid {
		t.Error("seeded content release has no publication timestamp")
	}

	var rootKey string
	var nodeCount int
	if err := conn.QueryRow(ctx, `
		SELECT upgrade_tree->>'rootNodeKey', jsonb_array_length(upgrade_tree->'nodes')
		FROM keeper_definition_versions
		WHERE id = '00000000-0000-0000-0000-000000000021'
	`).Scan(&rootKey, &nodeCount); err != nil {
		t.Fatalf("load seeded Keeper definition: %v", err)
	}
	if rootKey != "base" {
		t.Errorf("seeded Keeper root node = %q, want base", rootKey)
	}
	if nodeCount != 3 {
		t.Errorf("seeded Keeper node count = %d, want 3", nodeCount)
	}

	var nodeRows int
	var nodeRoots int
	if err := conn.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE is_root)
		FROM keeper_definition_upgrade_nodes
		WHERE keeper_definition_version_id = '00000000-0000-0000-0000-000000000021'
	`).Scan(&nodeRows, &nodeRoots); err != nil {
		t.Fatalf("count backfilled upgrade nodes: %v", err)
	}
	if nodeRows != 3 || nodeRoots != 1 {
		t.Errorf("backfilled nodes = %d rows / %d roots, want 3 rows / 1 root", nodeRows, nodeRoots)
	}
	var deepCrownDepth int16
	if err := conn.QueryRow(ctx, `
		SELECT depth FROM keeper_definition_upgrade_nodes
		WHERE keeper_definition_version_id = '00000000-0000-0000-0000-000000000021'
		  AND node_key = 'deep_crown'
	`).Scan(&deepCrownDepth); err != nil {
		t.Fatalf("load deep_crown depth: %v", err)
	}
	if deepCrownDepth != 1 {
		t.Errorf("deep_crown depth = %d, want 1", deepCrownDepth)
	}

	var standardStatus string
	var legacyStatus string
	if err := conn.QueryRow(ctx, "SELECT status FROM voyage_definition_versions WHERE definition_key = 'standard'").Scan(&standardStatus); err != nil {
		t.Fatalf("load standard voyage definition: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT status FROM voyage_definition_versions WHERE definition_key = 'legacy'").Scan(&legacyStatus); err != nil {
		t.Fatalf("load legacy voyage definition: %v", err)
	}
	if standardStatus != "PUBLISHED" || legacyStatus != "PUBLISHED" {
		t.Errorf("voyage definition statuses = standard:%s legacy:%s, want PUBLISHED/PUBLISHED", standardStatus, legacyStatus)
	}

	var columnExists, indexExists, fkExists bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'keeper_instances' AND column_name = 'voyage_id')").Scan(&columnExists); err != nil {
		t.Fatalf("check keeper_instances.voyage_id column: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND tablename = 'keeper_instances' AND indexname = 'keeper_instances_voyage_idx')").Scan(&indexExists); err != nil {
		t.Fatalf("check keeper_instances_voyage_idx: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'keeper_instances_voyage_fk' AND conrelid = (current_schema() || '.keeper_instances')::regclass)").Scan(&fkExists); err != nil {
		t.Fatalf("check keeper_instances_voyage_fk: %v", err)
	}
	if !columnExists || !indexExists || !fkExists {
		t.Errorf("voyage ownership present = column:%t index:%t fk:%t, want true/true/true", columnExists, indexExists, fkExists)
	}

	applyMigration(t, ctx, conn, "000007_keeper_definition_upgrade_nodes.down.sql")

	applyMigration(t, ctx, conn, "000006_voyage_lifecycle.down.sql")

	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'keeper_instances' AND column_name = 'voyage_id')").Scan(&columnExists); err != nil {
		t.Fatalf("recheck keeper_instances.voyage_id column: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE schemaname = current_schema() AND tablename = 'keeper_instances' AND indexname = 'keeper_instances_voyage_idx')").Scan(&indexExists); err != nil {
		t.Fatalf("recheck keeper_instances_voyage_idx: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'keeper_instances_voyage_fk' AND conrelid = (current_schema() || '.keeper_instances')::regclass)").Scan(&fkExists); err != nil {
		t.Fatalf("recheck keeper_instances_voyage_fk: %v", err)
	}
	if columnExists || indexExists || fkExists {
		t.Errorf("voyage ownership removed = column:%t index:%t fk:%t, want false/false/false", columnExists, indexExists, fkExists)
	}

	var playerColumn, levelColumn, acquiredAtColumn bool
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'keeper_instances' AND column_name = 'player_id')").Scan(&playerColumn); err != nil {
		t.Fatalf("recheck keeper_instances.player_id: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'keeper_instances' AND column_name = 'level')").Scan(&levelColumn); err != nil {
		t.Fatalf("recheck keeper_instances.level: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'keeper_instances' AND column_name = 'acquired_at')").Scan(&acquiredAtColumn); err != nil {
		t.Fatalf("recheck keeper_instances.acquired_at: %v", err)
	}
	if !playerColumn || !levelColumn || !acquiredAtColumn {
		t.Errorf("post-000004 ownership restored = player:%t level:%t acquired_at:%t, want true/true/true", playerColumn, levelColumn, acquiredAtColumn)
	}

	var keeperTableExists bool
	if err := conn.QueryRow(ctx, "SELECT to_regclass(current_schema() || '.keeper_definition_versions') IS NOT NULL").Scan(&keeperTableExists); err != nil {
		t.Fatalf("check keeper_definition_versions survives: %v", err)
	}
	if !keeperTableExists {
		t.Error("keeper_definition_versions missing after 000006 down migration")
	}
}
