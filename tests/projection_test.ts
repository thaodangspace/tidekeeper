import { assertEquals, assertThrows } from "@std/assert";
import {
  decodeAndValidateProjection,
  decodeProjectionIdentity,
  projectionsCompatible,
} from "../src/domain/projection.ts";
import { buildInitialProjection } from "../src/content/voyage_definition.ts";

function projection(): Record<string, unknown> {
  return buildInitialProjection({
    fleetSlotCount: 2,
    instances: [{
      publicId: "keeper-1",
      key: "crest_sovereign",
      name: "Crest Sovereign",
      sector: "CREST",
      role: "VANGUARD",
      rarity: "COMMON",
    }],
    offers: [{
      keeperKey: "harbor_warden",
      name: "Harbor Warden",
      sector: "HARBOR",
      role: "WARDEN",
      rarity: "COMMON",
      cost: 5,
    }],
  }) as Record<string, unknown>;
}

Deno.test("projection: validates schema 1 and extracts compatible identity", () => {
  const value = projection();
  const identity = decodeProjectionIdentity(1, value);
  assertEquals(identity.modifierId, "mod_default");
  assertEquals(identity.objectiveId, "obj_default");
  assertEquals(identity.strategyIds, []);
  assertEquals(identity.selectedStrategyId, null);
  assertEquals(projectionsCompatible(identity, { ...identity }), true);
  decodeAndValidateProjection(1, JSON.stringify(value));
});

Deno.test("projection: rejects unsupported schemas, unknown fields, and malformed slots", () => {
  assertThrows(() => decodeAndValidateProjection(2, projection()));
  const unknown = { ...projection(), unexpected: true };
  assertThrows(() => decodeAndValidateProjection(1, unknown));
  const malformed = projection();
  const lineup = malformed.lineup as Record<string, unknown>;
  lineup.slots = [{ index: 1, keeper: null }, { index: 1, keeper: null }];
  assertThrows(() => decodeAndValidateProjection(1, malformed));
});

Deno.test("projection: selected strategy must be available", () => {
  const value = projection();
  value.strategies = [{
    id: "strategy",
    name: "Strategy",
    description: "Description",
    upside: "Up",
    downside: "Down",
    available: false,
  }];
  value.selectedStrategyId = "strategy";
  assertThrows(() => decodeAndValidateProjection(1, value));
});
