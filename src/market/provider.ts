/** Market provider boundary used by scheduled settlement jobs. */

import type { DailyTide } from "../domain/types.ts";
import { targetKeys } from "./sector.ts";

export interface MarketWindow {
  id: string;
  dailyTideId: string;
  providerKey: string;
  start: string;
  end: string;
  toleranceMs: number;
}

export interface MarketPrice {
  assetId: string;
  observedAt: string;
  priceUnits: number;
  status: "VALID" | "PROVIDER_REJECTED";
}

export interface MarketProvider {
  readonly key: string;
  fetchWindow(tide: DailyTide): Promise<MarketWindow>;
  fetchPrices(window: MarketWindow): Promise<MarketPrice[]>;
  fetchSectorScores(
    tide: DailyTide,
    window: MarketWindow,
    prices: MarketPrice[],
  ): Promise<Record<string, number>>;
}

/** Deterministic local provider for development and isolated tests. */
export class StaticMarketProvider implements MarketProvider {
  readonly key = "static";

  async fetchWindow(tide: DailyTide): Promise<MarketWindow> {
    return {
      id: `window_${tide.id}`,
      dailyTideId: tide.id,
      providerKey: this.key,
      start: tide.dayKey + "T00:00:00.000Z",
      end: tide.dayKey + "T23:59:59.999Z",
      toleranceMs: 60 * 60 * 1000,
    };
  }

  async fetchPrices(_window: MarketWindow): Promise<MarketPrice[]> {
    return [];
  }

  async fetchSectorScores(
    _tide: DailyTide,
    _window: MarketWindow,
    _prices: MarketPrice[],
  ): Promise<Record<string, number>> {
    return Object.fromEntries(targetKeys().map((sector) => [sector, 5_000]));
  }
}

export function createMarketProvider(provider: string): MarketProvider {
  if (provider === "static") return new StaticMarketProvider();
  throw new Error(`unsupported market provider: ${provider}`);
}
