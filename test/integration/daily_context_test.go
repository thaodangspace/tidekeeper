//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/daily"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

const (
	accID      = "00000000-0000-0000-0000-0000000000a1"
	accTwoID   = "00000000-0000-0000-0000-0000000000a2"
	plrID      = "00000000-0000-0000-0000-0000000000b1"
	plrTwoID   = "00000000-0000-0000-0000-0000000000b2"
	voyID      = "00000000-0000-0000-0000-0000000000c1"
	voyTwoID   = "00000000-0000-0000-0000-0000000000c2"
	tideID     = "00000000-0000-0000-0000-0000000000d1"
	stateID    = "00000000-0000-0000-0000-0000000000e1"
	stateTwoID = "00000000-0000-0000-0000-0000000000e2"
)

func TestGetCurrentDailyContextSuccess(t *testing.T) {
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

	schema := "daily_context_success_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, conn, "SET search_path TO "+schema)

	applyMigration(t, ctx, conn, "000001_foundation.up.sql")

	mustExec(t, ctx, conn,
		"INSERT INTO accounts (id, email, password_hash) VALUES ($1, 'a@a.test', 'hash-a')",
		accID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO players (id, account_id, public_id) VALUES ($1, $2, 'plr_one')",
		plrID, accID,
	)
	mustExec(t, ctx, conn,
		`INSERT INTO voyages (id, public_id, player_id, status, definition_version_key, current_day_number, fund_health, max_fund_health, capital, score, started_at) VALUES ($1, 'voy_one', $2, 'ACTIVE', 'standard', 3, 82, 100, 11, 245.0000, transaction_timestamp())`,
		voyID, plrID,
	)
	mustExec(t, ctx, conn,
		"UPDATE players SET current_voyage_id = $1 WHERE id = $2",
		voyID, plrID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO daily_tides (id, day_key, sequence_number, phase, lock_at, settle_after, content_version) VALUES ($1, '2026-07-30', 42, 'REGULAR', transaction_timestamp(), transaction_timestamp() + interval '10 minutes', 1)",
		tideID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO player_daily_states (id, player_id, voyage_id, daily_tide_id, day_number, phase, version) VALUES ($1, $2, $3, $4, 3, 'PREPARATION', 9)",
		stateID, plrID, voyID, tideID,
	)

	projection := validProjection(t)
	projJSON, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}

	mustExec(t, ctx, conn,
		"INSERT INTO player_daily_context_views (player_daily_state_id, schema_version, projection) VALUES ($1, 1, $2)",
		stateID, projJSON,
	)

	var databasePlayerID pgtype.UUID
	scanErr := databasePlayerID.Scan(plrID)
	if scanErr != nil {
		t.Fatalf("parse player ID: %v", scanErr)
	}

	row, err := query.New(conn).GetCurrentDailyContextForPlayer(ctx, databasePlayerID)
	if err != nil {
		t.Fatalf("query daily context: %v", err)
	}

	if !row.CurrentVoyageID.Valid {
		t.Fatal("current_voyage_id is not valid")
	}
	if row.VoyagePublicID == nil || *row.VoyagePublicID != "voy_one" {
		t.Fatalf("voyage_public_id = %v, want voy_one", row.VoyagePublicID)
	}
	if row.VoyageStatus == nil || *row.VoyageStatus != "ACTIVE" {
		t.Fatalf("voyage_status = %v, want ACTIVE", row.VoyageStatus)
	}
	if row.CurrentDayNumber == nil || *row.CurrentDayNumber != 3 {
		t.Fatalf("current_day_number = %v, want 3", row.CurrentDayNumber)
	}
	if row.PlayerPhase == nil || *row.PlayerPhase != "PREPARATION" {
		t.Fatalf("player_phase = %v, want PREPARATION", row.PlayerPhase)
	}
	if row.PlayerStateVersion == nil || *row.PlayerStateVersion != 9 {
		t.Fatalf("player_state_version = %v, want 9", row.PlayerStateVersion)
	}
	if row.ProjectionSchemaVersion == nil || *row.ProjectionSchemaVersion != 1 {
		t.Fatalf("projection_schema_version = %v, want 1", row.ProjectionSchemaVersion)
	}
	if !row.ServerNow.Valid {
		t.Fatal("server_now is not valid")
	}
}

func TestGetCurrentDailyContextNoActiveVoyage(t *testing.T) {
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

	schema := "daily_context_no_voyage_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, conn, "SET search_path TO "+schema)

	applyMigration(t, ctx, conn, "000001_foundation.up.sql")

	mustExec(t, ctx, conn,
		"INSERT INTO accounts (id, email, password_hash) VALUES ($1, 'c@c.test', 'hash-c')",
		accID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO players (id, account_id, public_id) VALUES ($1, $2, 'plr_no_voyage')",
		plrID, accID,
	)

	repo := daily.NewRepository(createPool(t, schema))
	_, err = repo.GetCurrentDailyContext(ctx, daily.ID(plrID))
	if !errors.Is(err, daily.ErrNoActiveVoyage) {
		t.Fatalf("err = %v, want ErrNoActiveVoyage", err)
	}
}

