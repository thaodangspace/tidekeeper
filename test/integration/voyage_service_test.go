//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/database/query"
	"github.com/thaodangspace/tidekeepers-server/voyage"
)

var voyagePlrOneID = "00000000-0000-0000-0000-0000000000b1"
var voyagePlrTwoID = "00000000-0000-0000-0000-0000000000b2"
var tideVoyageID = "00000000-0000-0000-0000-0000000000d1"

func TestVoyageCreateAndRejectSecond(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	conn, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_create")
	defer cleanup()

	svc := voyage.NewService(pool)

	resp, err := svc.Create(ctx, voyagePlrOneID, "key-create-only")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	assertActiveVoyage(t, resp, 100, 10)

	_, err = svc.Create(ctx, voyagePlrOneID, "key-create-second")
	if !errors.Is(err, voyage.ErrActiveVoyageExists) {
		t.Fatalf("second Create err = %v, want ErrActiveVoyageExists", err)
	}

	checkLedgerEntries(t, ctx, conn, resp.PublicID, 2)
}

func TestVoyageCreateGrantsRootNodeStarterInstance(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	conn, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_starter")
	defer cleanup()

	svc := voyage.NewService(pool)

	resp, err := svc.Create(ctx, voyagePlrOneID, "key-starter-create")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	voyageRow, err := query.New(conn).GetVoyageByPublicID(ctx, resp.PublicID)
	if err != nil {
		t.Fatalf("GetVoyageByPublicID: %v", err)
	}

	var (
		upgradeNodeKey string
		acquiredDay    int16
		acquiredSource string
		unlockCount    int
	)
	if err := conn.QueryRow(ctx, `
		SELECT i.upgrade_node_key, i.acquired_day, i.acquired_source
		FROM keeper_instances AS i
		WHERE i.voyage_id = $1
	`, voyageRow.ID).Scan(&upgradeNodeKey, &acquiredDay, &acquiredSource); err != nil {
		t.Fatalf("load starter instance: %v", err)
	}
	if upgradeNodeKey != "base" {
		t.Errorf("starter upgrade node = %q, want base", upgradeNodeKey)
	}
	if acquiredDay != 1 {
		t.Errorf("starter acquired day = %d, want 1", acquiredDay)
	}
	if acquiredSource != "STARTER" {
		t.Errorf("starter acquired source = %q, want STARTER", acquiredSource)
	}

	inventory, err := svc.GetActiveVoyageKeepers(ctx, voyagePlrOneID)
	if err != nil {
		t.Fatalf("GetActiveVoyageKeepers: %v", err)
	}
	if inventory.VoyageID != resp.PublicID {
		t.Errorf("inventory voyage ID = %q, want %q", inventory.VoyageID, resp.PublicID)
	}
	if len(inventory.Keepers) != 1 {
		t.Fatalf("inventory keeper count = %d, want 1", len(inventory.Keepers))
	}
	keeper := inventory.Keepers[0]
	if keeper.DefinitionKey != "crest_sovereign" {
		t.Errorf("keeper definition key = %q, want crest_sovereign", keeper.DefinitionKey)
	}
	if keeper.DefinitionVersion != 1 {
		t.Errorf("keeper definition version = %d, want 1", keeper.DefinitionVersion)
	}
	if keeper.UpgradeNodeKey != "base" {
		t.Errorf("keeper upgrade node = %q, want base", keeper.UpgradeNodeKey)
	}
	if keeper.Level != 1 {
		t.Errorf("keeper level = %d, want 1", keeper.Level)
	}
	if keeper.AcquiredDay != 1 || keeper.AcquiredSource != "STARTER" {
		t.Errorf("keeper acquisition = day %d source %q, want 1/STARTER", keeper.AcquiredDay, keeper.AcquiredSource)
	}

	if err := conn.QueryRow(ctx, "SELECT count(*) FROM player_keeper_unlocks WHERE player_id = $1", voyagePlrOneID).Scan(&unlockCount); err != nil {
		t.Fatalf("count unlocks: %v", err)
	}
	if unlockCount != 0 {
		t.Errorf("unlock count = %d, want 0 (starter instances are not permanent unlocks)", unlockCount)
	}
}

