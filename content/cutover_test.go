package content

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thaodangspace/tidekeepers-server/database/query"
)

func TestBuildCutoverPlanMapsDevLegacyTides(t *testing.T) {
	tideID := testUUID(t, "10000000-0000-0000-0000-000000000001")
	stateID := testUUID(t, "10000000-0000-0000-0000-000000000002")
	modifierID := testUUID(t, "20000000-0000-0000-0000-000000000001")
	objectiveID := testUUID(t, "20000000-0000-0000-0000-000000000002")
	rulesetID := testUUID(t, "20000000-0000-0000-0000-000000000003")

	plan, err := buildCutoverPlan(cutoverInput{
		sourceVersion:       1,
		targetVersion:       3,
		sourceCatalog:       SourceCatalogV1,
		expectedModifierID:  modifierID,
		expectedObjectiveID: objectiveID,
		expectedRulesetID:   rulesetID,
		strategyByKey:       map[string]pgtype.UUID{},
		tides: []query.ListLegacyCutoverTidesRow{{
			ID: tideID, ContentVersion: 1,
		}},
		projections: []query.ListCutoverProjectionsForTidesRow{{
			DailyTideID: tideID, StateID: stateID, SchemaVersion: 1,
			Projection: cutoverProjectionJSON("mod_default", "obj_default", nil, nil),
		}},
	})
	if err != nil {
		t.Fatalf("buildCutoverPlan() error = %v", err)
	}
	if plan.idempotent {
		t.Fatalf("plan must not be idempotent")
	}
	if plan.tideCount != 1 || plan.changedTideCount != 1 || plan.stateCount != 1 {
		t.Fatalf("plan counts = %+v", plan)
	}
	if len(plan.tideUpdates) != 1 {
		t.Fatalf("tideUpdates = %d, want 1", len(plan.tideUpdates))
	}
	if plan.tideUpdates[0].ContentVersion != 3 ||
		plan.tideUpdates[0].ModifierDefinitionVersionID != modifierID ||
		plan.tideUpdates[0].ObjectiveDefinitionVersionID != objectiveID ||
		plan.tideUpdates[0].GameRuleSetVersionID != rulesetID {
		t.Fatalf("tide update = %+v", plan.tideUpdates[0])
	}
	if len(plan.availability) != 0 || len(plan.selections) != 0 {
		t.Fatalf("availability/selections must be empty, got %+v / %+v", plan.availability, plan.selections)
	}
	if len(plan.fingerprintLabels) != 1 || plan.fingerprintLabels[0] != "migration-seeded-development" {
		t.Fatalf("fingerprint labels = %v", plan.fingerprintLabels)
	}
}

func TestBuildCutoverPlanMapsStrategyAvailabilityAndSelection(t *testing.T) {
	tideID := testUUID(t, "10000000-0000-0000-0000-000000000010")
	stateID := testUUID(t, "10000000-0000-0000-0000-000000000011")
	modifierID := testUUID(t, "20000000-0000-0000-0000-000000000010")
	objectiveID := testUUID(t, "20000000-0000-0000-0000-000000000011")
	rulesetID := testUUID(t, "20000000-0000-0000-0000-000000000012")
	strategyID := testUUID(t, "20000000-0000-0000-0000-000000000013")
	selected := "momentum"

	plan, err := buildCutoverPlan(cutoverInput{
		sourceVersion:       1,
		targetVersion:       3,
		sourceCatalog:       SourceCatalogV1,
		expectedModifierID:  modifierID,
		expectedObjectiveID: objectiveID,
		expectedRulesetID:   rulesetID,
		strategyByKey:       map[string]pgtype.UUID{"momentum": strategyID},
		tides: []query.ListLegacyCutoverTidesRow{{
			ID: tideID, ContentVersion: 1,
		}},
		projections: []query.ListCutoverProjectionsForTidesRow{{
			DailyTideID: tideID, StateID: stateID, SchemaVersion: 1,
			Projection: cutoverProjectionJSON("mod_default", "obj_default", []string{"momentum"}, &selected),
		}},
	})
	if err != nil {
		t.Fatalf("buildCutoverPlan() error = %v", err)
	}
	if len(plan.availability) != 1 {
		t.Fatalf("availability = %d, want 1", len(plan.availability))
	}
	if plan.availability[0].StrategyDefinitionVersionID != strategyID || plan.availability[0].ContentVersion != 3 {
		t.Fatalf("availability = %+v", plan.availability[0])
	}
	if len(plan.selections) != 1 {
		t.Fatalf("selections = %d, want 1", len(plan.selections))
	}
	if plan.selections[0].SelectedStrategyDefinitionVersionID != strategyID {
		t.Fatalf("selection = %+v", plan.selections[0])
	}
	if plan.strategyCount != 1 || plan.availabilityCount != 1 {
		t.Fatalf("plan counts = %+v", plan)
	}
}

