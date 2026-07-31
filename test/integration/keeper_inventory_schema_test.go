//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	invPlayerID    = "00000000-0000-0000-0000-0000000000b1"
	invVoyageID    = "00000000-0000-0000-0000-0000000000c1"
	invKeeperDefID = "00000000-0000-0000-0000-000000000021"
)

func connectInventorySchema(t *testing.T, ctx context.Context, prefix string) (*pgx.Conn, string) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}
	schema := prefix + "_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _ = conn.Close(ctx) })
	t.Cleanup(func() {
		conn2, err := pgx.Connect(ctx, databaseURL)
		if err == nil {
			_, _ = conn2.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
			_ = conn2.Close(ctx)
		}
	})
	mustExec(t, ctx, conn, "SET search_path TO "+schema)
	return conn, schema
}

func applyLegacyInventoryMigrations(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	applyMigration(t, ctx, conn, "000001_foundation.up.sql")
	applyMigration(t, ctx, conn, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, conn, "000003_auth_credentials.up.sql")
	applyMigration(t, ctx, conn, "000004_player_keeper_instances.up.sql")
	applyMigration(t, ctx, conn, "000006_voyage_lifecycle.up.sql")
	applyMigration(t, ctx, conn, "000007_keeper_definition_upgrade_nodes.up.sql")
}

func seedLegacyInventory(t *testing.T, ctx context.Context, conn *pgx.Conn, playerID, voyageID string) {
	t.Helper()
	mustExec(t, ctx, conn,
		"INSERT INTO accounts (id, email, password_hash) VALUES ($1, 'a@inv.test', 'hash-a')",
		playerID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO players (id, account_id, public_id) VALUES ($1, $2, 'plr_inv_legacy')",
		playerID, playerID,
	)
	mustExec(t, ctx, conn, `
		INSERT INTO voyages (
			id, public_id, player_id, status, current_day_number,
			fund_health, max_fund_health, capital, started_at,
			voyage_definition_version_id
		) VALUES ($1, 'voy_inv_legacy', $2, 'ACTIVE', 1, 100, 100, 10, transaction_timestamp(),
			'00000000-0000-0000-0000-000000000002')
	`, voyageID, playerID)
}

