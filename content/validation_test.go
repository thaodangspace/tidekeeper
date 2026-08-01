package content

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/thaodangspace/tidekeepers-server/keeper"
)

// testGameplayRegistry exercises validation mechanics for every content kind
// using test-only descriptors so production balance rules are never invented.
func testGameplayRegistry() *Registry {
	return NewRegistry(
		NewSpecValidator(RuleSpec{Kind: KindModifier, RuleKey: RuleStandardConditions, SchemaVersion: 1}),
		NewSpecValidator(RuleSpec{Kind: KindObjective, RuleKey: RuleNavigate, SchemaVersion: 1}),
		NewSpecValidator(RuleSpec{Kind: KindGameRuleSet, RuleKey: RuleStandardRules, SchemaVersion: 1}),
		NewSpecValidator(RuleSpec{Kind: KindStrategy, RuleKey: "TEST_STRATEGY", SchemaVersion: 1}),
		NewSpecValidator(RuleSpec{
			Kind:          KindStrategy,
			RuleKey:       "REFERENCING_STRATEGY",
			SchemaVersion: 1,
			Parameters: []ParameterSpec{
				{Name: "modifierKey", Type: ParameterReference, Required: true, RefKind: KindModifier},
			},
		}),
		NewSpecValidator(RuleSpec{
			Kind:          KindRelic,
			RuleKey:       "TEST_RELIC",
			SchemaVersion: 1,
			Parameters: []ParameterSpec{
				{Name: "coverage", Type: ParameterInteger, Required: true, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 0, Max: ContentWeightScale}},
			},
		}),
		NewSpecValidator(RuleSpec{
			Kind:          KindSynergy,
			RuleKey:       "TEST_SYNERGY",
			SchemaVersion: 1,
			Parameters: []ParameterSpec{
				{Name: "memberCount", Type: ParameterInteger, Required: true, Fixed: &FixedPoint{Scale: ContentWeightScale, Min: 1, Max: ContentWeightScale}},
			},
		}),
	)
}

