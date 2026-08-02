import { assertEquals } from "@std/assert";
import { completeReleaseV1 } from "../src/content/releases.ts";
import { ContentPublisher } from "../src/content/publisher.ts";
import { ContentCutoverService } from "../src/content/cutover_service.ts";
import { buildInitialProjection } from "../src/content/voyage_definition.ts";
import { KvCutoverRepository } from "../src/repositories/cutover_repository.ts";
import { ContentRepository } from "../src/repositories/content_repository.ts";
import {
  dailyTideKey,
  Store,
  voyageDailyStateKey,
  voyageDailyViewKey,
} from "../src/repositories/kv.ts";
import type {
  DailyTide,
  PlayerDailyContextView,
  PlayerDailyState,
} from "../src/domain/types.ts";

Deno.test("content cutover service: dry-run and apply use one KV persistence boundary", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    await new ContentPublisher(store).publish(completeReleaseV1(3));
    const tide: DailyTide = {
      id: "tide-legacy",
      dayKey: "2026-08-01",
      sequenceNumber: 1,
      phase: "REGULAR",
      lockAt: "2026-08-01T12:00:00Z",
      settleAfter: "2026-08-01T15:00:00Z",
      contentVersion: 1,
      createdAt: "2026-08-01T00:00:00Z",
      updatedAt: "2026-08-01T00:00:00Z",
    };
    const state: PlayerDailyState = {
      id: "state-legacy",
      playerId: "player-1",
      voyageId: "voyage-1",
      dailyTideId: tide.id,
      dayNumber: 1,
      phase: "PREPARATION",
      version: 1,
      selectedStrategyId: null,
      pendingRewardCount: 0,
      createdAt: tide.createdAt,
      updatedAt: tide.createdAt,
    };
    const view: PlayerDailyContextView = {
      voyageId: "voyage-1",
      dayNumber: 1,
      schemaVersion: 1,
      projection: buildInitialProjection({
        fleetSlotCount: 1,
        instances: [],
        offers: [],
      }),
      validatedAt: tide.createdAt,
    };
    await store.set(dailyTideKey(tide.dayKey), tide);
    await store.set(
      voyageDailyStateKey(state.voyageId, state.dayNumber),
      state,
    );
    await store.set(voyageDailyViewKey(view.voyageId, view.dayNumber), view);

    const service = new ContentCutoverService(
      new ContentRepository(store),
      new KvCutoverRepository(store),
    );
    const dryRun = await service.run({ sourceVersion: 1, targetVersion: 3 });
    assertEquals(dryRun.applied, false);
    assertEquals(dryRun.changedTideCount, 1);
    assertEquals(
      (await store.get<DailyTide>(dailyTideKey(tide.dayKey)))?.contentVersion,
      1,
    );

    const applied = await service.run({
      sourceVersion: 1,
      targetVersion: 3,
      apply: true,
    });
    assertEquals(applied.applied, true);
    const updatedTide = await store.get<DailyTide>(dailyTideKey(tide.dayKey));
    assertEquals(updatedTide?.contentVersion, 3);
    assertEquals(
      updatedTide?.modifierDefinitionVersionId,
      "standard_conditions",
    );
  } finally {
    kv.close();
  }
});
