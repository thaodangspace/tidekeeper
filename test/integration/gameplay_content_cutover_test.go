//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thaodangspace/tidekeepers-server/content"
	"github.com/thaodangspace/tidekeepers-server/database/query"
	"github.com/thaodangspace/tidekeepers-server/keeper"
)

var cutoverSeedCounter int64

func TestGameplayContentCutoverMapsMigrationSeededFixture(t *testing.T) {
	ctx := context.Background()
	pool := cutoverTestPool(t, ctx, "gameplay_cutover_dev")

	baseline := loadBaselineIDs(t, ctx, pool, content.BaselineV1Version)
	tideID, stateID := seedCutoverTide(t, ctx, pool, cutoverTide{
		contentVersion: 1,
		projection:     cutoverProjectionJSON("mod_default", "obj_default", nil, nil),
	})

	cutover := content.NewCutover(pool)

	report, err := cutover.Run(ctx, content.CutoverParams{SourceVersion: 1, TargetVersion: content.BaselineV1Version})
	if err != nil {
		t.Fatalf("dry-run cutover: %v", err)
	}
	if report.Applied || report.Idempotent || report.TideCount != 1 || report.StateCount != 1 || report.ChangedTideCount != 1 {
		t.Fatalf("dry-run report = %+v", report)
	}
	if len(report.FingerprintLabels) != 1 || report.FingerprintLabels[0] != "migration-seeded-development" {
		t.Fatalf("fingerprint labels = %v", report.FingerprintLabels)
	}
	assertTideState(t, ctx, pool, tideID, stateID, 1, nil, nil, nil)

	applied, err := cutover.Run(ctx, content.CutoverParams{
		SourceVersion: 1, TargetVersion: content.BaselineV1Version, Apply: true,
	})
	if err != nil {
		t.Fatalf("apply cutover: %v", err)
	}
	if !applied.Applied || applied.ChangedTideCount != 1 {
		t.Fatalf("applied report = %+v", applied)
	}
	assertTideState(t, ctx, pool, tideID, stateID, content.BaselineV1Version, &baseline.modifier, &baseline.objective, &baseline.ruleset)

	rerun, err := cutover.Run(ctx, content.CutoverParams{
		SourceVersion: 1, TargetVersion: content.BaselineV1Version, Apply: true,
	})
	if err != nil {
		t.Fatalf("idempotent cutover: %v", err)
	}
	if !rerun.Idempotent || rerun.ChangedTideCount != 0 || rerun.TideCount != 0 {
		t.Fatalf("idempotent report = %+v", rerun)
	}
}

func TestGameplayContentCutoverMapsKnownStrategySelection(t *testing.T) {
	ctx := context.Background()
	pool := cutoverTestPool(t, ctx, "gameplay_cutover_strategy")

	strategyID := publishStrategyTarget(t, ctx, pool, 5, "momentum")
	selected := "momentum"
	tideID, stateID := seedCutoverTide(t, ctx, pool, cutoverTide{
		contentVersion: 1,
		projection:     cutoverProjectionJSON("mod_default", "obj_default", []string{"momentum"}, &selected),
	})

	report, err := content.NewCutover(pool).Run(ctx, content.CutoverParams{
		SourceVersion: 1, TargetVersion: 5, Apply: true,
	})
	if err != nil {
		t.Fatalf("apply strategy cutover: %v", err)
	}
	if report.StrategyCount != 1 || report.AvailabilityCount != 1 || report.ChangedTideCount != 1 {
		t.Fatalf("strategy report = %+v", report)
	}

	var availabilityStrategyID pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT strategy_definition_version_id FROM daily_tide_strategy_versions WHERE daily_tide_id = $1 AND content_version = 5",
		tideID,
	).Scan(&availabilityStrategyID); err != nil {
		t.Fatalf("load availability row: %v", err)
	}
	if !availabilityStrategyID.Valid || availabilityStrategyID != strategyID {
		t.Fatalf("availability strategy = %v, want %v", availabilityStrategyID, strategyID)
	}

	var selectedStrategyID pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT selected_strategy_definition_version_id FROM player_daily_states WHERE id = $1",
		stateID,
	).Scan(&selectedStrategyID); err != nil {
		t.Fatalf("load selected strategy: %v", err)
	}
	if !selectedStrategyID.Valid || selectedStrategyID != strategyID {
		t.Fatalf("selected strategy = %v, want %v", selectedStrategyID, strategyID)
	}
}

