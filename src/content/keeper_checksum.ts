/** Keeper catalog canonical serialization and checksum (ported from keeper/checksum.go). */

import type { CatalogRelease } from "./keeper_model.ts";
import { validateKeeperCatalog } from "./keeper_validation.ts";

interface CanonicalComponent {
  assetKey: string;
  weight: number;
}

interface CanonicalBasket {
  key: string;
  sector: string;
  minimumCoveredWeight: number;
  expectedTurbulencePolicyKey: string;
  normalizationCap: number;
  benchmarkEligible: boolean;
  components: CanonicalComponent[];
}

interface CanonicalDefinition {
  key: string;
  name: string;
  currentKey: string;
  currentName: string;
  sector: string;
  role: string;
  rarity: string;
  baseRisk: string;
  expectedTurbulenceBps: number | null;
  passiveRuleKey: string;
  passiveRuleConfig: unknown;
  upgradeTree: unknown;
  basketMappingKey: string;
}

interface CanonicalRelease {
  version: number;
  assets: { key: string; symbol: string }[];
  sectorDefinitions: {
    sector: string;
    benchmarkMethod: string;
    minimumEligibleBaskets: number;
    relativeScale: number;
    relativeBlendWeight: number;
    rankBlendWeight: number;
    scoreCap: number;
  }[];
  turbulencePolicies: {
    key: string;
    type: string;
    staticValue: number;
    floor: number;
    roundingMode: string;
  }[];
  baskets: CanonicalBasket[];
  definitions: CanonicalDefinition[];
}

/** Returns the stable SHA-256 checksum (hex) for a validated catalog release. */
export async function keeperChecksum(release: CatalogRelease): Promise<string> {
  validateKeeperCatalog(release);
  const encoded = canonicalReleaseJson(release);
  const digest = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode(encoded),
  );
  return bytesToHex(new Uint8Array(digest));
}

/** Returns the canonical schema-1 serialization of a catalog release (no validation). */
export function keeperCanonicalJson(release: CatalogRelease): string {
  return canonicalReleaseJson(release);
}

function canonicalReleaseJson(release: CatalogRelease): string {
  const canonical: CanonicalRelease = {
    version: release.version,
    assets: [...release.assets].sort((
      a,
      b,
    ) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0)),
    sectorDefinitions: [...release.sectorDefinitions].sort((
      a,
      b,
    ) => (a.sector < b.sector ? -1 : a.sector > b.sector ? 1 : 0)),
    turbulencePolicies: [...release.turbulencePolicies].sort((
      a,
      b,
    ) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0)),
    baskets: [...release.baskets]
      .map((entry) => {
        const components = [...entry.components]
          .map((c) => ({ assetKey: c.assetKey, weight: c.weight }))
          .sort((
            a,
            b,
          ) => (a.assetKey < b.assetKey
            ? -1
            : a.assetKey > b.assetKey
            ? 1
            : 0)
          );
        return {
          key: entry.key,
          sector: entry.sector,
          minimumCoveredWeight: entry.minimumCoveredWeight,
          expectedTurbulencePolicyKey: entry.expectedTurbulencePolicyKey,
          normalizationCap: entry.normalizationCap,
          benchmarkEligible: entry.benchmarkEligible,
          components,
        } satisfies CanonicalBasket;
      })
      .sort((a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0)),
    definitions: [...release.definitions]
      .map((d) => ({
        key: d.key,
        name: d.name,
        currentKey: d.currentKey,
        currentName: d.currentName,
        sector: d.sector,
        role: d.role,
        rarity: d.rarity,
        baseRisk: d.baseRisk,
        expectedTurbulenceBps: d.expectedTurbulenceBPS,
        passiveRuleKey: d.passiveRuleKey,
        passiveRuleConfig: canonicalJsonObject(d.passiveRuleConfig),
        upgradeTree: canonicalJsonObject(d.upgradeTree),
        basketMappingKey: d.basketMappingKey,
      } satisfies CanonicalDefinition))
      .sort((a, b) => (a.key < b.key ? -1 : a.key > b.key ? 1 : 0)),
  };
  return JSON.stringify(canonical);
}

function canonicalJsonObject(raw: string): unknown {
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
