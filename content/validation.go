package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/thaodangspace/tidekeepers-server/keeper"
)

const (
	maxNameLength        = 128
	maxDescriptionLength = 512
	maxDirectionLength   = 256
	maxRequiredCount     = 64
)

// Validate confirms that a candidate complete gameplay release is safe to
// publish under the production baseline registry.
func Validate(release Release) error {
	return validateWithRegistry(release, StandardRegistry())
}

// validateWithRegistry validates a candidate release against a supplied rule
// registry. Candidate indexes are built first so all references and semantics
// are resolved in memory before any publisher opens a transaction.
func validateWithRegistry(release Release, registry *Registry) error {
	if registry == nil {
		return errors.New("a rule registry is required")
	}
	if release.Version <= 0 {
		return validationErrorf(Kind("release"), "", "release version must be positive")
	}
	if err := keeper.Validate(release.Keeper); err != nil {
		return fmt.Errorf("keeper catalog: %w", err)
	}

	indexes := make(map[Kind]map[string]struct{}, 6)
	for _, group := range kindGroups(release) {
		keys, err := collectKeys(group.kind, group.definitions)
		if err != nil {
			return err
		}
		indexes[group.kind] = keys
	}
	ctx := &ReleaseContext{indexes: indexes}

	for _, definition := range release.Strategies {
		if err := validateStrategy(ctx, registry, definition); err != nil {
			return err
		}
	}
	for _, definition := range release.Relics {
		if err := validateRelic(ctx, registry, definition); err != nil {
			return err
		}
	}
	for _, definition := range release.Synergies {
		if err := validateSynergy(ctx, registry, definition); err != nil {
			return err
		}
	}
	for _, definition := range release.Modifiers {
		if err := validateModifier(ctx, registry, definition); err != nil {
			return err
		}
	}
	for _, definition := range release.Objectives {
		if err := validateObjective(ctx, registry, definition); err != nil {
			return err
		}
	}
	for _, definition := range release.GameRuleSets {
		if err := validateGameRuleSet(ctx, registry, definition); err != nil {
			return err
		}
	}
	return nil
}

func validateStrategy(ctx *ReleaseContext, registry *Registry, definition Strategy) error {
	if !contentKeyPattern.MatchString(definition.Key) {
		return validationErrorf(KindStrategy, definition.Key, "has an invalid key")
	}
	if err := validatePublicText(KindStrategy, definition.Key, "name", definition.Name, maxNameLength, true); err != nil {
		return err
	}
	if err := validatePublicText(KindStrategy, definition.Key, "description", definition.Description, maxDescriptionLength, true); err != nil {
		return err
	}
	if err := validatePublicText(KindStrategy, definition.Key, "upside", definition.Upside, maxDirectionLength, true); err != nil {
		return err
	}
	if err := validatePublicText(KindStrategy, definition.Key, "downside", definition.Downside, maxDirectionLength, true); err != nil {
		return err
	}
	return validateRule(KindStrategy, definition.Key, definition.RuleKey, definition.RuleConfig, ctx, registry)
}

func validateRelic(ctx *ReleaseContext, registry *Registry, definition Relic) error {
	if !contentKeyPattern.MatchString(definition.Key) {
		return validationErrorf(KindRelic, definition.Key, "has an invalid key")
	}
	if err := validatePublicText(KindRelic, definition.Key, "name", definition.Name, maxNameLength, true); err != nil {
		return err
	}
	if err := validatePublicText(KindRelic, definition.Key, "description", definition.Description, maxDescriptionLength, true); err != nil {
		return err
	}
	return validateRule(KindRelic, definition.Key, definition.RuleKey, definition.RuleConfig, ctx, registry)
}

func validateSynergy(ctx *ReleaseContext, registry *Registry, definition Synergy) error {
	if !contentKeyPattern.MatchString(definition.Key) {
		return validationErrorf(KindSynergy, definition.Key, "has an invalid key")
	}
	if err := validatePublicText(KindSynergy, definition.Key, "name", definition.Name, maxNameLength, true); err != nil {
		return err
	}
	if err := validatePublicText(KindSynergy, definition.Key, "description", definition.Description, maxDescriptionLength, true); err != nil {
		return err
	}
	if definition.RequiredCount < 1 || definition.RequiredCount > maxRequiredCount {
		return validationErrorf(KindSynergy, definition.Key, "required count must be between 1 and %d", maxRequiredCount)
	}
	return validateRule(KindSynergy, definition.Key, definition.RuleKey, definition.RuleConfig, ctx, registry)
}

