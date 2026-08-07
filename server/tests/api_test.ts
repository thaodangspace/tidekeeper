/** End-to-end HTTP handler tests through the Hono router against in-memory KV. */

import { assertEquals, assertStrictEquals } from "@std/assert";
import { Store } from "../src/repositories/kv.ts";
import { AuthRepository } from "../src/repositories/auth_repository.ts";
import { PlayerRepository } from "../src/repositories/player_repository.ts";
import { VoyageRepository } from "../src/repositories/voyage_repository.ts";
import { DailyRepository } from "../src/repositories/daily_repository.ts";
import { IdempotencyRepository } from "../src/repositories/idempotency_repository.ts";
import { AuthService } from "../src/auth/service.ts";
import { PlayerService } from "../src/services/player_service.ts";
import { DailyService } from "../src/services/daily_service.ts";
import { VoyageService } from "../src/services/voyage_service.ts";
import { FleetService } from "../src/services/fleet_service.ts";
import { ShopService } from "../src/services/shop_service.ts";
import { SettlementService } from "../src/services/settlement_service.ts";
import { buildRouter } from "../src/http/router.ts";
import type { Config } from "../src/config.ts";
import { ensureDailyTide } from "../src/jobs/cron.ts";

const config: Config = {
  environment: "test",
  http: {
    address: ":8080",
    readTimeoutMs: 10_000,
    writeTimeoutMs: 15_000,
    shutdownTimeoutMs: 15_000,
  },
  session: {
    cookieName: "tidekeepers_session",
    cookieSecure: false,
    cookieDomain: "",
    ttlMs: 30 * 24 * 60 * 60 * 1000,
  },
  rateLimit: { requests: 120, windowMs: 60_000 },
  logLevel: "error",
  serviceVersion: "test",
  gameTimezone: "UTC",
  lockTime: "23:55",
  contentVersion: 1,
  market: { provider: "static", apiKey: "" },
  playerIdSecret: "test-player-id-secret",
  proxy: { trusted: false, header: "x-forwarded-for" },
};

const base = "http://tidekeepers.test";

interface ApiHarness {
  kv: Deno.Kv;
  app: ReturnType<typeof buildRouter>;
}

async function setup(): Promise<ApiHarness> {
  const kv = await Deno.openKv(":memory:");
  const store = new Store(kv);
  await ensureDailyTide(config, new DailyRepository(store), new Date());

  const authRepository = new AuthRepository(store);
  const playerRepository = new PlayerRepository(store);
  const voyageRepository = new VoyageRepository(store);
  const dailyRepository = new DailyRepository(store);
  const idempotencyRepository = new IdempotencyRepository(store);

  const auth = new AuthService(authRepository, config.session.ttlMs);
  const players = new PlayerService(playerRepository);
  const daily = new DailyService(dailyRepository, voyageRepository);
  const voyages = new VoyageService(
    store,
    voyageRepository,
    dailyRepository,
    idempotencyRepository,
    config.gameTimezone,
  );
  const fleet = new FleetService(dailyRepository, voyageRepository);
  const shop = new ShopService(store, dailyRepository, voyageRepository);
  const settlement = new SettlementService(
    store,
    dailyRepository,
    voyageRepository,
  );
  const app = buildRouter({
    config,
    auth,
    players,
    daily,
    voyages,
    fleet,
    shop,
    settlement,
    ready: () => true,
    log: () => {},
  });
  return { kv, app };
}

async function register(
  app: ReturnType<typeof buildRouter>,
  username = "Thao Dang",
): Promise<string> {
  const res = await app.request(`${base}/auth/session`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ username }),
  }, { clientIp: "192.0.2.1" });
  assertStrictEquals(res.status, 201);
  const setCookie = res.headers.get("set-cookie") ?? "";
  const match = /tidekeepers_session=([^;]+)/.exec(setCookie);
  assertStrictEquals(match !== null, true);
  return match![1]!;
}

async function json(res: Response): Promise<Record<string, unknown>> {
  return (await res.json()) as Record<string, unknown>;
}

Deno.test("api: health endpoints", async () => {
  const h = await setup();
  try {
    const live = await h.app.request(`${base}/health/live`);
    assertStrictEquals(live.status, 200);
    const ready = await h.app.request(`${base}/health/ready`);
    assertStrictEquals(ready.status, 200);
  } finally {
    h.kv.close();
  }
});

