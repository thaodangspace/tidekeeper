import { assertEquals, assertThrows } from "@std/assert";
import {
  buildCutoverPlan,
  CutoverError,
  type CutoverInput,
} from "../src/content/cutover.ts";
import { buildInitialProjection } from "../src/content/voyage_definition.ts";

function baseProjection(): Record<string, unknown> {
  return buildInitialProjection({
    fleetSlotCount: 1,
    instances: [],
    offers: [],
  }) as Record<string, unknown>;
}

function input(overrides: Partial<CutoverInput> = {}): CutoverInput {
  return {
    sourceVersion: 1,
    targetVersion: 3,
    sourceCatalog: "v1",
    expectedModifierId: "modifier-target",
    expectedObjectiveId: "objective-target",
    expectedRulesetId: "ruleset-target",
    strategyByKey: new Map([["strategy-a", "strategy-target"]]),
    tides: [{ id: "tide-1", contentVersion: 1 }],
    projections: [{
      dailyTideId: "tide-1",
      stateId: "state-1",
      schemaVersion: 1,
      projection: baseProjection(),
    }],
    ...overrides,
  };
}

Deno.test("cutover planner: maps recognized projections and selected strategies", () => {
  const projection = baseProjection();
  projection.strategies = [{
    id: "strategy-a",
    name: "Strategy A",
    description: "A strategy",
    upside: "Up",
    downside: "Down",
    available: true,
  }];
  projection.selectedStrategyId = "strategy-a";
  const plan = buildCutoverPlan(input({
    projections: [{
      dailyTideId: "tide-1",
      stateId: "state-1",
      schemaVersion: 1,
      projection,
    }],
  }));
  assertEquals(plan.result.idempotent, false);
  assertEquals(plan.result.fingerprintLabels, ["migration-seeded-development"]);
  assertEquals(plan.result.tideCount, 1);
  assertEquals(plan.result.stateCount, 1);
  assertEquals(plan.result.strategyCount, 1);
  assertEquals(plan.availability.length, 1);
  assertEquals(plan.selections, [{
    stateId: "state-1",
    selectedStrategyDefinitionVersionId: "strategy-target",
  }]);
  assertEquals(plan.tideUpdates[0]?.contentVersion, 3);
});

Deno.test("cutover planner: completed target rows are idempotent", () => {
  const plan = buildCutoverPlan(input({
    tides: [{
      id: "tide-1",
      contentVersion: 3,
      modifierDefinitionVersionId: "modifier-target",
      objectiveDefinitionVersionId: "objective-target",
      gameRuleSetVersionId: "ruleset-target",
    }],
    projections: [],
  }));
  assertEquals(plan.result.idempotent, true);
  assertEquals(plan.tideUpdates, []);
});

Deno.test("cutover planner: fails closed for missing, contradictory, and unknown projections", () => {
  const missing = assertThrows(
    () => buildCutoverPlan(input({ projections: [] })),
    CutoverError,
  );
  assertEquals(missing.category, "missing-projections");

  const other = baseProjection();
  (other.objective as Record<string, unknown>).id = "different-objective";
  const contradiction = assertThrows(
    () =>
      buildCutoverPlan(input({
        projections: [
          input().projections[0]!,
          {
            dailyTideId: "tide-1",
            stateId: "state-2",
            schemaVersion: 1,
            projection: other,
          },
        ],
      })),
    CutoverError,
  );
  assertEquals(contradiction.category, "mixed-contradictory-projections");

  const unknown = baseProjection();
  (unknown.modifier as Record<string, unknown>).id = "unknown-modifier";
  const source = assertThrows(
    () =>
      buildCutoverPlan(input({
        projections: [{
          dailyTideId: "tide-1",
          stateId: "state-1",
          schemaVersion: 1,
          projection: unknown,
        }],
      })),
    CutoverError,
  );
  assertEquals(source.category, "unknown-source-fingerprint");
});
