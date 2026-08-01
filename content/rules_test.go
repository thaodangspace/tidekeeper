package content

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStandardRegistryServesBaselineRules(t *testing.T) {
	registry := StandardRegistry()
	for _, tc := range []struct {
		kind    Kind
		ruleKey string
		schema  int
	}{
		{KindModifier, RuleStandardConditions, 1},
		{KindObjective, RuleNavigate, 1},
		{KindGameRuleSet, RuleStandardRules, 1},
	} {
		validator, ok := registry.lookup(tc.kind, tc.ruleKey)
		if !ok {
			t.Fatalf("StandardRegistry missing %s/%s", tc.kind, tc.ruleKey)
		}
		if validator.SchemaVersion() != tc.schema {
			t.Errorf("%s/%s schema version = %d, want %d", tc.kind, tc.ruleKey, validator.SchemaVersion(), tc.schema)
		}
	}
	if _, ok := registry.lookup(KindStrategy, "TEST_STRATEGY"); ok {
		t.Fatal("StandardRegistry must not publish invented Strategy rules")
	}
}

func TestRegistryRejectsUnknownRuleKeyAndSchema(t *testing.T) {
	registry := NewRegistry(
		NewSpecValidator(RuleSpec{Kind: KindStrategy, RuleKey: "KNOWN_RULE", SchemaVersion: 1}),
	)
	if _, ok := registry.lookup(KindStrategy, "UNKNOWN_RULE"); ok {
		t.Fatal("lookup(unknown rule) succeeded")
	}

	err := registry.validateRuleBinding(nil, KindStrategy, "s1", "UNKNOWN_RULE", json.RawMessage(`{"schemaVersion":1,"parameters":{}}`))
	if err == nil || !strings.Contains(err.Error(), "unknown rule key") {
		t.Fatalf("unknown rule key error = %v, want substring %q", err, "unknown rule key")
	}

	err = registry.validateRuleBinding(nil, KindStrategy, "s1", "KNOWN_RULE", json.RawMessage(`{"schemaVersion":2,"parameters":{}}`))
	if err == nil || !strings.Contains(err.Error(), "unsupported rule schema version") {
		t.Fatalf("schema version error = %v, want substring %q", err, "unsupported rule schema version")
	}

	err = registry.validateRuleBinding(nil, KindStrategy, "s1", "KNOWN_RULE", json.RawMessage(`{"schemaVersion":1,"parameters":{},"extra":true}`))
	if err == nil || !strings.Contains(err.Error(), "envelope") {
		t.Fatalf("envelope error = %v, want substring %q", err, "envelope")
	}
}

func TestSpecValidatorRejectsUnknownFields(t *testing.T) {
	validator := NewSpecValidator(RuleSpec{
		Kind:          KindRelic,
		RuleKey:       "TEST_RELIC",
		SchemaVersion: 1,
	})
	err := validator.ValidateParameters(&ReleaseContext{}, "r1", json.RawMessage(`{"surprise":1}`))
	if err == nil || !strings.Contains(err.Error(), "unknown parameter") {
		t.Fatalf("unknown parameter error = %v, want substring %q", err, "unknown parameter")
	}
}

func TestSpecValidatorRejectsWrongParameterType(t *testing.T) {
	validator := NewSpecValidator(RuleSpec{
		Kind:          KindRelic,
		RuleKey:       "TEST_RELIC",
		SchemaVersion: 1,
		Parameters: []ParameterSpec{
			{Name: "bonusBps", Type: ParameterInteger, Required: true, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: ContentWeightScale}},
			{Name: "active", Type: ParameterBoolean},
		},
	})
	for config, want := range map[string]string{
		`{"bonusBps":"one"}`:           "must be an integer",
		`{"bonusBps":1.5}`:             "must be an integer",
		`{"bonusBps":true}`:            "must be an integer",
		`{"bonusBps":1000,"active":5}`: "must be a boolean",
		`{}`:                           "missing required parameter",
	} {
		err := validator.ValidateParameters(&ReleaseContext{}, "r1", json.RawMessage(config))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("config %s error = %v, want substring %q", config, err, want)
		}
	}
}

