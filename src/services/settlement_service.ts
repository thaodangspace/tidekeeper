/** Persistence boundary for deterministic settlement results and rewards. */

import type {
  PlayerDailyContextView,
  PlayerDailyState,
  SettlementRecord,
  Voyage,
} from "../domain/types.ts";
import {
  settle,
  type SettlementInput,
  type SettlementOutput,
} from "../domain/settlement.ts";
import type { DailyRepository } from "../repositories/daily_repository.ts";
import type { VoyageRepository } from "../repositories/voyage_repository.ts";
import {
  rewardClaimKey,
  settlementKey,
  type Store,
  voyageActiveKey,
  voyageDailyStateKey,
  voyageDailyViewKey,
  voyageKey,
} from "../repositories/kv.ts";
import { applyHullDelta, newHull } from "../domain/hull.ts";
import { conflict, internal, notFound } from "../utils/errors.ts";
import { newId } from "../utils/ids.ts";
import { getPublishedVoyageDefinition } from "../content/voyage_definition.ts";

export interface RewardClaimResponse {
  voyageId: string;
  dayNumber: number;
  version: number;
  reward: NonNullable<SettlementRecord["reward"]>;
}

export interface AdvanceResponse {
  voyageId: string;
  dayNumber: number;
  status: string;
  version: number;
}

interface Projection {
  pendingRewardCount?: number;
  [key: string]: unknown;
}

export class SettlementService {
  #store: Store;
  #daily: DailyRepository;
  #voyages: VoyageRepository;

  constructor(
    store: Store,
    daily: DailyRepository,
    voyages: VoyageRepository,
  ) {
    this.#store = store;
    this.#daily = daily;
    this.#voyages = voyages;
  }

  /** Settles a locked day. Repeating the call returns the persisted result. */
  async settleDay(
    voyageId: string,
    input: SettlementInput,
    now: Date,
  ): Promise<SettlementRecord> {
    const existing = await this.#store.get<SettlementRecord>(
      settlementKey(voyageId, input.dayNumber),
    );
    if (existing) return existing;

