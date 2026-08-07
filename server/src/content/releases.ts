/** Approved complete baseline releases (ported from content/releases.go). */

import type { Release } from "./model.ts";
import { catalogV1 } from "./keeper_catalog_v1.ts";
import { catalogV2 } from "./keeper_catalog_v2.ts";
import {
  RuleNavigate,
  RuleStandardConditions,
  RuleStandardRules,
} from "./rules.ts";

/** Baseline definition keys carried by every approved complete baseline release. */
export const BaselineModifierKey = "standard_conditions";
export const BaselineObjectiveKey = "navigate";
export const BaselineRuleSetKey = "standard_rules";

/** Baseline release versions. Approved complete baselines begin at version 3. */
export const BaselineV1Version = 3;
export const BaselineV2Version = 4;

/** Approved legacy source catalog labels. */
export const SourceCatalogV1 = "v1";
export const SourceCatalogV2 = "v2";

function standardRuleConfig(): string {
  return `{"schemaVersion":1,"parameters":{}}`;
}

/** Returns the approved complete baseline embedding CatalogV1 at `version`. */
export function completeReleaseV1(version: number): Release {
  const catalog = catalogV1();
  catalog.version = version;
  return completeBaseline(version, catalog);
}

/** Returns the approved complete baseline embedding CatalogV2 at `version`. */
export function completeReleaseV2(version: number): Release {
  const catalog = catalogV2();
  catalog.version = version;
  return completeBaseline(version, catalog);
}

/** Returns the approved complete baseline for a recognized source catalog. */
export function completeReleaseForSource(
  sourceCatalog: string,
  version: number,
): Release {
  if (sourceCatalog === SourceCatalogV2) {
    return completeReleaseV2(version);
  }
  return completeReleaseV1(version);
}

function completeBaseline(
  version: number,
  catalog: Release["keeper"],
): Release {
  return {
    version,
    keeper: catalog,
    strategies: [],
    relics: [],
    synergies: [],
    modifiers: [{
      key: BaselineModifierKey,
      name: "Standard Conditions",
      description: "Standard market conditions for this Daily Tide.",
      ruleKey: RuleStandardConditions,
      ruleConfig: standardRuleConfig(),
    }],
    objectives: [{
      key: BaselineObjectiveKey,
      name: "Navigate",
      description:
        "Hold a balanced position and outperform the sector benchmark.",
      progressLabel: null,
      rewardLabel: null,
      ruleKey: RuleNavigate,
      ruleConfig: standardRuleConfig(),
    }],
    gameRuleSets: [{
      key: BaselineRuleSetKey,
      name: "Standard Rules",
      description: "The standard Tidekeepers game-rule set.",
      ruleKey: RuleStandardRules,
      ruleConfig: standardRuleConfig(),
    }],
  };
}

export interface SourceFingerprint {
  label: string;
  sourceVersion: number;
  modifierID: string;
  objectiveID: string;
  modifierKey: string;
  objectiveKey: string;
  sourceCatalog: string;
}

const recognizedSourceFingerprints: SourceFingerprint[] = [
  {
    label: "migration-seeded-development",
    sourceVersion: 1,
    modifierID: "mod_default",
    objectiveID: "obj_default",
    modifierKey: BaselineModifierKey,
    objectiveKey: BaselineObjectiveKey,
    sourceCatalog: SourceCatalogV1,
  },
  {
    label: "keepers-v1",
    sourceVersion: 1,
    modifierID: BaselineModifierKey,
    objectiveID: BaselineObjectiveKey,
    modifierKey: BaselineModifierKey,
    objectiveKey: BaselineObjectiveKey,
    sourceCatalog: SourceCatalogV1,
  },
  {
    label: "sectors-v2",
    sourceVersion: 2,
    modifierID: BaselineModifierKey,
    objectiveID: BaselineObjectiveKey,
    modifierKey: BaselineModifierKey,
    objectiveKey: BaselineObjectiveKey,
    sourceCatalog: SourceCatalogV2,
  },
];

/** Returns the approved fingerprint matching a legacy source version and identity. */
export function recognizeSourceFingerprint(
  version: number,
  identity: { modifierID: string; objectiveID: string },
): SourceFingerprint | null {
  for (const fingerprint of recognizedSourceFingerprints) {
    if (
      fingerprint.sourceVersion === version &&
      fingerprint.modifierID === identity.modifierID &&
      fingerprint.objectiveID === identity.objectiveID
    ) {
      return fingerprint;
    }
  }
  return null;
}

/** Maps a legacy source content version to the approved source catalog. */
export function sourceCatalogForVersion(version: number): string | null {
  switch (version) {
    case 1:
      return SourceCatalogV1;
    case 2:
      return SourceCatalogV2;
    default:
      return null;
  }
}
