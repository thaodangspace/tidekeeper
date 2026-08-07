/** VoyageService lifecycle tests against an in-memory Deno KV instance. */

import { assertEquals, assertStrictEquals } from "@std/assert";
import { playerKey, Store } from "../src/repositories/kv.ts";
import { VoyageRepository } from "../src/repositories/voyage_repository.ts";
import { DailyRepository } from "../src/repositories/daily_repository.ts";
import { IdempotencyRepository } from "../src/repositories/idempotency_repository.ts";
import { VoyageService } from "../src/services/voyage_service.ts";
import { DailyService } from "../src/services/daily_service.ts";
import { Player } from "../src/domain/types.ts";
import { dayKeyInTimezone } from "../src/utils/time.ts";
import { newId } from "../src/utils/ids.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

interface TestHarness {
  kv: Deno.Kv;
  voyages: VoyageService;
  daily: DailyService;
  playerId: string;
  now: Date;
}

const now = new Date("2026-08-01T12:00:00Z");

async function setup(): Promise<TestHarness> {
  const kv = await Deno.openKv(":memory:");
  const store = new Store(kv);
  const playerId = newId();
  const player: Player = {
    id: playerId,
    publicId: "plr_test",
    displayName: "Test Player",
    username: "test_player",
    onboardingCompleted: false,
    locale: "en-US",
    timezone: "UTC",
    currentVoyageId: null,
    createdAt: now.toISOString(),
    updatedAt: now.toISOString(),
  };
  await store.set(playerKey(playerId), player);

  const voyages = new VoyageRepository(store);
  const daily = new DailyRepository(store);

  const tide = {
    id: newId(),
    dayKey: dayKeyInTimezone(now, "UTC"),
    sequenceNumber: 0,
    phase: "REGULAR",
    lockAt: now.toISOString(),
    settleAfter: now.toISOString(),
    contentVersion: 1,
    createdAt: now.toISOString(),
    updatedAt: now.toISOString(),
  };
  await daily.ensureDailyTide(tide);

  const idempotency = new IdempotencyRepository(store);
  return {
    kv,
    voyages: new VoyageService(store, voyages, daily, idempotency, "UTC"),
    daily: new DailyService(daily, voyages),
    playerId,
    now,
  };
}

