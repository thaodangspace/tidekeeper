/**
 * SHA-256 digest helpers used for session tokens and canonical checksums.
 */

/** Returns the SHA-256 digest of the input bytes. */
export async function sha256(input: Uint8Array): Promise<Uint8Array> {
  const copy = new Uint8Array(input.length);
  copy.set(input);
  const digest = await crypto.subtle.digest("SHA-256", copy.buffer);
  return new Uint8Array(digest);
}

/** Returns the SHA-256 digest as a lowercase hex string. */
export async function sha256Hex(input: Uint8Array): Promise<string> {
  return bytesToHex(await sha256(input));
}

/** Returns the SHA-256 digest as a base64url-raw string. */
export async function sha256Base64Url(input: Uint8Array): Promise<string> {
  return base64UrlEncodeBytes(await sha256(input));
}

export function bytesToHex(bytes: Uint8Array): string {
  const alphabet = "0123456789abcdef";
  let out = "";
  for (const b of bytes) {
    out += alphabet[b >>> 4] + alphabet[b & 0x0f];
  }
  return out;
}

export function hexToBytes(hex: string): Uint8Array {
  if (hex.length % 2 !== 0) {
    throw new Error("invalid hex string");
  }
  const bytes = new Uint8Array(hex.length / 2);
  for (let i = 0; i < bytes.length; i++) {
    bytes[i] = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }
  return bytes;
}

export function base64UrlEncodeBytes(bytes: Uint8Array): string {
  let binary = "";
  for (const b of bytes) {
    binary += String.fromCharCode(b);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(
    /=+$/,
    "",
  );
}

/**
 * Constant-time equality for byte arrays of equal length. Returns false for
 * length mismatches without early-exiting on content comparison.
 */
export function timingSafeEqual(left: Uint8Array, right: Uint8Array): boolean {
  if (left.length !== right.length) {
    return false;
  }
  let difference = 0;
  for (let i = 0; i < left.length; i++) {
    difference |= left[i]! ^ right[i]!;
  }
  return difference === 0;
}
