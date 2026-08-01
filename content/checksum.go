package content

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/thaodangspace/tidekeepers-server/keeper"
)

// Checksum returns the canonical schema-2 SHA-256 digest for a complete
// validated gameplay release. The production baseline registry validates the
// candidate before hashing.
func Checksum(release Release) ([sha256.Size]byte, error) {
	if err := validateWithRegistry(release, StandardRegistry()); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("validate release before checksumming: %w", err)
	}
	return canonicalChecksum(release)
}

// checksumWithRegistry validates and checksums a candidate under a supplied
// registry so unit tests can exercise ordering fixtures without a database.
func checksumWithRegistry(release Release, registry *Registry) ([sha256.Size]byte, error) {
	if err := validateWithRegistry(release, registry); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("validate release before checksumming: %w", err)
	}
	return canonicalChecksum(release)
}

func canonicalChecksum(release Release) ([sha256.Size]byte, error) {
	encoded, err := canonicalReleaseJSON(release)
	if err != nil {
		return [sha256.Size]byte{}, err
	}
	return sha256.Sum256(encoded), nil
}

// canonicalReleaseJSON builds the deterministic schema-2 payload with explicit
// sections, stable-key sorting, canonical JSON objects, and fixed-point
// integers. UUIDs and timestamps are excluded by construction.
func canonicalReleaseJSON(release Release) ([]byte, error) {
	keeperJSON, err := keeper.CanonicalJSON(release.Keeper)
	if err != nil {
		return nil, fmt.Errorf("canonicalize Keeper catalog: %w", err)
	}

	canonical := struct {
		ChecksumSchema int                    `json:"checksumSchema"`
		Version        int64                  `json:"version"`
		Keepers        json.RawMessage        `json:"keepers"`
		Strategies     []canonicalStrategy    `json:"strategies"`
		Relics         []canonicalRelic       `json:"relics"`
		Synergies      []canonicalSynergy     `json:"synergies"`
		Modifiers      []canonicalModifier    `json:"modifiers"`
		Objectives     []canonicalObjective   `json:"objectives"`
		GameRuleSets   []canonicalGameRuleSet `json:"gameRuleSets"`
	}{
		ChecksumSchema: ChecksumSchemaV2,
		Version:        release.Version,
		Keepers:        keeperJSON,
	}

	if canonical.Strategies, err = canonicalStrategies(release.Strategies); err != nil {
		return nil, err
	}
	if canonical.Relics, err = canonicalRelics(release.Relics); err != nil {
		return nil, err
	}
	if canonical.Synergies, err = canonicalSynergies(release.Synergies); err != nil {
		return nil, err
	}
	if canonical.Modifiers, err = canonicalModifiers(release.Modifiers); err != nil {
		return nil, err
	}
	if canonical.Objectives, err = canonicalObjectives(release.Objectives); err != nil {
		return nil, err
	}
	if canonical.GameRuleSets, err = canonicalGameRuleSets(release.GameRuleSets); err != nil {
		return nil, err
	}

	encoded, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("marshal canonical release: %w", err)
	}
	return encoded, nil
}

type canonicalStrategy struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Upside      string          `json:"upside"`
	Downside    string          `json:"downside"`
	RuleKey     string          `json:"ruleKey"`
	RuleConfig  json.RawMessage `json:"ruleConfig"`
}

type canonicalRelic struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	RuleKey     string          `json:"ruleKey"`
	RuleConfig  json.RawMessage `json:"ruleConfig"`
}

type canonicalSynergy struct {
	Key           string          `json:"key"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	RequiredCount int             `json:"requiredCount"`
	RuleKey       string          `json:"ruleKey"`
	RuleConfig    json.RawMessage `json:"ruleConfig"`
}

type canonicalModifier struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	RuleKey     string          `json:"ruleKey"`
	RuleConfig  json.RawMessage `json:"ruleConfig"`
}

type canonicalObjective struct {
	Key           string          `json:"key"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	ProgressLabel *string         `json:"progressLabel"`
	RewardLabel   *string         `json:"rewardLabel"`
	RuleKey       string          `json:"ruleKey"`
	RuleConfig    json.RawMessage `json:"ruleConfig"`
}