Deno.test("voyage: create returns standard definition and persists state", async () => {
  const h = await setup();
  try {
    const resp = await h.voyages.create(h.playerId, "key-create-1", h.now);
    assertStrictEquals(resp.status, "ACTIVE");
    assertStrictEquals(resp.definitionKey, "standard");
    assertStrictEquals(resp.definitionVersion, 1);
    assertStrictEquals(resp.dayNumber, 1);
    assertStrictEquals(resp.fundHealth, 100);
    assertStrictEquals(resp.maxFundHealth, 100);
    assertStrictEquals(resp.capital, 10);
    assertStrictEquals(resp.score, "0.0000");
    assertStrictEquals(resp.startedAt, h.now.toISOString());
    assertStrictEquals(resp.completedAt, null);
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: create is idempotent for the same key", async () => {
  const h = await setup();
  try {
    const first = await h.voyages.create(h.playerId, "key-idem-1", h.now);
    const second = await h.voyages.create(h.playerId, "key-idem-1", h.now);
    assertEquals(second, first);
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: second active voyage conflicts", async () => {
  const h = await setup();
  try {
    await h.voyages.create(h.playerId, "key-conf-1", h.now);
    await assertRejectsCode(
      () => h.voyages.create(h.playerId, "key-conf-2", h.now),
      "ACTIVE_VOYAGE_EXISTS",
    );
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: getCurrent resolves active voyage", async () => {
  const h = await setup();
  try {
    const created = await h.voyages.create(h.playerId, "key-cur-1", h.now);
    const current = await h.voyages.getCurrent(h.playerId);
    assertEquals(current, created);
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: getCurrent without voyage is not found", async () => {
  const h = await setup();
  try {
    await assertRejectsCode(
      () => h.voyages.getCurrent(h.playerId),
      "NO_ACTIVE_VOYAGE",
    );
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: getById resolves by public id and scopes ownership", async () => {
  const h = await setup();
  try {
    const created = await h.voyages.create(h.playerId, "key-gid-1", h.now);
    const byId = await h.voyages.getById(h.playerId, created.id);
    assertEquals(byId, created);
    await assertRejectsCode(
      () => h.voyages.getById("someone-else", created.id),
      "VOYAGE_NOT_FOUND",
    );
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: abandon transitions to ABANDONED and clears active", async () => {
  const h = await setup();
  try {
    const created = await h.voyages.create(h.playerId, "key-ab-1", h.now);
    const abandoned = await h.voyages.abandon(
      h.playerId,
      created.id,
      "key-abx-1",
      h.now,
    );
    assertStrictEquals(abandoned.status, "ABANDONED");
    assertStrictEquals(abandoned.rowVersion, 2);
    assertStrictEquals(abandoned.completedAt !== null, true);
    // A new voyage can now be created.
    const fresh = await h.voyages.create(h.playerId, "key-ab-2", h.now);
    assertStrictEquals(fresh.status, "ACTIVE");
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: abandon is idempotent", async () => {
  const h = await setup();
  try {
    const created = await h.voyages.create(h.playerId, "key-abid-1", h.now);
    const first = await h.voyages.abandon(
      h.playerId,
      created.id,
      "key-abx-1",
      h.now,
    );
    const second = await h.voyages.abandon(
      h.playerId,
      created.id,
      "key-abx-1",
      h.now,
    );
    assertEquals(second, first);
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: history exposes created and abandoned events", async () => {
  const h = await setup();
  try {
    const created = await h.voyages.create(h.playerId, "key-hist-1", h.now);
    await h.voyages.abandon(h.playerId, created.id, "key-histx-1", h.now);
    const history = await h.voyages.getHistory(h.playerId, created.id);
    assertEquals(
      [...history.events.map((e) => e.eventType)].sort(),
      ["ABANDONED", "CREATED"],
    );
    assertStrictEquals(history.voyage.status, "ABANDONED");
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: keepers list starter keepers", async () => {
  const h = await setup();
  try {
    const created = await h.voyages.create(h.playerId, "key-k-1", h.now);
    const keepers = await h.voyages.getActiveVoyageKeepers(h.playerId);
    assertStrictEquals(keepers.voyageId, created.id);
    assertStrictEquals(keepers.keepers.length, 1);
    const keeper = keepers.keepers[0]!;
    assertStrictEquals(keeper.definitionKey, "crest_sovereign");
    assertStrictEquals(keeper.level, 1);
    assertStrictEquals(keeper.acquiredDay, 1);
    assertStrictEquals(keeper.acquiredSource, "STARTER");
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: create seeds initial daily context for day 1", async () => {
  const h = await setup();
  try {
    await h.voyages.create(h.playerId, "key-ctx-1", h.now);
    const context = await h.daily.getCurrentDailyContext(h.playerId, h.now);
    assertStrictEquals(context.voyage.dayNumber, 1);
    assertStrictEquals(context.daily.phase, "PREPARATION");
    assertStrictEquals(context.daily.version, 1);
    assertStrictEquals(context.daily.modifier.id, "mod_default");
    assertStrictEquals(context.daily.objective.id, "obj_default");
    assertStrictEquals(context.daily.lineup.maxSlots, 3);
    assertStrictEquals(context.daily.shop.offers.length, 1);
    assertStrictEquals(context.daily.shop.offers[0]!.cost, 5);
    assertStrictEquals(context.daily.shop.rerollCost, 2);
    assertStrictEquals(context.daily.inventory.length, 1);
  } finally {
    h.kv.close();
  }
});

Deno.test("voyage: create without a daily tide is unavailable", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const playerId = newId();
    await store.set(playerKey(playerId), {
      id: playerId,
      publicId: "plr_test",
      displayName: "Test Player",
      username: "test_player",
      onboardingCompleted: false,
      locale: "en-US",
      timezone: "UTC",
      currentVoyageId: null,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    });
    const voyages = new VoyageRepository(store);
    const daily = new DailyRepository(store);
    const idempotency = new IdempotencyRepository(store);
    const service = new VoyageService(
      store,
      voyages,
      daily,
      idempotency,
      "UTC",
    );
    await assertRejectsCode(
      () =>
        service.create(
          playerId,
          "key-tide-1",
          new Date("2026-08-02T12:00:00Z"),
        ),
      "VOYAGE_INITIALIZATION_UNAVAILABLE",
    );
  } finally {
    kv.close();
  }
});

async function assertRejectsCode(
  fn: () => Promise<unknown>,
  code: string,
): Promise<void> {
  try {
    await fn();
  } catch (err) {
    if (err instanceof TidekeepersError && err.code === code) {
      return;
    }
    throw new Error(
      `expected TidekeepersError code ${code}, got ${String(err)}`,
    );
  }
  throw new Error(
    `expected TidekeepersError code ${code}, but no error thrown`,
  );
}
