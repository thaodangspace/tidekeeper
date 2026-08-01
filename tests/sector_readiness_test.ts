import { assertEquals } from "@std/assert";
import {
  type Benchmark,
  type Definition,
  evaluateReadiness,
  StatusReady,
  targetKeys,
} from "../src/market/sector.ts";

const definitions: Definition[] = targetKeys().map((sector, index) => ({
  id: `definition-${index}`,
  sector,
  benchmarkMethod: "EQUAL_WEIGHT",
  minimumEligibleBaskets: 2,
  relativeScale: 500_000,
  relativeBlendWeight: 700_000,
  rankBlendWeight: 300_000,
  scoreCap: 1_000_000,
}));

const benchmarks: Benchmark[] = targetKeys().map((sector) => ({
  sector,
  status: StatusReady,
  eligibleBasketCount: 2,
  value: 100,
  members: [],
  inputChecksum: `${sector}-checksum`,
}));

Deno.test("sector readiness: all target sectors must have ready benchmarks", () => {
  const ready = evaluateReadiness(definitions, benchmarks);
  assertEquals(ready.ready, true);
  assertEquals(ready.sectors.length, 4);

  const incomplete = evaluateReadiness(definitions, benchmarks.slice(0, 3));
  assertEquals(incomplete.ready, false);
  assertEquals(incomplete.sectors[3]?.ready, false);
  assertEquals(incomplete.sectors[3]?.status, null);
});

Deno.test("sector readiness: insufficient baskets is not ready", () => {
  const result = evaluateReadiness(
    definitions,
    benchmarks.map((benchmark, index) =>
      index === 0 ? { ...benchmark, eligibleBasketCount: 1 } : benchmark
    ),
  );
  assertEquals(result.ready, false);
  assertEquals(result.sectors[0]?.minimumRequiredBaskets, 2);
});