func TestBuildCutoverPlanRejectsUnknownStrategy(t *testing.T) {
	tideID := testUUID(t, "10000000-0000-0000-0000-000000000020")
	stateID := testUUID(t, "10000000-0000-0000-0000-000000000021")
	selected := "ghost"

	_, err := buildCutoverPlan(cutoverInput{
		sourceVersion: 1,
		targetVersion: 3,
		sourceCatalog: SourceCatalogV1,
		strategyByKey: map[string]pgtype.UUID{},
		tides:         []query.ListLegacyCutoverTidesRow{{ID: tideID, ContentVersion: 1}},
		projections: []query.ListCutoverProjectionsForTidesRow{{
			DailyTideID: tideID, StateID: stateID, SchemaVersion: 1,
			Projection: cutoverProjectionJSON("mod_default", "obj_default", []string{"ghost"}, &selected),
		}},
	})
	if err == nil {
		t.Fatalf("unknown strategy must fail")
	}
	var cutoverErr *CutoverError
	if !asCutoverError(err, &cutoverErr) || cutoverErr.Category != "unknown-strategy" {
		t.Fatalf("error = %v", err)
	}
	if len(cutoverErr.StrategyIDs) != 1 || cutoverErr.StrategyIDs[0] != "ghost" {
		t.Fatalf("strategy ids = %v", cutoverErr.StrategyIDs)
	}
}

