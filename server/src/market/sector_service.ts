/** Sector benchmark orchestration and telemetry. */

import {
  type Benchmark,
  BenchmarkCalculator,
  type BenchmarkInput,
  type Definition,
} from "./sector.ts";
import { type MarketRecorder, NopMarketRecorder } from "./telemetry.ts";

export interface SectorBenchmarkRepository {
  persistBenchmark(
    dailyTideId: string,
    definition: Definition,
    benchmark: Benchmark,
  ): Promise<{ id: string; idempotent: boolean }>;
}

export class SectorBenchmarkService {
  #calculator = new BenchmarkCalculator();
  #repository: SectorBenchmarkRepository;
  #recorder: MarketRecorder;

  constructor(
    repository: SectorBenchmarkRepository,
    recorder: MarketRecorder = new NopMarketRecorder(),
  ) {
    this.#repository = repository;
    this.#recorder = recorder;
  }

  async calculateAndPersist(
    input: BenchmarkInput,
  ): Promise<{ benchmark: Benchmark; id: string; idempotent: boolean }> {
    const started = performance.now();
    let benchmark: Benchmark | null = null;
    try {
      benchmark = await this.#calculator.calculate(input);
      const persisted = await this.#repository.persistBenchmark(
        input.dailyTideId,
        input.definition,
        benchmark,
      );
      return { benchmark, ...persisted };
    } finally {
      this.#recorder.recordMarketCalculation({
        operation: "calculate_sector_benchmark",
        dailyTideId: input.dailyTideId,
        sector: input.definition.sector,
        mappingId: input.definition.id,
        status: benchmark?.status ?? "ERROR",
        coveredWeight: 0,
        checksum: benchmark?.inputChecksum ?? "",
        durationMs: performance.now() - started,
      });
    }
  }
}
