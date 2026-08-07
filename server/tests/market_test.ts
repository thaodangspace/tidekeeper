/** Market math tests: precision arithmetic, canonical checksums, basket, sector. */

import { assertEquals, assertStrictEquals, assertThrows } from "@std/assert";
import * as precision from "../src/market/precision.ts";
import { canonicalChecksum, sortedStrings } from "../src/market/canonical.ts";
import {
  Calculator,
  ComponentStatusMissingOpen,
  ComponentStatusValid,
  MetricStatusDataIncomplete,
  MetricStatusReady,
} from "../src/market/basket.ts";
import {
  BenchmarkCalculator,
  EqualWeight,
  ScoreCalculator,
  targetKeys,
} from "../src/market/sector.ts";

Deno.test("precision: add/subtract are exact integer math", () => {
  assertStrictEquals(precision.add(100, 200), 300);
  assertStrictEquals(precision.subtract(200, 100), 100);
});

Deno.test("precision: multiplyDivide rounds half away from zero", () => {
  // 5 * 1 / 2 = 2.5 -> 3 (half away from zero)
  assertStrictEquals(precision.multiplyDivide(5, 1, 2), 3);
  // -5 * 1 / 2 = -2.5 -> -3
  assertStrictEquals(precision.multiplyDivide(-5, 1, 2), -3);
  // 1 * 1 / 3 = 0.333 -> 0
  assertStrictEquals(precision.multiplyDivide(1, 1, 3), 0);
  // 4 * 1 / 3 = 1.333 -> 1
  assertStrictEquals(precision.multiplyDivide(4, 1, 3), 1);
});

Deno.test("precision: division by zero rejects", () => {
  assertThrows(() => precision.divide(1, 0));
  assertThrows(() => precision.multiplyDivide(1, 1, 0));
});

Deno.test("precision: clamp enforces bounds", () => {
  assertStrictEquals(precision.clamp(50, 0, 100), 50);
  assertStrictEquals(precision.clamp(-5, 0, 100), 0);
  assertStrictEquals(precision.clamp(150, 0, 100), 100);
});

Deno.test("canonical: sortedStrings does not mutate input", () => {
  const input = ["b", "a", "c"];
  const sorted = sortedStrings(input);
  assertEquals(sorted, ["a", "b", "c"]);
  assertEquals(input, ["b", "a", "c"]);
});

Deno.test("canonical: checksum is deterministic and length-prefixed", async () => {
  const a = await canonicalChecksum(["x", "y"]);
  const b = await canonicalChecksum(["x", "y"]);
  const c = await canonicalChecksum(["x", "y", "z"]);
  assertStrictEquals(a, b);
  assertStrictEquals(a.length, 64);
  assertStrictEquals(a === c, false);
  // Reordering parts changes the checksum.
  const d = await canonicalChecksum(["y", "x"]);
  assertStrictEquals(a === d, false);
});

Deno.test("basket: single-component full-coverage metric computes returns", async () => {
  const mapping = {
    id: "mapping-1",
    key: "crest_large_cap",
    sector: "CREST",
    minimumCoveredWeight: 80_000_000,
    normalizationCap: 2_000_000,
    benchmarkEligible: true,
    expectedTurbulence: {
      id: "crest_static",
      type: "STATIC_CONTENT_VALUE",
      staticValue: 4_000_000,
      floor: 250_000,
    },
    components: [
      {
        marketAssetId: "asset-a",
        targetWeight: 100_000_000,
        enabled: true,
        sequence: 0,
      },
    ],
  };
  const window = {
    id: "window-1",
    dailyTideId: "tide-1",
    providerKey: "provider-1",
    start: "2026-08-01T00:00:00Z",
    end: "2026-08-02T00:00:00Z",
    toleranceMs: 3_600_000,
  };
  const observations = [
    {
      id: "obs-open",
      providerKey: "provider-1",
      marketAssetId: "asset-a",
      observedAt: "2026-08-01T00:00:00Z",
      priceUnits: 100_000_000,
      status: "VALID" as const,
    },
    {
      id: "obs-close",
      providerKey: "provider-1",
      marketAssetId: "asset-a",
      observedAt: "2026-08-02T00:00:00Z",
      priceUnits: 110_000_000,
      status: "VALID" as const,
    },
  ];
  const metric = await new Calculator().calculate({
    mapping,
    window,
    observations,
  });
  assertStrictEquals(metric.status, MetricStatusReady);
  // 10% return on 100M weight => rawReturn 10_000_000 (10% of 1e8)
  assertStrictEquals(metric.rawReturn, 10_000_000);
  assertStrictEquals(metric.coveredWeight, 100_000_000);
  assertStrictEquals(metric.normalizedPerformance !== null, true);
  assertStrictEquals(metric.components[0]!.status, ComponentStatusValid);
  assertStrictEquals(metric.components[0]!.componentReturn, 10_000_000);
  assertStrictEquals(metric.inputChecksum.length, 64);
});

