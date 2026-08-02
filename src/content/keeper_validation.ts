/** Keeper catalog validation (ported from keeper/validation.go). */

import type {
  BasketMapping,
  CatalogRelease,
  Definition,
  ExpectedTurbulencePolicy,
  SectorDefinition,
} from "./keeper_model.ts";
import { WeightScale } from "./keeper_model.ts";
import { parseUpgradeTree } from "./upgrade_tree.ts";

const contentKeyPattern = /^[a-z][a-z0-9_]*$/;
const symbolPattern = /^[A-Za-z][A-Za-z0-9]*$/;

const validSectors = tokenSet(
  "CREST",
  "HARBOR",
  "FORGE",
  "CURRENT",
  "ECHO",
  "VEIL",
  "BLOOM",
  "EMBER",
);
const validRoles = tokenSet(
  "VANGUARD",
  "WARDEN",
  "COMPOUNDER",
  "SUPPORT",
  "ORACLE",
  "CONTRARIAN",
  "TRICKSTER",
  "GROWTH",
);
const validRarities = tokenSet("COMMON", "RARE", "EPIC");
const validBaseRisks = tokenSet(
  "VERY_LOW",
  "LOW_TO_MEDIUM",
  "MEDIUM",
  "MEDIUM_TO_HIGH",
  "HIGH",
  "VERY_HIGH",
);

const passiveRuleParameters: Record<string, Set<string>> = {
  "CROWN_OF_THE_TIDE": parameterSet(
    "minimumOutperformingComponents",
    "globalRelativeScoreBonusBps",
    "broadDeclineDepthPenaltyReductionBps",
  ),
  "UNBROKEN_PEG": parameterSet(
    "depthPenaltyReductionBps",
    "stabilityThreshold",
    "hullDamageReductionBps",
  ),
  "FEE_ENGINE": parameterSet(
    "riskAdjustedScoreBonusBpsPerStack",
    "maxStacks",
    "removeStacksOnFleetExit",
    "shockPressurePenaltyReductionPerStackBps",
  ),
  "FLOW_TRANSFER": parameterSet(
    "minimumAdditionalOutperformingSectors",
    "diversificationScoreBonusBps",
    "volumeTrendBonusBps",
  ),
  "READ_THE_WAKE": parameterSet(
    "signalTypes",
    "selectionPolicy",
    "objectiveContributionBonusBps",
  ),
  "FROM_THE_DEPTHS": parameterSet(
    "depthExceedsExpectedTurbulence",
    "minimumRecoveryFromLowBps",
    "resurfaceScoreMultiplierBps",
    "fallbackDepthPenaltyReductionBps",
  ),
  "ACCRUED_TIDE": parameterSet(
    "skillEffectBonusBpsPerDay",
    "maxConsecutiveDays",
    "extraSuppliesProbabilityBps",
    "deterministicSeedScope",
  ),
  "WILD_SURGE": parameterSet(
    "positiveNormalizedPerformanceThresholdMicros",
    "sectorRelativeScoreMultiplierBps",
    "negativeNormalizedPerformanceThresholdMicros",
    "pressurePenaltyIncreaseBps",
    "pressurePenaltyApplicationCap",
  ),
  "DISTRIBUTED_ENGINE": parameterSet(
    "requiresPositiveBasketReturn",
    "requiresPositiveVolumeTrend",
    "minimumPositiveComponents",
    "performanceCeilingBonusBps",
    "outlierDepthPenaltyReductionBps",
  ),
  "SECOND_CURRENT": parameterSet(
    "requiresPositiveLayerOneBenchmark",
    "fleetBonusSectorRelativeScoreBps",
    "layerOneDeepDeclineDepthPenaltyIncreaseBps",
  ),
};

const calculationSectorKeys = tokenSet("CREST", "EMBER", "CURRENT", "HARBOR");