func TestVoyageCreateIdempotent(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_idem_create")
	defer cleanup()

	svc := voyage.NewService(pool)

	resp1, err := svc.Create(ctx, voyagePlrOneID, "key-idem-create")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	resp2, err := svc.Create(ctx, voyagePlrOneID, "key-idem-create")
	if err != nil {
		t.Fatalf("replay Create: %v", err)
	}
	if resp1.PublicID != resp2.PublicID {
		t.Errorf("replayed ID = %q, want %q", resp2.PublicID, resp1.PublicID)
	}
	if resp2.Status != "ACTIVE" {
		t.Errorf("replayed status = %q, want ACTIVE", resp2.Status)
	}

	resp3, err := svc.Create(ctx, voyagePlrOneID, "key-idem-create")
	if err != nil {
		t.Fatalf("third Create (same key, same request): %v", err)
	}
	if resp3.PublicID != resp1.PublicID {
		t.Errorf("third replay ID = %q, want %q", resp3.PublicID, resp1.PublicID)
	}

	_, err = svc.Create(ctx, voyagePlrOneID, "different-key")
	if !errors.Is(err, voyage.ErrActiveVoyageExists) {
		t.Fatalf("Create with different key err = %v, want ErrActiveVoyageExists", err)
	}
}

func TestVoyageAbandonAndRecreate(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_abandon")
	defer cleanup()

	svc := voyage.NewService(pool)

	created, err := svc.Create(ctx, voyagePlrOneID, "key-abandon-cycle")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	abandoned, err := svc.Abandon(ctx, voyagePlrOneID, created.PublicID, "key-abandon")
	if err != nil {
		t.Fatalf("Abandon: %v", err)
	}
	if abandoned.Status != "ABANDONED" {
		t.Errorf("status = %q, want ABANDONED", abandoned.Status)
	}
	if abandoned.CompletedAt == nil {
		t.Error("CompletedAt should be set after abandon")
	}
	if abandoned.RowVersion != created.RowVersion+1 {
		t.Errorf("rowVersion = %d, want %d", abandoned.RowVersion, created.RowVersion+1)
	}

	byID, err := svc.GetByID(ctx, voyagePlrOneID, created.PublicID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Status != "ABANDONED" {
		t.Errorf("GetByID status = %q, want ABANDONED", byID.Status)
	}

	_, err = svc.GetCurrent(ctx, voyagePlrOneID)
	if !errors.Is(err, voyage.ErrNoActiveVoyage) {
		t.Fatalf("GetCurrent after abandon err = %v, want ErrNoActiveVoyage", err)
	}

	recreated, err := svc.Create(ctx, voyagePlrOneID, "key-abandon-recreate")
	if err != nil {
		t.Fatalf("Create after abandon: %v", err)
	}
	if recreated.Status != "ACTIVE" {
		t.Errorf("recreated status = %q, want ACTIVE", recreated.Status)
	}
	if recreated.PublicID == created.PublicID {
		t.Error("recreated voyage should have a different ID")
	}

	current, err := svc.GetCurrent(ctx, voyagePlrOneID)
	if err != nil {
		t.Fatalf("GetCurrent after recreate: %v", err)
	}
	if current.PublicID != recreated.PublicID {
		t.Errorf("current voyage = %q, want newly created %q", current.PublicID, recreated.PublicID)
	}
}

func TestVoyageAbandonIdempotent(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_idem_abandon")
	defer cleanup()

	svc := voyage.NewService(pool)

	created, err := svc.Create(ctx, voyagePlrOneID, "key-abandon-idem-create")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	abandon1, err := svc.Abandon(ctx, voyagePlrOneID, created.PublicID, "key-abandon-idem")
	if err != nil {
		t.Fatalf("first Abandon: %v", err)
	}
	abandon2, err := svc.Abandon(ctx, voyagePlrOneID, created.PublicID, "key-abandon-idem")
	if err != nil {
		t.Fatalf("replay Abandon: %v", err)
	}
	if abandon1.PublicID != abandon2.PublicID {
		t.Errorf("replayed voyage ID = %q, want %q", abandon2.PublicID, abandon1.PublicID)
	}
	if abandon2.Status != "ABANDONED" {
		t.Errorf("replayed status = %q, want ABANDONED", abandon2.Status)
	}
}

func TestVoyageOwnershipIsolation(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_owner")
	defer cleanup()

	svc := voyage.NewService(pool)

	_, err := svc.Create(ctx, voyagePlrOneID, "key-owner-plr1")
	if err != nil {
		t.Fatalf("player one Create: %v", err)
	}

	_, err = svc.Create(ctx, voyagePlrTwoID, "key-owner-plr2")
	if err != nil {
		t.Fatalf("player two Create: %v", err)
	}

	plr2Resp, err := svc.GetCurrent(ctx, voyagePlrTwoID)
	if err != nil {
		t.Fatalf("player two GetCurrent: %v", err)
	}

	_, err = svc.GetByID(ctx, voyagePlrOneID, plr2Resp.PublicID)
	if !errors.Is(err, voyage.ErrVoyageNotFound) {
		t.Fatalf("player one GetByID for plr2 voyage err = %v, want ErrVoyageNotFound", err)
	}

	_, err = svc.Abandon(ctx, voyagePlrOneID, plr2Resp.PublicID, "key-owner-cross")
	if !errors.Is(err, voyage.ErrVoyageNotFound) {
		t.Fatalf("player one Abandon plr2 voyage err = %v, want ErrVoyageNotFound", err)
	}

	_, err = svc.GetHistory(ctx, voyagePlrOneID, plr2Resp.PublicID)
	if !errors.Is(err, voyage.ErrVoyageNotFound) {
		t.Fatalf("player one GetHistory plr2 voyage err = %v, want ErrVoyageNotFound", err)
	}
}

func TestVoyageHistory(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_history")
	defer cleanup()

	svc := voyage.NewService(pool)

	created, err := svc.Create(ctx, voyagePlrOneID, "key-history-create")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	history, err := svc.GetHistory(ctx, voyagePlrOneID, created.PublicID)
	if err != nil {
		t.Fatalf("GetHistory after create: %v", err)
	}
	if len(history.Events) != 1 {
		t.Fatalf("event count = %d, want 1", len(history.Events))
	}
	if history.Events[0].EventType != "CREATED" {
		t.Errorf("event type = %q, want CREATED", history.Events[0].EventType)
	}
	if history.Events[0].ResultingStatus != "ACTIVE" {
		t.Errorf("resultingStatus = %q, want ACTIVE", history.Events[0].ResultingStatus)
	}

	_, err = svc.Abandon(ctx, voyagePlrOneID, created.PublicID, "key-history-abandon")
	if err != nil {
		t.Fatalf("Abandon: %v", err)
	}

	history, err = svc.GetHistory(ctx, voyagePlrOneID, created.PublicID)
	if err != nil {
		t.Fatalf("GetHistory after abandon: %v", err)
	}
	if len(history.Events) != 2 {
		t.Fatalf("event count = %d, want 2", len(history.Events))
	}
	if history.Events[1].EventType != "ABANDONED" {
		t.Errorf("event type = %q, want ABANDONED", history.Events[1].EventType)
	}
	if history.Events[1].ResultingStatus != "ABANDONED" {
		t.Errorf("resultingStatus = %q, want ABANDONED", history.Events[1].ResultingStatus)
	}

	if history.Voyage.PublicID != created.PublicID {
		t.Errorf("history voyage ID = %q, want %q", history.Voyage.PublicID, created.PublicID)
	}
	if history.Voyage.Status != "ABANDONED" {
		t.Errorf("history voyage status = %q, want ABANDONED", history.Voyage.Status)
	}
}

func TestVoyageAbandonIdempotencyKeyScopedToVoyageID(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_key_scope")
	defer cleanup()

	svc := voyage.NewService(pool)

	v1, err := svc.Create(ctx, voyagePlrOneID, "key-scope-v1")
	if err != nil {
		t.Fatalf("Create v1: %v", err)
	}

	_, err = svc.Abandon(ctx, voyagePlrOneID, v1.PublicID, "key-scope-abandon")
	if err != nil {
		t.Fatalf("Abandon v1: %v", err)
	}

	v2, err := svc.Create(ctx, voyagePlrOneID, "key-scope-v2")
	if err != nil {
		t.Fatalf("Create v2: %v", err)
	}

	_, err = svc.Abandon(ctx, voyagePlrOneID, v2.PublicID, "key-scope-abandon")
	if !errors.Is(err, voyage.ErrIdempotencyKeyReused) {
		t.Fatalf("abandon v2 with same key err = %v, want ErrIdempotencyKeyReused", err)
	}
}

func TestVoyageConcurrentCreateOnlyOneSucceeds(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	_, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_concurrent")
	defer cleanup()

	svc := voyage.NewService(pool)

	const goroutines = 5
	var (
		mu     sync.Mutex
		wg     sync.WaitGroup
		winner string
		count  int
	)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		key := "key-concurrent-" + string(rune('a'+i))
		go func(k string) {
			defer wg.Done()
			resp, err := svc.Create(ctx, voyagePlrOneID, k)
			mu.Lock()
			if err == nil {
				count++
				winner = resp.PublicID
			}
			mu.Unlock()
		}(key)
	}
	wg.Wait()

	if count != 1 {
		t.Fatalf("exactly 1 creation should succeed, got %d", count)
	}

	current, err := svc.GetCurrent(ctx, voyagePlrOneID)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if current.PublicID != winner {
		t.Errorf("current voyage = %q, want winning voyage %q", current.PublicID, winner)
	}
}

