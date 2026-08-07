/** Daily context read service (ported from daily/). */

import type { Voyage } from "../domain/types.ts";
import { decodeAndValidateProjection } from "../domain/projection.ts";
import { notFound, TidekeepersError } from "../utils/errors.ts";
import type { DailyRepository } from "../repositories/daily_repository.ts";
import type { VoyageRepository } from "../repositories/voyage_repository.ts";

export interface DailyContext {
  serverNow: string;
  voyage: VoyageSummary;
  daily: DailyDetail;
}

export interface VoyageSummary {
  id: string;
  status: string;
  dayNumber: number;
  fundHealth: number;
  maxFundHealth: number;
  capital: number;
  score: string;
}

export interface DailyDetail {
  phase: string;
  lockAt: string;
  settleAfter: string;
  version: number;
  modifier: Modifier;
  objective: Objective;
  signals: Signal[];
  lineup: Lineup;
  inventory: Keeper[];
  shop: Shop;
  strategies: Strategy[];
  selectedStrategyId: string | null;
  pendingRewardCount: number;
}

export interface Modifier {
  id: string;
  name: string;
  description: string;
}

export interface Objective {
  id: string;
  name: string;
  description: string;
  progressLabel: string | null;
  rewardLabel: string | null;
}

export interface Signal {
  id: string;
  name: string;
  description: string;
  direction: string;
  strength: string;
  observedFrom: string;
  observedTo: string;
}

export interface Lineup {
  lockedAt: string | null;
  maxSlots: number;
  slots: FleetSlot[];
  synergies: Synergy[];
  warnings: FleetWarning[];
}

export interface FleetSlot {
  index: number;
  keeper: Keeper | null;
}

export interface FleetWarning {
  code: string;
  message: string;
  severity: string;
}

export interface Synergy {
  id: string;
  name: string;
  description: string;
  state: string;
  currentCount: number;
  requiredCount: number;
}

export interface Keeper {
  id: string;
  definitionId: string;
  name: string;
  level: number;
  rarity: string;
  role: string;
  sector: string;
  passiveSummary: string;
  artworkUrl: string | null;
}

export interface Shop {
  offers: ShopOffer[];
  refreshAt: string | null;
  rerollCost: number;
  rerollIndex: number;
}

export interface ShopOffer {
  id: string;
  keeper: Keeper;
  cost: number;
  available: boolean;
  synergyHint: string | null;
}

export interface Strategy {
  id: string;
  name: string;
  description: string;
  upside: string;
  downside: string;
  available: boolean;
}

export class DailyService {
  #daily: DailyRepository;
  #voyages: VoyageRepository;

  constructor(daily: DailyRepository, voyages: VoyageRepository) {
    this.#daily = daily;
    this.#voyages = voyages;
  }

  async getCurrentDailyContext(
    playerId: string,
    now: Date,
  ): Promise<DailyContext> {
    const activeId = await this.#voyages.getActiveVoyageId(playerId);
    if (activeId === null) {
      throw notFound("no active voyage", { code: "NO_ACTIVE_VOYAGE" });
    }
    const voyage = await this.#voyages.getVoyageOrFail(activeId);

    const state = await this.#daily.getPlayerDailyState(
      voyage.id,
      voyage.currentDayNumber,
    );
    if (!state) {
      throw new TidekeepersError("missing daily state", "internal", {
        code: "INTERNAL_ERROR",
      });
    }
    const view = await this.#daily.getPlayerDailyView(
      voyage.id,
      voyage.currentDayNumber,
    );
    if (!view) {
      throw new TidekeepersError("missing projection", "internal", {
        code: "INTERNAL_ERROR",
      });
    }
    const tide = await this.#daily.getDailyTideById(state.dailyTideId);
    if (!tide) {
      throw new TidekeepersError("missing daily tide", "internal", {
        code: "INTERNAL_ERROR",
      });
    }

    const proj = decodeAndValidateProjection(
      view.schemaVersion,
      view.projection,
    ) as unknown as ProjectionV1;
    return {
      serverNow: now.toISOString(),
      voyage: {
        id: voyage.publicId,
        status: voyage.status,
        dayNumber: voyage.currentDayNumber,
        fundHealth: voyage.fundHealth,
        maxFundHealth: voyage.maxFundHealth,
        capital: voyage.capital,
        score: safeScore(voyage.score),
      },
      daily: {
        phase: state.phase,
        lockAt: tide.lockAt,
        settleAfter: tide.settleAfter,
        version: state.version,
        modifier: proj.modifier,
        objective: proj.objective,
        signals: proj.signals ?? [],
        lineup: proj.lineup,
        inventory: proj.inventory ?? [],
        shop: proj.shop,
        strategies: proj.strategies ?? [],
        selectedStrategyId: proj.selectedStrategyId ?? null,
        pendingRewardCount: proj.pendingRewardCount ?? 0,
      },
    };
  }
}

interface ProjectionV1 {
  modifier: Modifier;
  objective: Objective;
  signals: Signal[];
  lineup: Lineup;
  inventory: Keeper[];
  shop: Shop;
  strategies: Strategy[];
  selectedStrategyId: string | null;
  pendingRewardCount: number;
}

function safeScore(score: string): string {
  return score === "" ? "0.0000" : score;
}

export function safeVoyageSummary(voyage: Voyage): VoyageSummary {
  return {
    id: voyage.publicId,
    status: voyage.status,
    dayNumber: voyage.currentDayNumber,
    fundHealth: voyage.fundHealth,
    maxFundHealth: voyage.maxFundHealth,
    capital: voyage.capital,
    score: safeScore(voyage.score),
  };
}