func TestBuildCutoverPlanRejectsContradictoryProjections(t *testing.T) {
	tideID := testUUID(t, "10000000-0000-0000-0000-000000000030")
	first := testUUID(t, "10000000-0000-0000-0000-000000000031")
	second := testUUID(t, "10000000-0000-0000-0000-000000000032")

	_, err := buildCutoverPlan(cutoverInput{
		sourceVersion: 1,
		targetVersion: 3,
		sourceCatalog: SourceCatalogV1,
		strategyByKey: map[string]pgtype.UUID{},
		tides:         []query.ListLegacyCutoverTidesRow{{ID: tideID, ContentVersion: 1}},
		projections: []query.ListCutoverProjectionsForTidesRow{
			{DailyTideID: tideID, StateID: first, SchemaVersion: 1, Projection: cutoverProjectionJSON("mod_default", "obj_default", nil, nil)},
			{DailyTideID: tideID, StateID: second, SchemaVersion: 1, Projection: cutoverProjectionJSON("mod_default", "other_objective", nil, nil)},
		},
	})
	if err == nil {
		t.Fatalf("contradictory projections must fail")
	}
	var cutoverErr *CutoverError
	if !asCutoverError(err, &cutoverErr) || cutoverErr.Category != "mixed-contradictory-projections" {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildCutoverPlanRejectsUnknownSourceFingerprint(t *testing.T) {
	tideID := testUUID(t, "10000000-0000-0000-0000-000000000040")
	stateID := testUUID(t, "10000000-0000-0000-0000-000000000041")

	_, err := buildCutoverPlan(cutoverInput{
		sourceVersion: 1,
		targetVersion: 3,
		sourceCatalog: SourceCatalogV1,
		strategyByKey: map[string]pgtype.UUID{},
		tides:         []query.ListLegacyCutoverTidesRow{{ID: tideID, ContentVersion: 1}},
		projections: []query.ListCutoverProjectionsForTidesRow{{
			DailyTideID: tideID, StateID: stateID, SchemaVersion: 1,
			Projection: cutoverProjectionJSON("mystery_mod", "obj_default", nil, nil),
		}},
	})
	if err == nil {
		t.Fatalf("unknown source fingerprint must fail")
	}
	var cutoverErr *CutoverError
	if !asCutoverError(err, &cutoverErr) || cutoverErr.Category != "unknown-source-fingerprint" {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildCutoverPlanRejectsPartiallyMappedState(t *testing.T) {
	legacyTide := testUUID(t, "10000000-0000-0000-0000-000000000050")
	stateID := testUUID(t, "10000000-0000-0000-0000-000000000051")
	partialTide := testUUID(t, "10000000-0000-0000-0000-000000000052")
	modifierID := testUUID(t, "20000000-0000-0000-0000-000000000050")

	_, err := buildCutoverPlan(cutoverInput{
		sourceVersion:      1,
		targetVersion:      3,
		sourceCatalog:      SourceCatalogV1,
		expectedModifierID: modifierID,
		strategyByKey:      map[string]pgtype.UUID{},
		tides: []query.ListLegacyCutoverTidesRow{
			{ID: legacyTide, ContentVersion: 1},
			{ID: partialTide, ContentVersion: 3, ModifierDefinitionVersionID: modifierID},
		},
		projections: []query.ListCutoverProjectionsForTidesRow{{
			DailyTideID: legacyTide, StateID: stateID, SchemaVersion: 1,
			Projection: cutoverProjectionJSON("mod_default", "obj_default", nil, nil),
		}},
	})
	if err == nil {
		t.Fatalf("partially mapped state must fail")
	}
	var cutoverErr *CutoverError
	if !asCutoverError(err, &cutoverErr) || cutoverErr.Category != "already-partially-mapped" {
		t.Fatalf("error = %v", err)
	}
}

func TestBuildCutoverPlanIsIdempotentWhenComplete(t *testing.T) {
	tideID := testUUID(t, "10000000-0000-0000-0000-000000000060")
	modifierID := testUUID(t, "20000000-0000-0000-0000-000000000060")
	objectiveID := testUUID(t, "20000000-0000-0000-0000-000000000061")
	rulesetID := testUUID(t, "20000000-0000-0000-0000-000000000062")

	plan, err := buildCutoverPlan(cutoverInput{
		sourceVersion:       1,
		targetVersion:       3,
		sourceCatalog:       SourceCatalogV1,
		expectedModifierID:  modifierID,
		expectedObjectiveID: objectiveID,
		expectedRulesetID:   rulesetID,
		strategyByKey:       map[string]pgtype.UUID{},
		tides: []query.ListLegacyCutoverTidesRow{{
			ID:                           tideID,
			ContentVersion:               3,
			ModifierDefinitionVersionID:  modifierID,
			ObjectiveDefinitionVersionID: objectiveID,
			GameRuleSetVersionID:         rulesetID,
		}},
		projections: nil,
	})
	if err != nil {
		t.Fatalf("buildCutoverPlan() error = %v", err)
	}
	if !plan.idempotent {
		t.Fatalf("plan must be idempotent")
	}
	if plan.tideCount != 0 || len(plan.tideUpdates) != 0 {
		t.Fatalf("idempotent plan must not change rows, got %+v", plan)
	}
}

func TestBuildCutoverPlanRejectsMissingProjections(t *testing.T) {
	tideID := testUUID(t, "10000000-0000-0000-0000-000000000070")

	_, err := buildCutoverPlan(cutoverInput{
		sourceVersion: 1,
		targetVersion: 3,
		sourceCatalog: SourceCatalogV1,
		strategyByKey: map[string]pgtype.UUID{},
		tides:         []query.ListLegacyCutoverTidesRow{{ID: tideID, ContentVersion: 1}},
		projections:   nil,
	})
	if err == nil {
		t.Fatalf("missing projections must fail")
	}
	var cutoverErr *CutoverError
	if !asCutoverError(err, &cutoverErr) || cutoverErr.Category != "missing-projections" {
		t.Fatalf("error = %v", err)
	}
}

func TestCutoverErrorRendersOffendingIdentifiers(t *testing.T) {
	err := &CutoverError{
		Category:    "unknown-strategy",
		Detail:      "projection references a strategy not present in the target release",
		TideIDs:     []string{"tide-b"},
		StrategyIDs: []string{"ghost", "alpha"},
	}
	message := err.Error()
	if !strings.Contains(message, "cutover unknown-strategy: projection references") {
		t.Fatalf("message = %q", message)
	}
	if !strings.Contains(message, "strategies: alpha, ghost") {
		t.Fatalf("message must sort strategy ids, got %q", message)
	}
}

func testUUID(t *testing.T, value string) pgtype.UUID {
	t.Helper()
	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", value, err)
	}
	var out pgtype.UUID
	copy(out.Bytes[:], id[:])
	out.Valid = true
	return out
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

func asCutoverError(err error, target **CutoverError) bool {
	var cutoverErr *CutoverError
	if !errors.As(err, &cutoverErr) {
		return false
	}
	*target = cutoverErr
	return true
}
