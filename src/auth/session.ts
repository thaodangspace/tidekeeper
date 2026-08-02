/** Opaque session token generation and digest helpers (ported from auth/token.go). */

import { newSessionToken } from "../utils/ids.ts";
import { base64UrlEncodeBytes, sha256Hex } from "../utils/crypto.ts";

const tokenBytes = 32;

export interface NewTokenResult {
  token: string;
  digestHex: string;
}

/** Generates a 256-bit unpadded base64url token and its SHA-256 digest hex. */
export async function newToken(): Promise<NewTokenResult> {
  const { raw, token } = newSessionToken();
  return { token, digestHex: await sha256Hex(raw) };
}

/**
 * Validates and hashes a raw 256-bit unpadded base64url token. Returns null
 * for malformed tokens so callers can treat them as expired sessions.
 */
export async function digestToken(raw: string): Promise<string | null> {
  // Raw base64url for exactly 32 bytes is always 43 characters. Reject
  // padding and non-canonical encodings rather than letting `atob` accept
  // alternate spellings of a session token.
  if (!/^[A-Za-z0-9_-]{43}$/.test(raw)) {
    return null;
  }
  const b64 = raw.replace(/-/g, "+").replace(/_/g, "/");
  const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
  let decoded: Uint8Array;
  try {
    const binary = atob(padded);
    decoded = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
      decoded[i] = binary.charCodeAt(i);
    }
  } catch {
    return null;
  }
  if (
    decoded.length !== tokenBytes || base64UrlEncodeBytes(decoded) !== raw
  ) {
    return null;
  }
  return await sha256Hex(decoded);
}
