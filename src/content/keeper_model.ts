/** Versioned Keeper catalog content model (ported from keeper/catalog.go). */

/** Fixed decimal scale for basket component weights (1 unit = 1e-8). */
export const WeightScale = 100_000_000;
/** Fixed scale used by normalized calculation policy values (1 unit = 1e-6). */
export const CalculationScale = 1_000_000;

export interface MarketAsset {
  key: string;
  symbol: string;
}

export interface SectorDefinition {
  sector: string;
  benchmarkMethod: string;
  minimumEligibleBaskets: number;
  relativeScale: number;
  relativeBlendWeight: number;
  rankBlendWeight: number;
  scoreCap: number;
}

export interface ExpectedTurbulencePolicy {
  key: string;
  type: string;
  staticValue: number;
  floor: number;
  roundingMode: string;
}

export interface BasketComponent {
  assetKey: string;
  weight: number;
}

export interface BasketMapping {
  key: string;
  sector: string;
  minimumCoveredWeight: number;
  expectedTurbulencePolicyKey: string;
  normalizationCap: number;
  benchmarkEligible: boolean;
  components: BasketComponent[];
}

export interface Definition {
  key: string;
  name: string;
  currentKey: string;
  currentName: string;
  sector: string;
  role: string;
  rarity: string;
  baseRisk: string;
  expectedTurbulenceBPS: number | null;
  passiveRuleKey: string;
  passiveRuleConfig: string;
  upgradeTree: string;
  basketMappingKey: string;
}

export interface CatalogRelease {
  version: number;
  assets: MarketAsset[];
  sectorDefinitions?: SectorDefinition[];
  turbulencePolicies?: ExpectedTurbulencePolicy[];
  baskets: BasketMapping[];
  definitions: Definition[];
}
