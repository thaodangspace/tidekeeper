/** Basket calculation orchestration and evidence persistence boundary. */

import { type CalculationInput, Calculator, type Metric } from "./basket.ts";
import { type MarketRecorder, NopMarketRecorder } from "./telemetry.ts";

export interface BasketMetricRepository {
  persistMetric(
    input: CalculationInput,
    metric: Metric,
  ): Promise<{ id: string; idempotent: boolean }>;
}

export class BasketCalculationService {
  #calculator = new Calculator();
  #repository: BasketMetricRepository;
  #recorder: MarketRecorder;

  constructor(
    repository: BasketMetricRepository,
    recorder: MarketRecorder = new NopMarketRecorder(),
  ) {
    this.#repository = repository;
    this.#recorder = recorder;
  }

  async calculateAndPersist(
    input: CalculationInput,
  ): Promise<{ metric: Metric; id: string; idempotent: boolean }> {
    const started = performance.now();
    let metric: Metric | null = null;
    try {
      metric = await this.#calculator.calculate(input);
      const persisted = await this.#repository.persistMetric(input, metric);
      return { metric, ...persisted };
    } finally {
      this.#recorder.recordMarketCalculation({
        operation: "calculate_basket_metrics",
        dailyTideId: input.window.dailyTideId,
        sector: input.mapping.sector,
        mappingId: input.mapping.id,
        status: metric?.status ?? "ERROR",
        coveredWeight: metric?.coveredWeight ?? 0,
        checksum: metric?.inputChecksum ?? "",
        durationMs: performance.now() - started,
      });
    }
  }
}