func TestGameplayContentCutoverRejectsUnknownStrategy(t *testing.T) {
	ctx := context.Background()
	pool := cutoverTestPool(t, ctx, "gameplay_cutover_unknown_strategy")

	selected := "ghost"
	tideID, stateID := seedCutoverTide(t, ctx, pool, cutoverTide{
		contentVersion: 1,
		projection:     cutoverProjectionJSON("mod_default", "obj_default", []string{"ghost"}, &selected),
	})

	_, err := content.NewCutover(pool).Run(ctx, content.CutoverParams{
		SourceVersion: 1, TargetVersion: content.BaselineV1Version, Apply: true,
	})
	assertCutoverError(t, err, "unknown-strategy")
	assertTideState(t, ctx, pool, tideID, stateID, 1, nil, nil, nil)
}

func TestGameplayContentCutoverRejectsContradictoryProjections(t *testing.T) {
	ctx := context.Background()
	pool := cutoverTestPool(t, ctx, "gameplay_cutover_contradiction")

	tideID, firstState := seedCutoverTide(t, ctx, pool, cutoverTide{
		contentVersion: 1,
		projection:     cutoverProjectionJSON("mod_default", "obj_default", nil, nil),
	})
	secondState := seedCutoverState(t, ctx, pool, tideID, cutoverProjectionJSON("mod_default", "other_objective", nil, nil))

	_, err := content.NewCutover(pool).Run(ctx, content.CutoverParams{
		SourceVersion: 1, TargetVersion: content.BaselineV1Version, Apply: true,
	})
	assertCutoverError(t, err, "mixed-contradictory-projections")
	assertTideState(t, ctx, pool, tideID, firstState, 1, nil, nil, nil)
	assertTideState(t, ctx, pool, tideID, secondState, 1, nil, nil, nil)
}

func TestGameplayContentCutoverRejectsUnknownSourceFingerprint(t *testing.T) {
	ctx := context.Background()
	pool := cutoverTestPool(t, ctx, "gameplay_cutover_unknown_source")

	tideID, stateID := seedCutoverTide(t, ctx, pool, cutoverTide{
		contentVersion: 1,
		projection:     cutoverProjectionJSON("mystery_mod", "obj_default", nil, nil),
	})

	_, err := content.NewCutover(pool).Run(ctx, content.CutoverParams{
		SourceVersion: 1, TargetVersion: content.BaselineV1Version, Apply: true,
	})
	assertCutoverError(t, err, "unknown-source-fingerprint")
	assertTideState(t, ctx, pool, tideID, stateID, 1, nil, nil, nil)
}

func TestGameplayContentCutoverRejectsTargetMismatch(t *testing.T) {
	ctx := context.Background()
	pool := cutoverTestPool(t, ctx, "gameplay_cutover_mismatch")

	if _, err := content.NewPublisher(pool).Publish(ctx, *content.CompleteReleaseV2(content.BaselineV2Version)); err != nil {
		t.Fatalf("publish CatalogV2 baseline target: %v", err)
	}
	tideID, stateID := seedCutoverTide(t, ctx, pool, cutoverTide{
		contentVersion: 1,
		projection:     cutoverProjectionJSON("mod_default", "obj_default", nil, nil),
	})

	_, err := content.NewCutover(pool).Run(ctx, content.CutoverParams{
		SourceVersion: 1, TargetVersion: content.BaselineV2Version, Apply: true,
	})
	assertCutoverError(t, err, "target-mismatch")
	assertTideState(t, ctx, pool, tideID, stateID, 1, nil, nil, nil)
}