func insertLegacyKeeperInstance(t *testing.T, ctx context.Context, conn *pgx.Conn, id, publicID, playerID, acquiredAt string, level int, voyageID *string) {
	t.Helper()
	args := []any{id, publicID, playerID, invKeeperDefID, level, acquiredAt}
	sql := `
		INSERT INTO keeper_instances (
			id, public_id, player_id, keeper_definition_version_id, level, acquired_at, voyage_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	if voyageID == nil {
		args = append(args, nil)
	} else {
		args = append(args, *voyageID)
	}
	mustExec(t, ctx, conn, sql, args...)
}

func readMigrationFile(t *testing.T, filename string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", filename))
	if err != nil {
		t.Fatalf("read migration %s: %v", filename, err)
	}
	return string(content)
}

func TestKeeperInventoryMigrationLegacyRoundTrip(t *testing.T) {
	ctx := context.Background()
	conn, _ := connectInventorySchema(t, ctx, "inv_roundtrip")
	applyLegacyInventoryMigrations(t, ctx, conn)
	seedLegacyInventory(t, ctx, conn, invPlayerID, invVoyageID)

	voyageID := invVoyageID
	insertLegacyKeeperInstance(t, ctx, conn, "00000000-0000-0000-0000-0000000000d1", "kpr_legacy_one", invPlayerID, "2026-01-01T00:00:00Z", 3, nil)
	insertLegacyKeeperInstance(t, ctx, conn, "00000000-0000-0000-0000-0000000000d2", "kpr_legacy_two", invPlayerID, "2026-01-10T00:00:00Z", 5, nil)
	insertLegacyKeeperInstance(t, ctx, conn, "00000000-0000-0000-0000-0000000000d3", "kpr_legacy_three", invPlayerID, "2026-01-15T00:00:00Z", 2, &voyageID)

	applyMigration(t, ctx, conn, "000008_meta_and_voyage_keeper_inventory.up.sql")

	var (
		unlockKey          string
		unlockSource       string
		unlockDefVersionID string
		unlockedAt         time.Time
	)
	if err := conn.QueryRow(ctx, `
		SELECT u.keeper_key, u.unlock_source, u.unlocked_definition_version_id, u.unlocked_at
		FROM player_keeper_unlocks AS u
		WHERE u.player_id = $1
	`, invPlayerID).Scan(&unlockKey, &unlockSource, &unlockDefVersionID, &unlockedAt); err != nil {
		t.Fatalf("load unlock: %v", err)
	}
	if unlockKey != "crest_sovereign" {
		t.Errorf("unlock key = %q, want crest_sovereign", unlockKey)
	}
	if unlockSource != "LEGACY_MIGRATION" {
		t.Errorf("unlock source = %q, want LEGACY_MIGRATION", unlockSource)
	}
	if unlockDefVersionID != invKeeperDefID {
		t.Errorf("unlock definition version = %q, want %q", unlockDefVersionID, invKeeperDefID)
	}
	if want := timeMustParse(t, "2026-01-01T00:00:00Z"); !unlockedAt.Equal(want) {
		t.Errorf("unlocked at = %v, want earliest legacy acquisition %v", unlockedAt, want)
	}

	var (
		instanceVoyageID string
		upgradeNodeKey   string
		acquiredDay      int
		acquiredSource   string
		instanceCount    int
		unlockCount      int
		playerColExists  bool
		levelColExists   bool
		voyageNotNull    bool
	)
	if err := conn.QueryRow(ctx, `
		SELECT i.voyage_id::text, i.upgrade_node_key, i.acquired_day, i.acquired_source
		FROM keeper_instances AS i
	`).Scan(&instanceVoyageID, &upgradeNodeKey, &acquiredDay, &acquiredSource); err != nil {
		t.Fatalf("load retained instance: %v", err)
	}
	if instanceVoyageID != invVoyageID {
		t.Errorf("instance voyage = %q, want %q", instanceVoyageID, invVoyageID)
	}
	if upgradeNodeKey != "base" {
		t.Errorf("instance upgrade node = %q, want base", upgradeNodeKey)
	}
	if acquiredDay != 1 {
		t.Errorf("instance acquired day = %d, want 1", acquiredDay)
	}
	if acquiredSource != "STARTER" {
		t.Errorf("instance acquired source = %q, want STARTER", acquiredSource)
	}

	if err := conn.QueryRow(ctx, "SELECT count(*) FROM keeper_instances").Scan(&instanceCount); err != nil {
		t.Fatalf("count instances: %v", err)
	}
	if instanceCount != 1 {
		t.Errorf("retained instance count = %d, want 1", instanceCount)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM player_keeper_unlocks").Scan(&unlockCount); err != nil {
		t.Fatalf("count unlocks: %v", err)
	}
	if unlockCount != 1 {
		t.Errorf("unlock count = %d, want 1 (duplicate player-only rows collapse)", unlockCount)
	}

	for _, check := range []struct {
		column string
		exists *bool
	}{
		{"player_id", &playerColExists},
		{"level", &levelColExists},
	} {
		if err := conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = current_schema() AND table_name = 'keeper_instances' AND column_name = $1
			)
		`, check.column).Scan(check.exists); err != nil {
			t.Fatalf("check column %q: %v", check.column, err)
		}
	}
	if playerColExists || levelColExists {
		t.Errorf("keeper_instances still has player_id=%v level=%v, want both dropped", playerColExists, levelColExists)
	}
	if err := conn.QueryRow(ctx, `
		SELECT is_nullable = 'NO'
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'keeper_instances' AND column_name = 'voyage_id'
	`).Scan(&voyageNotNull); err != nil {
		t.Fatalf("check voyage_id nullability: %v", err)
	}
	if !voyageNotNull {
		t.Error("keeper_instances.voyage_id should be NOT NULL after up")
	}

	applyMigration(t, ctx, conn, "000008_meta_and_voyage_keeper_inventory.down.sql")

	var (
		restoredPlayerID string
		restoredLevel    int
	)
	if err := conn.QueryRow(ctx, "SELECT player_id, level FROM keeper_instances").Scan(&restoredPlayerID, &restoredLevel); err != nil {
		t.Fatalf("load restored instance: %v", err)
	}
	if restoredPlayerID != invPlayerID {
		t.Errorf("restored player ID = %q, want %q", restoredPlayerID, invPlayerID)
	}
	if restoredLevel != 1 {
		t.Errorf("restored level = %d, want 1 (root node depth 0 + 1)", restoredLevel)
	}

	var unlockTableExists bool
	if err := conn.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = current_schema() AND table_name = 'player_keeper_unlocks'
		)
	`).Scan(&unlockTableExists); err != nil {
		t.Fatalf("check unlock table: %v", err)
	}
	if unlockTableExists {
		t.Error("player_keeper_unlocks should be dropped after down")
	}

	for _, indexName := range []string{"keeper_instances_player_acquired_idx", "keeper_instances_voyage_idx"} {
		var idxExists bool
		if err := conn.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = current_schema() AND tablename = 'keeper_instances' AND indexname = $1
			)
		`, indexName).Scan(&idxExists); err != nil {
			t.Fatalf("check index %q: %v", indexName, err)
		}
		if !idxExists {
			t.Errorf("index %s missing after down", indexName)
		}
	}
}

