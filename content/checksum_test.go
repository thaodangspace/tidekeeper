package content

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"

	"github.com/thaodangspace/tidekeepers-server/keeper"
)

func gameplayReleaseFixture() Release {
	return Release{
		Version: 1,
		Keeper:  keeper.CatalogV1(),
		Strategies: []Strategy{
			{Key: "mod_follower", Name: "Mod Follower", Description: "test", Upside: "u", Downside: "d", RuleKey: "REFERENCING_STRATEGY", RuleConfig: rawConfig(1, `{"modifierKey":"mod_default"}`)},
			{Key: "tide_rider", Name: "Tide Rider", Description: "test", Upside: "u", Downside: "d", RuleKey: "TEST_STRATEGY", RuleConfig: rawConfig(1, `{}`)},
		},
		Relics: []Relic{
			{Key: "shell_brand", Name: "Shell Brand", Description: "test", RuleKey: "TEST_RELIC", RuleConfig: rawConfig(1, `{"coverage":500000}`)},
		},
		Synergies: []Synergy{
			{Key: "twin_tides", Name: "Twin Tides", Description: "test", RequiredCount: 2, RuleKey: "TEST_SYNERGY", RuleConfig: rawConfig(1, `{"memberCount":2}`)},
		},
		Modifiers: []Modifier{
			{Key: "mod_default", Name: "Standard Conditions", Description: "Default voyage conditions.", RuleKey: "STANDARD_CONDITIONS", RuleConfig: rawConfig(1, `{}`)},
		},
		Objectives: []Objective{
			{Key: "obj_default", Name: "Navigate", Description: "Complete the daily voyage.", RuleKey: "NAVIGATE", RuleConfig: rawConfig(1, `{}`)},
		},
		GameRuleSets: []GameRuleSet{
			{Key: "rules_standard", Name: "Standard Rules", Description: "Standard game rules.", RuleKey: "STANDARD_RULES", RuleConfig: rawConfig(1, `{}`)},
		},
	}
}

func TestChecksumIsIndependentOfOrdering(t *testing.T) {
	registry := testGameplayRegistry()
	want, err := checksumWithRegistry(gameplayReleaseFixture(), registry)
	if err != nil {
		t.Fatalf("checksum(baseline): %v", err)
	}

	reordered := gameplayReleaseFixture()
	slices.Reverse(reordered.Strategies)
	slices.Reverse(reordered.Relics)
	slices.Reverse(reordered.Synergies)
	slices.Reverse(reordered.Modifiers)
	slices.Reverse(reordered.Objectives)
	slices.Reverse(reordered.GameRuleSets)
	slices.Reverse(reordered.Keeper.Assets)
	slices.Reverse(reordered.Keeper.Baskets)
	slices.Reverse(reordered.Keeper.Definitions)
	for index := range reordered.Keeper.Baskets {
		slices.Reverse(reordered.Keeper.Baskets[index].Components)
	}
	// Reordered JSON object keys inside a rule configuration and its nested
	// parameter object must not change the digest.
	reordered.Synergies[0].RuleConfig = json.RawMessage(`{"parameters":{"memberCount":2},"schemaVersion":1}`)
	reordered.Relics[0].RuleConfig = rawConfig(1, `{"coverage":500000}`)

	got, err := checksumWithRegistry(reordered, registry)
	if err != nil {
		t.Fatalf("checksum(reordered): %v", err)
	}
	if !bytes.Equal(got[:], want[:]) {
		t.Fatalf("checksum = %x, want %x", got, want)
	}
}

func TestChecksumIsIndependentOfJSONObjectOrderingInKeeperCatalog(t *testing.T) {
	release := gameplayReleaseFixture()
	want, err := checksumWithRegistry(release, testGameplayRegistry())
	if err != nil {
		t.Fatalf("checksum(baseline): %v", err)
	}

	reordered := gameplayReleaseFixture()
	for index := range reordered.Keeper.Definitions {
		if reordered.Keeper.Definitions[index].Key == "crest_sovereign" {
			reordered.Keeper.Definitions[index].PassiveRuleConfig = []byte(`{"parameters":{"broadDeclineDepthPenaltyReductionBps":1500,"globalRelativeScoreBonusBps":2000,"minimumOutperformingComponents":3},"schemaVersion":1}`)
		}
	}
	got, err := checksumWithRegistry(reordered, testGameplayRegistry())
	if err != nil {
		t.Fatalf("checksum(reordered keeper config): %v", err)
	}
	if !bytes.Equal(got[:], want[:]) {
		t.Fatalf("checksum = %x, want %x", got, want)
	}
}

func TestChecksumRejectsInvalidRelease(t *testing.T) {
	release := validBaselineRelease()
	release.Modifiers[0].RuleConfig = json.RawMessage(`[]`)
	if _, err := Checksum(release); err == nil {
		t.Fatal("Checksum(invalid release) succeeded")
	}
}

func TestCanonicalJSONExcludesUUIDsAndTimestamps(t *testing.T) {
	encoded, err := canonicalReleaseJSON(gameplayReleaseFixture())
	if err != nil {
		t.Fatalf("canonicalReleaseJSON: %v", err)
	}
	payload := string(encoded)
	for _, forbidden := range []string{"00000000-0000-0000-0000-00000000", "createdAt", "publishedAt", "updatedAt", "ruleConfigOverride"} {
		if bytes.Contains([]byte(payload), []byte(forbidden)) {
			t.Fatalf("canonical payload contains forbidden fragment %q", forbidden)
		}
	}
}
