import { assertEquals, assertRejects } from "@std/assert";
import type { CalculationInput, Metric } from "../src/market/basket.ts";
import type { Benchmark, Definition } from "../src/market/sector.ts";
import { MarketRepository } from "../src/repositories/market_repository.ts";
import { Store } from "../src/repositories/kv.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

const input: CalculationInput = {
  mapping: {
    id: "mapping-1",
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
    dailyTideId: "tide-1",
    providerKey: "fixture",
    start: "2026-08-01T00:00:00Z",
    end: "2026-08-02T00:00:00Z",
    toleranceMs: 0,
  },
  observations: [],
};

const metric: Metric = {
  basketMappingVersionId: "mapping-1",
  sector: "CREST",
  status: "DATA_INCOMPLETE",
  coveredWeight: 0,
  rawReturn: null,
  expectedTurbulence: null,
  normalizedPerformance: null,
  components: [],
  inputChecksum: "checksum-1",
};

const definition: Definition = {
  id: "definition-1",
  sector: "CREST",
  benchmarkMethod: "EQUAL_WEIGHT",
  minimumEligibleBaskets: 2,
  relativeScale: 500_000,
  relativeBlendWeight: 700_000,
  rankBlendWeight: 300_000,
  scoreCap: 1_000_000,
};

const benchmark: Benchmark = {
  sector: "CREST",
  status: "DATA_INCOMPLETE",
  eligibleBasketCount: 0,
  value: null,
  members: [],
  inputChecksum: "benchmark-checksum",
};

Deno.test("market repository: calculation evidence is idempotent and immutable", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const repository = new MarketRepository(new Store(kv));
    const first = await repository.persistMetric(input, metric);
    const second = await repository.persistMetric(input, metric);
    assertEquals(first.idempotent, false);
    assertEquals(second, { id: first.id, idempotent: true });
    const different = { ...metric, inputChecksum: "different" };
    const error = await assertRejects<TidekeepersError>(
      () => repository.persistMetric(input, different),
      TidekeepersError,
    );
    assertEquals(error.code, "MARKET_METRIC_CONFLICT");
  } finally {
    kv.close();
  }
});

Deno.test("market repository: benchmark evidence is persisted by tide and definition", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const repository = new MarketRepository(new Store(kv));
    const result = await repository.persistBenchmark(
      "tide-1",
      definition,
      benchmark,
    );
    assertEquals(result.idempotent, false);
    const replay = await repository.persistBenchmark(
      "tide-1",
      definition,
      benchmark,
    );
    assertEquals(replay, { id: result.id, idempotent: true });
    const readiness = await repository.loadReadiness("tide-1", [definition]);
    assertEquals(readiness.ready, false);
    assertEquals(readiness.sectors.length, 4);
    assertEquals(
      (await repository.getBenchmark("tide-1", definition.id))?.id,
      result.id,
    );

    const scoreId = await repository.persistKeeperScore({
      dailyTideId: "tide-1",
      keeperDefinitionId: "keeper-1",
      basketMetricId: "metric-1",
      sectorBenchmarkId: result.id,
      percentile: 500_000,
      score: {
        relativePerformance: 10,
        relativeComponent: 20,
        rankComponent: 0,
        normalizedScore: 14,
        scorePoints: 1,
      },
    });
    assertEquals(
      (await repository.getKeeperScore("tide-1", "keeper-1"))?.id,
      scoreId,
    );
  } finally {
    kv.close();
  }
});