func TestGameplayContentCutoverRejectsPartiallyMappedTide(t *testing.T) {
	ctx := context.Background()
	pool := cutoverTestPool(t, ctx, "gameplay_cutover_partial")

	baseline := loadBaselineIDs(t, ctx, pool, content.BaselineV1Version)
	partialTideID, _ := seedCutoverTide(t, ctx, pool, cutoverTide{
		contentVersion: content.BaselineV1Version,
		projection:     cutoverProjectionJSON("mod_default", "obj_default", nil, nil),
	})
	mustExec(t, ctx, pool,
		"UPDATE daily_tides SET modifier_definition_version_id = $1 WHERE id = $2",
		baseline.modifier, partialTideID,
	)
	legacyTideID, legacyStateID := seedCutoverTide(t, ctx, pool, cutoverTide{
		contentVersion: 1,
		projection:     cutoverProjectionJSON("mod_default", "obj_default", nil, nil),
	})

	_, err := content.NewCutover(pool).Run(ctx, content.CutoverParams{
		SourceVersion: 1, TargetVersion: content.BaselineV1Version, Apply: true,
	})
	assertCutoverError(t, err, "already-partially-mapped")
	assertTideState(t, ctx, pool, partialTideID, "", content.BaselineV1Version, &baseline.modifier, nil, nil)
	assertTideState(t, ctx, pool, legacyTideID, legacyStateID, 1, nil, nil, nil)
}

type baselineIDs struct {
	modifier  pgtype.UUID
	objective pgtype.UUID
	ruleset   pgtype.UUID
}

func cutoverTestPool(t *testing.T, ctx context.Context, prefix string) *pgxpool.Pool {
	t.Helper()
	pool := gameplayContentTestPool(t, ctx, prefix)
	if _, err := keeper.NewPublisher(pool).Publish(ctx, keeper.CatalogV1()); err != nil {
		t.Fatalf("publish source CatalogV1 release 1: %v", err)
	}
	if _, err := content.NewPublisher(pool).Publish(ctx, *content.CompleteReleaseV1(content.BaselineV1Version)); err != nil {
		t.Fatalf("publish baseline target release 3: %v", err)
	}
	return pool
}

func loadBaselineIDs(t *testing.T, ctx context.Context, pool *pgxpool.Pool, version int64) baselineIDs {
	t.Helper()
	return baselineIDs{
		modifier:  loadDefinitionID(t, ctx, pool, "daily_modifier_definition_versions", "modifier_key", content.BaselineModifierKey, version),
		objective: loadDefinitionID(t, ctx, pool, "daily_objective_definition_versions", "objective_key", content.BaselineObjectiveKey, version),
		ruleset:   loadDefinitionID(t, ctx, pool, "game_rule_set_versions", "rule_set_key", content.BaselineRuleSetKey, version),
	}
}

func loadDefinitionID(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table, keyColumn, key string, version int64) pgtype.UUID {
	t.Helper()
	var id pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT id FROM "+table+" WHERE "+keyColumn+" = $1 AND content_version = $2",
		key, version,
	).Scan(&id); err != nil {
		t.Fatalf("load %s %q at version %d: %v", table, key, version, err)
	}
	if !id.Valid {
		t.Fatalf("load %s %q at version %d returned null id", table, key, version)
	}
	return id
}