Deno.test("basket: missing component returns DATA_INCOMPLETE", async () => {
  const mapping = {
    id: "mapping-2",
    key: "two_asset",
    sector: "CURRENT",
    minimumCoveredWeight: 100_000_000,
    normalizationCap: 2_000_000,
    benchmarkEligible: false,
    expectedTurbulence: {
      id: "current_static",
      type: "STATIC_CONTENT_VALUE",
      staticValue: 8_000_000,
      floor: 500_000,
    },
    components: [
      {
        marketAssetId: "asset-a",
        targetWeight: 50_000_000,
        enabled: true,
        sequence: 0,
      },
      {
        marketAssetId: "asset-b",
        targetWeight: 50_000_000,
        enabled: true,
        sequence: 1,
      },
    ],
  };
  const window = {
    id: "window-2",
    dailyTideId: "tide-2",
    providerKey: "provider-2",
    start: "2026-08-01T00:00:00Z",
    end: "2026-08-02T00:00:00Z",
    toleranceMs: 3_600_000,
  };
  const observations = [
    {
      id: "obs-open",
      providerKey: "provider-2",
      marketAssetId: "asset-a",
      observedAt: "2026-08-01T00:00:00Z",
      priceUnits: 100_000_000,
      status: "VALID" as const,
    },
    {
      id: "obs-close",
      providerKey: "provider-2",
      marketAssetId: "asset-a",
      observedAt: "2026-08-02T00:00:00Z",
      priceUnits: 110_000_000,
      status: "VALID" as const,
    },
  ];
  const metric = await new Calculator().calculate({
    mapping,
    window,
    observations,
  });
  assertStrictEquals(metric.status, MetricStatusDataIncomplete);
  assertStrictEquals(metric.rawReturn, null);
  assertStrictEquals(metric.coveredWeight, 50_000_000);
  assertStrictEquals(metric.components[1]!.status, ComponentStatusMissingOpen);
});

Deno.test("sector: targetKeys are the four MVP sectors in order", () => {
  assertEquals(targetKeys(), ["CREST", "EMBER", "CURRENT", "HARBOR"]);
});

Deno.test("sector: benchmark computes equal-weight from eligible baskets", async () => {
  const definition = {
    id: "def-1",
    sector: "CREST" as const,
    benchmarkMethod: EqualWeight,
    minimumEligibleBaskets: 2,
    relativeScale: 500_000,
    relativeBlendWeight: 700_000,
    rankBlendWeight: 300_000,
    scoreCap: 1_000_000,
  };
  const baskets = [
    {
      basketMetricId: "metric-a",
      basketMappingVersionId: "mapping-a",
      sector: "CREST" as const,
      benchmarkEligible: true,
      metricStatus: "READY",
      normalizedPerformance: 200_000,
    },
    {
      basketMetricId: "metric-b",
      basketMappingVersionId: "mapping-b",
      sector: "CREST" as const,
      benchmarkEligible: true,
      metricStatus: "READY",
      normalizedPerformance: 300_000,
    },
  ];
  const benchmark = await new BenchmarkCalculator().calculate({
    dailyTideId: "tide-3",
    definition,
    baskets,
  });
  assertStrictEquals(benchmark.status, "READY");
  assertStrictEquals(benchmark.eligibleBasketCount, 2);
  // Equal weight: (200k + 300k) / 2 = 250k.
  assertStrictEquals(benchmark.value, 250_000);
  assertStrictEquals(benchmark.members.length, 2);
});

Deno.test("sector: score calculator applies blend and cap", () => {
  const definition = {
    id: "def-1",
    sector: "CREST" as const,
    benchmarkMethod: EqualWeight,
    minimumEligibleBaskets: 2,
    relativeScale: 500_000,
    relativeBlendWeight: 700_000,
    rankBlendWeight: 300_000,
    scoreCap: 1_000_000,
  };
  const output = new ScoreCalculator().score({
    definition,
    keeperNormalizedPerformance: 500_000,
    sectorBenchmark: 200_000,
    sectorPercentile: 750_000,
  });
  assertStrictEquals(output.relativePerformance, 300_000);
  // relativeComponent = 300k * 1e6 / 500k = 600k.
  assertStrictEquals(output.relativeComponent, 600_000);
  // rankComponent = (2 * 750k) - 1e6 = 500k.
  assertStrictEquals(output.rankComponent, 500_000);
  // normalized = 600k*(0.7) + 500k*(0.3) = 420k + 150k = 570k.
  assertStrictEquals(output.normalizedScore, 570_000);
  // points = 570k * 100 / 1e6 = 57.
  assertStrictEquals(output.scorePoints, 57);
});
