/** Validation and identity extraction for daily projection schema 1. */

export const supportedProjectionSchemaVersion = 1;

export interface ProjectionIdentity {
  modifierId: string;
  modifierName: string;
  objectiveId: string;
  objectiveName: string;
  strategyIds: string[];
  selectedStrategyId: string | null;
}

/**
 * Parses and validates a persisted projection before it is exposed or used for
 * content identity. Unknown fields are rejected to match Go's
 * json.Decoder.DisallowUnknownFields behavior.
 */
export function decodeAndValidateProjection(
  schemaVersion: number,
  data: unknown,
): Record<string, unknown> {
  if (schemaVersion !== supportedProjectionSchemaVersion) {
    throw new Error(`unsupported projection schema version ${schemaVersion}`);
  }
  const value = typeof data === "string" ? parseJson(data) : data;
  if (!isObject(value)) throw new Error("empty projection data");
  validateProjection(value);
  return value;
}

export function decodeProjectionIdentity(
  schemaVersion: number,
  data: unknown,
): ProjectionIdentity {
  const projection = decodeAndValidateProjection(schemaVersion, data);
  const modifier = object(projection.modifier, "modifier");
  const objective = object(projection.objective, "objective");
  const strategies = array(projection.strategies, "strategies");
  const ids = strategies.map((value, index) =>
    requiredString(
      object(value, `strategies[${index}]`),
      "id",
      `strategies[${index}]`,
    )
  ).sort();
  const selected = optionalString(
    projection.selectedStrategyId,
    "selectedStrategyId",
  );
  return {
    modifierId: requiredString(modifier, "id", "modifier"),
    modifierName: requiredString(modifier, "name", "modifier"),
    objectiveId: requiredString(objective, "id", "objective"),
    objectiveName: requiredString(objective, "name", "objective"),
    strategyIds: ids,
    selectedStrategyId: selected || null,
  };
}

export function projectionsCompatible(
  left: ProjectionIdentity,
  right: ProjectionIdentity,
): boolean {
  return left.modifierId === right.modifierId &&
    left.objectiveId === right.objectiveId &&
    left.strategyIds.length === right.strategyIds.length &&
    left.strategyIds.every((id, index) => id === right.strategyIds[index]);
}

export function validateProjection(value: Record<string, unknown>): void {
  keys(value, [
    "modifier",
    "objective",
    "signals",
    "lineup",
    "inventory",
    "shop",
    "strategies",
    "selectedStrategyId",
    "pendingRewardCount",
  ], "projection");
  const modifier = object(value.modifier, "modifier");
  keys(modifier, ["id", "name", "description"], "modifier");
  requireNonEmpty(modifier, "id", "modifier");
  requireNonEmpty(modifier, "name", "modifier");
  requireNonEmpty(modifier, "description", "modifier");

  const objective = object(value.objective, "objective");
  keys(
    objective,
    ["id", "name", "description", "progressLabel", "rewardLabel"],
    "objective",
  );
  requireNonEmpty(objective, "id", "objective");
  requireNonEmpty(objective, "name", "objective");
  requireNonEmpty(objective, "description", "objective");
  optionalString(objective.progressLabel, "objective.progressLabel");
  optionalString(objective.rewardLabel, "objective.rewardLabel");

  const signals = array(value.signals, "signals");
  signals.forEach((entry, index) => {
    const path = `signals[${index}]`;
    const signal = object(entry, path);
    keys(signal, [
      "id",
      "name",
      "description",
      "direction",
      "strength",
      "observedFrom",
      "observedTo",
    ], path);
    for (
      const field of ["id", "name", "description", "direction", "strength"]
    ) {
      requireNonEmpty(signal, field, path);
    }
    const from = requiredDate(signal, "observedFrom", path);
    const to = requiredDate(signal, "observedTo", path);
    if (to < from) {
      throw new Error(`${path}.observedTo must not be before observedFrom`);
    }
  });

  const lineup = object(value.lineup, "lineup");
  keys(
    lineup,
    ["lockedAt", "maxSlots", "slots", "synergies", "warnings"],
    "lineup",
  );
  optionalDate(lineup.lockedAt, "lineup.lockedAt");
  const maxSlots = positiveInteger(lineup.maxSlots, "lineup.maxSlots");
  const slots = array(lineup.slots, "lineup.slots");
  if (slots.length !== maxSlots) {
    throw new Error(
      `lineup.slots length ${slots.length} does not match maxSlots ${maxSlots}`,
    );
  }
  slots.forEach((entry, index) => {
    const path = `lineup.slots[${index}]`;
    const slot = object(entry, path);
    keys(slot, ["index", "keeper"], path);
    if (slot.index !== index) {
      throw new Error(`${path}.index is ${String(slot.index)}, want ${index}`);
    }
    if (slot.keeper !== null) validateKeeper(slot.keeper, `${path}.keeper`);
  });
  validateSynergies(lineup.synergies);
  validateWarnings(lineup.warnings);

  const inventory = array(value.inventory, "inventory");
  inventory.forEach((entry, index) =>
    validateKeeper(entry, `inventory[${index}]`)
  );

  const shop = object(value.shop, "shop");
  keys(shop, ["offers", "refreshAt", "rerollCost", "rerollIndex"], "shop");
  const offers = array(shop.offers, "shop.offers");
  offers.forEach((entry, index) => {
    const path = `shop.offers[${index}]`;
    const offer = object(entry, path);
    keys(offer, ["id", "keeper", "cost", "available", "synergyHint"], path);
    requireNonEmpty(offer, "id", path);
    nonNegativeInteger(offer.cost, `${path}.cost`);
    validateKeeper(offer.keeper, `${path}.keeper`);
    if (typeof offer.available !== "boolean") {
      throw new Error(`${path}.available must be boolean`);
    }
    optionalString(offer.synergyHint, `${path}.synergyHint`);
  });
  optionalDate(shop.refreshAt, "shop.refreshAt");
  nonNegativeInteger(shop.rerollCost, "shop.rerollCost");
  nonNegativeInteger(shop.rerollIndex, "shop.rerollIndex");

  const strategies = array(value.strategies, "strategies");
  for (const [index, entry] of strategies.entries()) {
    const path = `strategies[${index}]`;
    const strategy = object(entry, path);
    keys(strategy, [
      "id",
      "name",
      "description",
      "upside",
      "downside",
      "available",
    ], path);
    for (const field of ["id", "name", "description", "upside", "downside"]) {
      requireNonEmpty(strategy, field, path);
    }
    if (typeof strategy.available !== "boolean") {
      throw new Error(`${path}.available must be boolean`);
    }
  }
  const selected = optionalString(
    value.selectedStrategyId,
    "selectedStrategyId",
  );
  if (selected) {
    const match = strategies.some((entry) => {
      const strategy = object(entry, "strategy");
      return strategy.id === selected && strategy.available === true;
    });
    if (!match) {
      throw new Error(
        `selectedStrategyId ${
          JSON.stringify(selected)
        } does not match an available strategy`,
      );
    }
  }
  nonNegativeInteger(value.pendingRewardCount, "pendingRewardCount");
}