/** Confirms a candidate release can be safely persisted as content. */
export function validateKeeperCatalog(release: CatalogRelease): void {
  if (release.version <= 0) {
    throw new Error("release version must be positive");
  }

  const { sectorDefinitions, turbulencePolicies } = validateCalculationPolicies(
    release.sectorDefinitions ?? [],
    release.turbulencePolicies ?? [],
  );

  const assets = new Map<string, string>();
  const symbols = new Set<string>();
  for (const item of release.assets) {
    if (!contentKeyPattern.test(item.key)) {
      throw new Error(`asset ${item.key} has an invalid key`);
    }
    if (!symbolPattern.test(item.symbol)) {
      throw new Error(`asset ${item.key} has an invalid symbol`);
    }
    if (assets.has(item.key)) {
      throw new Error(`duplicate asset key ${item.key}`);
    }
    if (symbols.has(item.symbol)) {
      throw new Error(`duplicate asset symbol ${item.symbol}`);
    }
    assets.set(item.key, item.symbol);
    symbols.add(item.symbol);
  }

  const baskets = new Map<string, BasketMapping>();
  for (const entry of release.baskets) {
    if (!contentKeyPattern.test(entry.key)) {
      throw new Error(`basket ${entry.key} has an invalid key`);
    }
    if (baskets.has(entry.key)) {
      throw new Error(`duplicate basket key ${entry.key}`);
    }
    if (entry.components.length === 0) {
      throw new Error(`basket ${entry.key} has no components`);
    }
    const componentAssets = new Set<string>();
    let total = 0;
    for (const componentEntry of entry.components) {
      if (!assets.has(componentEntry.assetKey)) {
        throw new Error(
          `basket ${entry.key} references unknown asset ${componentEntry.assetKey}`,
        );
      }
      if (componentAssets.has(componentEntry.assetKey)) {
        throw new Error(
          `basket ${entry.key} repeats asset ${componentEntry.assetKey}`,
        );
      }
      if (componentEntry.weight <= 0 || componentEntry.weight > WeightScale) {
        throw new Error(
          `basket ${entry.key} has invalid weight for asset ${componentEntry.assetKey}`,
        );
      }
      if (total > WeightScale - componentEntry.weight) {
        throw new Error(
          `basket ${entry.key} component weights overflow fixed scale`,
        );
      }
      componentAssets.add(componentEntry.assetKey);
      total += componentEntry.weight;
    }
    if (total !== WeightScale) {
      throw new Error(
        `basket ${entry.key} component weights total ${total}, want ${WeightScale}`,
      );
    }
    baskets.set(entry.key, entry);
  }

  const definitionKeys = new Set<string>();
  for (const definitionEntry of release.definitions) {
    validateDefinition(definitionEntry, baskets);
    if (definitionKeys.has(definitionEntry.key)) {
      throw new Error(`duplicate Keeper key ${definitionEntry.key}`);
    }
    definitionKeys.add(definitionEntry.key);
  }
  validateBasketCalculationConfiguration(
    baskets,
    release.definitions,
    sectorDefinitions,
    turbulencePolicies,
  );
}

function validateDefinition(
  definition: Definition,
  baskets: Map<string, BasketMapping>,
): void {
  if (!contentKeyPattern.test(definition.key)) {
    throw new Error(`Keeper ${definition.key} has an invalid key`);
  }
  if (
    definition.name === "" || definition.currentName === "" ||
    !contentKeyPattern.test(definition.currentKey)
  ) {
    throw new Error(
      `Keeper ${definition.key} has invalid display Current data`,
    );
  }
  if (
    !validSectors.has(definition.sector) || !validRoles.has(definition.role) ||
    !validRarities.has(definition.rarity) ||
    !validBaseRisks.has(definition.baseRisk)
  ) {
    throw new Error(`Keeper ${definition.key} has an invalid classification`);
  }
  if (
    definition.expectedTurbulenceBPS !== null &&
    (definition.expectedTurbulenceBPS <= 0 ||
      definition.expectedTurbulenceBPS > 1_000_000)
  ) {
    throw new Error(`Keeper ${definition.key} has invalid expected turbulence`);
  }
  if (!baskets.has(definition.basketMappingKey)) {
    throw new Error(
      `Keeper ${definition.key} references unknown basket ${definition.basketMappingKey}`,
    );
  }
  validatePassiveRule(definition.passiveRuleKey, definition.passiveRuleConfig);
  parseUpgradeTree(definition.upgradeTree);
}

function validatePassiveRule(key: string, config: string): void {
  const expectedParameters = passiveRuleParameters[key];
  if (!expectedParameters) {
    throw new Error(`unknown passive rule key ${key}`);
  }
  let envelope: {
    schemaVersion: number;
    parameters: Record<string, unknown> | null;
  };
  try {
    const value = JSON.parse(config);
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
      throw new Error("must be a JSON object");
    }
    envelope = value;
  } catch {
    throw new Error(
      "invalid passive rule configuration: must be a JSON object",
    );
  }
  if (envelope.schemaVersion !== 1) {
    throw new Error(
      "invalid passive rule configuration: schemaVersion must be 1",
    );
  }
  if (
    envelope.parameters === null || typeof envelope.parameters !== "object" ||
    Array.isArray(envelope.parameters)
  ) {
    throw new Error(
      "invalid passive rule configuration: parameters must be a JSON object",
    );
  }
  for (const parameter of expectedParameters) {
    if (!(parameter in envelope.parameters)) {
      throw new Error(
        `invalid passive rule configuration: missing parameter ${parameter}`,
      );
    }
  }
  for (const parameter of Object.keys(envelope.parameters)) {
    if (!expectedParameters.has(parameter)) {
      throw new Error(
        `invalid passive rule configuration: unknown parameter ${parameter}`,
      );
    }
  }
}

