/**
 * Tidekeepers API server (Deno port).
 * Entry point wiring config, KV store, repositories, services, HTTP router,
 * and scheduled jobs.
 */

import { loadConfig } from "./src/config.ts";
import { createLogger } from "./src/utils/log.ts";
import { Store } from "./src/repositories/kv.ts";
import { AuthRepository } from "./src/repositories/auth_repository.ts";
import { PlayerRepository } from "./src/repositories/player_repository.ts";
import { VoyageRepository } from "./src/repositories/voyage_repository.ts";
import { DailyRepository } from "./src/repositories/daily_repository.ts";
import { IdempotencyRepository } from "./src/repositories/idempotency_repository.ts";
import { AuthService } from "./src/auth/service.ts";
import { PlayerService } from "./src/services/player_service.ts";
import { DailyService } from "./src/services/daily_service.ts";
import { VoyageService } from "./src/services/voyage_service.ts";
import { FleetService } from "./src/services/fleet_service.ts";
import { ShopService } from "./src/services/shop_service.ts";
import { SettlementService } from "./src/services/settlement_service.ts";
import { buildRouter } from "./src/http/router.ts";
import {
  cleanupSessions,
  ensureDailyTide,
  runSettlementJob,
} from "./src/jobs/cron.ts";
import { createMarketProvider } from "./src/market/provider.ts";

const config = loadConfig();
const logger = createLogger({
  level: config.logLevel,
  service: "tidekeepers-api",
  version: config.serviceVersion,
});
const log = (fields: Record<string, unknown>) =>
  logger.info(fields.msg as string, fields);

const kv = await Deno.openKv();
const store = new Store(kv);

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
const marketProvider = createMarketProvider(config.market.provider);

// Provision today's tide before accepting requests. Cron keeps future days
// populated, while this startup path prevents a fresh deployment from
// returning an initialization error until the next midnight.
const initialTide = await ensureDailyTide(config, dailyRepository, new Date());
log({ msg: "daily tide initialized", result: initialTide });

const readiness = { ready: true };
const app = buildRouter({
  config,
  auth,
  players,
  daily,
  voyages,
  fleet,
  shop,
  settlement,
  ready: () => readiness.ready,
  log,
});

Deno.cron("daily-tide ensure", "0 0 * * *", async () => {
  const result = await ensureDailyTide(config, dailyRepository, new Date());
  log({ msg: "daily tide ensure completed", result });
});

Deno.cron("settlement", "*/5 * * * *", async () => {
  const result = await runSettlementJob(
    store,
    dailyRepository,
    voyageRepository,
    settlement,
    marketProvider,
    new Date(),
  );
  log({ msg: "settlement job completed", result });
});

Deno.cron("session cleanup", "0 * * * *", async () => {
  const removed = await cleanupSessions(store, new Date());
  log({ msg: "session cleanup completed", removed });
});

const url = parseAddress(config.http.address);
logger.info("api starting", { address: config.http.address });

const handler = async (request: Request): Promise<Response> => {
  const pathname = new URL(request.url).pathname;
  if (isApiPath(pathname)) {
    return app.fetch(request, Deno.env);
  }
  return serveFrontend(request);
};
Deno.serve(
  { hostname: url.hostname, port: url.port, onListen: () => {} },
  handler,
);

const frontendRoot = new URL("../build/", import.meta.url);

function isApiPath(pathname: string): boolean {
  return ["/api", "/auth", "/health"].some(
    (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
  );
}

async function serveFrontend(request: Request): Promise<Response> {
  if (request.method !== "GET" && request.method !== "HEAD") {
    return new Response("Method Not Allowed", {
      status: 405,
      headers: { Allow: "GET, HEAD" },
    });
  }

  const url = new URL(request.url);
  const fileUrl = frontendFileUrl(url.pathname);
  if (!fileUrl) return new Response("Bad Request", { status: 400 });

  try {
    const body = await Deno.readFile(fileUrl);
    const headers = new Headers({
      "Content-Type": contentType(fileUrl.pathname),
      "Cache-Control": url.pathname.startsWith("/_app/immutable/")
        ? "public, max-age=31536000, immutable"
        : "no-cache",
      "X-Content-Type-Options": "nosniff",
      "X-Frame-Options": "DENY",
      "Referrer-Policy": "no-referrer",
      "Content-Security-Policy":
        "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
    });
    return new Response(request.method === "HEAD" ? null : body, { headers });
  } catch (error) {
    if (error instanceof Deno.errors.NotFound) {
      return new Response("Not Found", { status: 404 });
    }
    throw error;
  }
}

function frontendFileUrl(pathname: string): URL | null {
  let decodedPath: string;
  try {
    decodedPath = decodeURIComponent(pathname);
  } catch {
    return null;
  }
  const segments = decodedPath.split("/");
  if (segments.includes("..")) return null;

  const normalized = decodedPath.replace(/^\/+/, "");
  const candidate = normalized === "" || normalized.endsWith("/")
    ? `${normalized}index.html`
    : normalized.includes(".")
    ? normalized
    : `${normalized}/index.html`;
  return new URL(candidate, frontendRoot);
}

function contentType(pathname: string): string {
  const extension = pathname.slice(pathname.lastIndexOf(".")).toLowerCase();
  return ({
    ".css": "text/css; charset=utf-8",
    ".gif": "image/gif",
    ".html": "text/html; charset=utf-8",
    ".jpeg": "image/jpeg",
    ".jpg": "image/jpeg",
    ".js": "application/javascript; charset=utf-8",
    ".json": "application/json; charset=utf-8",
    ".png": "image/png",
    ".svg": "image/svg+xml",
    ".webp": "image/webp",
    ".woff": "font/woff",
    ".woff2": "font/woff2",
  } as Record<string, string>)[extension] ?? "application/octet-stream";
}

function parseAddress(address: string): { hostname: string; port: number } {
  const trimmed = address.trim();
  if (trimmed.startsWith(":")) {
    return { hostname: "0.0.0.0", port: Number(trimmed.slice(1)) };
  }
  const lastColon = trimmed.lastIndexOf(":");
  if (lastColon > 0) {
    const hostname = trimmed.slice(0, lastColon);
    const port = Number(trimmed.slice(lastColon + 1));
    return { hostname, port };
  }
  return { hostname: "0.0.0.0", port: 8080 };
}
