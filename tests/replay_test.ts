import { assertEquals, assertRejects } from "@std/assert";
import { type CalculationInput, Calculator } from "../src/market/basket.ts";
import {
  BenchmarkCalculator,
  type BenchmarkInput,
  EqualWeight,
} from "../src/market/sector.ts";
import {
  replayMismatchMessage,
  verifyBasket,
  verifyBenchmark,
} from "../src/market/replay.ts";

Deno.test("replay: verifies basket and benchmark outputs exactly", async () => {
  const basketInput: CalculationInput = {
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
      components: [
        {
          marketAssetId: "a",
          targetWeight: 50_000_000,
          enabled: true,
          sequence: 0,
        },
        {
          marketAssetId: "b",
          targetWeight: 50_000_000,
          enabled: true,
          sequence: 1,
        },
      ],
    },
    window: {
      id: "window",
      dailyTideId: "tide",
      providerKey: "fixture",
      start: "2026-07-30T00:00:00Z",
      end: "2026-07-31T00:00:00Z",
      toleranceMs: 0,
    },
    observations: [
      {
        id: "a-open",
        providerKey: "fixture",
        marketAssetId: "a",
        observedAt: "2026-07-30T00:00:00Z",
        priceUnits: 100_000_000,
        status: "VALID",
      },
      {
        id: "a-close",
        providerKey: "fixture",
        marketAssetId: "a",
        observedAt: "2026-07-31T00:00:00Z",
        priceUnits: 110_000_000,
        status: "VALID",
      },
      {
        id: "b-open",
        providerKey: "fixture",
        marketAssetId: "b",
        observedAt: "2026-07-30T00:00:00Z",
        priceUnits: 100_000_000,
        status: "VALID",
      },
      {
        id: "b-close",
        providerKey: "fixture",
        marketAssetId: "b",
        observedAt: "2026-07-31T00:00:00Z",
        priceUnits: 90_000_000,
        status: "VALID",
      },
    ],
  };
  const basketExpected = await new Calculator().calculate(basketInput);
  await verifyBasket(basketInput, basketExpected);
  basketExpected.rawReturn = 123;
  await assertRejects(
    () => verifyBasket(basketInput, basketExpected),
    Error,
    replayMismatchMessage,
  );

  const benchmarkInput: BenchmarkInput = {
    dailyTideId: "tide",
    definition: {
      id: "crest-v2",
      sector: "CREST",
      benchmarkMethod: EqualWeight,
      minimumEligibleBaskets: 2,
      relativeScale: 500_000,
      relativeBlendWeight: 700_000,
      rankBlendWeight: 300_000,
      scoreCap: 1_000_000,
    },
    baskets: [
      {
        basketMetricId: "metric-a",
        basketMappingVersionId: "map-a",
        sector: "CREST",
        benchmarkEligible: true,
        metricStatus: "READY",
        normalizedPerformance: -100,
      },
      {
        basketMetricId: "metric-b",
        basketMappingVersionId: "map-b",
        sector: "CREST",
        benchmarkEligible: true,
        metricStatus: "READY",
        normalizedPerformance: 100,
      },
    ],
  };
  const benchmarkExpected = await new BenchmarkCalculator().calculate(
    benchmarkInput,
  );
  await verifyBenchmark(benchmarkInput, benchmarkExpected);
  benchmarkExpected.eligibleBasketCount++;
  await assertRejects(
    () => verifyBenchmark(benchmarkInput, benchmarkExpected),
    Error,
    replayMismatchMessage,
  );
  assertEquals(benchmarkExpected.eligibleBasketCount, 3);
});
