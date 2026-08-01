/**
 * Standard voyage definition content, ported from the MVP seed in migration
 * 000006. The Keeper archetype references resolve against the embedded CatalogV1
 * universe (content version 1), matching the migration's seeded rows.
 */

import type {
  VoyageDefinition,
  VoyageDefinitionInitialOffer,
  VoyageDefinitionStarterKeeper,
} from "../domain/types.ts";
import { catalogV1 } from "./keeper_catalog_v1.ts";

export const defaultDefinitionKey = "standard";

export interface ProjectionKeeper {
  id: string;
  definitionId: string;
  name: string;
  level: number;
  rarity: string;
  role: string;
  sector: string;
  passiveSummary: string;
  artworkUrl: string | null;
}

export interface InitialProjectionInput {
  fleetSlotCount: number;
  instances: {
    publicId: string;
    key: string;
    name: string;
    sector: string;
    role: string;
    rarity: string;
  }[];
  offers: {
    keeperKey: string;
    name: string;
    sector: string;
    role: string;
    rarity: string;
    cost: number;
  }[];
}

export const standardVoyageDefinition: VoyageDefinition = {
  id: "00000000-0000-0000-0000-000000000002",
  definitionKey: "standard",
  version: 1,
  status: "PUBLISHED",
  durationDays: 7,
  startingHull: 100,
  startingSupplies: 10,
  fleetSlotCount: 3,
  initialPhase: "PREPARATION",
  launchPresentation: {
    modifier: {
      id: "mod_default",
      name: "Standard Conditions",
      description: "Default voyage conditions.",
    },
    objective: {
      id: "obj_default",
      name: "Navigate",
      description: "Complete the daily voyage.",
      progressLabel: null,
      rewardLabel: null,
    },
    signals: [],
    lineup: {
      lockedAt: null,
      maxSlots: 3,
      slots: [{ index: 0, keeper: null }, { index: 1, keeper: null }, {
        index: 2,
        keeper: null,
      }],
      synergies: [],
      warnings: [{
        code: "EMPTY_SLOT",
        severity: "WARNING",
        message: "All slots are empty.",
      }],
    },
    inventory: [],
    shop: {
      offers: [{
        id: "crest_sovereign_offer",
        keeper: {
          id: "crest_sovereign_preview",
          definitionId: "crest_sovereign",
          name: "Crest Sovereign",
          level: 1,
          rarity: "COMMON",
          role: "VANGUARD",
          sector: "CREST",
          passiveSummary: "Commands the Crest sector with authority.",
          artworkUrl: null,
        },
        cost: 5,
        available: true,
        synergyHint: null,
      }],
      refreshAt: null,
      rerollCost: 2,
      rerollIndex: 0,
    },
    strategies: [],
    selectedStrategyId: null,
    pendingRewardCount: 0,
  },
  checksum: "7374616e64617264000000000000000000000000000000000000000000000000",
};

const starter: VoyageDefinitionStarterKeeper = {
  keeperDefinitionVersionId: "00000000-0000-0000-0000-000000000021",
  keeperKey: "crest_sovereign",
  name: "Crest Sovereign",
  currentName: "Sovereign Current",
  sectorKey: "CREST",
  roleKey: "VANGUARD",
  rarityKey: "COMMON",
  passiveRuleKey: "CROWN_OF_THE_TIDE",
  passiveRuleConfig:
    `{"schemaVersion":1,"parameters":{"minimumOutperformingComponents":3,"globalRelativeScoreBonusBps":2000,"broadDeclineDepthPenaltyReductionBps":1500}}`,
  rootUpgradeNodeKey: "base",
};

const initialOffer: VoyageDefinitionInitialOffer = {
  keeperDefinitionVersionId: "00000000-0000-0000-0000-000000000021",
  keeperKey: "crest_sovereign",
  name: "Crest Sovereign",
  currentName: "Sovereign Current",
  sectorKey: "CREST",
  roleKey: "VANGUARD",
  rarityKey: "COMMON",
  passiveRuleKey: "CROWN_OF_THE_TIDE",
  passiveRuleConfig:
    `{"schemaVersion":1,"parameters":{"minimumOutperformingComponents":3,"globalRelativeScoreBonusBps":2000,"broadDeclineDepthPenaltyReductionBps":1500}}`,
  cost: 5,
};