func TestSpecValidatorRejectsFixedPointRangeAndOverflow(t *testing.T) {
	validator := NewSpecValidator(RuleSpec{
		Kind:          KindRelic,
		RuleKey:       "TEST_RELIC",
		SchemaVersion: 1,
		Parameters: []ParameterSpec{
			{Name: "coverage", Type: ParameterInteger, Required: true, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: ContentWeightScale}},
		},
	})
	if err := validator.ValidateParameters(&ReleaseContext{}, "r1", json.RawMessage(`{"coverage":-1}`)); err == nil || !strings.Contains(err.Error(), "outside the range") {
		t.Fatalf("negative range error = %v, want substring %q", err, "outside the range")
	}
	if err := validator.ValidateParameters(&ReleaseContext{}, "r1", json.RawMessage(`{"coverage":1000001}`)); err == nil || !strings.Contains(err.Error(), "outside the range") {
		t.Fatalf("upper range error = %v, want substring %q", err, "outside the range")
	}
	if err := validator.ValidateParameters(&ReleaseContext{}, "r1", json.RawMessage(`{"coverage":500000}`)); err != nil {
		t.Fatalf("valid coverage rejected: %v", err)
	}

	overflowValidator := NewSpecValidator(RuleSpec{
		Kind:          KindRelic,
		RuleKey:       "OVERFLOW_RELIC",
		SchemaVersion: 1,
		Parameters: []ParameterSpec{
			{Name: "a", Type: ParameterInteger, Required: true, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: 1 << 62}},
			{Name: "b", Type: ParameterInteger, Required: true, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: 1 << 62}},
		},
		Total: &TotalSpec{Fields: []string{"a", "b"}, Total: 1},
	})
	err := overflowValidator.ValidateParameters(&ReleaseContext{}, "r1", json.RawMessage(`{"a":4611686018427387904,"b":4611686018427387904}`))
	if err == nil || !strings.Contains(err.Error(), "overflows") {
		t.Fatalf("overflow error = %v, want substring %q", err, "overflows")
	}
}

func TestSpecValidatorRejectsTotalMismatch(t *testing.T) {
	validator := NewSpecValidator(RuleSpec{
		Kind:          KindSynergy,
		RuleKey:       "TEST_WEIGHTED_SYNERGY",
		SchemaVersion: 1,
		Parameters: []ParameterSpec{
			{Name: "alpha", Type: ParameterInteger, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: ContentWeightScale}},
			{Name: "beta", Type: ParameterInteger, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: ContentWeightScale}},
		},
		Total: &TotalSpec{Fields: []string{"alpha", "beta"}, Total: ContentWeightScale},
	})
	if err := validator.ValidateParameters(&ReleaseContext{}, "s1", json.RawMessage(`{"alpha":600000,"beta":300000}`)); err == nil || !strings.Contains(err.Error(), "total 900000, want 1000000") {
		t.Fatalf("total mismatch error = %v, want substring %q", err, "total 900000, want 1000000")
	}
	if err := validator.ValidateParameters(&ReleaseContext{}, "s1", json.RawMessage(`{"alpha":600000}`)); err == nil || !strings.Contains(err.Error(), "total requires parameter") {
		t.Fatalf("missing total field error = %v, want substring %q", err, "total requires parameter")
	}
	if err := validator.ValidateParameters(&ReleaseContext{}, "s1", json.RawMessage(`{"alpha":600000,"beta":400000}`)); err != nil {
		t.Fatalf("valid total rejected: %v", err)
	}
}

func TestSpecValidatorRejectsMutuallyExclusiveParameters(t *testing.T) {
	validator := NewSpecValidator(RuleSpec{
		Kind:          KindStrategy,
		RuleKey:       "TEST_STRATEGY",
		SchemaVersion: 1,
		Parameters: []ParameterSpec{
			{Name: "flat", Type: ParameterInteger, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: ContentWeightScale}},
			{Name: "scaled", Type: ParameterInteger, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: ContentWeightScale}},
		},
		MutuallyExclusive: [][]string{{"flat", "scaled"}},
	})
	if err := validator.ValidateParameters(&ReleaseContext{}, "s1", json.RawMessage(`{"flat":100,"scaled":200}`)); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("mutual exclusion error = %v, want substring %q", err, "mutually exclusive")
	}
	if err := validator.ValidateParameters(&ReleaseContext{}, "s1", json.RawMessage(`{"flat":100}`)); err != nil {
		t.Fatalf("valid exclusive parameter rejected: %v", err)
	}
}

func TestSpecValidatorRejectsMissingAndCrossKindReferences(t *testing.T) {
	ctx := &ReleaseContext{indexes: map[Kind]map[string]struct{}{
		KindModifier: {"mod_default": {}},
		KindStrategy: {"tide_rider": {}},
	}}
	validator := NewSpecValidator(RuleSpec{
		Kind:          KindStrategy,
		RuleKey:       "REFERENCING_STRATEGY",
		SchemaVersion: 1,
		Parameters: []ParameterSpec{
			{Name: "modifierKey", Type: ParameterReference, Required: true, RefKind: KindModifier},
		},
	})
	if err := validator.ValidateParameters(ctx, "mod_follower", json.RawMessage(`{"modifierKey":"absent_modifier"}`)); err == nil || !strings.Contains(err.Error(), "references unknown modifier") {
		t.Fatalf("missing reference error = %v, want substring %q", err, "references unknown modifier")
	}
	if err := validator.ValidateParameters(ctx, "mod_follower", json.RawMessage(`{"modifierKey":"tide_rider"}`)); err == nil || !strings.Contains(err.Error(), "references unknown modifier") {
		t.Fatalf("cross-kind reference error = %v, want substring %q", err, "references unknown modifier")
	}
	if err := validator.ValidateParameters(ctx, "mod_follower", json.RawMessage(`{"modifierKey":"mod_default"}`)); err != nil {
		t.Fatalf("valid reference rejected: %v", err)
	}
}
