/** Scheduled jobs: daily-tide provisioning and session cleanup. */

import type { DailyTide } from "../domain/types.ts";
import type { Config } from "../config.ts";
import { newId } from "../utils/ids.ts";
import { addMs, dayKeyInTimezone, instantAtMinutesUtc } from "../utils/time.ts";
import { lockMinutesOfDay } from "../config.ts";
import type { DailyRepository } from "../repositories/daily_repository.ts";
import { sessionKey, type Store } from "../repositories/kv.ts";
import type { VoyageRepository } from "../repositories/voyage_repository.ts";
import type { SettlementService } from "../services/settlement_service.ts";
import type { MarketProvider } from "../market/provider.ts";
import type { SettlementInput } from "../domain/settlement.ts";
import { decodeAndValidateProjection } from "../domain/projection.ts";

const tidePhase = "REGULAR";
const tideSettleDelayMs = 3 * 60 * 60 * 1000;

/**
 * Ensures a Daily Tide exists for today. Idempotent: concurrent invocations
 * agree on a single tide per day-key.
 */
export async function ensureDailyTide(
  config: Config,
  daily: DailyRepository,
  now: Date,
): Promise<"created" | "exists"> {
  const dayKey = dayKeyInTimezone(now, config.gameTimezone);
  const existing = await daily.getDailyTideByDayKey(dayKey);
  if (existing) {
    return "exists";
  }

  // DailyRepository allocates the sequence transactionally with the tide.
  // The provisional value is replaced before persistence.
  const lockAt = instantAtMinutesUtc(
    dayKey,
    lockMinutesOfDay(config),
    config.gameTimezone,
  );
  const settleAfter = addMs(lockAt, tideSettleDelayMs);
  const tide: DailyTide = {
    id: newId(),
    dayKey,
    sequenceNumber: 0,
    phase: tidePhase,
    lockAt: lockAt.toISOString(),
    settleAfter: settleAfter.toISOString(),
    contentVersion: config.contentVersion,
    createdAt: now.toISOString(),
    updatedAt: now.toISOString(),
  };
  return daily.ensureDailyTide(tide);
}

/** Deletes sessions whose expiration has passed, plus stale in-progress records. */
export async function runSettlementJob(
  store: Store,
  daily: DailyRepository,
  voyages: VoyageRepository,
  settlement: SettlementService,
  provider: MarketProvider,
  now: Date,
): Promise<{ settled: number; skipped: number; failed: number }> {
  let settled = 0;
  let skipped = 0;
  let failed = 0;
  for await (
    const entry of store.list<import("../domain/types.ts").Voyage>(["voyage"])
  ) {
    const voyage = entry.value;
    if (voyage.status !== "ACTIVE") {
      skipped++;
      continue;
    }
    const state = await daily.getPlayerDailyState(
      voyage.id,
      voyage.currentDayNumber,
    );
    if (!state || state.phase !== "LOCKED") {
      skipped++;
      continue;
    }
    const tide = await daily.getDailyTideById(state.dailyTideId);
    if (!tide || new Date(tide.settleAfter).getTime() > now.getTime()) {
      skipped++;
      continue;
    }
    try {
      const view = await daily.getPlayerDailyView(
        voyage.id,
        voyage.currentDayNumber,
      );
      if (!view) throw new Error("missing daily settlement view");
      const window = await provider.fetchWindow(tide);
      const prices = await provider.fetchPrices(window);
      const input = await settlementInput(
        view.projection,
        provider,
        tide,
        window,
        prices,
        voyage.currentDayNumber,
      );
      await settlement.settleDay(voyage.id, input, now);
      settled++;
    } catch {
      failed++;
    }
  }
  // Keep the repository dependency explicit in the job contract; it also
  // ensures callers cannot accidentally run settlement without voyage access.
  void voyages;
  return { settled, skipped, failed };
}

async function settlementInput(
  projectionValue: unknown,
  provider: MarketProvider,
  tide: DailyTide,
  window: Awaited<ReturnType<MarketProvider["fetchWindow"]>>,
  prices: Awaited<ReturnType<MarketProvider["fetchPrices"]>>,
  dayNumber: number,
): Promise<SettlementInput> {
  const projection = decodeAndValidateProjection(1, projectionValue);
  const lineup = projection.lineup as {
    slots?: Array<{ keeper?: Record<string, unknown> | null }>;
  };
  const keepers = (lineup.slots ?? []).flatMap((slot) => {
    const keeper = slot.keeper;
    if (!keeper) return [];
    return [{
      id: String(keeper.id),
      definitionKey: String(keeper.definitionId),
      sector: String(keeper.sector),
      role: String(keeper.role),
      rarity: String(keeper.rarity),
    }];
  });
  const scores = await provider.fetchSectorScores(tide, window, prices);
  return {
    dayNumber,
    keepers,
    market: { sectorScores: scores },
    modifier: {
      id: String(
        (projection.modifier as Record<string, unknown> | undefined)?.id ??
          "default",
      ),
      scoreBonusBps: 0,
      hullDelta: 0,
      suppliesDelta: 0,
    },
    strategy: {
      id: (projection.selectedStrategyId as string | null | undefined) ?? null,
      scoreBonusBps: 0,
      hullDelta: 0,
      suppliesDelta: 0,
    },
    rules: {
      version: tide.contentVersion,
      passScoreBps: 5_000,
      damageScoreBps: 3_000,
      diversityBonusBps: 0,
      successHullDelta: 1,
      deficitHullDelta: -2,
      successSuppliesDelta: 2,
      deficitSuppliesDelta: 0,
    },
  };
}

export async function cleanupSessions(
  store: Store,
  now: Date,
): Promise<number> {
  const expired: string[] = [];
  const nowMs = now.getTime();
  for await (const entry of store.list<{ expiresAt: string }>(["session"])) {
    const digest = entry.key[1];
    const expiresAtMs = new Date(entry.value.expiresAt).getTime();
    if (expiresAtMs <= nowMs && typeof digest === "string") {
      expired.push(digest);
    }
  }
  for (const digest of expired) {
    await store.delete(sessionKey(digest));
  }
  return expired.length;
}