function validateCalculationPolicies(
  definitions: SectorDefinition[],
  policies: ExpectedTurbulencePolicy[],
): {
  sectorDefinitions: Map<string, SectorDefinition>;
  turbulencePolicies: Map<string, ExpectedTurbulencePolicy>;
} {
  const sectorDefinitions = new Map<string, SectorDefinition>();
  const turbulencePolicies = new Map<string, ExpectedTurbulencePolicy>();
  if (definitions.length === 0 && policies.length === 0) {
    return { sectorDefinitions, turbulencePolicies };
  }
  if (
    definitions.length !== calculationSectorKeys.size || policies.length === 0
  ) {
    throw new Error(
      "calculation content requires four sector definitions and turbulence policies",
    );
  }
  for (const definition of definitions) {
    if (!calculationSectorKeys.has(definition.sector)) {
      throw new Error(`unsupported calculation sector ${definition.sector}`);
    }
    if (sectorDefinitions.has(definition.sector)) {
      throw new Error(`duplicate sector definition ${definition.sector}`);
    }
    if (
      definition.benchmarkMethod !== "EQUAL_WEIGHT" ||
      definition.minimumEligibleBaskets < 2 ||
      definition.relativeScale <= 0 || definition.scoreCap <= 0 ||
      definition.relativeBlendWeight < 0 || definition.rankBlendWeight < 0 ||
      definition.relativeBlendWeight + definition.rankBlendWeight !== 1_000_000
    ) {
      throw new Error(
        `invalid calculation definition for sector ${definition.sector}`,
      );
    }
    sectorDefinitions.set(definition.sector, definition);
  }
  for (const policy of policies) {
    if (!contentKeyPattern.test(policy.key)) {
      throw new Error(`turbulence policy ${policy.key} has an invalid key`);
    }
    if (turbulencePolicies.has(policy.key)) {
      throw new Error(`duplicate turbulence policy ${policy.key}`);
    }
    if (
      policy.type !== "STATIC_CONTENT_VALUE" || policy.staticValue <= 0 ||
      policy.floor <= 0 ||
      policy.floor > policy.staticValue ||
      policy.roundingMode !== "ROUND_HALF_AWAY_FROM_ZERO"
    ) {
      throw new Error(`invalid turbulence policy ${policy.key}`);
    }
    turbulencePolicies.set(policy.key, policy);
  }
  return { sectorDefinitions, turbulencePolicies };
}

function validateBasketCalculationConfiguration(
  baskets: Map<string, BasketMapping>,
  definitions: Definition[],
  sectors: Map<string, SectorDefinition>,
  policies: Map<string, ExpectedTurbulencePolicy>,
): void {
  if (sectors.size === 0) {
    for (const entry of baskets.values()) {
      if (
        entry.sector !== "" || entry.minimumCoveredWeight !== 0 ||
        entry.expectedTurbulencePolicyKey !== "" ||
        entry.normalizationCap !== 0 || entry.benchmarkEligible
      ) {
        throw new Error(
          `basket ${entry.key} has calculation configuration without sector definitions`,
        );
      }
    }
    return;
  }
  const eligibleCounts = new Map<string, number>();
  for (const entry of baskets.values()) {
    const configured = entry.sector !== "" ||
      entry.minimumCoveredWeight !== 0 ||
      entry.expectedTurbulencePolicyKey !== "" ||
      entry.normalizationCap !== 0 || entry.benchmarkEligible;
    if (!configured) {
      continue;
    }
    if (
      !sectors.has(entry.sector) || entry.minimumCoveredWeight <= 0 ||
      entry.minimumCoveredWeight > WeightScale ||
      !contentKeyPattern.test(entry.expectedTurbulencePolicyKey) ||
      entry.normalizationCap <= 0
    ) {
      throw new Error(
        `basket ${entry.key} has invalid calculation configuration`,
      );
    }
    if (!policies.has(entry.expectedTurbulencePolicyKey)) {
      throw new Error(
        `basket ${entry.key} references unknown turbulence policy ${entry.expectedTurbulencePolicyKey}`,
      );
    }
    if (entry.benchmarkEligible) {
      eligibleCounts.set(
        entry.sector,
        (eligibleCounts.get(entry.sector) ?? 0) + 1,
      );
    }
  }
  for (const [sectorKey, definition] of sectors) {
    const count = eligibleCounts.get(sectorKey) ?? 0;
    if (count < definition.minimumEligibleBaskets) {
      throw new Error(
        `sector ${sectorKey} has ${count} eligible baskets, want at least ${definition.minimumEligibleBaskets}`,
      );
    }
  }
  for (const definition of definitions) {
    const entry = baskets.get(definition.basketMappingKey)!;
    if (entry.sector !== "" && entry.sector !== definition.sector) {
      throw new Error(
        `Keeper ${definition.key} sector ${definition.sector} does not match basket sector ${entry.sector}`,
      );
    }
  }
}

function tokenSet(...values: string[]): Set<string> {
  return new Set(values);
}

function parameterSet(...parameters: string[]): Set<string> {
  return new Set(parameters);
}
