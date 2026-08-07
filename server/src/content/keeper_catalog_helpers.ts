/** Shared constructors used to build embedded Keeper catalog content. */

import type {
  BasketComponent,
  BasketMapping,
  Definition,
  MarketAsset,
} from "./keeper_model.ts";

export function asset(key: string, symbol: string): MarketAsset {
  return { key, symbol };
}

export function basket(
  key: string,
  ...components: BasketComponent[]
): BasketMapping {
  return {
    key,
    sector: "",
    minimumCoveredWeight: 0,
    expectedTurbulencePolicyKey: "",
    normalizationCap: 0,
    benchmarkEligible: false,
    components,
  };
}

export function component(
  assetKey: string,
  percentage: number,
): BasketComponent {
  return { assetKey, weight: percentage * 1_000_000 };
}

export function upgradeTree(
  firstKey: string,
  firstName: string,
  firstEffects: string,
  secondKey: string,
  secondName: string,
  secondEffects: string,
): string {
  return JSON.stringify({
    rootNodeKey: "base",
    nodes: [
      { key: "base", name: "Base", next: [firstKey, secondKey], effects: {} },
      {
        key: firstKey,
        name: firstName,
        next: [],
        effects: JSON.parse(firstEffects),
      },
      {
        key: secondKey,
        name: secondName,
        next: [],
        effects: JSON.parse(secondEffects),
      },
    ],
  });
}

export function definition(
  key: string,
  name: string,
  currentKey: string,
  currentName: string,
  sector: string,
  role: string,
  rarity: string,
  baseRisk: string,
  ruleKey: string,
  ruleConfig: string,
  basketKey: string,
  upgrades: string,
): Definition {
  return {
    key,
    name,
    currentKey,
    currentName,
    sector,
    role,
    rarity,
    baseRisk,
    expectedTurbulenceBPS: null,
    passiveRuleKey: ruleKey,
    passiveRuleConfig: JSON.stringify(JSON.parse(ruleConfig)),
    upgradeTree: upgrades,
    basketMappingKey: basketKey,
  };
}
