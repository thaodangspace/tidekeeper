/** Complete gameplay release canonical checksum (ported from content/checksum.go). */

import type {
  GameRuleSet,
  Modifier,
  Objective,
  Release,
  Relic,
  Strategy,
  Synergy,
} from "./model.ts";
import { ChecksumSchemaV2 } from "./model.ts";
import { keeperCanonicalJson } from "./keeper_checksum.ts";
import { validateRelease } from "./validation.ts";

interface CanonicalGameplay<T> {
  key: string;
  name: string;
  description: string;
  ruleKey: string;
  ruleConfig: unknown;
  [extra: string]: unknown;
}

/** Returns the canonical schema-2 SHA-256 digest (hex) for a validated release. */
export async function contentChecksum(release: Release): Promise<string> {
  validateRelease(release);
  const encoded = canonicalReleaseJson(release);
  const digest = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(encoded),
  );
  return bytesToHex(new Uint8Array(digest));
}

export function canonicalReleaseJson(release: Release): string {
  const canonical = {
    checksumSchema: ChecksumSchemaV2,
    version: release.version,
    keepers: JSON.parse(keeperCanonicalJson(release.keeper)),
    strategies: canonicalStrategies(release.strategies),
    relics: canonicalRelics(release.relics),
    synergies: canonicalSynergies(release.synergies),
    modifiers: canonicalModifiers(release.modifiers),
    objectives: canonicalObjectives(release.objectives),
    gameRuleSets: canonicalGameRuleSets(release.gameRuleSets),
  };
  return JSON.stringify(canonical);
}

function canonicalStrategies(
  definitions: Strategy[],
): CanonicalGameplay<Strategy>[] {
  return definitions
    .map((d) => ({
      key: d.key,
      name: d.name,
      description: d.description,
      upside: d.upside,
      downside: d.downside,
      ruleKey: d.ruleKey,
      ruleConfig: canonicalJson(d.ruleConfig),
    }))
    .sort(byKey);
}

function canonicalRelics(definitions: Relic[]): CanonicalGameplay<Relic>[] {
  return definitions
    .map((d) => ({
      key: d.key,
      name: d.name,
      description: d.description,
      ruleKey: d.ruleKey,
      ruleConfig: canonicalJson(d.ruleConfig),
    }))
    .sort(byKey);
}

function canonicalSynergies(
  definitions: Synergy[],
): CanonicalGameplay<Synergy>[] {
  return definitions
    .map((d) => ({
      key: d.key,
      name: d.name,
      description: d.description,
      requiredCount: d.requiredCount,
      ruleKey: d.ruleKey,
      ruleConfig: canonicalJson(d.ruleConfig),
    }))
    .sort(byKey);
}

function canonicalModifiers(
  definitions: Modifier[],
): CanonicalGameplay<Modifier>[] {
  return definitions
    .map((d) => ({
      key: d.key,
      name: d.name,
      description: d.description,
      ruleKey: d.ruleKey,
      ruleConfig: canonicalJson(d.ruleConfig),
    }))
    .sort(byKey);
}

function canonicalObjectives(
  definitions: Objective[],
): CanonicalGameplay<Objective>[] {
  return definitions
    .map((d) => ({
      key: d.key,
      name: d.name,
      description: d.description,
      progressLabel: d.progressLabel,
      rewardLabel: d.rewardLabel,
      ruleKey: d.ruleKey,
      ruleConfig: canonicalJson(d.ruleConfig),
    }))
    .sort(byKey);
}

function canonicalGameRuleSets(
  definitions: GameRuleSet[],
): CanonicalGameplay<GameRuleSet>[] {
  return definitions
    .map((d) => ({
      key: d.key,
      name: d.name,
      description: d.description,
      ruleKey: d.ruleKey,
      ruleConfig: canonicalJson(d.ruleConfig),
    }))
    .sort(byKey);
}

function byKey(a: { key: string }, b: { key: string }): number {
  return a.key < b.key ? -1 : a.key > b.key ? 1 : 0;
}

function canonicalJson(raw: string): unknown {
  const decoded: unknown = JSON.parse(raw);
  return JSON.parse(JSON.stringify(decoded));
}

function bytesToHex(bytes: Uint8Array): string {
  const alphabet = "0123456789abcdef";
  let out = "";
  for (const b of bytes) {
    out += alphabet[b >>> 4] + alphabet[b & 0x0f];
  }
  return out;
}
