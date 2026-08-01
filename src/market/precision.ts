/**
 * Fixed-point arithmetic for authoritative market calculations (ported from
 * market/precision). Values are scaled integers; binary floating point must
 * not be used for settlement inputs or outputs.
 */

export const WeightScale = 100_000_000;
export const ReturnScale = 100_000_000;
export const NormalizedScale = 1_000_000;
export const ScoreScale = 10_000;

const MAX_SAFE = Number.MAX_SAFE_INTEGER;

export function errDivisionByZero(): Error {
  return new Error("precision division by zero");
}

export function errOverflow(): Error {
  return new Error("precision overflow");
}

export function errInvalidRange(): Error {
  return new Error("precision invalid clamp range");
}

/** Returns left + right when the exact result fits in a safe integer. */
export function add(left: number, right: number): number {
  const result = BigInt(left) + BigInt(right);
  return checked(result);
}

/** Returns left - right when the exact result fits in a safe integer. */
export function subtract(left: number, right: number): number {
  const result = BigInt(left) - BigInt(right);
  return checked(result);
}

/** Calculates left*right/divisor, rounding half away from zero. */
export function multiplyDivide(
  left: number,
  right: number,
  divisor: number,
): number {
  if (divisor === 0) {
    throw errDivisionByZero();
  }
  const product = BigInt(left) * BigInt(right);
  return roundHalfAwayFromZero(product, BigInt(divisor));
}

/** Calculates numerator/divisor, rounding half away from zero. */
export function divide(numerator: number, divisor: number): number {
  if (divisor === 0) {
    throw errDivisionByZero();
  }
  return roundHalfAwayFromZero(BigInt(numerator), BigInt(divisor));
}

/** Bounds value inclusively; the range must be ordered. */
export function clamp(value: number, minimum: number, maximum: number): number {
  if (minimum > maximum) {
    throw errInvalidRange();
  }
  if (value < minimum) {
    return minimum;
  }
  if (value > maximum) {
    return maximum;
  }
  return value;
}

function roundHalfAwayFromZero(numerator: bigint, divisor: bigint): number {
  if (divisor === 0n) {
    throw errDivisionByZero();
  }
  const negative = numerator !== 0n && (numerator < 0n) !== (divisor < 0n);
  const absoluteNumerator = numerator < 0n ? -numerator : numerator;
  const absoluteDivisor = divisor < 0n ? -divisor : divisor;
  let quotient = absoluteNumerator / absoluteDivisor;
  const remainder = absoluteNumerator % absoluteDivisor;
  if (remainder * 2n >= absoluteDivisor) {
    quotient += 1n;
  }
  if (negative) {
    quotient = -quotient;
  }
  return checked(quotient);
}

function checked(value: bigint): number {
  if (value > BigInt(MAX_SAFE) || value < -BigInt(MAX_SAFE)) {
    throw errOverflow();
  }
  return Number(value);
}
