/** Deno KV persistence boundary for legacy content cutover plans. */

import type {
  DailyTide,
  PlayerDailyContextView,
  PlayerDailyState,
} from "../domain/types.ts";
import type {
  CutoverAvailability,
  CutoverProjection,
  CutoverSelection,
  CutoverTide,
  CutoverTideUpdate,
} from "../content/cutover.ts";
import { conflict } from "../utils/errors.ts";
import {
  dailyTideKey,
  dailyTideStrategyKey,
  type Store,
  voyageDailyStateKey,
  voyageDailyViewKey,
} from "./kv.ts";

export interface CutoverSnapshot {
  tides: CutoverTide[];
  projections: CutoverProjection[];
}

export interface CutoverPersistence {
  loadSnapshot(): Promise<CutoverSnapshot>;
  apply(
    updates: CutoverTideUpdate[],
    availability: CutoverAvailability[],
    selections: CutoverSelection[],
    now?: Date,
  ): Promise<void>;
}

export class KvCutoverRepository implements CutoverPersistence {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async loadSnapshot(): Promise<CutoverSnapshot> {
    const tides = await this.#store.listValues<DailyTide>(["daily_tide"]);
    const states: PlayerDailyState[] = [];
    for await (
      const entry of this.#store.list<PlayerDailyState>([
        "voyage_daily_state",
      ])
    ) {
      states.push(entry.value);
    }
    const views: PlayerDailyContextView[] = [];
    for await (
      const entry of this.#store.list<PlayerDailyContextView>([
        "voyage_daily_view",
      ])
    ) {
      views.push(entry.value);
    }
    const stateByDay = new Map(
      states.map((state) => [`${state.voyageId}:${state.dayNumber}`, state]),
    );
    const projections: CutoverProjection[] = [];
    for (const view of views) {
      const state = stateByDay.get(`${view.voyageId}:${view.dayNumber}`);
      if (!state) continue;
      projections.push({
        dailyTideId: state.dailyTideId,
        stateId: state.id,
        schemaVersion: view.schemaVersion,
        projection: view.projection,
      });
    }
    return {
      tides: tides.map((tide) => ({
        id: tide.id,
        contentVersion: tide.contentVersion,
        modifierDefinitionVersionId: tide.modifierDefinitionVersionId,
        objectiveDefinitionVersionId: tide.objectiveDefinitionVersionId,
        gameRuleSetVersionId: tide.gameRuleSetVersionId,
      })),
      projections,
    };
  }

  async apply(
    updates: CutoverTideUpdate[],
    availability: CutoverAvailability[],
    selections: CutoverSelection[],
    now = new Date(),
  ): Promise<void> {
    const tideEntries = await this.#readTidesWithVersions();
    const tideById = new Map(
      tideEntries.map((entry) => [entry.value.id, entry]),
    );
    const stateEntries = await this.#readStatesWithVersions();
    const stateById = new Map(
      stateEntries.map((entry) => [entry.value.id, entry]),
    );
    const atomic = this.#store.atomic();

    for (const update of updates) {
      const entry = tideById.get(update.id);
      if (!entry) {
        throw conflict(`cutover tide ${update.id} was not found`, {
          code: "CUTOVER_CONFLICT",
        });
      }
      if (
        entry.value.contentVersion === update.contentVersion &&
        entry.value.modifierDefinitionVersionId ===
          update.modifierDefinitionVersionId &&
        entry.value.objectiveDefinitionVersionId ===
          update.objectiveDefinitionVersionId &&
        entry.value.gameRuleSetVersionId === update.gameRuleSetVersionId
      ) {
        continue;
      }
      if (
        entry.value.modifierDefinitionVersionId != null ||
        entry.value.objectiveDefinitionVersionId != null ||
        entry.value.gameRuleSetVersionId != null
      ) {
        throw conflict(`cutover tide ${update.id} changed concurrently`, {
          code: "CUTOVER_CONFLICT",
        });
      }
      const updated: DailyTide = {
        ...entry.value,
        contentVersion: update.contentVersion,
        modifierDefinitionVersionId: update.modifierDefinitionVersionId,
        objectiveDefinitionVersionId: update.objectiveDefinitionVersionId,
        gameRuleSetVersionId: update.gameRuleSetVersionId,
        updatedAt: now.toISOString(),
      };
      atomic.check({
        key: dailyTideKey(entry.value.dayKey),
        versionstamp: entry.versionstamp,
      });
      atomic.set(dailyTideKey(entry.value.dayKey), updated);
    }

    for (const selection of selections) {
      const entry = stateById.get(selection.stateId);
      if (!entry) {
        throw conflict(`cutover state ${selection.stateId} was not found`, {
          code: "CUTOVER_CONFLICT",
        });
      }
      const updated: PlayerDailyState = {
        ...entry.value,
        selectedStrategyId: selection.selectedStrategyDefinitionVersionId,
        updatedAt: now.toISOString(),
      };
      atomic.check({
        key: voyageDailyStateKey(entry.value.voyageId, entry.value.dayNumber),
        versionstamp: entry.versionstamp,
      });
      atomic.set(
        voyageDailyStateKey(entry.value.voyageId, entry.value.dayNumber),
        updated,
      );
    }

    for (const item of availability) {
      atomic.set(
        dailyTideStrategyKey(
          item.dailyTideId,
          item.strategyDefinitionVersionId,
        ),
        {
          dailyTideId: item.dailyTideId,
          strategyId: item.strategyDefinitionVersionId,
          contentVersion: item.contentVersion,
        },
      );
    }
    if (
      updates.length === 0 && selections.length === 0 &&
      availability.length === 0
    ) return;
    const result = await atomic.commit();
    if (!result.ok) {
      throw conflict("content cutover conflicted with a concurrent write", {
        code: "CUTOVER_CONFLICT",
      });
    }
  }

  async #readTidesWithVersions(): Promise<
    { value: DailyTide; versionstamp: string }[]
  > {
    const entries: { value: DailyTide; versionstamp: string }[] = [];
    for await (
      const entry of this.#store.kv.list<DailyTide>({ prefix: ["daily_tide"] })
    ) {
      if (entry.key.length === 2 && entry.versionstamp) {
        entries.push({ value: entry.value, versionstamp: entry.versionstamp });
      }
    }
    return entries;
  }

  async #readStatesWithVersions(): Promise<
    { value: PlayerDailyState; versionstamp: string }[]
  > {
    const entries: { value: PlayerDailyState; versionstamp: string }[] = [];
    for await (
      const entry of this.#store.kv.list<PlayerDailyState>({
        prefix: ["voyage_daily_state"],
      })
    ) {
      if (entry.versionstamp) {
        entries.push({ value: entry.value, versionstamp: entry.versionstamp });
      }
    }
    return entries;
  }
}

// Keep the view key helper referenced here as a reminder that snapshot joins
// use the same identity as DailyRepository; it also prevents accidental key
// drift if the KV layout changes.
export function dailyViewKeyForCutover(voyageId: string, dayNumber: number) {
  return voyageDailyViewKey(voyageId, dayNumber);
}
