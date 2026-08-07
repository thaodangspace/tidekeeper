/** Argon2id password hashing, byte-compatible with the Go server's hashes. */

import { argon2id } from "hash-wasm";
import { newSalt } from "../utils/ids.ts";
import { base64UrlEncodeBytes, timingSafeEqual } from "../utils/crypto.ts";
import { TidekeepersError, unauthorized } from "../utils/errors.ts";

export const minPasswordLength = 8;
export const maxPasswordLength = 128;

const argonTime = 3;
const argonMemory = 64 * 1024;
const argonThreads = 4;
const argonKeyLength = 32;
const saltLength = 16;

const encodedPattern =
  /^\$argon2id\$v=19\$m=65536,t=3,p=4\$([A-Za-z0-9_-]+)\$([A-Za-z0-9_-]+)$/;

function validPassword(password: string): boolean {
  const length = Array.from(password).length;
  return length >= minPasswordLength && length <= maxPasswordLength;
}

/**
 * Derives an encoded Argon2id credential hash matching the Go format
 * `$argon2id$v=19$m=65536,t=3,p=4$<salt b64url>$<key b64url>`.
 */
export async function hashPassword(password: string): Promise<string> {
  if (!validPassword(password)) {
    throw new TidekeepersError(
      "invalid authentication input",
      "invalid_request",
      { code: "INVALID_AUTH_INPUT" },
    );
  }
  const salt = newSalt();
  const derived = await argon2id({
    password,
    salt,
    parallelism: argonThreads,
    iterations: argonTime,
    memorySize: argonMemory,
    hashLength: argonKeyLength,
    outputType: "binary",
  });
  return `$${
    [
      "argon2id",
      "v=19",
      `m=${argonMemory},t=${argonTime},p=${argonThreads}`,
      base64UrlEncodeBytes(salt),
      base64UrlEncodeBytes(derived),
    ].join("$")
  }`;
}

/** Compares a plaintext password with an encoded Argon2id hash. */
export async function verifyPassword(
  encoded: string,
  password: string,
): Promise<boolean> {
  const match = encodedPattern.exec(encoded);
  if (!match || !validPassword(password)) {
    return false;
  }
  const salt = base64UrlDecode(match[1]!);
  const expected = base64UrlDecode(match[2]!);
  if (salt.length !== saltLength || expected.length !== argonKeyLength) {
    return false;
  }
  const actual = await argon2id({
    password,
    salt,
    parallelism: argonThreads,
    iterations: argonTime,
    memorySize: argonMemory,
    hashLength: expected.length,
    outputType: "binary",
  });
  return timingSafeEqual(actual, expected);
}

export function invalidCredentials(): TidekeepersError {
  return unauthorized("invalid credentials", { code: "INVALID_CREDENTIALS" });
}

function base64UrlDecode(value: string): Uint8Array {
  const b64 = value.replace(/-/g, "+").replace(/_/g, "/");
  const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes;
}
