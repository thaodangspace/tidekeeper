/** Controlled KV write boundary for provider market evidence. */

import type { Store } from "../repositories/kv.ts";
import { conflict } from "../utils/errors.ts";

export interface CalculationWindowRecord {
  id: string;
  dailyTideId: string;
  providerKey: string;
  start: string;
  end: string;
  requiredGranularitySeconds: number;
  observationToleranceSeconds: number;
}

export type ObservationQuality =
  | "VALID"
  | "INVALID_PRICE"
  | "OUTLIER_FLAGGED"
  | "PROVIDER_REJECTED";

export interface MarketObservationRecord {
  id: string;
  calculationWindowId: string;
  marketAssetId: string;
  observedAt: string;
  rawPrice: string;
  price: string | null;
  qualityStatus: ObservationQuality;
  qualityFlags: string[];
  rawResponseHash: string | null;
}

export class MarketObservationWriter {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async insertWindow(window: CalculationWindowRecord): Promise<void> {
    validateWindow(window);
    const key = windowKey(window.id);
    const committed = await this.#store.atomic()
      .check({ key, versionstamp: null })
      .set(key, window)
      .commit();
    if (!committed.ok) {
      throw conflict("calculation window already exists", {
        code: "MARKET_WINDOW_CONFLICT",
      });
    }
  }

  async insertObservation(observation: MarketObservationRecord): Promise<void> {
    validateObservation(observation);
    const key = observationKey(observation.id);
    const committed = await this.#store.atomic()
      .check({ key, versionstamp: null })
      .set(key, {
        ...observation,
        qualityFlags: [...observation.qualityFlags],
      })
      .commit();
    if (!committed.ok) {
      throw conflict("market observation already exists", {
        code: "MARKET_OBSERVATION_CONFLICT",
      });
    }
  }

  async getWindow(id: string): Promise<CalculationWindowRecord | null> {
    return this.#store.get<CalculationWindowRecord>(windowKey(id));
  }

  async getObservation(id: string): Promise<MarketObservationRecord | null> {
    return this.#store.get<MarketObservationRecord>(observationKey(id));
  }
}

export function validateWindow(window: CalculationWindowRecord): void {
  if (
    window.id === "" || window.dailyTideId === "" ||
    window.providerKey === "" ||
    !(new Date(window.start).getTime() < new Date(window.end).getTime()) ||
    !Number.isInteger(window.requiredGranularitySeconds) ||
    window.requiredGranularitySeconds <= 0 ||
    !Number.isInteger(window.observationToleranceSeconds) ||
    window.observationToleranceSeconds < 0
  ) {
    throw new Error("invalid market observation window");
  }
}

export function validateObservation(
  observation: MarketObservationRecord,
): void {
  if (
    observation.id === "" || observation.calculationWindowId === "" ||
    observation.marketAssetId === "" ||
    !Number.isFinite(new Date(observation.observedAt).getTime()) ||
    observation.rawPrice === "" ||
    !["VALID", "INVALID_PRICE", "OUTLIER_FLAGGED", "PROVIDER_REJECTED"]
      .includes(
        observation.qualityStatus,
      ) ||
    !Array.isArray(observation.qualityFlags)
  ) {
    throw new Error("invalid market observation");
  }
  if (
    observation.qualityStatus === "VALID" &&
    (observation.price === null || !isPositiveDecimal(observation.price))
  ) {
    throw new Error("invalid market observation");
  }
  if (
    observation.rawResponseHash !== null &&
    !/^[0-9a-f]{64}$/i.test(observation.rawResponseHash)
  ) {
    throw new Error("invalid market observation");
  }
}

function isPositiveDecimal(value: string): boolean {
  if (!/^\+?(?:\d+(?:\.\d*)?|\.\d+)$/.test(value.trim())) return false;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0;
}

function windowKey(id: string): Deno.KvKey {
  return ["market_window", id];
}

function observationKey(id: string): Deno.KvKey {
  return ["market_observation", id];
}