Deno.test("api: register sets cookie and me returns player", async () => {
  const h = await setup();
  try {
    const cookie = await register(h.app);
    const me = await h.app.request(`${base}/api/v1/me`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(me.status, 200);
    const body = await json(me);
    assertStrictEquals(typeof body.playerId, "string");
    assertStrictEquals(body.onboardingCompleted, false);
    assertStrictEquals(body.activeVoyageId, null);
  } finally {
    h.kv.close();
  }
});

Deno.test("api: protected route without cookie is unauthorized", async () => {
  const h = await setup();
  try {
    const res = await h.app.request(`${base}/api/v1/me`);
    assertStrictEquals(res.status, 401);
    const body = await json(res);
    assertStrictEquals(body.code, "AUTH_REQUIRED");
  } finally {
    h.kv.close();
  }
});

Deno.test("api: repeated name/IP resumes the player with a fresh session", async () => {
  const h = await setup();
  try {
    const firstCookie = await register(h.app);
    const resumed = await h.app.request(`${base}/auth/session`, {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({ username: "  thao   dang " }),
    }, { clientIp: "192.0.2.1" });
    assertStrictEquals(resumed.status, 204);
    const secondCookie = /tidekeepers_session=([^;]+)/.exec(
      resumed.headers.get("set-cookie") ?? "",
    )![1]!;
    const firstMe = await h.app.request(`${base}/api/v1/me`, {
      headers: { cookie: `tidekeepers_session=${firstCookie}` },
    });
    const secondMe = await h.app.request(`${base}/api/v1/me`, {
      headers: { cookie: `tidekeepers_session=${secondCookie}` },
    });
    assertStrictEquals(
      (await json(firstMe)).playerId,
      (await json(secondMe)).playerId,
    );
  } finally {
    h.kv.close();
  }
});

Deno.test("api: removed account endpoints are not exposed", async () => {
  const h = await setup();
  try {
    const registerResponse = await h.app.request(`${base}/auth/register`, {
      method: "POST",
    });
    const loginResponse = await h.app.request(`${base}/auth/login`, {
      method: "POST",
    });
    assertStrictEquals(registerResponse.status, 404);
    assertStrictEquals(loginResponse.status, 404);
  } finally {
    h.kv.close();
  }
});

Deno.test("api: full voyage lifecycle through the router", async () => {
  const h = await setup();
  try {
    const cookie = await register(h.app);

    // No active voyage yet.
    let res = await h.app.request(`${base}/api/v1/voyages/current`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 404);

    // Create voyage.
    res = await h.app.request(`${base}/api/v1/voyages`, {
      method: "POST",
      headers: {
        cookie: `tidekeepers_session=${cookie}`,
        "Idempotency-Key": "voyage-create-1",
      },
    });
    assertStrictEquals(res.status, 201);
    const created = await json(res);
    assertStrictEquals(created.status, "ACTIVE");
    const voyageId = created.id as string;

    // Create with the same idempotency key replays.
    const replay = await h.app.request(`${base}/api/v1/voyages`, {
      method: "POST",
      headers: {
        cookie: `tidekeepers_session=${cookie}`,
        "Idempotency-Key": "voyage-create-1",
      },
    });
    assertStrictEquals(replay.status, 201);
    assertEquals(await json(replay), created);

    // Get current.
    res = await h.app.request(`${base}/api/v1/voyages/current`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 200);
    assertEquals(await json(res), created);

    // Current keepers.
    res = await h.app.request(`${base}/api/v1/voyages/current/keepers`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 200);
    const keepers = await json(res);
    const keeperList = keepers.keepers as Array<Record<string, unknown>>;
    assertStrictEquals(keeperList.length, 1);
    assertStrictEquals(keeperList[0]!.definitionKey, "crest_sovereign");

    // Daily context.
    res = await h.app.request(`${base}/api/v1/voyages/current/daily-context`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 200);
    const context = await json(res);
    const daily = context.daily as Record<string, unknown>;
    assertStrictEquals(daily.phase, "PREPARATION");

    // Voyage history.
    res = await h.app.request(`${base}/api/v1/voyages/${voyageId}/history`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 200);
    const history = await json(res);
    const events = history.events as Array<{ eventType: string }>;
    assertStrictEquals(events.length, 1);
    assertStrictEquals(events[0]!.eventType, "CREATED");

    // Abandon.
    res = await h.app.request(`${base}/api/v1/voyages/${voyageId}/abandon`, {
      method: "POST",
      headers: {
        cookie: `tidekeepers_session=${cookie}`,
        "Idempotency-Key": "voyage-abandon-1",
      },
    });
    assertStrictEquals(res.status, 200);
    const abandoned = await json(res);
    assertStrictEquals(abandoned.status, "ABANDONED");

    // Current voyage is gone.
    res = await h.app.request(`${base}/api/v1/voyages/current`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 404);

    // Get by id still resolves.
    res = await h.app.request(`${base}/api/v1/voyages/${voyageId}`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 200);
    assertEquals(await json(res), abandoned);
  } finally {
    h.kv.close();
  }
});

Deno.test("api: voyage create without idempotency key is invalid", async () => {
  const h = await setup();
  try {
    const cookie = await register(h.app);
    const res = await h.app.request(`${base}/api/v1/voyages`, {
      method: "POST",
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 400);
    const body = await json(res);
    assertStrictEquals(body.code, "INVALID_IDEMPOTENCY_KEY");
  } finally {
    h.kv.close();
  }
});

Deno.test("api: abandoned voyage cannot be abandoned again without key reuse", async () => {
  const h = await setup();
  try {
    const cookie = await register(h.app);
    const createdRes = await h.app.request(`${base}/api/v1/voyages`, {
      method: "POST",
      headers: {
        cookie: `tidekeepers_session=${cookie}`,
        "Idempotency-Key": "v-abx-key-1",
      },
    });
    const created = await json(createdRes);
    const voyageId = created.id as string;
    await h.app.request(`${base}/api/v1/voyages/${voyageId}/abandon`, {
      method: "POST",
      headers: {
        cookie: `tidekeepers_session=${cookie}`,
        "Idempotency-Key": "v-abx-1",
      },
    });
    // Reusing a different key on the already-abandoned voyage replays as 200.
    const second = await h.app.request(
      `${base}/api/v1/voyages/${voyageId}/abandon`,
      {
        method: "POST",
        headers: {
          cookie: `tidekeepers_session=${cookie}`,
          "Idempotency-Key": "v-abx-2",
        },
      },
    );
    assertStrictEquals(second.status, 200);
  } finally {
    h.kv.close();
  }
});

Deno.test("api: keeper unlocks starts empty", async () => {
  const h = await setup();
  try {
    const cookie = await register(h.app);
    const res = await h.app.request(`${base}/api/v1/me/keeper-unlocks`, {
      headers: { cookie: `tidekeepers_session=${cookie}` },
    });
    assertStrictEquals(res.status, 200);
    const body = await json(res);
    assertEquals(body.unlocks, []);
  } finally {
    h.kv.close();
  }
});
