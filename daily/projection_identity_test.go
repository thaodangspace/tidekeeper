package daily

import (
	"encoding/json"
	"testing"
)

func TestDecodeProjectionIdentityDefaultProjection(t *testing.T) {
	identity, err := DecodeProjectionIdentity(1, defaultProjection(t))
	if err != nil {
		t.Fatalf("DecodeProjectionIdentity() error = %v", err)
	}
	if identity.ModifierID != "mod_default" || identity.ObjectiveID != "obj_default" {
		t.Fatalf("identity = %+v", identity)
	}
	if len(identity.StrategyIDs) != 0 {
		t.Fatalf("identity.StrategyIDs = %v, want empty", identity.StrategyIDs)
	}
	if identity.SelectedStrategyID != nil {
		t.Fatalf("identity.SelectedStrategyID = %v, want nil", *identity.SelectedStrategyID)
	}
}

func TestDecodeProjectionIdentityWithStrategySelection(t *testing.T) {
	projection := defaultProjection(t)
	projection = mutateProjection(t, projection, map[string]any{
		"strategies": []map[string]any{{
			"id": "momentum", "name": "Momentum", "description": "Ride the trend",
			"upside": "High", "downside": "Low", "available": true,
		}},
		"selectedStrategyId": "momentum",
	})

	identity, err := DecodeProjectionIdentity(1, projection)
	if err != nil {
		t.Fatalf("DecodeProjectionIdentity() error = %v", err)
	}
	if len(identity.StrategyIDs) != 1 || identity.StrategyIDs[0] != "momentum" {
		t.Fatalf("identity.StrategyIDs = %v", identity.StrategyIDs)
	}
	if identity.SelectedStrategyID == nil || *identity.SelectedStrategyID != "momentum" {
		t.Fatalf("identity.SelectedStrategyID = %v", identity.SelectedStrategyID)
	}
}

func TestDecodeProjectionIdentitySortsStrategies(t *testing.T) {
	projection := defaultProjection(t)
	projection = mutateProjection(t, projection, map[string]any{
		"strategies": []map[string]any{
			{"id": "zebra", "name": "Z", "description": "Z", "upside": "U", "downside": "D", "available": true},
			{"id": "alpha", "name": "A", "description": "A", "upside": "U", "downside": "D", "available": true},
		},
	})
	identity, err := DecodeProjectionIdentity(1, projection)
	if err != nil {
		t.Fatalf("DecodeProjectionIdentity() error = %v", err)
	}
	if identity.StrategyIDs[0] != "alpha" || identity.StrategyIDs[1] != "zebra" {
		t.Fatalf("identity.StrategyIDs = %v", identity.StrategyIDs)
	}
}

func TestProjectionIdentityCompatibleIgnoresSelection(t *testing.T) {
	base := ProjectionIdentity{ModifierID: "mod_default", ObjectiveID: "obj_default", StrategyIDs: []string{"momentum"}}
	withSelection := base
	selection := "momentum"
	withSelection.SelectedStrategyID = &selection

	if !base.Compatible(withSelection) {
		t.Fatalf("selection differences must be compatible")
	}
	other := ProjectionIdentity{ModifierID: "other", ObjectiveID: "obj_default", StrategyIDs: []string{"momentum"}}
	if base.Compatible(other) {
		t.Fatalf("modifier differences must be incompatible")
	}
}

func TestDecodeProjectionIdentityRejectsUnknownSchemaAndMalformedData(t *testing.T) {
	if _, err := DecodeProjectionIdentity(2, defaultProjection(t)); err == nil {
		t.Fatalf("unsupported schema version must fail")
	}
	if _, err := DecodeProjectionIdentity(1, nil); err == nil {
		t.Fatalf("empty projection must fail")
	}
	if _, err := DecodeProjectionIdentity(1, []byte(`{"modifier":{"id":""}}`)); err == nil {
		t.Fatalf("invalid projection must fail")
	}
}

// defaultProjection returns the exact schema-1 projection written by the
// voyage service for a fresh day (mod_default / obj_default, empty strategy set).
func defaultProjection(t *testing.T) []byte {
	t.Helper()
	value := map[string]any{
		"modifier":  map[string]any{"id": "mod_default", "name": "Standard Conditions", "description": "Default"},
		"objective": map[string]any{"id": "obj_default", "name": "Navigate", "description": "Default"},
		"signals":   []any{},
		"lineup": map[string]any{
			"maxSlots":  1,
			"slots":     []any{map[string]any{"index": 0}},
			"synergies": []any{},
			"warnings":  []any{},
		},
		"inventory":          []any{},
		"shop":               map[string]any{"offers": []any{}, "rerollCost": 2, "rerollIndex": 0},
		"strategies":         []any{},
		"selectedStrategyId": nil,
		"pendingRewardCount": 0,
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}
	return encoded
}

func mutateProjection(t *testing.T, projection []byte, changes map[string]any) []byte {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(projection, &value); err != nil {
		t.Fatalf("unmarshal projection: %v", err)
	}
	for key, change := range changes {
		value[key] = change
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal projection: %v", err)
	}
	return encoded
}