func TestVoyageAtomicRollbackOnFailure(t *testing.T) {
	ctx := context.Background()
	databaseURL := databaseURL(t)

	conn, pool, cleanup := setupVoyageTest(t, ctx, databaseURL, "voyage_rollback")
	defer cleanup()

	svc := voyage.NewService(pool)

	resp, err := svc.Create(ctx, voyagePlrOneID, "key-rollback-ok")
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	firstVoyageRow, err := query.New(conn).GetVoyageByPublicID(ctx, resp.PublicID)
	if err != nil {
		t.Fatalf("GetVoyageByPublicID: %v", err)
	}
	firstVoyageID := firstVoyageRow.ID

	_, err = svc.Create(ctx, voyagePlrOneID, "key-rollback-fail")
	if !errors.Is(err, voyage.ErrActiveVoyageExists) {
		t.Fatalf("second Create err = %v, want ErrActiveVoyageExists", err)
	}

	var counts struct {
		voyages       int
		ledgerEntries int
		keeperInst    int
		lifecycleEvts int
		states        int
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM voyages WHERE player_id = $1", voyagePlrOneID).Scan(&counts.voyages); err != nil {
		t.Fatalf("count voyages: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM voyage_ledger_entries WHERE voyage_id = $1", firstVoyageID).Scan(&counts.ledgerEntries); err != nil {
		t.Fatalf("count ledger entries: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM keeper_instances AS i JOIN voyages AS v ON v.id = i.voyage_id WHERE v.player_id = $1", voyagePlrOneID).Scan(&counts.keeperInst); err != nil {
		t.Fatalf("count keeper instances: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM voyage_lifecycle_events WHERE voyage_id = $1", firstVoyageID).Scan(&counts.lifecycleEvts); err != nil {
		t.Fatalf("count lifecycle events: %v", err)
	}
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM player_daily_states WHERE player_id = $1", voyagePlrOneID).Scan(&counts.states); err != nil {
		t.Fatalf("count daily states: %v", err)
	}

	t.Logf("rollback check: voyages=%d ledger=%d keepers=%d events=%d states=%d",
		counts.voyages, counts.ledgerEntries, counts.keeperInst, counts.lifecycleEvts, counts.states)

	if counts.voyages != 1 {
		t.Errorf("voyages count = %d, want 1", counts.voyages)
	}
	if counts.ledgerEntries != 2 {
		t.Errorf("ledger entries count = %d, want 2", counts.ledgerEntries)
	}
	if counts.keeperInst != 1 {
		t.Errorf("keeper instances count = %d, want 1", counts.keeperInst)
	}
	if counts.lifecycleEvts != 1 {
		t.Errorf("lifecycle events count = %d, want 1", counts.lifecycleEvts)
	}
	if counts.states != 1 {
		t.Errorf("daily states count = %d, want 1", counts.states)
	}

	rows, err := query.New(conn).ListLifecycleEventsForVoyage(ctx, firstVoyageID)
	if err != nil {
		t.Fatalf("list lifecycle events: %v", err)
	}
	if len(rows) != 1 || rows[0].EventType != "CREATED" {
		t.Errorf("lifecycle events = %+v, want [CREATED]", rows)
	}
}

func databaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Fatal("DATABASE_URL is required for integration tests")
	}
	return url
}

