//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

func TestListKeeperInstancesForPlayerReturnsOnlyOwnedInstances(t *testing.T) {
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

	schema := "player_keepers_test_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, conn, "SET search_path TO "+schema)

	applyMigration(t, ctx, conn, "000001_foundation.up.sql")
	applyMigration(t, ctx, conn, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, conn, "000003_auth_credentials.up.sql")
	applyMigration(t, ctx, conn, "000004_player_keeper_instances.up.sql")
	seedValidDraft(t, ctx, conn)

	const (
		accountOneID = "00000000-0000-0000-0000-000000000041"
		accountTwoID = "00000000-0000-0000-0000-000000000042"
		playerOneID  = "00000000-0000-0000-0000-000000000051"
		playerTwoID  = "00000000-0000-0000-0000-000000000052"
	)
	mustExec(t, ctx, conn,
		"INSERT INTO accounts (id, email, password_hash) VALUES ($1, 'one@example.test', 'hash-one'), ($2, 'two@example.test', 'hash-two')",
		accountOneID, accountTwoID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO players (id, account_id, public_id) VALUES ($1, $2, 'plr_one'), ($3, $4, 'plr_two')",
		playerOneID, accountOneID, playerTwoID, accountTwoID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO keeper_instances (id, public_id, player_id, keeper_definition_version_id) VALUES ($1, 'kpr_owned', $2, $3), ($4, 'kpr_other', $5, $3)",
		"00000000-0000-0000-0000-000000000061", playerOneID, keeperOneID,
		"00000000-0000-0000-0000-000000000062", playerTwoID,
	)

	var databasePlayerOneID pgtype.UUID
	if err := databasePlayerOneID.Scan(playerOneID); err != nil {
		t.Fatalf("parse player ID: %v", err)
	}
	keepers, err := query.New(conn).ListKeeperInstancesForPlayer(ctx, databasePlayerOneID)
	if err != nil {
		t.Fatalf("list Keeper instances: %v", err)
	}
	if len(keepers) != 1 {
		t.Fatalf("Keeper count = %d, want 1", len(keepers))
	}
	if got := keepers[0]; got.KeeperPublicID != "kpr_owned" || got.DefinitionKey != "crest_sovereign" || got.Level != 1 {
		t.Errorf("Keeper = %+v, want owned instance with crest_sovereign definition", got)
	}

	expectExecError(t, ctx, conn,
		"INSERT INTO keeper_instances (id, public_id, player_id, keeper_definition_version_id) VALUES ($1, 'kpr_invalid', $2, $3)",
		"00000000-0000-0000-0000-000000000063", playerOneID, "00000000-0000-0000-0000-000000000999",
	)

	applyMigration(t, ctx, conn, "000004_player_keeper_instances.down.sql")
	var tableExists bool
	if err := conn.QueryRow(ctx, "SELECT to_regclass(current_schema() || '.keeper_instances') IS NOT NULL").Scan(&tableExists); err != nil {
		t.Fatalf("check Keeper migration rollback: %v", err)
	}
	if tableExists {
		t.Fatal("keeper_instances table remains after down migration")
	}
}
