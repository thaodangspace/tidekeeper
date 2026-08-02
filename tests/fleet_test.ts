/** Integration tests for KV-backed lineup edits and fleet locking. */

import { assertEquals, assertRejects } from "@std/assert";
import { DailyRepository } from "../src/repositories/daily_repository.ts";
import { Store, voyageActiveKey, voyageKey } from "../src/repositories/kv.ts";
import { VoyageRepository } from "../src/repositories/voyage_repository.ts";
import { FleetService } from "../src/services/fleet_service.ts";
import type {
  PlayerDailyContextView,
  PlayerDailyState,
  Voyage,
} from "../src/domain/types.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

const now = new Date("2026-08-01T12:00:00Z");

function seedProjection(): PlayerDailyContextView {
  const keeper = {
    id: "kpr_1",
    definitionId: "crest_sovereign",
    name: "Crest Sovereign",
    level: 1,
    rarity: "COMMON",
    role: "VANGUARD",
    sector: "CREST",
    passiveSummary: "Commands the Crest sector with authority.",
    artworkUrl: null,
  };
  return {
    voyageId: "voyage-1",
    dayNumber: 1,
    schemaVersion: 1,
    projection: {
      modifier: { id: "modifier", name: "Normal", description: "Normal." },
      objective: {
        id: "objective",
        name: "Navigate",
        description: "Navigate.",
        progressLabel: null,
        rewardLabel: null,
      },
      signals: [],
      lineup: {
        lockedAt: null,
        maxSlots: 2,
        slots: [{ index: 0, keeper: null }, { index: 1, keeper: null }],
        synergies: [],
        warnings: [{
          code: "EMPTY_SLOT",
          message: "All slots are empty.",
          severity: "WARNING",
        }],
      },
      inventory: [keeper],
      shop: { offers: [], refreshAt: null, rerollCost: 2, rerollIndex: 0 },
      strategies: [],
      selectedStrategyId: null,
      pendingRewardCount: 0,
    },
    validatedAt: now.toISOString(),
  };
}

function seedVoyage(): Voyage {
  return {
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
}

async function setup() {
  const kv = await Deno.openKv(":memory:");
  const store = new Store(kv);
  const daily = new DailyRepository(store);
  const voyages = new VoyageRepository(store);
  const fleet = new FleetService(daily, voyages);
  const voyage = seedVoyage();
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
  await store.set(voyageActiveKey(voyage.playerId), voyage.id);
  await store.set(voyageKey(voyage.id), voyage);
  await daily.upsertPlayerDailyState(state);
  await daily.upsertPlayerDailyView(seedProjection());
  return { kv, fleet };
}

Deno.test("fleet: lineup update is versioned and preserves inventory", async () => {
  const { kv, fleet } = await setup();
  try {
    const result = await fleet.updateLineup("player-1", {
      expectedVersion: 1,
      slots: [{ index: 0, keeperId: "kpr_1" }, { index: 1, keeperId: null }],
    }, now);
    assertEquals(result.version, 2);
    assertEquals(result.lineup.slots[0]?.keeper?.id, "kpr_1");
    assertEquals(result.lineup.slots[1]?.keeper, null);
  } finally {
    kv.close();
  }
});

Deno.test("fleet: lock transitions phase and rejects later edits", async () => {
  const { kv, fleet } = await setup();
  try {
    const result = await fleet.lockFleet("player-1", {
      expectedVersion: 1,
      slots: [{ index: 0, keeperId: "kpr_1" }, { index: 1, keeperId: null }],
    }, now);
    assertEquals(result.version, 2);
    assertEquals(result.lineup.lockedAt, now.toISOString());
    const error = await assertRejects<TidekeepersError>(
      () =>
        fleet.updateLineup("player-1", {
          expectedVersion: 2,
          slots: [{ index: 0, keeperId: "kpr_1" }, {
            index: 1,
            keeperId: null,
          }],
        }, now),
      TidekeepersError,
    );
    assertEquals(error.code, "INVALID_PHASE");
  } finally {
    kv.close();
  }
});

Deno.test("fleet: concurrent writes produce one version conflict", async () => {
  const { kv, fleet } = await setup();
  try {
    const results = await Promise.allSettled([
      fleet.updateLineup("player-1", {
        expectedVersion: 1,
        slots: [{ index: 0, keeperId: "kpr_1" }, { index: 1, keeperId: null }],
      }, now),
      fleet.updateLineup("player-1", {
        expectedVersion: 1,
        slots: [{ index: 0, keeperId: null }, { index: 1, keeperId: "kpr_1" }],
      }, now),
    ]);
    assertEquals(
      results.filter((result) => result.status === "fulfilled").length,
      1,
    );
    assertEquals(
      results.filter((result) =>
        result.status === "rejected" &&
        result.reason instanceof TidekeepersError &&
        result.reason.code === "VERSION_CONFLICT"
      ).length,
      1,
    );
  } finally {
    kv.close();
  }
});
