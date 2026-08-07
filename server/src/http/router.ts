/** Hono router composing middleware and handlers (ported from httpapi/router.go). */

import { Hono } from "hono";
import type { Config } from "../config.ts";
import type { AuthService } from "../auth/service.ts";
import type { PlayerService } from "../services/player_service.ts";
import type { DailyService } from "../services/daily_service.ts";
import type { VoyageService } from "../services/voyage_service.ts";
import type { FleetService } from "../services/fleet_service.ts";
import type { ShopService } from "../services/shop_service.ts";
import type { SettlementService } from "../services/settlement_service.ts";
import {
  AuthHandler,
  DailyContextHandler,
  FleetHandler,
  HealthHandler,
  KeeperUnlocksHandler,
  MeHandler,
  SettlementHandler,
  ShopHandler,
  VoyageHandler,
  VoyageKeepersHandler,
} from "./handlers.ts";
import {
  requestLogger,
  requireAuthentication,
  securityHeaders,
  withRequestId,
} from "./middleware.ts";

export interface RouterDeps {
  config: Config;
  auth: AuthService;
  players: PlayerService;
  daily: DailyService;
  voyages: VoyageService;
  fleet: FleetService;
  shop: ShopService;
  settlement: SettlementService;
  ready: () => boolean;
  log: (fields: Record<string, unknown>) => void;
}

export function buildRouter(deps: RouterDeps): Hono {
  const app = new Hono();
  app.use("*", withRequestId);
  app.use("*", securityHeaders);
  app.use("*", requestLogger(deps.log, "tidekeepers-api"));

  const health = new HealthHandler(deps.ready);
  const auth = new AuthHandler(
    deps.auth,
    deps.config.session.cookieName,
    deps.config.session.cookieSecure,
    deps.config.session.cookieDomain,
  );
  const me = new MeHandler(deps.players);
  const keeperUnlocks = new KeeperUnlocksHandler(deps.players);
  const voyageKeepers = new VoyageKeepersHandler(deps.voyages);
  const dailyContext = new DailyContextHandler(deps.daily);
  const voyage = new VoyageHandler(deps.voyages);
  const fleet = new FleetHandler(deps.fleet);
  const shop = new ShopHandler(deps.shop);
  const settlement = new SettlementHandler(deps.settlement);

  app.post("/auth/register", auth.register);
  app.post("/auth/login", auth.login);
  app.get("/health/live", health.live);
  app.get("/health/ready", health.readyHandler);

  const protectedRouter = new Hono();
  protectedRouter.use(
    "*",
    requireAuthentication(deps.auth, deps.config.session.cookieName),
  );

  protectedRouter.get("/api/v1/me", me.get);
  protectedRouter.get("/api/v1/me/keeper-unlocks", keeperUnlocks.list);
  protectedRouter.get("/api/v1/voyages/current/keepers", voyageKeepers.list);
  protectedRouter.get(
    "/api/v1/voyages/current/daily-context",
    dailyContext.getCurrent,
  );
  protectedRouter.put("/api/v1/voyages/current/lineup", fleet.updateLineup);
  protectedRouter.post("/api/v1/voyages/current/lock", fleet.lock);
  protectedRouter.post(
    "/api/v1/voyages/current/shop/:offerId/recruit",
    shop.purchase,
  );
  protectedRouter.get("/api/v1/voyages/current/result", settlement.result);
  protectedRouter.post(
    "/api/v1/voyages/current/reward",
    settlement.claimReward,
  );
  protectedRouter.post("/api/v1/voyages/current/advance", settlement.advance);
  protectedRouter.post("/api/v1/voyages", voyage.create);
  protectedRouter.get("/api/v1/voyages/current", voyage.getCurrent);
  protectedRouter.get("/api/v1/voyages/:voyageId", voyage.getById);
  protectedRouter.post("/api/v1/voyages/:voyageId/abandon", voyage.abandon);
  protectedRouter.get("/api/v1/voyages/:voyageId/history", voyage.getHistory);

  app.route("/", protectedRouter);
  return app;
}
