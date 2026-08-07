/** Guest identity, session token, and concurrency tests. */

import { assertEquals, assertStrictEquals } from "@std/assert";
import { digestToken, newToken } from "../src/auth/session.ts";
import { AuthService } from "../src/auth/service.ts";
import {
  derivePlayerId,
  normalizeClientIp,
  normalizeUsername,
} from "../src/auth/identity.ts";
import { Store } from "../src/repositories/kv.ts";
import { AuthRepository } from "../src/repositories/auth_repository.ts";
import { TidekeepersError } from "../src/utils/errors.ts";

Deno.test("identity: username normalization", () => {
  assertEquals(normalizeUsername("Thao Dang"), "thao_dang");
  assertEquals(normalizeUsername("  Thao   Dang "), "thao_dang");
  assertEquals(normalizeUsername("Tide-Keeper"), "tide_keeper");
});

Deno.test("identity: invalid names are rejected", () => {
  for (const name of ["", "A", " ", "a\u0000b"]) {
    try {
      normalizeUsername(name);
      throw new Error("expected invalid name");
    } catch (error) {
      assertStrictEquals(
        (error as TidekeepersError).code,
        "INVALID_PLAYER_NAME",
      );
    }
  }
});

Deno.test("identity: IP representations are canonical", () => {
  assertEquals(normalizeClientIp("001.002.003.004"), "1.2.3.4");
  assertEquals(normalizeClientIp("2001:0db8:0:0:0:0:0:1"), "2001:db8::1");
  assertEquals(normalizeClientIp("[::1]"), "::1");
});

Deno.test("identity: HMAC ids are stable, opaque, and IP-sensitive", async () => {
  const a = await derivePlayerId("thao_dang", "192.0.2.1", "secret");
  assertEquals(a, await derivePlayerId("thao_dang", "192.0.2.1", "secret"));
  assertEquals(a.includes("192.0.2.1"), false);
  assertEquals(
    a === await derivePlayerId("thao_dang", "192.0.2.2", "secret"),
    false,
  );
  assertEquals(
    a === await derivePlayerId("other", "192.0.2.1", "secret"),
    false,
  );
});

Deno.test("session: token digest roundtrip", async () => {
  const token = await newToken();
  assertStrictEquals(await digestToken(token.token), token.digestHex);
  assertEquals(await digestToken("short"), null);
});

Deno.test("auth service: create, resume, and authenticate", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const repo = new AuthRepository(new Store(kv));
    const service = new AuthService(repo, 30 * 24 * 60 * 60 * 1000, "secret");
    const now = new Date("2026-08-01T12:00:00Z");
    const first = await service.createOrResumeSession(
      "Thao Dang",
      "192.0.2.1",
      now,
    );
    const second = await service.createOrResumeSession(
      "thao_dang",
      "192.0.2.1",
      now,
    );
    assertStrictEquals(first.created, true);
    assertStrictEquals(second.created, false);
    assertEquals(
      (await service.authenticate(first.token, now))?.playerId,
      (await service.authenticate(second.token, now))?.playerId,
    );
    assertEquals(
      (await service.createOrResumeSession("Thao Dang", "192.0.2.2", now))
        .playerId === first.playerId,
      false,
    );
  } finally {
    kv.close();
  }
});

Deno.test("auth service: concurrent first requests create one player", async () => {
  const kv = await Deno.openKv(":memory:");
  try {
    const repo = new AuthRepository(new Store(kv));
    const service = new AuthService(repo, 10000, "secret");
    const now = new Date("2026-08-01T12:00:00Z");
    const sessions = await Promise.all(
      Array.from(
        { length: 8 },
        () => service.createOrResumeSession("Concurrent", "192.0.2.3", now),
      ),
    );
    assertEquals(new Set(sessions.map((session) => session.playerId)).size, 1);
    assertEquals((await new Store(kv).listValues(["player"])).length, 1);
  } finally {
    kv.close();
  }
});
