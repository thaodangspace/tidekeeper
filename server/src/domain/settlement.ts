/**
 * Pure deterministic daily settlement engine.
 *
 * The engine accepts only immutable snapshots. It has no KV, clock, network,
 * or repository dependency, which makes replay and idempotent cron execution
 * straightforward.
 */

import { clamp, divide, ScoreScale } from "../market/precision.ts";

export interface SettlementKeeperInput {
  id: string;
  definitionKey: string;
  sector: string;
  role: string;
  rarity: string;
}

export interface SettlementMarketInput {
  /** Sector scores in ScoreScale units: 10_000 = 1.0000. */
  sectorScores: Record<string, number>;
}

export interface SettlementModifierInput {
  id: string;
  scoreBonusBps: number;
  hullDelta: number;
  suppliesDelta: number;
}

export interface SettlementStrategyInput {
  id: string | null;
  scoreBonusBps: number;
  hullDelta: number;
  suppliesDelta: number;
}

export interface SettlementRules {
  version: number;
  passScoreBps: number;
  damageScoreBps: number;
  diversityBonusBps: number;
  successHullDelta: number;
  deficitHullDelta: number;
  successSuppliesDelta: number;
  deficitSuppliesDelta: number;
}

export interface SettlementInput {
  dayNumber: number;
  keepers: SettlementKeeperInput[];
  market: SettlementMarketInput;
  modifier: SettlementModifierInput;
  strategy: SettlementStrategyInput;
  rules: SettlementRules;
}

export interface SettlementBreakdown {
  keeperScores: Array<{
    keeperId: string;
    definitionKey: string;
    sector: string;
    scoreBps: number;
  }>;
  averageScoreBps: number;
  diversityBonusBps: number;
  modifierBonusBps: number;
  strategyBonusBps: number;
}

export interface SettlementReward {
  type: "SUPPLIES";
  amount: number;
  reason: string;
}

export interface SettlementOutput {
  dayNumber: number;
  ruleVersion: number;
  scoreBps: number;
  score: string;
  hullDelta: number;
  suppliesDelta: number;
  reward: SettlementReward | null;
  breakdown: SettlementBreakdown;
}

/** Calculates a daily result using fixed-point integer arithmetic. */
export function settle(input: SettlementInput): SettlementOutput {
  validateInput(input);
  const keepers = [...input.keepers].sort((left, right) =>
    left.id.localeCompare(right.id)
  );
  const keeperScores = keepers.map((keeper) => {
    const scoreBps = input.market.sectorScores[keeper.sector];
    if (scoreBps === undefined) {
      throw new Error(`market data is incomplete for sector ${keeper.sector}`);
    }
    return {
      keeperId: keeper.id,
      definitionKey: keeper.definitionKey,
      sector: keeper.sector,
      scoreBps,
    };
  });
  const averageScoreBps = divide(
    keeperScores.reduce((total, component) => total + component.scoreBps, 0),
    keeperScores.length,
  );
  const distinctSectors = new Set(
    keeperScores.map((component) => component.sector),
  );
  const diversityBonusBps = distinctSectors.size > 1
    ? input.rules.diversityBonusBps
    : 0;
  const scoreBps = clamp(
    averageScoreBps +
      diversityBonusBps +
      input.modifier.scoreBonusBps +
      input.strategy.scoreBonusBps,
    0,
    ScoreScale,
  );
  const passed = scoreBps >= input.rules.passScoreBps;
  const damaged = scoreBps < input.rules.damageScoreBps;
  const hullDelta =
    (damaged ? input.rules.deficitHullDelta : input.rules.successHullDelta) +
    input.modifier.hullDelta + input.strategy.hullDelta;
  const suppliesDelta =
    (passed
      ? input.rules.successSuppliesDelta
      : input.rules.deficitSuppliesDelta) +
    input.modifier.suppliesDelta + input.strategy.suppliesDelta;
  const reward = passed && suppliesDelta > 0
    ? {
      type: "SUPPLIES" as const,
      amount: suppliesDelta,
      reason: "DAILY_SCORE",
    }
    : null;

  return {
    dayNumber: input.dayNumber,
    ruleVersion: input.rules.version,
    scoreBps,
    score: formatScore(scoreBps),
    hullDelta,
    suppliesDelta,
    reward,
    breakdown: {
      keeperScores,
      averageScoreBps,
      diversityBonusBps,
      modifierBonusBps: input.modifier.scoreBonusBps,
      strategyBonusBps: input.strategy.scoreBonusBps,
    },
  };
}

function validateInput(input: SettlementInput): void {
  if (!Number.isInteger(input.dayNumber) || input.dayNumber < 1) {
    throw new Error("settlement day number must be positive");
  }
  if (input.keepers.length === 0) {
    throw new Error("settlement requires a locked Keeper lineup");
  }
  if (!Number.isInteger(input.rules.version) || input.rules.version < 1) {
    throw new Error("settlement rule version must be positive");
  }
  if (
    input.rules.damageScoreBps < 0 ||
    input.rules.damageScoreBps > input.rules.passScoreBps ||
    input.rules.passScoreBps > ScoreScale
  ) {
    throw new Error("settlement score thresholds are invalid");
  }
  for (const score of Object.values(input.market.sectorScores)) {
    if (!Number.isInteger(score) || score < 0 || score > ScoreScale) {
      throw new Error("market sector scores must be in ScoreScale range");
    }
  }
}

function formatScore(scoreBps: number): string {
  const whole = Math.floor(scoreBps / ScoreScale);
  const fraction = String(scoreBps % ScoreScale).padStart(4, "0");
  return `${whole}.${fraction}`;
}