func publishStrategyTarget(t *testing.T, ctx context.Context, pool *pgxpool.Pool, version int64, strategyKey string) pgtype.UUID {
	t.Helper()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin strategy target: %v", err)
	}
	defer tx.Rollback(ctx)
	queries := query.New(tx)

	checksum := bytes.Repeat([]byte{byte(version)}, 32)
	if err := queries.CreateDraftContentRelease(ctx, query.CreateDraftContentReleaseParams{
		Version: version, Checksum: checksum, ChecksumSchemaVersion: int16(content.ChecksumSchemaV2),
	}); err != nil {
		t.Fatalf("create draft strategy target: %v", err)
	}

	catalog := keeper.CatalogV1()
	catalog.Version = version
	if err := keeper.NewCatalogWriter(queries).Write(ctx, catalog); err != nil {
		t.Fatalf("write strategy target keeper catalog: %v", err)
	}

	modifierID := pgtypeUUID(t)
	objectiveID := pgtypeUUID(t)
	rulesetID := pgtypeUUID(t)
	strategyID := pgtypeUUID(t)
	if err := queries.InsertDailyModifierDefinitionVersion(ctx, query.InsertDailyModifierDefinitionVersionParams{
		ID: modifierID, ModifierKey: content.BaselineModifierKey, ContentVersion: version,
		Name: "Standard Conditions", Description: "Standard market conditions for this Daily Tide.",
		RuleKey: content.RuleStandardConditions, RuleConfig: []byte(`{"schemaVersion":1,"parameters":{}}`),
	}); err != nil {
		t.Fatalf("insert strategy target modifier: %v", err)
	}
	if err := queries.InsertDailyObjectiveDefinitionVersion(ctx, query.InsertDailyObjectiveDefinitionVersionParams{
		ID: objectiveID, ObjectiveKey: content.BaselineObjectiveKey, ContentVersion: version,
		Name: "Navigate", Description: "Hold a balanced position and outperform the sector benchmark.",
		RuleKey: content.RuleNavigate, RuleConfig: []byte(`{"schemaVersion":1,"parameters":{}}`),
	}); err != nil {
		t.Fatalf("insert strategy target objective: %v", err)
	}
	if err := queries.InsertGameRuleSetVersion(ctx, query.InsertGameRuleSetVersionParams{
		ID: rulesetID, RuleSetKey: content.BaselineRuleSetKey, ContentVersion: version,
		Name: "Standard Rules", Description: "The standard Tidekeepers game-rule set.",
		RuleKey: content.RuleStandardRules, RuleConfig: []byte(`{"schemaVersion":1,"parameters":{}}`),
	}); err != nil {
		t.Fatalf("insert strategy target ruleset: %v", err)
	}
	if err := queries.InsertStrategyDefinitionVersion(ctx, query.InsertStrategyDefinitionVersionParams{
		ID: strategyID, StrategyKey: strategyKey, ContentVersion: version,
		Name: "Momentum", Description: "Ride winning positions.",
		Upside: "Higher upside", Downside: "Higher downside",
		RuleKey: "STANDARD_STRATEGY", RuleConfig: []byte(`{"schemaVersion":1,"parameters":{}}`),
	}); err != nil {
		t.Fatalf("insert strategy target strategy: %v", err)
	}

	if err := queries.PublishContentRelease(ctx, version); err != nil {
		t.Fatalf("publish strategy target: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit strategy target: %v", err)
	}
	return strategyID
}

type cutoverTide struct {
	contentVersion int64
	projection     []byte
}

func seedCutoverTide(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tide cutoverTide) (string, string) {
	t.Helper()
	tideID := uuid.New().String()
	mustExec(t, ctx, pool,
		"INSERT INTO daily_tides (id, day_key, sequence_number, phase, lock_at, settle_after, content_version) VALUES ($1, $2, $3, 'REGULAR', $4, $5, $6)",
		tideID, nextCutoverDayKey(t), nextCutoverSequence(t), nextCutoverLock(t), nextCutoverSettle(t), tide.contentVersion,
	)
	stateID := seedCutoverState(t, ctx, pool, tideID, tide.projection)
	return tideID, stateID
}

func seedCutoverState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tideID string, projection []byte) string {
	t.Helper()
	accountID := uuid.New().String()
	playerID := uuid.New().String()
	voyageID := uuid.New().String()
	stateID := uuid.New().String()

	mustExec(t, ctx, pool,
		"INSERT INTO accounts (id) VALUES ($1)",
		accountID,
	)
	mustExec(t, ctx, pool,
		"INSERT INTO players (id, account_id, public_id) VALUES ($1, $2, $3)",
		playerID, accountID, "plr_"+stateID,
	)
	mustExec(t, ctx, pool,
		`INSERT INTO voyages (id, public_id, player_id, status, definition_version_key, current_day_number, fund_health, max_fund_health, capital, score, started_at) VALUES ($1, $2, $3, 'ACTIVE', 'standard', 1, 100, 100, 100, 0.0000, transaction_timestamp())`,
		voyageID, "voy_"+stateID, playerID,
	)
	mustExec(t, ctx, pool,
		"INSERT INTO player_daily_states (id, player_id, voyage_id, daily_tide_id, day_number, phase, version) VALUES ($1, $2, $3, $4, 1, 'PREPARATION', 1)",
		stateID, playerID, voyageID, tideID,
	)
	mustExec(t, ctx, pool,
		"INSERT INTO player_daily_context_views (player_daily_state_id, schema_version, projection) VALUES ($1, 1, $2)",
		stateID, projection,
	)
	return stateID
}