    const voyageEntry = await this.#store.kv.get<Voyage>(voyageKey(voyageId));
    const voyage = voyageEntry.value;
    if (!voyage || !voyageEntry.versionstamp) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    if (voyage.status !== "ACTIVE") {
      throw conflict("voyage is not active", { code: "VOYAGE_NOT_ACTIVE" });
    }
    const stateEntry = await this.#daily.getPlayerDailyStateEntry(
      voyage.id,
      input.dayNumber,
    );
    const viewEntry = await this.#daily.getPlayerDailyViewEntry(
      voyage.id,
      input.dayNumber,
    );
    const state = stateEntry?.value;
    const view = viewEntry?.value;
    if (!state || !view) throw internal("missing daily settlement state");
    if (state.phase !== "LOCKED") {
      throw conflict("daily tide is not ready for settlement", {
        code: "INVALID_PHASE",
      });
    }
    const output = settle(input);
    const result = makeRecord(voyage.id, output, now);
    const updatedState: PlayerDailyState = {
      ...state,
      phase: "SETTLED",
      version: state.version + 1,
      pendingRewardCount: state.pendingRewardCount + (output.reward ? 1 : 0),
      updatedAt: now.toISOString(),
    };
    const updatedView: PlayerDailyContextView = {
      ...view,
      projection: withPendingRewardCount(
        view.projection,
        updatedState.pendingRewardCount,
      ),
      validatedAt: now.toISOString(),
    };
    const hullResult = applyHullDelta(
      newHull(voyage.fundHealth, voyage.maxFundHealth),
      output.hullDelta,
    );
    const updatedVoyage: Voyage = {
      ...voyage,
      fundHealth: hullResult.hull.current,
      capital: Math.max(0, voyage.capital + output.suppliesDelta),
      score: addScores(voyage.score, output.score),
      status: hullResult.change.destroyed ? "FAILED" : voyage.status,
      completedAt: hullResult.change.destroyed
        ? now.toISOString()
        : voyage.completedAt,
      rowVersion: voyage.rowVersion + 1,
      updatedAt: now.toISOString(),
    };
    const atomic = this.#store.atomic()
      .check({
        key: voyageKey(voyage.id),
        versionstamp: voyageEntry.versionstamp,
      })
      .check({
        key: voyageDailyStateKey(state.voyageId, state.dayNumber),
        versionstamp: stateEntry.versionstamp,
      })
      .check({
        key: voyageDailyViewKey(view.voyageId, view.dayNumber),
        versionstamp: viewEntry.versionstamp,
      })
      .set(voyageKey(voyage.id), updatedVoyage)
      .set(voyageDailyStateKey(state.voyageId, state.dayNumber), updatedState)
      .set(voyageDailyViewKey(view.voyageId, view.dayNumber), updatedView)
      .set(settlementKey(voyage.id, input.dayNumber), result);
    const committed = await atomic.commit();
    if (!committed.ok) {
      const winner = await this.#store.get<SettlementRecord>(
        settlementKey(voyageId, input.dayNumber),
      );
      if (winner) return winner;
      throw conflict("settlement state changed", { code: "VERSION_CONFLICT" });
    }
    return result;
  }

  async advanceDay(
    playerId: string,
    expectedVersion: number,
    now: Date,
  ): Promise<AdvanceResponse> {
    if (!Number.isInteger(expectedVersion) || expectedVersion < 1) {
      throw conflict("invalid advance version", { code: "VERSION_CONFLICT" });
    }
    const activeId = await this.#voyages.getActiveVoyageId(playerId);
    if (activeId === null) {
      throw notFound("no active voyage", { code: "NO_ACTIVE_VOYAGE" });
    }
    const voyageEntry = await this.#store.kv.get<Voyage>(voyageKey(activeId));
    const voyage = voyageEntry.value;
    if (!voyage || !voyageEntry.versionstamp) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    const stateEntry = await this.#daily.getPlayerDailyStateEntry(
      voyage.id,
      voyage.currentDayNumber,
    );
    const viewEntry = await this.#daily.getPlayerDailyViewEntry(
      voyage.id,
      voyage.currentDayNumber,
    );
    const state = stateEntry?.value;
    const view = viewEntry?.value;
    if (!state || !view) throw internal("missing daily advance state");
    if (state.version !== expectedVersion) {
      throw conflict("daily state version conflict", {
        code: "VERSION_CONFLICT",
      });
    }
    if (state.phase !== "SETTLED") {
      throw conflict("daily tide is not settled", { code: "INVALID_PHASE" });
    }
    if (state.pendingRewardCount > 0) {
      throw conflict("claim the pending reward before advancing", {
        code: "REWARD_NOT_CLAIMED",
      });
    }

    const definition = getPublishedVoyageDefinition();
    const voyageKeyValue = voyageKey(voyage.id);
    if (voyage.currentDayNumber >= definition.durationDays) {
      const completed: Voyage = {
        ...voyage,
        status: "COMPLETED",
        completedAt: now.toISOString(),
        rowVersion: voyage.rowVersion + 1,
        updatedAt: now.toISOString(),
      };
      const committed = await this.#store.atomic()
        .check({ key: voyageKeyValue, versionstamp: voyageEntry.versionstamp })
        .check({
          key: voyageDailyStateKey(state.voyageId, state.dayNumber),
          versionstamp: stateEntry.versionstamp,
        })
        .set(voyageKeyValue, completed)
        .delete(voyageActiveKey(playerId))
        .commit();
      if (!committed.ok) {
        throw conflict("voyage state changed", { code: "VERSION_CONFLICT" });
      }
      return {
        voyageId: voyage.publicId,
        dayNumber: voyage.currentDayNumber,
        status: completed.status,
        version: state.version,
      };
    }

    const currentTide = await this.#daily.getDailyTideById(state.dailyTideId);
    if (!currentTide) throw internal("missing current daily tide");
    const nextDayKey = nextDayKeyFor(currentTide.dayKey);
    const nextTide = await this.#daily.getDailyTideByDayKey(nextDayKey);
    if (!nextTide) {
      throw conflict("next daily tide is not ready", {
        code: "NEXT_TIDE_NOT_READY",
      });
    }
    const nextDay = voyage.currentDayNumber + 1;
    const nextState: PlayerDailyState = {
      id: newId(),
      playerId,
      voyageId: voyage.id,
      dailyTideId: nextTide.id,
      dayNumber: nextDay,
      phase: "PREPARATION",
      version: 1,
      selectedStrategyId: null,
      pendingRewardCount: 0,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
    const nextView: PlayerDailyContextView = {
      voyageId: voyage.id,
      dayNumber: nextDay,
      schemaVersion: view.schemaVersion,
      projection: nextDayProjection(view.projection),
      validatedAt: now.toISOString(),
    };
    const advanced: Voyage = {
      ...voyage,
      currentDayNumber: nextDay,
      rowVersion: voyage.rowVersion + 1,
      updatedAt: now.toISOString(),
    };
    const committed = await this.#store.atomic()
      .check({ key: voyageKeyValue, versionstamp: voyageEntry.versionstamp })
      .check({
        key: voyageDailyStateKey(state.voyageId, state.dayNumber),
        versionstamp: stateEntry.versionstamp,
      })
      .check({
        key: voyageDailyViewKey(view.voyageId, view.dayNumber),
        versionstamp: viewEntry.versionstamp,
      })
      .check({
        key: voyageDailyStateKey(voyage.id, nextDay),
        versionstamp: null,
      })
      .check({
        key: voyageDailyViewKey(voyage.id, nextDay),
        versionstamp: null,
      })
      .set(voyageKeyValue, advanced)
      .set(voyageDailyStateKey(voyage.id, nextDay), nextState)
      .set(voyageDailyViewKey(voyage.id, nextDay), nextView)
      .commit();
    if (!committed.ok) {
      throw conflict("voyage state changed", { code: "VERSION_CONFLICT" });
    }
    return {
      voyageId: voyage.publicId,
      dayNumber: nextDay,
      status: advanced.status,
      version: nextState.version,
    };
  }

  async getCurrentResult(playerId: string): Promise<SettlementRecord> {
    const activeId = await this.#voyages.getActiveVoyageId(playerId);
    if (activeId === null) {
      throw notFound("no active voyage", { code: "NO_ACTIVE_VOYAGE" });
    }
    const voyage = await this.#voyages.getVoyageOrFail(activeId);
    const result = await this.#store.get<SettlementRecord>(
      settlementKey(voyage.id, voyage.currentDayNumber),
    );
    if (!result) {
      throw notFound("result is not ready", { code: "RESULT_NOT_READY" });
    }
    return result;
  }

  async claimCurrentReward(
    playerId: string,
    expectedVersion: number,
    now: Date,
  ): Promise<RewardClaimResponse> {
    if (!Number.isInteger(expectedVersion) || expectedVersion < 1) {
      throw conflict("invalid reward version", { code: "REWARD_INVALID" });
    }
    const activeId = await this.#voyages.getActiveVoyageId(playerId);
    if (activeId === null) {
      throw notFound("no active voyage", { code: "NO_ACTIVE_VOYAGE" });
    }
    const voyageEntry = await this.#store.kv.get<Voyage>(voyageKey(activeId));
    const voyage = voyageEntry.value;
    if (!voyage || !voyageEntry.versionstamp) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    const stateEntry = await this.#daily.getPlayerDailyStateEntry(
      voyage.id,
      voyage.currentDayNumber,
    );
    const viewEntry = await this.#daily.getPlayerDailyViewEntry(
      voyage.id,
      voyage.currentDayNumber,
    );
    const state = stateEntry?.value;
    const view = viewEntry?.value;
    const resultEntry = await this.#store.kv.get<SettlementRecord>(
      settlementKey(voyage.id, voyage.currentDayNumber),
    );
    const result = resultEntry.value;
    if (!state || !view || !result || !resultEntry.versionstamp) {
      throw notFound("reward is not ready", { code: "RESULT_NOT_READY" });
    }
    if (state.version !== expectedVersion) {
      throw conflict("daily state version conflict", {
        code: "VERSION_CONFLICT",
      });
    }
    if (
      !result.reward || result.claimedAt !== null ||
      state.pendingRewardCount < 1
    ) {
      throw conflict("reward has already been claimed", {
        code: "REWARD_ALREADY_CLAIMED",
      });
    }
    const claimedAt = now.toISOString();
    const updatedState: PlayerDailyState = {
      ...state,
      version: state.version + 1,
      pendingRewardCount: state.pendingRewardCount - 1,
      updatedAt: claimedAt,
    };
    const updatedView: PlayerDailyContextView = {
      ...view,
      projection: withPendingRewardCount(
        view.projection,
        updatedState.pendingRewardCount,
      ),
      validatedAt: claimedAt,
    };
    const updatedResult: SettlementRecord = { ...result, claimedAt };
    const committed = await this.#store.atomic()
      .check({
        key: voyageKey(voyage.id),
        versionstamp: voyageEntry.versionstamp,
      })
      .check({
        key: voyageDailyStateKey(state.voyageId, state.dayNumber),
        versionstamp: stateEntry!.versionstamp,
      })
      .check({
        key: voyageDailyViewKey(view.voyageId, view.dayNumber),
        versionstamp: viewEntry!.versionstamp,
      })
      .check({
        key: settlementKey(voyage.id, voyage.currentDayNumber),
        versionstamp: resultEntry.versionstamp,
      })
      .set(voyageDailyStateKey(state.voyageId, state.dayNumber), updatedState)
      .set(voyageDailyViewKey(view.voyageId, view.dayNumber), updatedView)
      .set(settlementKey(voyage.id, voyage.currentDayNumber), updatedResult)
      .set(rewardClaimKey(voyage.id, voyage.currentDayNumber), { claimedAt })
      .commit();
    if (!committed.ok) {
      throw conflict("reward state changed", { code: "VERSION_CONFLICT" });
    }
    return {
      voyageId: voyage.publicId,
      dayNumber: voyage.currentDayNumber,
      version: updatedState.version,
      reward: result.reward,
    };
  }
}

