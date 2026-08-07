/** Complete gameplay release validation (ported from content/validation.go). */

import type {
  GameRuleSet,
  Kind,
  Modifier,
  Objective,
  Release,
  Relic,
  Strategy,
  Synergy,
} from "./model.ts";
import { contentKeyPattern } from "./model.ts";
import { Registry, ReleaseContext } from "./rules.ts";
import { standardRegistry } from "./rules.ts";
import { validateKeeperCatalog } from "./keeper_validation.ts";

const maxNameLength = 128;
const maxDescriptionLength = 512;
const maxDirectionLength = 256;
const maxRequiredCount = 64;

/** Confirms a candidate complete gameplay release is safe to publish. */
export function validateRelease(release: Release): void {
  validateWithRegistry(release, standardRegistry());
}

function validateWithRegistry(release: Release, registry: Registry): void {
  if (release.version <= 0) {
    throw new Error(`release content: release version must be positive`);
  }
  try {
    validateKeeperCatalog(release.keeper);
  } catch (error) {
    throw new Error(`keeper catalog: ${(error as Error).message}`);
  }

  const indexes = new Map<Kind, Set<string>>();
  for (const group of kindGroups(release)) {
    indexes.set(group.kind, collectKeys(group.kind, group.definitions));
  }
  const ctx = new ReleaseContext(indexes);

  for (const definition of release.strategies) {
    validateStrategy(ctx, registry, definition);
  }
  for (const definition of release.relics) {
    validateRelic(ctx, registry, definition);
  }
  for (const definition of release.synergies) {
    validateSynergy(ctx, registry, definition);
  }
  for (const definition of release.modifiers) {
    validateModifier(ctx, registry, definition);
  }
  for (const definition of release.objectives) {
    validateObjective(ctx, registry, definition);
  }
  for (const definition of release.gameRuleSets) {
    validateGameRuleSet(ctx, registry, definition);
  }
}

function validateStrategy(
  ctx: ReleaseContext,
  registry: Registry,
  definition: Strategy,
): void {
  if (!contentKeyPattern.test(definition.key)) {
    throw new Error(`strategy content ${definition.key}: has an invalid key`);
  }
  validatePublicText(
    "strategy",
    definition.key,
    "name",
    definition.name,
    maxNameLength,
    true,
  );
  validatePublicText(
    "strategy",
    definition.key,
    "description",
    definition.description,
    maxDescriptionLength,
    true,
  );
  validatePublicText(
    "strategy",
    definition.key,
    "upside",
    definition.upside,
    maxDirectionLength,
    true,
  );
  validatePublicText(
    "strategy",
    definition.key,
    "downside",
    definition.downside,
    maxDirectionLength,
    true,
  );
  validateRule(
    ctx,
    registry,
    "strategy",
    definition.key,
    definition.ruleKey,
    definition.ruleConfig,
  );
}

function validateRelic(
  ctx: ReleaseContext,
  registry: Registry,
  definition: Relic,
): void {
  if (!contentKeyPattern.test(definition.key)) {
    throw new Error(`relic content ${definition.key}: has an invalid key`);
  }
  validatePublicText(
    "relic",
    definition.key,
    "name",
    definition.name,
    maxNameLength,
    true,
  );
  validatePublicText(
    "relic",
    definition.key,
    "description",
    definition.description,
    maxDescriptionLength,
    true,
  );
  validateRule(
    ctx,
    registry,
    "relic",
    definition.key,
    definition.ruleKey,
    definition.ruleConfig,
  );
}

function validateSynergy(
  ctx: ReleaseContext,
  registry: Registry,
  definition: Synergy,
): void {
  if (!contentKeyPattern.test(definition.key)) {
    throw new Error(`synergy content ${definition.key}: has an invalid key`);
  }
  validatePublicText(
    "synergy",
    definition.key,
    "name",
    definition.name,
    maxNameLength,
    true,
  );
  validatePublicText(
    "synergy",
    definition.key,
    "description",
    definition.description,
    maxDescriptionLength,
    true,
  );
  if (
    definition.requiredCount < 1 || definition.requiredCount > maxRequiredCount
  ) {
    throw new Error(
      `synergy content ${definition.key}: required count must be between 1 and ${maxRequiredCount}`,
    );
  }
  validateRule(
    ctx,
    registry,
    "synergy",
    definition.key,
    definition.ruleKey,
    definition.ruleConfig,
  );
}

