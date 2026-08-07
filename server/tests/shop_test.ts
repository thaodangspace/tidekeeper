/** Integration tests for atomic shop recruitment. */

import { assertEquals, assertRejects } from "@std/assert";
import type {
  PlayerDailyContextView,
  PlayerDailyState,
  PlayerKeeperUnlock,
  Voyage,
} from "../src/domain/types.ts";
import { DailyRepository } from "../src/repositories/daily_repository.ts";
import {
  playerUnlockKey,
  Store,
  voyageActiveKey,
  voyageKey,
} from "../src/repositories/kv.ts";
import { VoyageRepository } from "../src/repositories/voyage_repository.ts";
import { ShopService } from "../src/services/shop_service.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

const now = new Date("2026-08-01T12:00:00Z");

Deno.test("shop: purchase atomically spends capital and adds Keeper", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const daily = new DailyRepository(store);
    const voyages = new VoyageRepository(store);
    const shop = new ShopService(store, daily, voyages);
    const voyage: Voyage = {
      id: "voyage-1",
      publicId: "voy_1",
      playerId: "player-1",
      status: "ACTIVE",
      definitionKey: "standard",
      definitionVersion: 1,
      currentDayNumber: 1,
      fundHealth: 100,
      maxFundHealth: 100,
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
      phase: "PREPARATION",
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
        lineup: {
          lockedAt: null,
          maxSlots: 1,
          slots: [{ index: 0, keeper: null }],
          synergies: [],
          warnings: [],
        },
        inventory: [],
        shop: {
          offers: [{
            id: "harbor_warden_offer",
            keeper: {
              id: "harbor_warden_preview",
              definitionId: "harbor_warden",
              name: "Harbor Warden",
              level: 1,
              rarity: "COMMON",
              role: "WARDEN",
              sector: "HARBOR",
              passiveSummary: "Guards the Harbor sector vigilantly.",
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
      },
      validatedAt: now.toISOString(),
    };
    await store.set(voyageActiveKey(voyage.playerId), voyage.id);
    await store.set(voyageKey(voyage.id), voyage);
    await daily.upsertPlayerDailyState(state);
    await daily.upsertPlayerDailyView(view);

    const result = await shop.purchase("player-1", "harbor_warden_offer", {
      expectedVersion: 1,
    }, now);
    assertEquals(result.capital, 5);
    assertEquals(result.version, 2);
    assertEquals(result.keeper.definitionId, "harbor_warden");
    assertEquals(
      (await store.get<Voyage>(voyageKey(voyage.id)))?.capital,
      5,
    );
    assertEquals(
      (await store.get<PlayerKeeperUnlock>(
        playerUnlockKey("player-1", "harbor_warden"),
      ))?.unlockSource,
      "SHOP",
    );
  } finally {
    kv.close();
  }
});

Deno.test("shop: stale purchase does not mutate state", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const daily = new DailyRepository(store);
    const voyages = new VoyageRepository(store);
    const shop = new ShopService(store, daily, voyages);
    await store.set(voyageActiveKey("player-1"), "missing");
    await assertRejects(
      () => shop.purchase("player-1", "offer", { expectedVersion: 1 }, now),
      TidekeepersError,
    );
  } finally {
    kv.close();
  }
});