func mustParseUUID(t *testing.T, dst *pgtype.UUID, src string) {
	t.Helper()
	if err := dst.Scan(src); err != nil {
		t.Fatalf("parse UUID %q: %v", src, err)
	}
}

func setupVoyageTest(t *testing.T, ctx context.Context, databaseURL, schemaPrefix string) (*pgx.Conn, *pgxpool.Pool, func()) {
	t.Helper()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect PostgreSQL: %v", err)
	}

	schema := schemaPrefix + "_" + randomHex(t)
	mustExec(t, ctx, conn, "CREATE SCHEMA "+schema)
	mustExec(t, ctx, conn, "SET search_path TO "+schema)

	applyMigration(t, ctx, conn, "000001_foundation.up.sql")
	applyMigration(t, ctx, conn, "000002_keepers_catalog.up.sql")
	applyMigration(t, ctx, conn, "000003_auth_credentials.up.sql")
	applyMigration(t, ctx, conn, "000004_player_keeper_instances.up.sql")
	applyMigration(t, ctx, conn, "000006_voyage_lifecycle.up.sql")
	applyMigration(t, ctx, conn, "000007_keeper_definition_upgrade_nodes.up.sql")
	applyMigration(t, ctx, conn, "000008_meta_and_voyage_keeper_inventory.up.sql")

	mustExec(t, ctx, conn,
		"INSERT INTO accounts (id, email, password_hash) VALUES ($1, 'a@voy.test', 'hash-a'), ($2, 'b@voy.test', 'hash-b')",
		voyagePlrOneID, voyagePlrTwoID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO players (id, account_id, public_id) VALUES ($1, $2, 'plr_voy_one'), ($3, $4, 'plr_voy_two')",
		voyagePlrOneID, voyagePlrOneID, voyagePlrTwoID, voyagePlrTwoID,
	)
	mustExec(t, ctx, conn,
		"INSERT INTO daily_tides (id, day_key, sequence_number, phase, lock_at, settle_after, content_version) VALUES ($1, current_date, 1, 'REGULAR', current_date + interval '1 day', current_date + interval '1 day' + interval '10 minutes', 1)",
		tideVoyageID,
	)

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse pool config: %v", err)
	}
	poolConfig.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}

	cleanup := func() {
		pool.Close()
		_, _ = conn.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		conn.Close(ctx)
	}

	return conn, pool, cleanup
}