type canonicalGameRuleSet struct {
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	RuleKey     string          `json:"ruleKey"`
	RuleConfig  json.RawMessage `json:"ruleConfig"`
}

func canonicalStrategies(definitions []Strategy) ([]canonicalStrategy, error) {
	result := make([]canonicalStrategy, 0, len(definitions))
	for _, definition := range definitions {
		config, err := canonicalJSON(definition.RuleConfig)
		if err != nil {
			return nil, fmt.Errorf("canonicalize strategy %q rule config: %w", definition.Key, err)
		}
		result = append(result, canonicalStrategy{
			Key: definition.Key, Name: definition.Name, Description: definition.Description,
			Upside: definition.Upside, Downside: definition.Downside,
			RuleKey: definition.RuleKey, RuleConfig: config,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

func canonicalRelics(definitions []Relic) ([]canonicalRelic, error) {
	result := make([]canonicalRelic, 0, len(definitions))
	for _, definition := range definitions {
		config, err := canonicalJSON(definition.RuleConfig)
		if err != nil {
			return nil, fmt.Errorf("canonicalize relic %q rule config: %w", definition.Key, err)
		}
		result = append(result, canonicalRelic{
			Key: definition.Key, Name: definition.Name, Description: definition.Description,
			RuleKey: definition.RuleKey, RuleConfig: config,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

func canonicalSynergies(definitions []Synergy) ([]canonicalSynergy, error) {
	result := make([]canonicalSynergy, 0, len(definitions))
	for _, definition := range definitions {
		config, err := canonicalJSON(definition.RuleConfig)
		if err != nil {
			return nil, fmt.Errorf("canonicalize synergy %q rule config: %w", definition.Key, err)
		}
		result = append(result, canonicalSynergy{
			Key: definition.Key, Name: definition.Name, Description: definition.Description,
			RequiredCount: definition.RequiredCount,
			RuleKey:       definition.RuleKey, RuleConfig: config,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

func canonicalModifiers(definitions []Modifier) ([]canonicalModifier, error) {
	result := make([]canonicalModifier, 0, len(definitions))
	for _, definition := range definitions {
		config, err := canonicalJSON(definition.RuleConfig)
		if err != nil {
			return nil, fmt.Errorf("canonicalize modifier %q rule config: %w", definition.Key, err)
		}
		result = append(result, canonicalModifier{
			Key: definition.Key, Name: definition.Name, Description: definition.Description,
			RuleKey: definition.RuleKey, RuleConfig: config,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

func canonicalObjectives(definitions []Objective) ([]canonicalObjective, error) {
	result := make([]canonicalObjective, 0, len(definitions))
	for _, definition := range definitions {
		config, err := canonicalJSON(definition.RuleConfig)
		if err != nil {
			return nil, fmt.Errorf("canonicalize objective %q rule config: %w", definition.Key, err)
		}
		result = append(result, canonicalObjective{
			Key: definition.Key, Name: definition.Name, Description: definition.Description,
			ProgressLabel: definition.ProgressLabel, RewardLabel: definition.RewardLabel,
			RuleKey: definition.RuleKey, RuleConfig: config,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

func canonicalGameRuleSets(definitions []GameRuleSet) ([]canonicalGameRuleSet, error) {
	result := make([]canonicalGameRuleSet, 0, len(definitions))
	for _, definition := range definitions {
		config, err := canonicalJSON(definition.RuleConfig)
		if err != nil {
			return nil, fmt.Errorf("canonicalize game rule set %q rule config: %w", definition.Key, err)
		}
		result = append(result, canonicalGameRuleSet{
			Key: definition.Key, Name: definition.Name, Description: definition.Description,
			RuleKey: definition.RuleKey, RuleConfig: config,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result, nil
}

// canonicalJSON re-encodes a JSON value with deterministically sorted object
// keys and number-literal-preserving decoding.
func canonicalJSON(raw json.RawMessage) (json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}
