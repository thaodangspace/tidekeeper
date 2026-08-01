/** Approved Keeper catalog v2 (ported verbatim from keeper/catalog_v2.go). */

import type { CatalogRelease } from "./keeper_model.ts";
import { CalculationScale } from "./keeper_model.ts";
import { catalogV1 } from "./keeper_catalog_v1.ts";
import {
  basket,
  component,
  definition,
  upgradeTree,
} from "./keeper_catalog_helpers.ts";

/**
 * Extends the immutable v1 roster with the four-sector calculation
 * configuration and one additional benchmark-eligible Keeper basket per sector.
 */
export function catalogV2(): CatalogRelease {
  const release = catalogV1();
  release.version = 2;
  release.sectorDefinitions = [
    {
      sector: "CREST",
      benchmarkMethod: "EQUAL_WEIGHT",
      minimumEligibleBaskets: 2,
      relativeScale: 500_000,
      relativeBlendWeight: 700_000,
      rankBlendWeight: 300_000,
      scoreCap: CalculationScale,
    },
    {
      sector: "EMBER",
      benchmarkMethod: "EQUAL_WEIGHT",
      minimumEligibleBaskets: 2,
      relativeScale: 500_000,
      relativeBlendWeight: 700_000,
      rankBlendWeight: 300_000,
      scoreCap: CalculationScale,
    },
    {
      sector: "CURRENT",
      benchmarkMethod: "EQUAL_WEIGHT",
      minimumEligibleBaskets: 2,
      relativeScale: 500_000,
      relativeBlendWeight: 700_000,
      rankBlendWeight: 300_000,
      scoreCap: CalculationScale,
    },
    {
      sector: "HARBOR",
      benchmarkMethod: "EQUAL_WEIGHT",
      minimumEligibleBaskets: 2,
      relativeScale: 500_000,
      relativeBlendWeight: 700_000,
      rankBlendWeight: 300_000,
      scoreCap: CalculationScale,
    },
  ];
  release.turbulencePolicies = [
    {
      key: "crest_static",
      type: "STATIC_CONTENT_VALUE",
      staticValue: 4_000_000,
      floor: 250_000,
      roundingMode: "ROUND_HALF_AWAY_FROM_ZERO",
    },
    {
      key: "ember_static",
      type: "STATIC_CONTENT_VALUE",
      staticValue: 15_000_000,
      floor: 1_000_000,
      roundingMode: "ROUND_HALF_AWAY_FROM_ZERO",
    },
    {
      key: "current_static",
      type: "STATIC_CONTENT_VALUE",
      staticValue: 8_000_000,
      floor: 500_000,
      roundingMode: "ROUND_HALF_AWAY_FROM_ZERO",
    },
    {
      key: "harbor_static",
      type: "STATIC_CONTENT_VALUE",
      staticValue: 500_000,
      floor: 250_000,
      roundingMode: "ROUND_HALF_AWAY_FROM_ZERO",
    },
  ];

  release.baskets.push(
    basket(
      "ember_breakers",
      component("dogecoin", 35),
      component("shiba_inu", 25),
      component("pepe", 20),
      component("bonk", 20),
    ),
    basket(
      "current_networks",
      component("mantle", 35),
      component("polygon", 35),
      component("arbitrum", 30),
    ),
    basket(
      "harbor_reserve",
      component("tether", 40),
      component("usdc", 35),
      component("dai", 15),
      component("usds", 10),
    ),
  );

  const calculationConfig = new Map<
    string,
    { sector: string; coverage: number; policy: string }
  >([
    ["crest_large_cap", {
      sector: "CREST",
      coverage: 80_000_000,
      policy: "crest_static",
    }],
    ["second_current", {
      sector: "CREST",
      coverage: 80_000_000,
      policy: "crest_static",
    }],
    ["wild_momentum", {
      sector: "EMBER",
      coverage: 85_000_000,
      policy: "ember_static",
    }],
    ["ember_breakers", {
      sector: "EMBER",
      coverage: 85_000_000,
      policy: "ember_static",
    }],
    ["exchange_flow", {
      sector: "CURRENT",
      coverage: 80_000_000,
      policy: "current_static",
    }],
    ["current_networks", {
      sector: "CURRENT",
      coverage: 80_000_000,
      policy: "current_static",
    }],
    ["harbor_stability", {
      sector: "HARBOR",
      coverage: 90_000_000,
      policy: "harbor_static",
    }],
    ["harbor_reserve", {
      sector: "HARBOR",
      coverage: 90_000_000,
      policy: "harbor_static",
    }],
  ]);
  for (const index of release.baskets.keys()) {
    const entry = release.baskets[index]!;
    const config = calculationConfig.get(entry.key);
    if (!config) {
      continue;
    }
    entry.sector = config.sector;
    entry.minimumCoveredWeight = config.coverage;
    entry.expectedTurbulencePolicyKey = config.policy;
    entry.normalizationCap = 2_000_000;
    entry.benchmarkEligible = true;
  }

  release.definitions.push(
    definition(
      "ember_runner",
      "Ember Runner",
      "breaker_current",
      "Breaker Current",
      "EMBER",
      "TRICKSTER",
      "RARE",
      "VERY_HIGH",
      "WILD_SURGE",
      `{"schemaVersion":1,"parameters":{"positiveNormalizedPerformanceThresholdMicros":10000,"sectorRelativeScoreMultiplierBps":15000,"negativeNormalizedPerformanceThresholdMicros":-10000,"pressurePenaltyIncreaseBps":3000,"pressurePenaltyApplicationCap":null}}`,
      "ember_breakers",
      upgradeTree(
        "burning_tail",
        "Burning Tail",
        `{"sectorRelativeScoreMultiplierBps":17500}`,
        "last_laugh",
        "Last Laugh",
        `{"pressurePenaltyApplicationCap":1}`,
      ),
    ),
    definition(
      "current_architect",
      "Current Architect",
      "network_current",
      "Network Current",
      "CURRENT",
      "SUPPORT",
      "RARE",
      "MEDIUM",
      "FLOW_TRANSFER",
      `{"schemaVersion":1,"parameters":{"minimumAdditionalOutperformingSectors":1,"diversificationScoreBonusBps":1500,"volumeTrendBonusBps":null}}`,
      "current_networks",
      upgradeTree(
        "cross_current",
        "Cross Current",
        `{"exchangeCurrentRequired":false,"minimumOutperformingSectors":2}`,
        "deep_liquidity",
        "Deep Liquidity",
        `{"volumeTrendBonusBps":null}`,
      ),
    ),
    definition(
      "harbor_sentinel",
      "Harbor Sentinel",
      "reserve_current",
      "Reserve Current",
      "HARBOR",
      "WARDEN",
      "RARE",
      "VERY_LOW",
      "UNBROKEN_PEG",
      `{"schemaVersion":1,"parameters":{"depthPenaltyReductionBps":3500,"stabilityThreshold":null,"hullDamageReductionBps":null}}`,
      "harbor_reserve",
      upgradeTree(
        "deep_shelter",
        "Deep Shelter",
        `{"depthPenaltyReductionBps":4500}`,
        "safe_yield",
        "Safe Yield",
        `{"riskAdjustedScoreBonusBps":null}`,
      ),
    ),
  );
  return release;
}
