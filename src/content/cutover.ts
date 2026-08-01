/** Pure legacy Daily Tide cutover planner. Persistence is intentionally separate. */

import {
  decodeProjectionIdentity,
  type ProjectionIdentity,
  projectionsCompatible,
} from "../domain/projection.ts";
import {
  recognizeSourceFingerprint,
  sourceCatalogForVersion,
} from "./releases.ts";

export interface CutoverParams {
  sourceVersion: number;
  targetVersion: number;
}

export interface CutoverTide {
  id: string;
  contentVersion: number;
  modifierDefinitionVersionId?: string | null;
  objectiveDefinitionVersionId?: string | null;
  gameRuleSetVersionId?: string | null;
}

export interface CutoverProjection {
  dailyTideId: string;
  stateId: string;
  schemaVersion: number;
  projection: unknown;
}

export interface CutoverInput extends CutoverParams {
  sourceCatalog?: string;
  expectedModifierId: string;
  expectedObjectiveId: string;
  expectedRulesetId: string;
  strategyByKey: ReadonlyMap<string, string>;
  tides: CutoverTide[];
  projections: CutoverProjection[];
}

export interface CutoverTideUpdate {
  id: string;
  contentVersion: number;
  modifierDefinitionVersionId: string;
  objectiveDefinitionVersionId: string;
  gameRuleSetVersionId: string;
}

export interface CutoverAvailability {
  dailyTideId: string;
  strategyDefinitionVersionId: string;
  contentVersion: number;
}

export interface CutoverSelection {
  stateId: string;
  selectedStrategyDefinitionVersionId: string;
}

export interface CutoverResult {
  sourceVersion: number;
  targetVersion: number;
  fingerprintLabels: string[];
  sourceCatalog: string;
  idempotent: boolean;
  tideCount: number;
  stateCount: number;
  strategyCount: number;
  availabilityCount: number;
  changedTideCount: number;
}

export interface CutoverPlan {
  result: CutoverResult;
  tideUpdates: CutoverTideUpdate[];
  availability: CutoverAvailability[];
  selections: CutoverSelection[];
}

export type CutoverErrorCategory =
  | "invalid-parameters"
  | "unknown-source-version"
  | "already-partially-mapped"
  | "missing-projections"
  | "invalid-projection"
  | "mixed-contradictory-projections"
  | "unknown-source-fingerprint"
  | "source-catalog-mismatch"
  | "unknown-strategy";

export class CutoverError extends Error {
  readonly category: CutoverErrorCategory;
  readonly tideIds: string[];
  readonly stateIds: string[];
  readonly strategyIds: string[];

  constructor(
    category: CutoverErrorCategory,
    detail: string,
    options: {
      tideIds?: string[];
      stateIds?: string[];
      strategyIds?: string[];
    } = {},
  ) {
    super(`cutover ${category}: ${detail}${formatIds(options)}`);
    this.name = "CutoverError";
    this.category = category;
    this.tideIds = [...(options.tideIds ?? [])].sort();
    this.stateIds = [...(options.stateIds ?? [])].sort();
    this.strategyIds = [...(options.strategyIds ?? [])].sort();
  }
}

