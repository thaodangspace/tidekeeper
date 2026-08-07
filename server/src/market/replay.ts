/** Deterministic market calculation replay verification. */

import { type CalculationInput, Calculator, type Metric } from "./basket.ts";
import {
  type Benchmark,
  BenchmarkCalculator,
  type BenchmarkInput,
} from "./sector.ts";

export const replayMismatchMessage = "calculation replay mismatch";

export async function verifyBasket(
  input: CalculationInput,
  expected: Metric,
): Promise<void> {
  const actual = await new Calculator().calculate(input);
  if (!deepEqual(actual, expected)) {
    throw new Error(replayMismatchMessage);
  }
}

export async function verifyBenchmark(
  input: BenchmarkInput,
  expected: Benchmark,
): Promise<void> {
  const actual = await new BenchmarkCalculator().calculate(input);
  if (!deepEqual(actual, expected)) {
    throw new Error(replayMismatchMessage);
  }
}

function deepEqual(left: unknown, right: unknown): boolean {
  if (Object.is(left, right)) return true;
  if (typeof left !== typeof right || left === null || right === null) {
    return false;
  }
  if (Array.isArray(left) || Array.isArray(right)) {
    if (
      !Array.isArray(left) || !Array.isArray(right) ||
      left.length !== right.length
    ) {
      return false;
    }
    return left.every((value, index) => deepEqual(value, right[index]));
  }
  if (typeof left === "object" && typeof right === "object") {
    const leftRecord = left as Record<string, unknown>;
    const rightRecord = right as Record<string, unknown>;
    const leftKeys = Object.keys(leftRecord).sort();
    const rightKeys = Object.keys(rightRecord).sort();
    return leftKeys.length === rightKeys.length &&
      leftKeys.every((key, index) =>
        key === rightKeys[index] && deepEqual(leftRecord[key], rightRecord[key])
      );
  }
  return false;
}