func assertTideState(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tideID, stateID string, contentVersion int64, modifier, objective, ruleset *pgtype.UUID) {
	t.Helper()
	var actualVersion int64
	var actualModifier, actualObjective, actualRuleset pgtype.UUID
	if err := pool.QueryRow(ctx,
		"SELECT content_version, modifier_definition_version_id, objective_definition_version_id, game_rule_set_version_id FROM daily_tides WHERE id = $1",
		tideID,
	).Scan(&actualVersion, &actualModifier, &actualObjective, &actualRuleset); err != nil {
		t.Fatalf("load tide %s: %v", tideID, err)
	}
	if actualVersion != contentVersion || !uuidEquals(actualModifier, modifier) || !uuidEquals(actualObjective, objective) || !uuidEquals(actualRuleset, ruleset) {
		t.Fatalf("tide %s state = version %d modifier %v objective %v ruleset %v, want version %d modifier %v objective %v ruleset %v",
			tideID, actualVersion, actualModifier, actualObjective, actualRuleset,
			contentVersion, modifier, objective, ruleset)
	}
	if stateID != "" {
		var selected pgtype.UUID
		if err := pool.QueryRow(ctx,
			"SELECT selected_strategy_definition_version_id FROM player_daily_states WHERE id = $1",
			stateID,
		).Scan(&selected); err != nil {
			t.Fatalf("load state %s: %v", stateID, err)
		}
		if selected.Valid {
			t.Fatalf("state %s selected strategy = %v, want null", stateID, selected)
		}
	}
}

func assertCutoverError(t *testing.T, err error, category string) {
	t.Helper()
	if err == nil {
		t.Fatalf("cutover unexpectedly succeeded, want category %q", category)
	}
	var cutoverErr *content.CutoverError
	if !errors.As(err, &cutoverErr) {
		t.Fatalf("cutover error = %v, want *content.CutoverError with category %q", err, category)
	}
	if cutoverErr.Category != category {
		t.Fatalf("cutover error category = %q, want %q (detail: %s)", cutoverErr.Category, category, cutoverErr.Detail)
	}
}

func uuidEquals(actual pgtype.UUID, expected *pgtype.UUID) bool {
	if expected == nil {
		return !actual.Valid
	}
	return actual.Valid && *expected == actual
}

func pgtypeUUID(t *testing.T) pgtype.UUID {
	t.Helper()
	id, err := uuid.NewUUID()
	if err != nil {
		t.Fatalf("generate uuid: %v", err)
	}
	var value pgtype.UUID
	copy(value.Bytes[:], id[:])
	value.Valid = true
	return value
}

func nextCutoverSequence(t *testing.T) int64 {
	t.Helper()
	_ = t
	return atomic.AddInt64(&cutoverSeedCounter, 1) + 1000
}

func nextCutoverDayKey(t *testing.T) string {
	t.Helper()
	sequence := atomic.AddInt64(&cutoverSeedCounter, 1)
	return time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, int(sequence)).Format("2006-01-02")
}

func nextCutoverLock(t *testing.T) time.Time {
	t.Helper()
	sequence := atomic.AddInt64(&cutoverSeedCounter, 1)
	return time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC).AddDate(0, 0, int(sequence))
}

func nextCutoverSettle(t *testing.T) time.Time {
	t.Helper()
	return nextCutoverLock(t).Add(10 * time.Minute)
}

func cutoverProjectionJSON(modifierID, objectiveID string, strategies []string, selected *string) []byte {
	strategyObjects := make([]map[string]any, 0, len(strategies))
	for _, strategyID := range strategies {
		strategyObjects = append(strategyObjects, map[string]any{
			"id": strategyID, "name": strategyID, "description": strategyID,
			"upside": "U", "downside": "D", "available": true,
		})
	}
	value := map[string]any{
		"modifier":  map[string]any{"id": modifierID, "name": modifierID, "description": modifierID},
		"objective": map[string]any{"id": objectiveID, "name": objectiveID, "description": objectiveID},
		"signals":   []any{},
		"lineup": map[string]any{
			"maxSlots":  1,
			"slots":     []any{map[string]any{"index": 0}},
			"synergies": []any{},
			"warnings":  []any{},
		},
		"inventory":          []any{},
		"shop":               map[string]any{"offers": []any{}, "rerollCost": 2, "rerollIndex": 0},
		"strategies":         strategyObjects,
		"selectedStrategyId": selected,
		"pendingRewardCount": 0,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return encoded
}
