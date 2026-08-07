import { assertEquals } from "@std/assert";
import {
  BasketCalculationService,
  type BasketMetricRepository,
} from "../src/market/basket_service.ts";
import type { CalculationInput, Metric } from "../src/market/basket.ts";
import type {
  MarketCalculationEvent,
  MarketRecorder,
} from "../src/market/telemetry.ts";

const input: CalculationInput = {
  mapping: {
    id: "mapping",
    key: "crest",
    sector: "CREST",
    minimumCoveredWeight: 100_000_000,
    normalizationCap: 2_000_000,
    benchmarkEligible: true,
    expectedTurbulence: {
      id: "turbulence",
      type: "STATIC_CONTENT_VALUE",
      staticValue: 1,
      floor: 1,
    },
    components: [{
      marketAssetId: "asset",
      targetWeight: 100_000_000,
      enabled: true,
      sequence: 0,
    }],
  },
  window: {
    id: "window",
    dailyTideId: "tide",
    providerKey: "fixture",
    start: "2026-08-01T00:00:00Z",
    end: "2026-08-02T00:00:00Z",
    toleranceMs: 0,
  },
  observations: [
    {
      id: "open",
      providerKey: "fixture",
      marketAssetId: "asset",
      observedAt: "2026-08-01T00:00:00Z",
      priceUnits: 100_000_000,
      status: "VALID",
    },
    {
      id: "close",
      providerKey: "fixture",
      marketAssetId: "asset",
      observedAt: "2026-08-02T00:00:00Z",
      priceUnits: 110_000_000,
      status: "VALID",
    },
  ],
};

Deno.test("basket service: calculates, persists, and records telemetry", async () => {
  let persisted: Metric | null = null;
  const repository: BasketMetricRepository = {
    async persistMetric(_input, metric) {
      persisted = metric;
      return { id: "metric-1", idempotent: false };
    },
  };
  const events: MarketCalculationEvent[] = [];
  const recorder: MarketRecorder = {
    recordMarketCalculation: (event) => events.push(event),
  };
  const result = await new BasketCalculationService(repository, recorder)
    .calculateAndPersist(input);
  assertEquals(result.id, "metric-1");
  assertEquals(result.idempotent, false);
  assertEquals(result.metric.status, "READY");
  assertEquals(persisted !== null, true);
  assertEquals(events.length, 1);
  assertEquals(events[0]?.operation, "calculate_basket_metrics");
  assertEquals(events[0]?.status, "READY");
});