func validateModifier(ctx *ReleaseContext, registry *Registry, definition Modifier) error {
	if !contentKeyPattern.MatchString(definition.Key) {
		return validationErrorf(KindModifier, definition.Key, "has an invalid key")
	}
	if err := validatePublicText(KindModifier, definition.Key, "name", definition.Name, maxNameLength, true); err != nil {
		return err
	}
	if err := validatePublicText(KindModifier, definition.Key, "description", definition.Description, maxDescriptionLength, true); err != nil {
		return err
	}
	return validateRule(KindModifier, definition.Key, definition.RuleKey, definition.RuleConfig, ctx, registry)
}

func validateObjective(ctx *ReleaseContext, registry *Registry, definition Objective) error {
	if !contentKeyPattern.MatchString(definition.Key) {
		return validationErrorf(KindObjective, definition.Key, "has an invalid key")
	}
	if err := validatePublicText(KindObjective, definition.Key, "name", definition.Name, maxNameLength, true); err != nil {
		return err
	}
	if err := validatePublicText(KindObjective, definition.Key, "description", definition.Description, maxDescriptionLength, true); err != nil {
		return err
	}
	if definition.ProgressLabel != nil {
		if err := validatePublicText(KindObjective, definition.Key, "progress label", *definition.ProgressLabel, maxDescriptionLength, false); err != nil {
			return err
		}
	}
	if definition.RewardLabel != nil {
		if err := validatePublicText(KindObjective, definition.Key, "reward label", *definition.RewardLabel, maxDescriptionLength, false); err != nil {
			return err
		}
	}
	return validateRule(KindObjective, definition.Key, definition.RuleKey, definition.RuleConfig, ctx, registry)
}

func validateGameRuleSet(ctx *ReleaseContext, registry *Registry, definition GameRuleSet) error {
	if !contentKeyPattern.MatchString(definition.Key) {
		return validationErrorf(KindGameRuleSet, definition.Key, "has an invalid key")
	}
	if err := validatePublicText(KindGameRuleSet, definition.Key, "name", definition.Name, maxNameLength, true); err != nil {
		return err
	}
	if err := validatePublicText(KindGameRuleSet, definition.Key, "description", definition.Description, maxDescriptionLength, true); err != nil {
		return err
	}
	return validateRule(KindGameRuleSet, definition.Key, definition.RuleKey, definition.RuleConfig, ctx, registry)
}

func validateRule(kind Kind, key, ruleKey string, config json.RawMessage, ctx *ReleaseContext, registry *Registry) error {
	if err := registry.validateRuleBinding(ctx, kind, key, ruleKey, config); err != nil {
		return validationErrorf(kind, key, "%v", err)
	}
	return nil
}

func validatePublicText(kind Kind, key, field, value string, maxLength int, required bool) error {
	if strings.TrimSpace(value) == "" {
		if required {
			return validationErrorf(kind, key, "has an empty %s", field)
		}
		return nil
	}
	if len(value) > maxLength {
		return validationErrorf(kind, key, "%s exceeds %d characters", field, maxLength)
	}
	return nil
}

func collectKeys(kind Kind, definitions []keyedDefinition) (map[string]struct{}, error) {
	keys := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		key := definition.stableKey()
		if _, exists := keys[key]; exists {
			return nil, validationErrorf(kind, key, "duplicate stable key")
		}
		keys[key] = struct{}{}
	}
	return keys, nil
}

func isJSONObject(raw json.RawMessage) bool {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return false
	}
	_, ok := value.(map[string]any)
	return ok
}

type kindGroup struct {
	kind        Kind
	definitions []keyedDefinition
}

func kindGroups(release Release) []kindGroup {
	return []kindGroup{
		{kind: KindStrategy, definitions: toKeyed(release.Strategies)},
		{kind: KindRelic, definitions: toKeyed(release.Relics)},
		{kind: KindSynergy, definitions: toKeyed(release.Synergies)},
		{kind: KindModifier, definitions: toKeyed(release.Modifiers)},
		{kind: KindObjective, definitions: toKeyed(release.Objectives)},
		{kind: KindGameRuleSet, definitions: toKeyed(release.GameRuleSets)},
	}
}

func toKeyed[T keyedDefinition](definitions []T) []keyedDefinition {
	result := make([]keyedDefinition, len(definitions))
	for i, definition := range definitions {
		result[i] = definition
	}
	return result
}
