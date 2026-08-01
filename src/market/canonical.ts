/** Deterministic ordering and checksum primitives (ported from market/canonical). */

import { bytesToHex } from "../utils/crypto.ts";

/** Returns a sorted copy of values without mutating caller input. */
export function sortedStrings(values: string[]): string[] {
  return [...values].sort();
}

/**
 * Hashes an unambiguous length-prefixed sequence of canonical parts and
 * returns the hex-encoded SHA-256 digest.
 */
export async function canonicalChecksum(parts: string[]): Promise<string> {
  const encoder = new TextEncoder();
  const chunks: Uint8Array[] = [];
  const lengthBuffer = new Uint8Array(8);
  const dataView = new DataView(lengthBuffer.buffer);
  for (const part of parts) {
    dataView.setBigUint64(0, BigInt(part.length), false);
    chunks.push(lengthBuffer);
    chunks.push(encoder.encode(part));
  }
  const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  const combined = new Uint8Array(totalLength);
  let offset = 0;
  for (const chunk of chunks) {
    combined.set(chunk, offset);
    offset += chunk.length;
  }
  const digest = await crypto.subtle.digest("SHA-256", combined);
  return bytesToHex(new Uint8Array(digest));
}