/** Returns the published standard voyage definition. */
export function getPublishedVoyageDefinition(): VoyageDefinition {
  return standardVoyageDefinition;
}

/** Returns the standard definition's starter Keepers in position order. */
export function getVoyageDefinitionStarterKeepers(
  definitionId: string,
): VoyageDefinitionStarterKeeper[] {
  if (definitionId !== standardVoyageDefinition.id) {
    return [];
  }
  return [starter];
}

/** Returns the standard definition's initial shop offers in position order. */
export function getVoyageDefinitionInitialOffers(
  definitionId: string,
): VoyageDefinitionInitialOffer[] {
  if (definitionId !== standardVoyageDefinition.id) {
    return [];
  }
  return [initialOffer];
}

/** Returns the root upgrade node key for a Keeper archetype, or null. */
export function rootUpgradeNodeKey(definitionKey: string): string | null {
  const definitions = catalogV1().definitions;
  const found = definitions.find((definition) =>
    definition.key === definitionKey
  );
  if (!found) {
    return null;
  }
  const tree = JSON.parse(found.upgradeTree) as { rootNodeKey: string };
  return tree.rootNodeKey;
}

function defaultPassiveSummary(keeperKey: string): string {
  switch (keeperKey) {
    case "crest_sovereign":
      return "Commands the Crest sector with authority.";
    case "harbor_warden":
      return "Guards the Harbor sector vigilantly.";
    case "current_weaver":
      return "Weaves through Current sector currents.";
    default:
      return "Adapts to changing market conditions.";
  }
}

function makeKeeperMap(
  id: string,
  definitionId: string,
  name: string,
  sector: string,
  role: string,
  rarity: string,
): ProjectionKeeper {
  return {
    id,
    definitionId,
    name,
    level: 1,
    rarity,
    role,
    sector,
    passiveSummary: defaultPassiveSummary(definitionId),
    artworkUrl: null,
  };
}

/**
 * Builds the initial schema-1 daily-context projection exactly as the Go
 * voyage service does at creation time.
 */
export function buildInitialProjection(input: InitialProjectionInput): unknown {
  const slots = [];
  for (let i = 0; i < input.fleetSlotCount; i++) {
    slots.push({ index: i, keeper: null });
  }

  const warnings: { code: string; severity: string; message: string }[] = [];
  if (input.fleetSlotCount > 0) {
    warnings.push({
      code: "EMPTY_SLOT",
      severity: "WARNING",
      message: "All slots are empty.",
    });
  }

  const inventory = input.instances.map((inst) =>
    makeKeeperMap(
      inst.publicId,
      inst.key,
      inst.name,
      inst.sector,
      inst.role,
      inst.rarity,
    )
  );

  const shopOffers = input.offers.map((offer) => ({
    id: `${offer.keeperKey}_offer`,
    keeper: makeKeeperMap(
      `${offer.keeperKey}_preview`,
      offer.keeperKey,
      offer.name,
      offer.sector,
      offer.role,
      offer.rarity,
    ),
    cost: offer.cost,
    available: true,
    synergyHint: null,
  }));

  return {
    modifier: {
      id: "mod_default",
      name: "Standard Conditions",
      description: "Default voyage conditions.",
    },
    objective: {
      id: "obj_default",
      name: "Navigate",
      description: "Complete the daily voyage.",
      progressLabel: null,
      rewardLabel: null,
    },
    signals: [],
    lineup: {
      lockedAt: null,
      maxSlots: input.fleetSlotCount,
      slots,
      synergies: [],
      warnings,
    },
    inventory,
    shop: {
      offers: shopOffers,
      refreshAt: null,
      rerollCost: 2,
      rerollIndex: 0,
    },
    strategies: [],
    selectedStrategyId: null,
    pendingRewardCount: 0,
  };
}
