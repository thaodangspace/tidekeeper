import type {
  MarketCalculationEvent,
  MarketRecorder,
} from "../market/telemetry.ts";
import type { Logger } from "../utils/log.ts";

/** Writes bounded market calculation events to the structured logger. */
export class StructuredMarketRecorder implements MarketRecorder {
  #logger: Logger;

  constructor(logger: Logger) {
    this.#logger = logger;
  }

  recordMarketCalculation(event: MarketCalculationEvent): void {
    const fields = {
      operation: event.operation,
      dailyTideId: event.dailyTideId,
      sector: event.sector,
      mappingId: event.mappingId,
      status: event.status,
      coveredWeight: event.coveredWeight,
      checksum: event.checksum,
      durationMs: event.durationMs,
    };
    if (event.error !== undefined && event.error !== null) {
      this.#logger.error("market calculation failed", {
        ...fields,
        error: event.error instanceof Error
          ? event.error.message
          : String(event.error),
      });
      return;
    }
    this.#logger.info("market calculation completed", fields);
  }
}
