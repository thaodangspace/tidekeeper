/** Bounded Hull resource rules ported from voyage/hull.go. */

export type HullState = "STABLE" | "DAMAGED" | "CRITICAL" | "DESTROYED";

export interface Hull {
  current: number;
  maximum: number;
}

export interface HullChange {
  before: number;
  requestedDelta: number;
  effectiveDelta: number;
  after: number;
  capped: boolean;
  destroyed: boolean;
}

export function newHull(current: number, maximum: number): Hull {
  if (!Number.isInteger(maximum) || maximum <= 0) {
    throw new Error("maximum hull must be greater than zero");
  }
  if (!Number.isInteger(current) || current < 0 || current > maximum) {
    throw new Error("current hull must be between zero and maximum");
  }
  return { current, maximum };
}

export function applyHullDelta(hull: Hull, delta: number): {
  hull: Hull;
  change: HullChange;
} {
  if (!Number.isInteger(delta)) {
    throw new Error("hull delta must be an integer");
  }
  const requestedAfter = BigInt(hull.current) + BigInt(delta);
  const maximum = BigInt(hull.maximum);
  const after = requestedAfter < 0n
    ? 0n
    : requestedAfter > maximum
    ? maximum
    : requestedAfter;
  const nextCurrent = Number(after);
  const effectiveDelta = nextCurrent - hull.current;
  return {
    hull: { current: nextCurrent, maximum: hull.maximum },
    change: {
      before: hull.current,
      requestedDelta: delta,
      effectiveDelta,
      after: nextCurrent,
      capped: effectiveDelta !== delta,
      destroyed: nextCurrent === 0,
    },
  };
}

export function hullState(hull: Hull): HullState {
  if (hull.current === 0) return "DESTROYED";
  const ratioBasisPoints = hullRatioBasisPoints(hull);
  if (ratioBasisPoints <= 2_500) return "CRITICAL";
  if (ratioBasisPoints <= 6_000) return "DAMAGED";
  return "STABLE";
}

export function hullRatioBasisPoints(hull: Hull): number {
  return Number((BigInt(hull.current) * 10_000n) / BigInt(hull.maximum));
}
