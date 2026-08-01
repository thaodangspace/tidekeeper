import { assertEquals, assertThrows } from "@std/assert";
import { settle, type SettlementInput } from "../src/domain/settlement.ts";

function input(): SettlementInput {
  return {
    dayNumber: 3,
    keepers: [
      {
        id: "kpr-b",
        definitionKey: "harbor_warden",
        sector: "HARBOR",
        role: "WARDEN",
        rarity: "COMMON",
      },
      {
        id: "kpr-a",
        definitionKey: "crest_sovereign",
        sector: "CREST",
        role: "VANGUARD",
        rarity: "COMMON",
      },
    ],
    market: { sectorScores: { CREST: 6_000, HARBOR: 8_000 } },
    modifier: {
      id: "mod-default",
      scoreBonusBps: 0,
      hullDelta: 0,
      suppliesDelta: 0,
    },
    strategy: {
      id: "balanced",
      scoreBonusBps: 0,
      hullDelta: 0,
      suppliesDelta: 0,
    },
    rules: {
      version: 1,
      passScoreBps: 7_000,
      damageScoreBps: 3_000,
      diversityBonusBps: 500,
      successHullDelta: 1,
      deficitHullDelta: -2,
      successSuppliesDelta: 2,
      deficitSuppliesDelta: 0,
    },
  };
}

Deno.test("settlement: deterministic fixed-point score and reward", () => {
  const result = settle(input());
  assertEquals(result.scoreBps, 7_500);
  assertEquals(result.score, "0.7500");
  assertEquals(result.hullDelta, 1);
  assertEquals(result.suppliesDelta, 2);
  assertEquals(result.reward, {
    type: "SUPPLIES",
    amount: 2,
    reason: "DAILY_SCORE",
  });
  assertEquals(result.breakdown.keeperScores.map((entry) => entry.keeperId), [
    "kpr-a",
    "kpr-b",
  ]);
});

Deno.test("settlement: missing market data fails closed", () => {
  const value = input();
  delete value.market.sectorScores.HARBOR;
  assertThrows(() => settle(value), Error, "market data is incomplete");
});

Deno.test("settlement: score is clamped and deficit damages hull", () => {
  const value = input();
  value.keepers = [value.keepers[0]!];
  value.market.sectorScores = { HARBOR: 0 };
  value.modifier.scoreBonusBps = -10_000;
  const result = settle(value);
  assertEquals(result.score, "0.0000");
  assertEquals(result.hullDelta, -2);
  assertEquals(result.reward, null);
});
