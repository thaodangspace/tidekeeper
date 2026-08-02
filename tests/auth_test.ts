/** Auth tests: password hashing roundtrip, token digest, service flows. */

import { assertEquals, assertStrictEquals } from "@std/assert";
import {
  hashPassword,
  invalidCredentials,
  verifyPassword,
} from "../src/auth/password.ts";
import { digestToken, newToken } from "../src/auth/session.ts";
import { AuthService } from "../src/auth/service.ts";
import { Store } from "../src/repositories/kv.ts";
import { AuthRepository } from "../src/repositories/auth_repository.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

async function assertRejectsCode(
  fn: () => Promise<unknown>,
  code: string,
): Promise<void> {
  try {
    await fn();
  } catch (err) {
    if (err instanceof TidekeepersError && err.code === code) {
      return;
    }
    throw new Error(
      `expected TidekeepersError code ${code}, got ${String(err)}`,
    );
  }
  throw new Error(
    `expected TidekeepersError code ${code}, but no error thrown`,
  );
}

Deno.test("password: roundtrip verifies", async () => {
  const encoded = await hashPassword("correct horse battery staple");
  assertStrictEquals(typeof encoded, "string");
  assertEquals(
    await verifyPassword(encoded, "correct horse battery staple"),
    true,
  );
  assertEquals(await verifyPassword(encoded, "wrong password"), false);
});

Deno.test("password: encoded format matches Go argon2id v19 shape", async () => {
  const encoded = await hashPassword("abcdefgh");
  assertStrictEquals(
    encoded.startsWith("$argon2id$v=19$m=65536,t=3,p=4$"),
    true,
  );
  const parts = encoded.split("$");
  assertEquals(parts.length, 6);
  // parts = ["", "argon2id", "v=19", "m=65536,t=3,p=4", salt, key].
  // Salt is 16 bytes, key is 32 bytes -> 22 and 43 unpadded base64url chars.
  assertEquals(parts[4]!.length, 22);
  assertEquals(parts[5]!.length, 43);
});

Deno.test("password: rejects short password in hashPassword", async () => {
  await assertRejectsCode(() => hashPassword("short"), "INVALID_AUTH_INPUT");
});

Deno.test("password: verify rejects malformed encodings", async () => {
  assertEquals(await verifyPassword("not-an-encoded-hash", "whatever"), false);
  assertEquals(
    await verifyPassword(
      "$argon2id$v=19$m=65536,t=3,p=4$AAAA$AAAA",
      "whatever",
    ),
    false,
  );
});

Deno.test("session: newToken produces raw base64url token and matching digest", async () => {
  const { token, digestHex } = await newToken();
  assertStrictEquals(digestHex.length, 64);
  const recomputed = await digestToken(token);
  assertEquals(recomputed, digestHex);
  // Malformed tokens do not digest.
  assertEquals(await digestToken("short"), null);
  assertEquals(await digestToken(token + "="), null);
});

Deno.test("session: digest matches direct sha256 of decoded raw bytes", async () => {
  const token = await newToken();
  const digest = await digestToken(token.token);
  const direct = await sha256OfHexRaw(token.token);
  assertEquals(digest, direct);
});

async function sha256OfHexRaw(token: string): Promise<string> {
  const b64 = token.replace(/-/g, "+").replace(/_/g, "/");
  const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
  const binary = atob(padded);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  const digest = await crypto.subtle.digest(
    "SHA-256",
    bytes.buffer as ArrayBuffer,
  );
  return Array.from(
    new Uint8Array(digest),
    (b) => b.toString(16).padStart(2, "0"),
  ).join("");
}

Deno.test("auth service: register issues session and persists account", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const repo = new AuthRepository(store);
    const service = new AuthService(repo, 30 * 24 * 60 * 60 * 1000);
    const now = new Date("2026-08-01T12:00:00Z");
    const session = await service.register(
      "Player@Example.com",
      "password-123",
      now,
    );
    assertStrictEquals(typeof session.token, "string");
    assertStrictEquals(session.expiresAt > now, true);
    const principal = await service.authenticate(
      session.token,
      new Date(now.getTime() + 1000),
    );
    assertStrictEquals(principal !== null, true);
    const account = await repo.findAccountByEmail("player@example.com");
    assertEquals(account?.email, "player@example.com");
  } finally {
    kv.close();
  }
});

Deno.test("auth service: duplicate email is a conflict", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const repo = new AuthRepository(store);
    const service = new AuthService(repo, 30 * 24 * 60 * 60 * 1000);
    const now = new Date("2026-08-01T12:00:00Z");
    await service.register("dup@example.com", "password-123", now);
    await assertRejectsCode(
      () => service.register("dup@example.com", "password-456", now),
      "EMAIL_ALREADY_REGISTERED",
    );
  } finally {
    kv.close();
  }
});

Deno.test("auth service: login verifies credentials and issues session", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const repo = new AuthRepository(store);
    const service = new AuthService(repo, 30 * 24 * 60 * 60 * 1000);
    const now = new Date("2026-08-01T12:00:00Z");
    await service.register("login@example.com", "password-123", now);
    const session = await service.login(
      "LOGIN@example.com",
      "password-123",
      now,
    );
    assertStrictEquals(typeof session.token, "string");
    const principal = await service.authenticate(
      session.token,
      new Date(now.getTime() + 1000),
    );
    assertStrictEquals(principal !== null, true);
    // Wrong password rejects with invalid credentials.
    await assertRejectsCode(
      () => service.login("login@example.com", "wrong-password", now),
      "INVALID_CREDENTIALS",
    );
  } finally {
    kv.close();
  }
});

Deno.test("auth service: expired session is not authenticated", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const store = new Store(kv);
    const repo = new AuthRepository(store);
    const service = new AuthService(repo, 30 * 24 * 60 * 60 * 1000);
    const now = new Date("2026-08-01T12:00:00Z");
    const session = await service.register(
      "exp@example.com",
      "password-123",
      now,
    );
    const past = new Date("2030-01-01T00:00:00Z");
    const principal = await service.authenticate(session.token, past);
    assertEquals(principal, null);
  } finally {
    kv.close();
  }
});

Deno.test("session invalid credentials error code", () => {
  const err = invalidCredentials();
  assertStrictEquals(err.code, "INVALID_CREDENTIALS");
});
