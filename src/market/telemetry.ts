/** Infrastructure-neutral market calculation telemetry. */

export interface MarketCalculationEvent {
  operation: string;
  dailyTideId: string;
  sector: string;
  mappingId: string;
  status: string;
  coveredWeight: number;
  checksum: string;
  durationMs: number;
  error?: unknown;
}

export interface MarketRecorder {
  recordMarketCalculation(event: MarketCalculationEvent): void;
}

export class NopMarketRecorder implements MarketRecorder {
  recordMarketCalculation(_event: MarketCalculationEvent): void {
    // Intentionally empty when observability is not configured.
  }
}