/** Builds the same fail-closed plan as the legacy PostgreSQL cutover. */
export function buildCutoverPlan(input: CutoverInput): CutoverPlan {
  validateParams(input);
  const sourceCatalog = input.sourceCatalog ??
    sourceCatalogForVersion(input.sourceVersion);
  if (!sourceCatalog) {
    throw new CutoverError(
      "unknown-source-version",
      `source version ${input.sourceVersion} is not a recognized legacy source`,
    );
  }

  const legacy: CutoverTide[] = [];
  const inconsistent: CutoverTide[] = [];
  for (const tide of input.tides) {
    if (tide.contentVersion === input.sourceVersion && !hasAnyExact(tide)) {
      legacy.push(tide);
    } else if (
      tide.contentVersion === input.targetVersion &&
      isCompleteExact(tide, input)
    ) {
      // Already completed rows are intentionally ignored for idempotent retry.
    } else {
      inconsistent.push(tide);
    }
  }
  if (inconsistent.length > 0) {
    throw new CutoverError(
      "already-partially-mapped",
      "found Daily Tides that are not fully mapped to the target release",
      { tideIds: inconsistent.map((tide) => tide.id) },
    );
  }

  if (legacy.length === 0) {
    return emptyPlan(input, sourceCatalog);
  }

  const projectionsByTide = new Map<string, CutoverProjection[]>();
  for (const projection of input.projections) {
    const rows = projectionsByTide.get(projection.dailyTideId) ?? [];
    rows.push(projection);
    projectionsByTide.set(projection.dailyTideId, rows);
  }

  const fingerprintLabels = new Set<string>();
  const strategyIds = new Set<string>();
  const tideUpdates: CutoverTideUpdate[] = [];
  const availability: CutoverAvailability[] = [];
  const selections: CutoverSelection[] = [];
  const missingTides: string[] = [];
  const contradictionTides: string[] = [];
  const unknownSourceTides: string[] = [];
  const unknownStrategyIds: string[] = [];
  const unknownStrategyStates: string[] = [];
  const unknownStrategyTides: string[] = [];
  let stateCount = 0;

  for (const tide of legacy) {
    const rows = projectionsByTide.get(tide.id) ?? [];
    if (rows.length === 0) {
      missingTides.push(tide.id);
      continue;
    }
    const decoded: { row: CutoverProjection; identity: ProjectionIdentity }[] =
      [];
    for (const row of rows) {
      let identity: ProjectionIdentity;
      try {
        identity = decodeProjectionIdentity(row.schemaVersion, row.projection);
      } catch (error) {
        throw new CutoverError(
          "invalid-projection",
          `projection could not be decoded: ${
            error instanceof Error ? error.message : String(error)
          }`,
          { tideIds: [tide.id], stateIds: [row.stateId] },
        );
      }
      decoded.push({ row, identity });
    }
    const first = decoded[0]!.identity;
    if (
      !decoded.slice(1).every((item) =>
        projectionsCompatible(first, item.identity)
      )
    ) {
      contradictionTides.push(tide.id);
      continue;
    }

    const fingerprint = recognizeSourceFingerprint(input.sourceVersion, {
      modifierID: first.modifierId,
      objectiveID: first.objectiveId,
    });
    if (!fingerprint) {
      unknownSourceTides.push(tide.id);
      continue;
    }
    if (fingerprint.sourceCatalog !== sourceCatalog) {
      throw new CutoverError(
        "source-catalog-mismatch",
        `tide source fingerprint ${
          JSON.stringify(fingerprint.label)
        } expects catalog ${fingerprint.sourceCatalog}`,
        { tideIds: [tide.id] },
      );
    }
    fingerprintLabels.add(fingerprint.label);
    stateCount += decoded.length;

    for (const strategyKey of first.strategyIds) {
      const targetId = input.strategyByKey.get(strategyKey);
      if (!targetId) {
        unknownStrategyIds.push(strategyKey);
        unknownStrategyTides.push(tide.id);
        continue;
      }
      strategyIds.add(strategyKey);
      availability.push({
        dailyTideId: tide.id,
        strategyDefinitionVersionId: targetId,
        contentVersion: input.targetVersion,
      });
    }
    tideUpdates.push({
      id: tide.id,
      contentVersion: input.targetVersion,
      modifierDefinitionVersionId: input.expectedModifierId,
      objectiveDefinitionVersionId: input.expectedObjectiveId,
      gameRuleSetVersionId: input.expectedRulesetId,
    });
    for (const item of decoded) {
      const selected = item.identity.selectedStrategyId;
      if (!selected) continue;
      const targetId = input.strategyByKey.get(selected);
      if (!targetId) {
        unknownStrategyIds.push(selected);
        unknownStrategyStates.push(item.row.stateId);
        unknownStrategyTides.push(tide.id);
        continue;
      }
      selections.push({
        stateId: item.row.stateId,
        selectedStrategyDefinitionVersionId: targetId,
      });
    }
  }

  if (missingTides.length > 0) {
    throw new CutoverError(
      "missing-projections",
      "Daily Tide has no attached projections to recognize",
      { tideIds: missingTides },
    );
  }
  if (contradictionTides.length > 0) {
    throw new CutoverError(
      "mixed-contradictory-projections",
      "Daily Tide projections disagree on modifier, objective, or available strategy set",
      { tideIds: contradictionTides },
    );
  }
  if (unknownSourceTides.length > 0) {
    throw new CutoverError(
      "unknown-source-fingerprint",
      "projection identity is not a recognized legacy source",
      { tideIds: unknownSourceTides },
    );
  }
  if (unknownStrategyIds.length > 0) {
    throw new CutoverError(
      "unknown-strategy",
      "projection references a strategy not present in the target release",
      {
        tideIds: unknownStrategyTides,
        stateIds: unknownStrategyStates,
        strategyIds: unique(unknownStrategyIds),
      },
    );
  }

  return {
    result: {
      sourceVersion: input.sourceVersion,
      targetVersion: input.targetVersion,
      fingerprintLabels: [...fingerprintLabels].sort(),
      sourceCatalog,
      idempotent: false,
      tideCount: legacy.length,
      stateCount,
      strategyCount: strategyIds.size,
      availabilityCount: availability.length,
      changedTideCount: legacy.length,
    },
    tideUpdates,
    availability,
    selections,
  };
}

function validateParams(input: CutoverInput): void {
  if (
    input.sourceVersion <= 0 || input.targetVersion <= 0 ||
    input.sourceVersion === input.targetVersion
  ) {
    throw new CutoverError(
      "invalid-parameters",
      "source and target versions must be positive and differ",
    );
  }
}

function emptyPlan(input: CutoverInput, sourceCatalog: string): CutoverPlan {
  return {
    result: {
      sourceVersion: input.sourceVersion,
      targetVersion: input.targetVersion,
      fingerprintLabels: [],
      sourceCatalog,
      idempotent: true,
      tideCount: 0,
      stateCount: 0,
      strategyCount: 0,
      availabilityCount: 0,
      changedTideCount: 0,
    },
    tideUpdates: [],
    availability: [],
    selections: [],
  };
}

function hasAnyExact(tide: CutoverTide): boolean {
  return tide.modifierDefinitionVersionId != null ||
    tide.objectiveDefinitionVersionId != null ||
    tide.gameRuleSetVersionId != null;
}

function isCompleteExact(tide: CutoverTide, input: CutoverInput): boolean {
  return tide.modifierDefinitionVersionId === input.expectedModifierId &&
    tide.objectiveDefinitionVersionId === input.expectedObjectiveId &&
    tide.gameRuleSetVersionId === input.expectedRulesetId;
}

function unique(values: string[]): string[] {
  return [...new Set(values)];
}

function formatIds(options: {
  tideIds?: string[];
  stateIds?: string[];
  strategyIds?: string[];
}): string {
  const parts: string[] = [];
  if (options.tideIds?.length) {
    parts.push(` (tides: ${[...new Set(options.tideIds)].sort().join(", ")})`);
  }
  if (options.stateIds?.length) {
    parts.push(
      ` (states: ${[...new Set(options.stateIds)].sort().join(", ")})`,
    );
  }
  if (options.strategyIds?.length) {
    parts.push(
      ` (strategies: ${[...new Set(options.strategyIds)].sort().join(", ")})`,
    );
  }
  return parts.join("");
}
