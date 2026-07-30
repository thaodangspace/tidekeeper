//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	assetOneID   = "00000000-0000-0000-0000-000000000001"
	assetTwoID   = "00000000-0000-0000-0000-000000000002"
	assetThreeID = "00000000-0000-0000-0000-000000000003"
	mappingOneID = "00000000-0000-0000-0000-000000000011"
	keeperOneID  = "00000000-0000-0000-0000-000000000021"
)

func TestKeepersCatalogSchema(t *testing.T) {
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

	schema := "keepers_catalog_test_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, conn, "SET search_path TO "+schema)

	applyMigration(t, ctx, conn, "000001_foundation.up.sql")
	applyMigration(t, ctx, conn, "000002_keepers_catalog.up.sql")

	seedValidDraft(t, ctx, conn)

	expectExecError(t, ctx, conn,
		"INSERT INTO basket_mapping_components (basket_mapping_version_id, market_asset_id, weight) VALUES ($1, $2, 0.10000000)",
		mappingOneID, assetOneID,
	)
	expectExecError(t, ctx, conn,
		"INSERT INTO basket_mapping_components (basket_mapping_version_id, market_asset_id, weight) VALUES ($1, $2, 0)",
		mappingOneID, assetThreeID,
	)
	expectExecError(t, ctx, conn,
		`INSERT INTO keeper_definition_versions (
			id, keeper_key, content_version, name, current_key, current_name,
			sector_key, role_key, rarity_key, base_risk_key, passive_rule_key,
			passive_rule_config, upgrade_tree, basket_mapping_version_id
		) VALUES ($1, 'invalid_sector', 1, 'Invalid Sector', 'invalid_current', 'Invalid Current',
			'INVALID', 'VANGUARD', 'COMMON', 'LOW_TO_MEDIUM', 'TEST_RULE', '{}'::jsonb, '{}'::jsonb, $2)`,
		"00000000-0000-0000-0000-000000000022", mappingOneID,
	)
	expectExecError(t, ctx, conn,
		`INSERT INTO keeper_definition_versions (
			id, keeper_key, content_version, name, current_key, current_name,
			sector_key, role_key, rarity_key, base_risk_key, passive_rule_key,
			passive_rule_config, upgrade_tree, basket_mapping_version_id
		) VALUES ($1, 'invalid_json', 1, 'Invalid JSON', 'invalid_current', 'Invalid Current',
			'CREST', 'VANGUARD', 'COMMON', 'LOW_TO_MEDIUM', 'TEST_RULE', '[]'::jsonb, '{}'::jsonb, $2)`,
		"00000000-0000-0000-0000-000000000023", mappingOneID,
	)
	expectExecError(t, ctx, conn,
		`INSERT INTO keeper_definition_versions (
			id, keeper_key, content_version, name, current_key, current_name,
			sector_key, role_key, rarity_key, base_risk_key, expected_turbulence_bps, passive_rule_key,
			passive_rule_config, upgrade_tree, basket_mapping_version_id
		) VALUES ($1, 'invalid_turbulence', 1, 'Invalid Turbulence', 'invalid_current', 'Invalid Current',
			'CREST', 'VANGUARD', 'COMMON', 'LOW_TO_MEDIUM', 0, 'TEST_RULE', '{}'::jsonb, '{}'::jsonb, $2)`,
		"00000000-0000-0000-0000-000000000025", mappingOneID,
	)

	mustExec(t, ctx, conn,
		"INSERT INTO content_releases (version, status, checksum) VALUES (2, 'DRAFT', $1)",
		bytes.Repeat([]byte{2}, 32),
	)
	expectExecError(t, ctx, conn,
		`INSERT INTO keeper_definition_versions (
			id, keeper_key, content_version, name, current_key, current_name,
			sector_key, role_key, rarity_key, base_risk_key, passive_rule_key,
			passive_rule_config, upgrade_tree, basket_mapping_version_id
		) VALUES ($1, 'cross_release', 2, 'Cross Release', 'cross_current', 'Cross Current',
			'CREST', 'VANGUARD', 'COMMON', 'LOW_TO_MEDIUM', 'TEST_RULE', '{}'::jsonb, '{}'::jsonb, $2)`,
		"00000000-0000-0000-0000-000000000024", mappingOneID,
	)

	assertDeferredWeightFailure(t, ctx, conn)

	mustExec(t, ctx, conn,
		"UPDATE content_releases SET status = 'PUBLISHED', published_at = transaction_timestamp() WHERE version = 1",
	)
	expectExecError(t, ctx, conn,
		"UPDATE basket_mapping_components SET weight = 0.50000000 WHERE basket_mapping_version_id = $1 AND market_asset_id = $2",
		mappingOneID, assetOneID,
	)
	expectExecError(t, ctx, conn,
		"UPDATE keeper_definition_versions SET name = 'Changed' WHERE id = $1", keeperOneID,
	)
	expectExecError(t, ctx, conn,
		"UPDATE market_assets SET symbol = 'BTX' WHERE id = $1", assetOneID,
	)

	applyMigration(t, ctx, conn, "000002_keepers_catalog.down.sql")
	var catalogTableExists bool
	if err := conn.QueryRow(ctx, "SELECT to_regclass(current_schema() || '.content_releases') IS NOT NULL").Scan(&catalogTableExists); err != nil {
		t.Fatalf("check catalog migration rollback: %v", err)
	}
	if catalogTableExists {
		t.Fatal("catalog tables remain after down migration")
	}
}

