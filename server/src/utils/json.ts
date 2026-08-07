/**
 * JSON helpers: strict decoding that rejects unknown fields (mirroring Go's
 * DisallowUnknownFields) and deterministic canonical JSON used by checksums.
 */

/** Parses JSON strictly, rejecting trailing data and unknown fields. */
export function parseStrict<T>(
  text: string,
  allowedFields: ReadonlySet<string>,
): T {
  let value: unknown;
  try {
    value = JSON.parse(text);
  } catch (error) {
    throw new TypeError(`invalid JSON: ${(error as Error).message}`);
  }
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new TypeError("expected a JSON object");
  }
  for (const key of Object.keys(value)) {
    if (!allowedFields.has(key)) {
      throw new TypeError(`unknown field ${key}`);
    }
  }
  return value as T;
}

/**
 * Re-encodes a JSON value with sorted object keys and number-preserving
 * decoding, matching the canonical JSON used by Keeper catalog checksums.
 */
export function canonicalJson(raw: string): string {
  const decoded = JSON.parse(raw);
  return JSON.stringify(decoded);
}

/** Returns true when the JSON value is a non-array object. */
export function isJsonObject(text: string): boolean {
  try {
    const value = JSON.parse(text);
    return value !== null && typeof value === "object" && !Array.isArray(value);
  } catch {
    return false;
  }
}