func TestKeeperInventoryMigrationRejectsOwnershipMismatch(t *testing.T) {
	ctx := context.Background()
	conn, _ := connectInventorySchema(t, ctx, "inv_owner_mismatch")
	applyLegacyInventoryMigrations(t, ctx, conn)
	seedLegacyInventory(t, ctx, conn, invPlayerID, invVoyageID)

	otherPlayerID := "00000000-0000-0000-0000-0000000000b2"
	voyageID := invVoyageID
	mustExec(t, ctx, conn,
		"INSERT INTO accounts (id, email, password_hash) VALUES ($1, 'b@inv.test', 'hash-b')",
		otherPlayerID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO players (id, account_id, public_id) VALUES ($1, $2, 'plr_inv_other')",
		otherPlayerID, otherPlayerID,
	)

	insertLegacyKeeperInstance(t, ctx, conn, "00000000-0000-0000-0000-0000000000d1", "kpr_mismatch", otherPlayerID, "2026-01-01T00:00:00Z", 1, &voyageID)

	expectExecError(t, ctx, conn, readMigrationFile(t, "000008_meta_and_voyage_keeper_inventory.up.sql"))
}

func TestKeeperInventoryUpgradePathFromLegacyEmptyTree(t *testing.T) {
	ctx := context.Background()
	conn, _ := connectInventorySchema(t, ctx, "inv_upgrade_path")
	applyMigration(t, ctx, conn, "000001_foundation.up.sql")
	applyMigration(t, ctx, conn, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, conn, "000003_auth_credentials.up.sql")
	applyMigration(t, ctx, conn, "000004_player_keeper_instances.up.sql")
	applyMigration(t, ctx, conn, "000005_four_sector_calculation.up.sql")
	applyMigration(t, ctx, conn, "000006_voyage_lifecycle.up.sql")

	// A database that already ran the originally shipped 000006 has the MVP
	// definition (content_version 1) with an empty upgrade_tree while its
	// release is already PUBLISHED. The shipped migration cannot be replayed on
	// a fresh schema (it violates the content_releases publication check and the
	// catalog draft-only guards), so reproduce its exact resulting state by
	// rewriting the seeded tree back to '{}' under the same guard bypass that
	// 000007 applies.
	mustExec(t, ctx, conn, "ALTER TABLE keeper_definition_versions DISABLE TRIGGER keeper_definition_versions_draft_only_trigger")
	mustExec(t, ctx, conn, `
		UPDATE keeper_definition_versions
		SET upgrade_tree = '{}'::jsonb
		WHERE id = $1 AND keeper_key = 'crest_sovereign' AND content_version = 1
	`, invKeeperDefID)
	mustExec(t, ctx, conn, "ALTER TABLE keeper_definition_versions ENABLE TRIGGER keeper_definition_versions_draft_only_trigger")

	var legacyTree string
	if err := conn.QueryRow(ctx, `
		SELECT upgrade_tree::text
		FROM keeper_definition_versions
		WHERE id = $1
	`, invKeeperDefID).Scan(&legacyTree); err != nil {
		t.Fatalf("load legacy definition tree: %v", err)
	}
	if legacyTree != "{}" {
		t.Fatalf("legacy definition upgrade_tree = %q, want {} to reproduce the shipped 000006 state", legacyTree)
	}

	applyMigration(t, ctx, conn, "000007_keeper_definition_upgrade_nodes.up.sql")

	var (
		nodeRows int
		nodeRoot int
		depth    int16
	)
	if err := conn.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE is_root)
		FROM keeper_definition_upgrade_nodes
		WHERE keeper_definition_version_id = $1
	`, invKeeperDefID).Scan(&nodeRows, &nodeRoot); err != nil {
		t.Fatalf("count repaired upgrade nodes: %v", err)
	}
	if nodeRows != 3 || nodeRoot != 1 {
		t.Fatalf("repaired nodes = %d rows / %d roots, want 3 rows / 1 root", nodeRows, nodeRoot)
	}
	if err := conn.QueryRow(ctx, `
		SELECT depth
		FROM keeper_definition_upgrade_nodes
		WHERE keeper_definition_version_id = $1 AND node_key = 'deep_crown'
	`, invKeeperDefID).Scan(&depth); err != nil {
		t.Fatalf("load repaired deep_crown depth: %v", err)
	}
	if depth != 1 {
		t.Errorf("repaired deep_crown depth = %d, want 1", depth)
	}

	applyMigration(t, ctx, conn, "000008_meta_and_voyage_keeper_inventory.up.sql")

	var unlockTableExists bool
	if err := conn.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = current_schema() AND table_name = 'player_keeper_unlocks'
		)
	`).Scan(&unlockTableExists); err != nil {
		t.Fatalf("check unlock table: %v", err)
	}
	if !unlockTableExists {
		t.Error("player_keeper_unlocks should exist after upgrading the shipped 000006 database")
	}

	var repairedRoot string
	if err := conn.QueryRow(ctx, `
		SELECT upgrade_tree->>'rootNodeKey'
		FROM keeper_definition_versions
		WHERE id = $1
	`, invKeeperDefID).Scan(&repairedRoot); err != nil {
		t.Fatalf("load repaired definition tree: %v", err)
	}
	if repairedRoot != "base" {
		t.Errorf("repaired definition root node = %q, want base", repairedRoot)
	}
}
