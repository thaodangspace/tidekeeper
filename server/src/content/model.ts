/** Gameplay content release model (ported from content/model.go). */

import type { CatalogRelease } from "./keeper_model.ts";

/** Checksum schemas identify canonical serializations. */
export const ChecksumSchemaV1 = 1;
export const ChecksumSchemaV2 = 2;

/** Fixed decimal scale for normalized gameplay-content weights (1 unit = 1e-6). */
export const ContentWeightScale = 1_000_000;

export const contentKeyPattern = /^[a-z][a-z0-9_]*$/;
export const ruleKeyPattern = /^[A-Z][A-Z0-9_]*$/;

export type Kind =
  | "strategy"
  | "relic"
  | "synergy"
  | "modifier"
  | "objective"
  | "game_rule_set";

export const KindStrategy: Kind = "strategy";
export const KindRelic: Kind = "relic";
export const KindSynergy: Kind = "synergy";
export const KindModifier: Kind = "modifier";
export const KindObjective: Kind = "objective";
export const KindGameRuleSet: Kind = "game_rule_set";

export interface Strategy {
  key: string;
  name: string;
  description: string;
  upside: string;
  downside: string;
  ruleKey: string;
  ruleConfig: string;
}

export interface Relic {
  key: string;
  name: string;
  description: string;
  ruleKey: string;
  ruleConfig: string;
}

export interface Synergy {
  key: string;
  name: string;
  description: string;
  requiredCount: number;
  ruleKey: string;
  ruleConfig: string;
}

export interface Modifier {
  key: string;
  name: string;
  description: string;
  ruleKey: string;
  ruleConfig: string;
}

export interface Objective {
  key: string;
  name: string;
  description: string;
  progressLabel: string | null;
  rewardLabel: string | null;
  ruleKey: string;
  ruleConfig: string;
}

export interface GameRuleSet {
  key: string;
  name: string;
  description: string;
  ruleKey: string;
  ruleConfig: string;
}

/** A complete candidate gameplay content release. */
export interface Release {
  version: number;
  keeper: CatalogRelease;
  strategies: Strategy[];
  relics: Relic[];
  synergies: Synergy[];
  modifiers: Modifier[];
  objectives: Objective[];
  gameRuleSets: GameRuleSet[];
}

export function stableKey(_kind: Kind, definition: { key: string }): string {
  return definition.key;
}
