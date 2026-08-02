import { assertRejects, assertThrows } from "@std/assert";
import {
  type MarketObservationRecord,
  MarketObservationWriter,
  validateObservation,
  validateWindow,
} from "../src/market/observations.ts";
import { Store } from "../src/repositories/kv.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

const window = {
  id: "window-1",
  dailyTideId: "tide-1",
  providerKey: "fixture",
  start: "2026-08-01T00:00:00Z",
  end: "2026-08-02T00:00:00Z",
  requiredGranularitySeconds: 60,
  observationToleranceSeconds: 30,
};

const observation: MarketObservationRecord = {
  id: "observation-1",
  calculationWindowId: "window-1",
  marketAssetId: "asset-1",
  observedAt: "2026-08-01T12:00:00Z",
  rawPrice: "100.00",
  price: "100.00",
  qualityStatus: "VALID",
  qualityFlags: [],
  rawResponseHash: "a".repeat(64),
};

Deno.test("observations: validates and persists immutable evidence", async () => {
  validateWindow(window);
  validateObservation(observation);
  const kv = await Deno.openKv(":memory:");
  try {
    const writer = new MarketObservationWriter(new Store(kv));
    await writer.insertWindow(window);
    await writer.insertObservation(observation);
    await assertRejects(
      () => writer.insertObservation(observation),
      TidekeepersError,
    );
    const stored = await writer.getObservation(observation.id);
    if (!stored) throw new Error("observation was not persisted");
    stored.qualityFlags.push("MUTATED");
    const reread = await writer.getObservation(observation.id);
    if (!reread) throw new Error("observation disappeared");
    // KV returns a serialized value, so mutation of the read copy cannot alter storage.
    if (reread.qualityFlags.includes("MUTATED")) {
      throw new Error("stored observation was mutated through a read value");
    }
  } finally {
    kv.close();
  }
});

Deno.test("observations: valid observations require a positive price", () => {
  assertThrows(() =>
    validateObservation({
      ...observation,
      price: "0",
    })
  );
  assertThrows(() =>
    validateObservation({
      ...observation,
      rawResponseHash: "not-a-hash",
    })
  );
  assertThrows(() =>
    validateWindow({
      ...window,
      requiredGranularitySeconds: 0,
    })
  );
});

Deno.test("observations: duplicate window and observation IDs are conflicts", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const writer = new MarketObservationWriter(new Store(kv));
    await writer.insertWindow(window);
    const windowError = await assertRejects<TidekeepersError>(
      () => writer.insertWindow(window),
      TidekeepersError,
    );
    if (windowError.code !== "MARKET_WINDOW_CONFLICT") {
      throw new Error(`unexpected code ${windowError.code}`);
    }
    await writer.insertObservation(observation);
    const observationError = await assertRejects<TidekeepersError>(
      () => writer.insertObservation(observation),
      TidekeepersError,
    );
    if (observationError.code !== "MARKET_OBSERVATION_CONFLICT") {
      throw new Error(`unexpected code ${observationError.code}`);
    }
  } finally {
    kv.close();
  }
});