func rawConfig(schemaVersion int, parameters string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"schemaVersion":%d,"parameters":%s}`, schemaVersion, parameters))
}

// validBaselineRelease returns a release valid under the production baseline
// registry, with empty Strategy, Relic, and Synergy collections.
func validBaselineRelease() Release {
	return Release{
		Version: 1,
		Keeper:  keeper.CatalogV1(),
		Modifiers: []Modifier{{
			Key: "mod_default", Name: "Standard Conditions", Description: "Default voyage conditions.",
			RuleKey: "STANDARD_CONDITIONS", RuleConfig: rawConfig(1, `{}`),
		}},
		Objectives: []Objective{{
			Key: "obj_default", Name: "Navigate", Description: "Complete the daily voyage.",
			RuleKey: "NAVIGATE", RuleConfig: rawConfig(1, `{}`),
		}},
		GameRuleSets: []GameRuleSet{{
			Key: "rules_standard", Name: "Standard Rules", Description: "Standard game rules.",
			RuleKey: "STANDARD_RULES", RuleConfig: rawConfig(1, `{}`),
		}},
	}
}

func TestValidateBaselineRelease(t *testing.T) {
	if err := Validate(validBaselineRelease()); err != nil {
		t.Fatalf("Validate(baseline): %v", err)
	}
}

func TestValidateAcceptsCompleteGameplayRelease(t *testing.T) {
	release := Release{
		Version: 1,
		Keeper:  keeper.CatalogV1(),
		Strategies: []Strategy{
			{Key: "tide_rider", Name: "Tide Rider", Description: "test", Upside: "u", Downside: "d", RuleKey: "TEST_STRATEGY", RuleConfig: rawConfig(1, `{}`)},
			{Key: "mod_follower", Name: "Mod Follower", Description: "test", Upside: "u", Downside: "d", RuleKey: "REFERENCING_STRATEGY", RuleConfig: rawConfig(1, `{"modifierKey":"mod_default"}`)},
		},
		Relics:       []Relic{{Key: "shell_brand", Name: "Shell Brand", Description: "test", RuleKey: "TEST_RELIC", RuleConfig: rawConfig(1, `{"coverage":500000}`)}},
		Synergies:    []Synergy{{Key: "twin_tides", Name: "Twin Tides", Description: "test", RequiredCount: 2, RuleKey: "TEST_SYNERGY", RuleConfig: rawConfig(1, `{"memberCount":2}`)}},
		Modifiers:    []Modifier{{Key: "mod_default", Name: "Standard Conditions", Description: "Default voyage conditions.", RuleKey: "STANDARD_CONDITIONS", RuleConfig: rawConfig(1, `{}`)}},
		Objectives:   []Objective{{Key: "obj_default", Name: "Navigate", Description: "Complete the daily voyage.", RuleKey: "NAVIGATE", RuleConfig: rawConfig(1, `{}`)}},
		GameRuleSets: []GameRuleSet{{Key: "rules_standard", Name: "Standard Rules", Description: "Standard game rules.", RuleKey: "STANDARD_RULES", RuleConfig: rawConfig(1, `{}`)}},
	}
	if err := validateWithRegistry(release, testGameplayRegistry()); err != nil {
		t.Fatalf("validateWithRegistry(complete): %v", err)
	}
}

func TestValidateRejectsInvalidGameplayContent(t *testing.T) {
	registry := testGameplayRegistry()
	base := func() Release {
		release := Release{
			Version: 1,
			Keeper:  keeper.CatalogV1(),
			Strategies: []Strategy{
				{Key: "tide_rider", Name: "Tide Rider", Description: "test", Upside: "u", Downside: "d", RuleKey: "TEST_STRATEGY", RuleConfig: rawConfig(1, `{}`)},
				{Key: "mod_follower", Name: "Mod Follower", Description: "test", Upside: "u", Downside: "d", RuleKey: "REFERENCING_STRATEGY", RuleConfig: rawConfig(1, `{"modifierKey":"mod_default"}`)},
			},
			Relics:       []Relic{{Key: "shell_brand", Name: "Shell Brand", Description: "test", RuleKey: "TEST_RELIC", RuleConfig: rawConfig(1, `{"coverage":500000}`)}},
			Synergies:    []Synergy{{Key: "twin_tides", Name: "Twin Tides", Description: "test", RequiredCount: 2, RuleKey: "TEST_SYNERGY", RuleConfig: rawConfig(1, `{"memberCount":2}`)}},
			Modifiers:    []Modifier{{Key: "mod_default", Name: "Standard Conditions", Description: "Default voyage conditions.", RuleKey: "STANDARD_CONDITIONS", RuleConfig: rawConfig(1, `{}`)}},
			Objectives:   []Objective{{Key: "obj_default", Name: "Navigate", Description: "Complete the daily voyage.", RuleKey: "NAVIGATE", RuleConfig: rawConfig(1, `{}`)}},
			GameRuleSets: []GameRuleSet{{Key: "rules_standard", Name: "Standard Rules", Description: "Standard game rules.", RuleKey: "STANDARD_RULES", RuleConfig: rawConfig(1, `{}`)}},
		}
		return release
	}

	tests := []struct {
		name   string
		mutate func(*Release)
		want   string
	}{
		{
			name: "duplicate strategy key",
			mutate: func(r *Release) {
				r.Strategies = append(r.Strategies, r.Strategies[0])
			},
			want: "duplicate stable key",
		},
		{
			name: "duplicate modifier key",
			mutate: func(r *Release) {
				r.Modifiers = append(r.Modifiers, r.Modifiers[0])
			},
			want: "duplicate stable key",
		},
		{
			name: "unknown strategy rule key",
			mutate: func(r *Release) {
				r.Strategies[0].RuleKey = "NOT_A_RULE"
			},
			want: "unknown rule key",
		},
		{
			name: "unsupported schema version",
			mutate: func(r *Release) {
				r.Modifiers[0].RuleConfig = rawConfig(2, `{}`)
			},
			want: "unsupported rule schema version",
		},
		{
			name: "unknown envelope field",
			mutate: func(r *Release) {
				r.Modifiers[0].RuleConfig = json.RawMessage(`{"schemaVersion":1,"parameters":{},"extra":true}`)
			},
			want: "envelope",
		},
		{
			name: "wrong parameter type",
			mutate: func(r *Release) {
				r.Synergies[0].RuleConfig = rawConfig(1, `{"memberCount":"two"}`)
			},
			want: "must be an integer",
		},
		{
			name: "fixed-point range failure",
			mutate: func(r *Release) {
				r.Relics[0].RuleConfig = rawConfig(1, `{"coverage":2000000}`)
			},
			want: "outside the range",
		},
		{
			name: "missing reference",
			mutate: func(r *Release) {
				r.Strategies[1].RuleConfig = rawConfig(1, `{"modifierKey":"absent_modifier"}`)
			},
			want: "references unknown modifier",
		},
		{
			name: "cross-kind reference",
			mutate: func(r *Release) {
				r.Strategies[1].RuleConfig = rawConfig(1, `{"modifierKey":"tide_rider"}`)
			},
			want: "references unknown modifier",
		},
		{
			name: "invalid strategy key",
			mutate: func(r *Release) {
				r.Strategies[0].Key = "Invalid_Key"
			},
			want: "invalid key",
		},
		{
			name: "empty strategy name",
			mutate: func(r *Release) {
				r.Strategies[0].Name = ""
			},
			want: "empty name",
		},
		{
			name: "synergy required count",
			mutate: func(r *Release) {
				r.Synergies[0].RequiredCount = 0
			},
			want: "required count",
		},
		{
			name: "non object rule configuration",
			mutate: func(r *Release) {
				r.Modifiers[0].RuleConfig = json.RawMessage(`[]`)
			},
			want: "must be a JSON object",
		},
		{
			name: "release version not positive",
			mutate: func(r *Release) {
				r.Version = 0
			},
			want: "version must be positive",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			release := base()
			test.mutate(&release)
			err := validateWithRegistry(release, registry)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validate() error = %v, want substring %q", err, test.want)
			}
		})
	}
}
