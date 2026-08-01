/** Voyage aggregate mutation rules ported from voyage/model.go. */

import { applyHullDelta, type Hull, type HullChange } from "./hull.ts";

export type VoyageStatus = "ACTIVE" | "COMPLETED" | "FAILED" | "ABANDONED";

export type HullSourceType =
  | "SETTLEMENT"
  | "DAILY_OBJECTIVE"
  | "REWARD"
  | "RELIC"
  | "SYNERGY"
  | "BOSS"
  | "EVENT"
  | "VOYAGE_START"
  | "ADMIN_REPAIR";

export interface VoyageAggregate {
  status: VoyageStatus;
  hull: Hull;
  completedAt: Date | null;
}

export interface HullMutation {
  delta: number;
  sourceType: HullSourceType;
  sourceId: string;
  reasonKey: string;
}

export interface VoyageMutationResult {
  voyage: VoyageAggregate;
  change: HullChange;
}

export function applyHullMutation(
  voyage: VoyageAggregate,
  mutation: HullMutation,
  occurredAt: Date,
): VoyageMutationResult {
  if (voyage.status !== "ACTIVE") {
    throw new Error("voyage is not active");
  }
  if (!Number.isInteger(mutation.delta) || mutation.delta === 0) {
    throw new Error("hull delta must not be zero");
  }
  const result = applyHullDelta(voyage.hull, mutation.delta);
  if (result.change.effectiveDelta === 0) {
    throw new Error("hull mutation has no effect");
  }
  return {
    voyage: {
      ...voyage,
      hull: result.hull,
      status: result.change.destroyed ? "FAILED" : voyage.status,
      completedAt: result.change.destroyed ? occurredAt : voyage.completedAt,
    },
    change: result.change,
  };
}

export function isTerminal(status: VoyageStatus | string): boolean {
  return status === "COMPLETED" || status === "FAILED" ||
    status === "ABANDONED";
}
