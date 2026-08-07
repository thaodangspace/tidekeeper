import { assertEquals, assertThrows } from "@std/assert";
import {
  applyHullDelta,
  hullRatioBasisPoints,
  hullState,
  newHull,
} from "../src/domain/hull.ts";

Deno.test("hull: validates bounds", () => {
  assertEquals(newHull(80, 100), { current: 80, maximum: 100 });
  assertEquals(newHull(0, 100).current, 0);
  assertThrows(() => newHull(0, 0));
  assertThrows(() => newHull(-1, 100));
  assertThrows(() => newHull(101, 100));
});

Deno.test("hull: applies bounded integer deltas", () => {
  const cases = [
    [80, 100, -12, 68, -12, false, false],
    [8, 100, -12, 0, -8, true, true],
    [70, 100, 10, 80, 10, false, false],
    [95, 100, 10, 100, 5, true, false],
    [
      2_147_483_637,
      2_147_483_647,
      2_147_483_647,
      2_147_483_647,
      10,
      true,
      false,
    ],
    [10, 100, -2_147_483_648, 0, -10, true, true],
  ] as const;
  for (
    const [current, maximum, delta, after, effective, capped, destroyed]
      of cases
  ) {
    const result = applyHullDelta(newHull(current, maximum), delta);
    assertEquals(result.change.after, after);
    assertEquals(result.change.effectiveDelta, effective);
    assertEquals(result.change.capped, capped);
    assertEquals(result.change.destroyed, destroyed);
  }
});

Deno.test("hull: state thresholds match Go domain", () => {
  assertEquals(hullState(newHull(61, 100)), "STABLE");
  assertEquals(hullState(newHull(60, 100)), "DAMAGED");
  assertEquals(hullState(newHull(26, 100)), "DAMAGED");
  assertEquals(hullState(newHull(25, 100)), "CRITICAL");
  assertEquals(hullState(newHull(1, 100)), "CRITICAL");
  assertEquals(hullState(newHull(0, 100)), "DESTROYED");
});

Deno.test("hull: ratio uses integer truncation without overflow", () => {
  assertEquals(hullRatioBasisPoints(newHull(1, 3)), 3_333);
  assertEquals(
    hullRatioBasisPoints(newHull(2_147_483_647, 2_147_483_647)),
    10_000,
  );
});