func TestGetCurrentDailyContextThroughRepository(t *testing.T) {
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

	schema := "daily_context_repo_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	t.Cleanup(func() { _, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE") })
	mustExec(t, ctx, conn, "SET search_path TO "+schema)

	applyMigration(t, ctx, conn, "000001_foundation.up.sql")

	mustExec(t, ctx, conn,
		"INSERT INTO accounts (id, email, password_hash) VALUES ($1, 'f@f.test', 'hash-f')",
		accID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO players (id, account_id, public_id) VALUES ($1, $2, 'plr_repo')",
		plrID, accID,
	)
	mustExec(t, ctx, conn,
		`INSERT INTO voyages (id, public_id, player_id, status, definition_version_key, current_day_number, fund_health, max_fund_health, capital, score, started_at, completed_at) VALUES ($1, 'voy_repo', $2, 'COMPLETED', 'standard', 5, 60, 100, 8, '320.0000', transaction_timestamp(), transaction_timestamp())`,
		voyID, plrID,
	)
	mustExec(t, ctx, conn,
		"UPDATE players SET current_voyage_id = $1 WHERE id = $2",
		voyID, plrID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO daily_tides (id, day_key, sequence_number, phase, lock_at, settle_after, content_version) VALUES ($1, '2026-07-30', 99, 'REGULAR', transaction_timestamp(), transaction_timestamp() + interval '10 minutes', 1)",
		tideID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO player_daily_states (id, player_id, voyage_id, daily_tide_id, day_number, phase, version, selected_strategy_id, pending_reward_count) VALUES ($1, $2, $3, $4, 5, 'RESULT_READY', 10, 'strategy_steady', 2)",
		stateID, plrID, voyID, tideID,
	)

	projection := validProjection(t)
	projJSON, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}
	mustExec(t, ctx, conn,
		"INSERT INTO player_daily_context_views (player_daily_state_id, schema_version, projection) VALUES ($1, 1, $2)",
		stateID, projJSON,
	)

	repo := daily.NewRepository(createPool(t, schema))
	ctxResult, err := repo.GetCurrentDailyContext(ctx, daily.ID(plrID))
	if err != nil {
		t.Fatalf("GetCurrentDailyContext: %v", err)
	}

	if ctxResult.Voyage.PublicID != "voy_repo" {
		t.Errorf("voyage public_id = %q, want voy_repo", ctxResult.Voyage.PublicID)
	}
	if ctxResult.Voyage.Status != "COMPLETED" {
		t.Errorf("voyage status = %q, want COMPLETED", ctxResult.Voyage.Status)
	}
	if ctxResult.Voyage.Score != "320.0000" {
		t.Errorf("voyage score = %q, want 320.0000", ctxResult.Voyage.Score)
	}
	if ctxResult.Daily.Phase != "RESULT_READY" {
		t.Errorf("daily phase = %q, want RESULT_READY", ctxResult.Daily.Phase)
	}
	if ctxResult.Daily.Version != 10 {
		t.Errorf("daily version = %d, want 10", ctxResult.Daily.Version)
	}
	if ctxResult.ServerNow.IsZero() {
		t.Error("serverNow is zero")
	}
}

func createPool(t *testing.T, schema string) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func validProjection(t *testing.T) map[string]any {
	t.Helper()
	return map[string]any{
		"modifier": map[string]any{
			"id": "mod_001", "name": "High Tide", "description": "All gains 1.5x",
		},
		"objective": map[string]any{
			"id": "obj_001", "name": "Growth", "description": "Grow portfolio",
			"progressLabel": "3/5%", "rewardLabel": "+1 Health",
		},
		"signals": []map[string]any{
			{
				"id": "sig_001", "name": "BTC Up", "description": "Bitcoin rising",
				"direction": "UP", "strength": "STRONG",
				"observedFrom": "2026-07-30T00:00:00Z", "observedTo": "2026-07-30T23:59:59Z",
			},
		},
		"lineup": map[string]any{
			"lockedAt": nil, "maxSlots": 3,
			"slots": []map[string]any{
				{"index": 0, "keeper": nil},
				{"index": 1, "keeper": map[string]any{
					"id": "kpr_001", "definitionId": "def_btc", "name": "Trader",
					"level": 5, "rarity": "RARE", "role": "TRADER", "sector": "CRYPTO",
					"passiveSummary": "Test", "artworkUrl": nil,
				}},
				{"index": 2, "keeper": nil},
			},
			"synergies": []map[string]any{},
			"warnings":  []map[string]any{},
		},
		"inventory": []map[string]any{
			{
				"id": "kpr_002", "definitionId": "def_eth", "name": "Holder",
				"level": 3, "rarity": "COMMON", "role": "HOLDER", "sector": "CRYPTO",
				"passiveSummary": "HODL", "artworkUrl": nil,
			},
		},
		"shop": map[string]any{
			"offers": []map[string]any{
				{
					"id": "off_001", "cost": 5, "available": true,
					"keeper": map[string]any{
						"id": "kpr_003", "definitionId": "def_sol", "name": "Sol Trader",
						"level": 2, "rarity": "UNCOMMON", "role": "TRADER", "sector": "CRYPTO",
						"passiveSummary": "Fast", "artworkUrl": nil,
					},
					"synergyHint": nil,
				},
			},
			"refreshAt": nil, "rerollCost": 3, "rerollIndex": 0,
		},
		"strategies": []map[string]any{
			{
				"id": "strategy_steady", "name": "Steady", "description": "Safe",
				"upside": "Low", "downside": "Minimal", "available": true,
			},
		},
		"selectedStrategyId": "strategy_steady",
		"pendingRewardCount": 0,
	}
}