func assertActiveVoyage(t *testing.T, resp *voyage.VoyageResponse, wantFundHealth, wantCapital int) {
	t.Helper()
	if resp.Status != "ACTIVE" {
		t.Errorf("status = %q, want ACTIVE", resp.Status)
	}
	if resp.DayNumber != 1 {
		t.Errorf("dayNumber = %d, want 1", resp.DayNumber)
	}
	if resp.FundHealth != wantFundHealth || resp.MaxFundHealth != wantFundHealth {
		t.Errorf("fundHealth = %d/%d, want %d/%d", resp.FundHealth, resp.MaxFundHealth, wantFundHealth, wantFundHealth)
	}
	if resp.Capital != wantCapital {
		t.Errorf("capital = %d, want %d", resp.Capital, wantCapital)
	}
	if resp.DefinitionKey != "standard" {
		t.Errorf("definitionKey = %q, want standard", resp.DefinitionKey)
	}
	if resp.Score != "0.0000" {
		t.Errorf("score = %q, want 0.0000", resp.Score)
	}
	if resp.RowVersion != 1 {
		t.Errorf("rowVersion = %d, want 1", resp.RowVersion)
	}
	if resp.CompletedAt != nil {
		t.Error("CompletedAt should be nil for active voyage")
	}
}

func checkLedgerEntries(t *testing.T, ctx context.Context, conn *pgx.Conn, voyagePublicID string, wantCount int) {
	t.Helper()
	var pgVoyageID pgtype.UUID
	voyageRow, err := query.New(conn).GetVoyageByPublicID(ctx, voyagePublicID)
	if err != nil {
		t.Fatalf("GetVoyageByPublicID: %v", err)
	}
	pgVoyageID = voyageRow.ID

	var count int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM voyage_ledger_entries WHERE voyage_id = $1", pgVoyageID).Scan(&count); err != nil {
		t.Fatalf("query ledger entries: %v", err)
	}
	if count != wantCount {
		t.Errorf("ledger entry count = %d, want %d", count, wantCount)
	}
}