function makeRecord(
  voyageId: string,
  output: SettlementOutput,
  now: Date,
): SettlementRecord {
  return {
    id: newId(),
    voyageId,
    dayNumber: output.dayNumber,
    ruleVersion: output.ruleVersion,
    scoreBps: output.scoreBps,
    score: output.score,
    hullDelta: output.hullDelta,
    suppliesDelta: output.suppliesDelta,
    reward: output.reward,
    breakdown: output.breakdown,
    settledAt: now.toISOString(),
    claimedAt: null,
  };
}

function withPendingRewardCount(value: unknown, count: number): unknown {
  if (!value || typeof value !== "object") {
    throw internal("invalid daily projection");
  }
  return { ...(value as Projection), pendingRewardCount: count };
}

function nextDayProjection(value: unknown): unknown {
  if (!value || typeof value !== "object") {
    throw internal("invalid daily projection");
  }
  const projection = value as Record<string, unknown>;
  const lineup = projection.lineup as { maxSlots: number };
  const slots = Array.from({ length: lineup.maxSlots }, (_, index) => ({
    index,
    keeper: null,
  }));
  return {
    ...projection,
    pendingRewardCount: 0,
    selectedStrategyId: null,
    lineup: {
      ...lineup,
      lockedAt: null,
      slots,
      warnings: lineup.maxSlots > 0
        ? [{
          code: "EMPTY_SLOT",
          severity: "WARNING",
          message: "All slots are empty.",
        }]
        : [],
    },
    shop: {
      ...(projection.shop as Record<string, unknown>),
      offers: [],
      refreshAt: null,
      rerollIndex: 0,
    },
  };
}

function nextDayKeyFor(dayKey: string): string {
  const [year, month, day] = dayKey.split("-").map(Number);
  const next = new Date(Date.UTC(year, month - 1, day + 1));
  return next.toISOString().slice(0, 10);
}

function addScores(left: string, right: string): string {
  return formatScore(parseScore(left) + parseScore(right));
}

function parseScore(value: string): number {
  const match = /^(\d+)\.(\d{1,4})$/.exec(value);
  if (!match) return 0;
  return Number.parseInt(match[1]!, 10) * 10_000 +
    Number.parseInt(match[2]!.padEnd(4, "0"), 10);
}

function formatScore(scoreBps: number): string {
  const whole = Math.floor(scoreBps / 10_000);
  const fraction = String(scoreBps % 10_000).padStart(4, "0");
  return `${whole}.${fraction}`;
}
