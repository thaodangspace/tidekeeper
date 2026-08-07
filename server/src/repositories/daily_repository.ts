/** Daily tide and player daily state persistence over Deno KV. */

import type {
  DailyTide,
  PlayerDailyContextView,
  PlayerDailyState,
} from "../domain/types.ts";
import {
  dailyTideIdKey,
  dailyTideKey,
  dailyTideSequenceKey,
  sequenceKey,
  type Store,
  voyageDailyStateKey,
  voyageDailyViewKey,
} from "./kv.ts";

export class DailyRepository {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async getDailyTideByDayKey(dayKey: string): Promise<DailyTide | null> {
    return this.#store.get<DailyTide>(dailyTideKey(dayKey));
  }

  async getDailyTideById(id: string): Promise<DailyTide | null> {
    const dayKey = await this.#store.get<string>(dailyTideIdKey(id));
    if (!dayKey) {
      return null;
    }
    return this.#store.get<DailyTide>(dailyTideKey(dayKey));
  }

  async getDailyTideBySequence(
    sequenceNumber: number,
  ): Promise<DailyTide | null> {
    const dayKey = await this.#store.get<string>(
      dailyTideSequenceKey(sequenceNumber),
    );
    if (!dayKey) {
      return null;
    }
    return this.#store.get<DailyTide>(dailyTideKey(dayKey));
  }

  /** Returns the largest allocated tide sequence number, or 0 when empty. */
  async getLatestTideSequence(): Promise<number> {
    let latest = 0;
    for await (const entry of this.#store.list<number>(["sequence"])) {
      if (entry.key[1] === "daily_tide" && typeof entry.value === "number") {
        latest = Math.max(latest, entry.value);
      }
    }
    return latest;
  }

  /**
   * Creates a daily tide exactly once and allocates its sequence in the same
   * transaction. Returning `exists` only means this day was won by another
   * invocation; a sequence conflict is retried so two different days cannot
   * accidentally strand one of their tides without a sequence.
   */
  async ensureDailyTide(tide: DailyTide): Promise<"created" | "exists"> {
    for (let attempt = 0; attempt < 16; attempt++) {
      const dayEntry = await this.#store.kv.get<DailyTide>(
        dailyTideKey(tide.dayKey),
      );
      if (dayEntry.value) {
        return "exists";
      }

      const counterEntry = await this.#store.kv.get<number>(
        sequenceKey("daily_tide"),
      );
      const sequenceNumber = (counterEntry.value ?? 0) + 1;
      const candidate = { ...tide, sequenceNumber };
      const result = await this.#store
        .atomic()
        .check({ key: dailyTideKey(tide.dayKey), versionstamp: null })
        .check({
          key: sequenceKey("daily_tide"),
          versionstamp: counterEntry.versionstamp,
        })
        .check({
          key: dailyTideSequenceKey(sequenceNumber),
          versionstamp: null,
        })
        .set(dailyTideKey(candidate.dayKey), candidate)
        .set(dailyTideSequenceKey(sequenceNumber), candidate.dayKey)
        .set(dailyTideIdKey(candidate.id), candidate.dayKey)
        .set(sequenceKey("daily_tide"), sequenceNumber)
        .commit();
      if (result.ok) {
        return "created";
      }
    }
    throw new Error("could not allocate a daily tide sequence");
  }

  async getPlayerDailyState(
    voyageId: string,
    dayNumber: number,
  ): Promise<PlayerDailyState | null> {
    const entry = await this.getPlayerDailyStateEntry(voyageId, dayNumber);
    return entry?.value ?? null;
  }

  async getPlayerDailyStateEntry(
    voyageId: string,
    dayNumber: number,
  ): Promise<{ value: PlayerDailyState; versionstamp: string } | null> {
    const entry = await this.#store.kv.get<PlayerDailyState>(
      voyageDailyStateKey(voyageId, dayNumber),
    );
    return entry.value && entry.versionstamp
      ? { value: entry.value, versionstamp: entry.versionstamp }
      : null;
  }

  async upsertPlayerDailyState(state: PlayerDailyState): Promise<void> {
    await this.#store.set(
      voyageDailyStateKey(state.voyageId, state.dayNumber),
      state,
    );
  }

  async versionstampForState(state: PlayerDailyState): Promise<string | null> {
    return this.#store.versionstamp(
      voyageDailyStateKey(state.voyageId, state.dayNumber),
    );
  }

  async versionstampForView(
    view: PlayerDailyContextView,
  ): Promise<string | null> {
    return this.#store.versionstamp(
      voyageDailyViewKey(view.voyageId, view.dayNumber),
    );
  }

  /** Atomically updates the daily state and its materialized projection. */
  async updatePlayerDaily(
    state: PlayerDailyState,
    view: PlayerDailyContextView,
    expectedStateVersionstamp: string,
    expectedViewVersionstamp: string,
  ): Promise<boolean> {
    const result = await this.#store
      .atomic()
      .check({
        key: voyageDailyStateKey(state.voyageId, state.dayNumber),
        versionstamp: expectedStateVersionstamp,
      })
      .check({
        key: voyageDailyViewKey(view.voyageId, view.dayNumber),
        versionstamp: expectedViewVersionstamp,
      })
      .set(voyageDailyStateKey(state.voyageId, state.dayNumber), state)
      .set(voyageDailyViewKey(view.voyageId, view.dayNumber), view)
      .commit();
    return result.ok;
  }

  async getPlayerDailyView(
    voyageId: string,
    dayNumber: number,
  ): Promise<PlayerDailyContextView | null> {
    const entry = await this.getPlayerDailyViewEntry(voyageId, dayNumber);
    return entry?.value ?? null;
  }

  async getPlayerDailyViewEntry(
    voyageId: string,
    dayNumber: number,
  ): Promise<{ value: PlayerDailyContextView; versionstamp: string } | null> {
    const entry = await this.#store.kv.get<PlayerDailyContextView>(
      voyageDailyViewKey(voyageId, dayNumber),
    );
    return entry.value && entry.versionstamp
      ? { value: entry.value, versionstamp: entry.versionstamp }
      : null;
  }

  async upsertPlayerDailyView(view: PlayerDailyContextView): Promise<void> {
    await this.#store.set(
      voyageDailyViewKey(view.voyageId, view.dayNumber),
      view,
    );
  }
}