function validateModifier(
  ctx: ReleaseContext,
  registry: Registry,
  definition: Modifier,
): void {
  if (!contentKeyPattern.test(definition.key)) {
    throw new Error(`modifier content ${definition.key}: has an invalid key`);
  }
  validatePublicText(
    "modifier",
    definition.key,
    "name",
    definition.name,
    maxNameLength,
    true,
  );
  validatePublicText(
    "modifier",
    definition.key,
    "description",
    definition.description,
    maxDescriptionLength,
    true,
  );
  validateRule(
    ctx,
    registry,
    "modifier",
    definition.key,
    definition.ruleKey,
    definition.ruleConfig,
  );
}

function validateObjective(
  ctx: ReleaseContext,
  registry: Registry,
  definition: Objective,
): void {
  if (!contentKeyPattern.test(definition.key)) {
    throw new Error(`objective content ${definition.key}: has an invalid key`);
  }
  validatePublicText(
    "objective",
    definition.key,
    "name",
    definition.name,
    maxNameLength,
    true,
  );
  validatePublicText(
    "objective",
    definition.key,
    "description",
    definition.description,
    maxDescriptionLength,
    true,
  );
  if (definition.progressLabel !== null) {
    validatePublicText(
      "objective",
      definition.key,
      "progress label",
      definition.progressLabel,
      maxDescriptionLength,
      false,
    );
  }
  if (definition.rewardLabel !== null) {
    validatePublicText(
      "objective",
      definition.key,
      "reward label",
      definition.rewardLabel,
      maxDescriptionLength,
      false,
    );
  }
  validateRule(
    ctx,
    registry,
    "objective",
    definition.key,
    definition.ruleKey,
    definition.ruleConfig,
  );
}

function validateGameRuleSet(
  ctx: ReleaseContext,
  registry: Registry,
  definition: GameRuleSet,
): void {
  if (!contentKeyPattern.test(definition.key)) {
    throw new Error(
      `game_rule_set content ${definition.key}: has an invalid key`,
    );
  }
  validatePublicText(
    "game_rule_set",
    definition.key,
    "name",
    definition.name,
    maxNameLength,
    true,
  );
  validatePublicText(
    "game_rule_set",
    definition.key,
    "description",
    definition.description,
    maxDescriptionLength,
    true,
  );
  validateRule(
    ctx,
    registry,
    "game_rule_set",
    definition.key,
    definition.ruleKey,
    definition.ruleConfig,
  );
}

function validateRule(
  ctx: ReleaseContext,
  registry: Registry,
  kind: Kind,
  key: string,
  ruleKey: string,
  config: string,
): void {
  try {
    registry.validateRuleBinding(ctx, kind, key, ruleKey, config);
  } catch (error) {
    throw new Error(`${kind} content ${key}: ${(error as Error).message}`);
  }
}

function validatePublicText(
  kind: Kind,
  key: string,
  field: string,
  value: string,
  maxLength: number,
  required: boolean,
): void {
  if (value.trim() === "") {
    if (required) {
      throw new Error(`${kind} content ${key}: has an empty ${field}`);
    }
    return;
  }
  if (value.length > maxLength) {
    throw new Error(
      `${kind} content ${key}: ${field} exceeds ${maxLength} characters`,
    );
  }
}

function collectKeys(kind: Kind, definitions: { key: string }[]): Set<string> {
  const keys = new Set<string>();
  for (const definition of definitions) {
    if (keys.has(definition.key)) {
      throw new Error(
        `${kind} content ${definition.key}: duplicate stable key`,
      );
    }
    keys.add(definition.key);
  }
  return keys;
}

function kindGroups(
  release: Release,
): { kind: Kind; definitions: { key: string }[] }[] {
  return [
    { kind: "strategy", definitions: release.strategies },
    { kind: "relic", definitions: release.relics },
    { kind: "synergy", definitions: release.synergies },
    { kind: "modifier", definitions: release.modifiers },
    { kind: "objective", definitions: release.objectives },
    { kind: "game_rule_set", definitions: release.gameRuleSets },
  ];
}