func seedValidDraft(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin valid catalog transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	mustExec(t, ctx, tx,
		"INSERT INTO content_releases (version, status, checksum) VALUES (1, 'DRAFT', $1)",
		bytes.Repeat([]byte{1}, 32),
	)
	mustExec(t, ctx, tx,
		"INSERT INTO market_assets (id, asset_key, symbol) VALUES ($1, 'bitcoin', 'BTC'), ($2, 'ethereum', 'ETH'), ($3, 'ethena_usde', 'USDe')",
		assetOneID, assetTwoID, assetThreeID,
	)
	mustExec(t, ctx, tx,
		"INSERT INTO basket_mapping_versions (id, mapping_key, content_version) VALUES ($1, 'crest_large_cap', 1)",
		mappingOneID,
	)
	mustExec(t, ctx, tx,
		"INSERT INTO basket_mapping_components (basket_mapping_version_id, market_asset_id, weight) VALUES ($1, $2, 0.40000000), ($1, $3, 0.60000000)",
		mappingOneID, assetOneID, assetTwoID,
	)
	mustExec(t, ctx, tx,
		`INSERT INTO keeper_definition_versions (
			id, keeper_key, content_version, name, current_key, current_name,
			sector_key, role_key, rarity_key, base_risk_key, passive_rule_key,
			passive_rule_config, upgrade_tree, basket_mapping_version_id
		) VALUES ($1, 'crest_sovereign', 1, 'Crest Sovereign', 'sovereign_current', 'Sovereign Current',
			'CREST', 'VANGUARD', 'COMMON', 'LOW_TO_MEDIUM', 'TEST_RULE', '{}'::jsonb, '{}'::jsonb, $2)`,
		keeperOneID, mappingOneID,
	)
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit valid catalog transaction: %v", err)
	}
}

func assertDeferredWeightFailure(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin invalid basket transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	mappingID := "00000000-0000-0000-0000-000000000012"
	mustExec(t, ctx, tx,
		"INSERT INTO basket_mapping_versions (id, mapping_key, content_version) VALUES ($1, 'invalid_total', 1)",
		mappingID,
	)
	mustExec(t, ctx, tx,
		"INSERT INTO basket_mapping_components (basket_mapping_version_id, market_asset_id, weight) VALUES ($1, $2, 0.40000000), ($1, $3, 0.50000000)",
		mappingID, assetOneID, assetTwoID,
	)
	if err := tx.Commit(ctx); err == nil {
		t.Fatal("commit invalid basket transaction succeeded")
	}
}

type dbExecutor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func mustExec(t *testing.T, ctx context.Context, executor dbExecutor, sql string, args ...any) {
	t.Helper()
	if _, err := executor.Exec(ctx, sql, args...); err != nil {
		t.Fatalf("execute SQL %q: %v", sql, err)
	}
}

func expectExecError(t *testing.T, ctx context.Context, conn *pgx.Conn, sql string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(ctx, sql, args...); err == nil {
		t.Fatalf("execute SQL %q unexpectedly succeeded", sql)
	}
}

func applyMigration(t *testing.T, ctx context.Context, conn *pgx.Conn, filename string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "db", "migrations", filename))
	if err != nil {
		t.Fatalf("read migration %s: %v", filename, err)
	}
	mustExec(t, ctx, conn, string(content))
}

func randomHex(t *testing.T) string {
	t.Helper()
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		t.Fatalf("generate random schema suffix: %v", err)
	}
	return hex.EncodeToString(bytes)
}
