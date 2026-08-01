import { assertEquals, assertRejects } from "@std/assert";
import type {
  PlayerDailyContextView,
  PlayerDailyState,
  Voyage,
} from "../src/domain/types.ts";
import { DailyRepository } from "../src/repositories/daily_repository.ts";
import {
  dailyTideIdKey,
  dailyTideKey,
  Store,
  voyageActiveKey,
  voyageKey,
} from "../src/repositories/kv.ts";
import { VoyageRepository } from "../src/repositories/voyage_repository.ts";
import { SettlementService } from "../src/services/settlement_service.ts";
import { TidekeepersError } from "../src/utils/errors.ts";
import { type SettlementInput } from "../src/domain/settlement.ts";

const now = new Date("2026-08-01T12:00:00Z");

function settlementInput(): SettlementInput {
  return {
    dayNumber: 1,
    keepers: [{
      id: "kpr-1",
      definitionKey: "crest_sovereign",
      sector: "CREST",
      role: "VANGUARD",
      rarity: "COMMON",
    }],
    market: { sectorScores: { CREST: 8_000 } },
    modifier: {
      id: "normal",
      scoreBonusBps: 0,
      hullDelta: 0,
      suppliesDelta: 0,
    },
    strategy: { id: null, scoreBonusBps: 0, hullDelta: 0, suppliesDelta: 0 },
    rules: {
      version: 1,
      passScoreBps: 7_000,
      damageScoreBps: 3_000,
      diversityBonusBps: 0,
      successHullDelta: 1,
      deficitHullDelta: -2,
      successSuppliesDelta: 2,
      deficitSuppliesDelta: 0,
    },
  };
}

Deno.test("settlement service: persists idempotent result and claims reward", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const daily = new DailyRepository(store);
    const voyages = new VoyageRepository(store);
    const service = new SettlementService(store, daily, voyages);
    const voyage: Voyage = {
      id: "voyage-1",
      publicId: "voy_1",
      playerId: "player-1",
      status: "ACTIVE",
      definitionKey: "standard",
      definitionVersion: 1,
      currentDayNumber: 1,
      fundHealth: 9,
      maxFundHealth: 10,
      capital: 10,
      score: "0.0000",
      rowVersion: 1,
      startedAt: now.toISOString(),
      completedAt: null,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
    const state: PlayerDailyState = {
      id: "state-1",
      playerId: voyage.playerId,
      voyageId: voyage.id,
      dailyTideId: "tide-1",
      dayNumber: 1,
      phase: "LOCKED",
      version: 1,
      selectedStrategyId: null,
      pendingRewardCount: 0,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
    const view: PlayerDailyContextView = {
      voyageId: voyage.id,
      dayNumber: 1,
      schemaVersion: 1,
      projection: {
        pendingRewardCount: 0,
        lineup: { maxSlots: 1 },
        shop: { offers: [] },
      },
      validatedAt: now.toISOString(),
    };
    await store.set(voyageActiveKey(voyage.playerId), voyage.id);
    await store.set(voyageKey(voyage.id), voyage);
    await store.set(dailyTideIdKey("tide-1"), "2026-08-01");
    await store.set(dailyTideKey("2026-08-01"), {
      id: "tide-1",
      dayKey: "2026-08-01",
      sequenceNumber: 1,
      phase: "REGULAR",
      lockAt: "2026-08-01T23:55:00Z",
      settleAfter: "2026-08-02T02:55:00Z",
      contentVersion: 4,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    });
    await store.set(dailyTideIdKey("tide-2"), "2026-08-02");
    await store.set(dailyTideKey("2026-08-02"), {
      id: "tide-2",
      dayKey: "2026-08-02",
      sequenceNumber: 2,
      phase: "REGULAR",
      lockAt: "2026-08-02T23:55:00Z",
      settleAfter: "2026-08-03T02:55:00Z",
      contentVersion: 4,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    });
    await daily.upsertPlayerDailyState(state);
    await daily.upsertPlayerDailyView(view);

    const result = await service.settleDay(voyage.id, settlementInput(), now);
    assertEquals(result.score, "0.8000");
    assertEquals(result.reward?.amount, 2);
    const replay = await service.settleDay(voyage.id, settlementInput(), now);
    assertEquals(replay.id, result.id);
    assertEquals((await store.get<Voyage>(voyageKey(voyage.id)))?.capital, 12);
    assertEquals(
      (await store.get<Voyage>(voyageKey(voyage.id)))?.fundHealth,
      10,
    );

    const claim = await service.claimCurrentReward("player-1", 2, now);
    assertEquals(claim.version, 3);
    assertEquals(claim.reward.amount, 2);
    const error = await assertRejects<TidekeepersError>(
      () => service.claimCurrentReward("player-1", 3, now),
      TidekeepersError,
    );
    assertEquals(error.code, "REWARD_ALREADY_CLAIMED");

    const advanced = await service.advanceDay("player-1", 3, now);
    assertEquals(advanced.dayNumber, 2);
    assertEquals(advanced.status, "ACTIVE");
    assertEquals(
      (await store.get<Voyage>(voyageKey(voyage.id)))?.currentDayNumber,
      2,
    );
  } finally {
    kv.close();
  }
});