function validateKeeper(value: unknown, path: string): void {
  const keeper = object(value, path);
  keys(keeper, [
    "id",
    "definitionId",
    "name",
    "level",
    "rarity",
    "role",
    "sector",
    "passiveSummary",
    "artworkUrl",
  ], path);
  for (
    const field of [
      "id",
      "definitionId",
      "name",
      "rarity",
      "role",
      "sector",
      "passiveSummary",
    ]
  ) {
    requireNonEmpty(keeper, field, path);
  }
  positiveInteger(keeper.level, `${path}.level`);
  optionalString(keeper.artworkUrl, `${path}.artworkUrl`);
}

function validateSynergies(value: unknown): void {
  const entries = array(value, "lineup.synergies");
  entries.forEach((entry, index) => {
    const path = `lineup.synergies[${index}]`;
    const synergy = object(entry, path);
    keys(synergy, [
      "id",
      "name",
      "description",
      "state",
      "currentCount",
      "requiredCount",
    ], path);
    for (const field of ["id", "name", "description", "state"]) {
      requireNonEmpty(synergy, field, path);
    }
    nonNegativeInteger(synergy.currentCount, `${path}.currentCount`);
    positiveInteger(synergy.requiredCount, `${path}.requiredCount`);
  });
}

function validateWarnings(value: unknown): void {
  const entries = array(value, "lineup.warnings");
  entries.forEach((entry, index) => {
    const path = `lineup.warnings[${index}]`;
    const warning = object(entry, path);
    keys(warning, ["code", "message", "severity"], path);
    for (const field of ["code", "message", "severity"]) {
      requireNonEmpty(warning, field, path);
    }
  });
}

function parseJson(value: string): unknown {
  try {
    return JSON.parse(value) as unknown;
  } catch (error) {
    throw new Error(
      `decode projection: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
  }
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function object(value: unknown, path: string): Record<string, unknown> {
  if (!isObject(value)) throw new Error(`${path} must be an object`);
  return value;
}

function array(value: unknown, path: string): unknown[] {
  if (!Array.isArray(value)) throw new Error(`${path} must not be null`);
  return value;
}

function keys(
  value: Record<string, unknown>,
  allowed: string[],
  path: string,
): void {
  const allowedSet = new Set(allowed);
  for (const key of Object.keys(value)) {
    if (!allowedSet.has(key)) throw new Error(`${path}.${key} is unknown`);
  }
}

function requiredString(
  value: Record<string, unknown>,
  field: string,
  path: string,
): string {
  if (typeof value[field] !== "string" || value[field] === "") {
    throw new Error(`${path}.${field} is required`);
  }
  return value[field] as string;
}

function requireNonEmpty(
  value: Record<string, unknown>,
  field: string,
  path: string,
): void {
  requiredString(value, field, path);
}

function optionalString(value: unknown, path: string): string | null {
  if (value === undefined || value === null) return null;
  if (typeof value !== "string") {
    throw new Error(`${path} must be a string or null`);
  }
  return value;
}

function requiredDate(
  value: Record<string, unknown>,
  field: string,
  path: string,
): number {
  const date = value[field];
  if (typeof date !== "string") throw new Error(`${path}.${field} is required`);
  const parsed = Date.parse(date);
  if (!Number.isFinite(parsed)) throw new Error(`${path}.${field} is required`);
  return parsed;
}

function optionalDate(value: unknown, path: string): number | null {
  if (value === undefined || value === null) return null;
  if (typeof value !== "string" || !Number.isFinite(Date.parse(value))) {
    throw new Error(`${path} must be a date or null`);
  }
  return Date.parse(value);
}

function positiveInteger(value: unknown, path: string): number {
  if (!Number.isInteger(value) || (value as number) <= 0) {
    throw new Error(`${path} must be positive`);
  }
  return value as number;
}

function nonNegativeInteger(value: unknown, path: string): number {
  if (!Number.isInteger(value) || (value as number) < 0) {
    throw new Error(`${path} must be non-negative`);
  }
  return value as number;
}
