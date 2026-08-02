/** Immutable market-calculation evidence stored in Deno KV. */

import type { CalculationInput, Metric } from "../market/basket.ts";
import {
  type Benchmark,
  type Definition,
  evaluateReadiness,
  type Readiness,
} from "../market/sector.ts";
import { conflict } from "../utils/errors.ts";
import { newId } from "../utils/ids.ts";
import type { Store } from "./kv.ts";

export interface BasketMetricRecord {
  id: string;
  dailyTideId: string;
  mappingId: string;
  input: CalculationInput;
  metric: Metric;
  createdAt: string;
}

export interface SectorBenchmarkRecord {
  id: string;
  dailyTideId: string;
  definitionId: string;
  definition: Definition;
  benchmark: Benchmark;
  createdAt: string;
}

export interface KeeperSectorScore {
  id: string;
  dailyTideId: string;
  keeperDefinitionId: string;
  basketMetricId: string;
  sectorBenchmarkId: string;
  percentile: number;
  score: {
    relativePerformance: number;
    relativeComponent: number;
    rankComponent: number;
    normalizedScore: number;
    scorePoints: number;
  };
  createdAt: string;
}

export class MarketRepository {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async persistMetric(
    input: CalculationInput,
    metric: Metric,
    now = new Date(),
  ): Promise<{ id: string; idempotent: boolean }> {
    const key = basketMetricKey(
      input.window.dailyTideId,
      input.mapping.id,
    );
    const existing = await this.#store.get<BasketMetricRecord>(key);
    if (existing) {
      if (existing.metric.inputChecksum !== metric.inputChecksum) {
        throw conflict("basket metric scope has different persisted input", {
          code: "MARKET_METRIC_CONFLICT",
        });
      }
      return { id: existing.id, idempotent: true };
    }
    const record: BasketMetricRecord = {
      id: newId(),
      dailyTideId: input.window.dailyTideId,
      mappingId: input.mapping.id,
      input,
      metric,
      createdAt: now.toISOString(),
    };
    const committed = await this.#store.atomic()
      .check({ key, versionstamp: null })
      .set(key, record)
      .set(basketMetricIdKey(record.id), record)
      .commit();
    if (!committed.ok) {
      const winner = await this.#store.get<BasketMetricRecord>(key);
      if (winner && winner.metric.inputChecksum === metric.inputChecksum) {
        return { id: winner.id, idempotent: true };
      }
      throw conflict("basket metric scope has different persisted input", {
        code: "MARKET_METRIC_CONFLICT",
      });
    }
    return { id: record.id, idempotent: false };
  }

  async persistBenchmark(
    dailyTideId: string,
    definition: Definition,
    benchmark: Benchmark,
    now = new Date(),
  ): Promise<{ id: string; idempotent: boolean }> {
    const key = benchmarkKey(dailyTideId, definition.id);
    const existing = await this.#store.get<SectorBenchmarkRecord>(key);
    if (existing) {
      if (existing.benchmark.inputChecksum !== benchmark.inputChecksum) {
        throw conflict("sector benchmark scope has different persisted input", {
          code: "MARKET_BENCHMARK_CONFLICT",
        });
      }
      return { id: existing.id, idempotent: true };
    }
    const record: SectorBenchmarkRecord = {
      id: newId(),
      dailyTideId,
      definitionId: definition.id,
      definition,
      benchmark,
      createdAt: now.toISOString(),
    };
    const committed = await this.#store.atomic()
      .check({ key, versionstamp: null })
      .set(key, record)
      .set(benchmarkIdKey(record.id), record)
      .commit();
    if (!committed.ok) {
      const winner = await this.#store.get<SectorBenchmarkRecord>(key);
      if (
        winner && winner.benchmark.inputChecksum === benchmark.inputChecksum
      ) {
        return { id: winner.id, idempotent: true };
      }
      throw conflict("sector benchmark scope has different persisted input", {
        code: "MARKET_BENCHMARK_CONFLICT",
      });
    }
    return { id: record.id, idempotent: false };
  }

  async persistKeeperScore(
    score: Omit<KeeperSectorScore, "id" | "createdAt">,
    now = new Date(),
  ): Promise<string> {
    const key = keeperScoreKey(score.dailyTideId, score.keeperDefinitionId);
    const existing = await this.#store.get<KeeperSectorScore>(key);
    if (existing) {
      if (
        JSON.stringify(existing) !==
          JSON.stringify({
            ...score,
            id: existing.id,
            createdAt: existing.createdAt,
          })
      ) {
        throw conflict(
          "keeper sector score scope has different persisted input",
          {
            code: "MARKET_KEEPER_SCORE_CONFLICT",
          },
        );
      }
      return existing.id;
    }
    const record: KeeperSectorScore = {
      ...score,
      id: newId(),
      createdAt: now.toISOString(),
    };
    const committed = await this.#store.atomic()
      .check({ key, versionstamp: null })
      .set(key, record)
      .set(keeperScoreIdKey(record.id), record)
      .commit();
    if (!committed.ok) {
      const winner = await this.#store.get<KeeperSectorScore>(key);
      if (winner) return winner.id;
      throw conflict(
        "keeper sector score scope has different persisted input",
        {
          code: "MARKET_KEEPER_SCORE_CONFLICT",
        },
      );
    }
    return record.id;
  }

  async getBenchmark(
    dailyTideId: string,
    definitionId: string,
  ): Promise<SectorBenchmarkRecord | null> {
    return this.#store.get<SectorBenchmarkRecord>(
      benchmarkKey(dailyTideId, definitionId),
    );
  }

  async getKeeperScore(
    dailyTideId: string,
    keeperDefinitionId: string,
  ): Promise<KeeperSectorScore | null> {
    return this.#store.get<KeeperSectorScore>(
      keeperScoreKey(dailyTideId, keeperDefinitionId),
    );
  }

  async loadReadiness(
    dailyTideId: string,
    definitions: Definition[],
  ): Promise<Readiness> {
    const records = await this.#store.listValues<SectorBenchmarkRecord>([
      "market_benchmark",
      dailyTideId,
    ]);
    return evaluateReadiness(
      definitions,
      records.map((record) => record.benchmark),
    );
  }

  async getMetric(
    dailyTideId: string,
    mappingId: string,
  ): Promise<BasketMetricRecord | null> {
    return this.#store.get<BasketMetricRecord>(
      basketMetricKey(dailyTideId, mappingId),
    );
  }
}

function basketMetricKey(dailyTideId: string, mappingId: string): Deno.KvKey {
  return ["market_metric", dailyTideId, mappingId];
}

function basketMetricIdKey(id: string): Deno.KvKey {
  return ["market_metric_id", id];
}

function benchmarkKey(dailyTideId: string, definitionId: string): Deno.KvKey {
  return ["market_benchmark", dailyTideId, definitionId];
}

function benchmarkIdKey(id: string): Deno.KvKey {
  return ["market_benchmark_id", id];
}

function keeperScoreKey(dailyTideId: string, definitionId: string): Deno.KvKey {
  return ["market_keeper_score", dailyTideId, definitionId];
}

function keeperScoreIdKey(id: string): Deno.KvKey {
  return ["market_keeper_score_id", id];
}
