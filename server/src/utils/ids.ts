/**
 * Random identifier and secret helpers.
 *
 * Mirrors the Go server: session tokens are 32 random bytes base64url-raw
 * encoded, and the stored digest is the SHA-256 of the raw token bytes.
 */

/** Encodes bytes as base64url without padding (RFC 4648, "raw" variant). */
export function base64UrlEncode(bytes: Uint8Array): string {
  let binary = "";
  for (const b of bytes) {
    binary += String.fromCharCode(b);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(
    /=+$/,
    "",
  );
}

/** Decodes raw base64url back to bytes. */
export function base64UrlDecode(value: string): Uint8Array {
  const b64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}

const HEX_ALPHABET = "0123456789abcdef";

function randomBytesHex(length: number): string {
  const bytes = crypto.getRandomValues(new Uint8Array(length));
  let out = "";
  for (const b of bytes) {
    out += HEX_ALPHABET[b >>> 4] + HEX_ALPHABET[b & 0x0f];
  }
  return out;
}

/**
 * Generates a random public ID. Kept as a hex string (32 hex chars = 16
 * bytes) to match the Go server's ids.New() output shape.
 */
export function newId(): string {
  return randomBytesHex(16);
}

/** Generates a random raw session token (32 bytes) and its base64url encoding. */
export function newSessionToken(): { raw: Uint8Array; token: string } {
  const raw = crypto.getRandomValues(new Uint8Array(32));
  return { raw, token: base64UrlEncode(raw) };
}

/** Generates a random salt for password hashing (16 bytes). */
export function newSalt(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(16));
}
