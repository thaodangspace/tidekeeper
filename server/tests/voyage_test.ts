import { assertEquals, assertThrows } from "@std/assert";
import { newHull } from "../src/domain/hull.ts";
import { applyHullMutation, isTerminal } from "../src/domain/voyage.ts";

const occurredAt = new Date("2026-07-30T12:00:00Z");

function voyage(status: "ACTIVE" | "COMPLETED" = "ACTIVE", current = 80) {
  return {
    status,
    hull: newHull(current, 100),
    completedAt: null,
  } as const;
}

Deno.test("voyage: active hull mutation preserves exact domain behavior", () => {
  const result = applyHullMutation(voyage(), {
    delta: -12,
    sourceType: "SETTLEMENT",
    sourceId: "tide_test",
    reasonKey: "settlement.score_deficit_damage",
  }, occurredAt);
  assertEquals(result.change.after, 68);
  assertEquals(result.voyage.status, "ACTIVE");
  assertEquals(result.voyage.completedAt, null);
});

Deno.test("voyage: lethal mutation fails with occurrence timestamp", () => {
  const result = applyHullMutation(voyage("ACTIVE", 8), {
    delta: -12,
    sourceType: "SETTLEMENT",
    sourceId: "tide_test",
    reasonKey: "settlement.score_deficit_damage",
  }, occurredAt);
  assertEquals(result.voyage.status, "FAILED");
  assertEquals(result.voyage.completedAt, occurredAt);
  assertEquals(result.change.destroyed, true);
});

Deno.test("voyage: terminal, zero, and no-effect mutations reject", () => {
  assertThrows(() =>
    applyHullMutation(voyage("COMPLETED"), {
      delta: -12,
      sourceType: "SETTLEMENT",
      sourceId: "tide_test",
      reasonKey: "test",
    }, occurredAt)
  );
  assertThrows(() =>
    applyHullMutation(voyage(), {
      delta: 0,
      sourceType: "SETTLEMENT",
      sourceId: "tide_test",
      reasonKey: "test",
    }, occurredAt)
  );
  assertThrows(() =>
    applyHullMutation(voyage("ACTIVE", 100), {
      delta: 10,
      sourceType: "REWARD",
      sourceId: "reward_test",
      reasonKey: "test",
    }, occurredAt)
  );
});

Deno.test("voyage: terminal status classification matches Go domain", () => {
  assertEquals(isTerminal("ACTIVE"), false);
  assertEquals(isTerminal("COMPLETED"), true);
  assertEquals(isTerminal("FAILED"), true);
  assertEquals(isTerminal("ABANDONED"), true);
  assertEquals(isTerminal("FUTURE"), false);
});
