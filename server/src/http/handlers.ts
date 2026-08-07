/** HTTP handlers for authentication, voyage lifecycle, fleet, and shop APIs. */

import type { Context } from "hono";
import type { AuthService, IssuedSession } from "../auth/service.ts";
import type { PlayerService } from "../services/player_service.ts";
import type { DailyService } from "../services/daily_service.ts";
import type { VoyageService } from "../services/voyage_service.ts";
import type {
  FleetMutationResponse,
  FleetService,
  LineupMutationRequest,
} from "../services/fleet_service.ts";
import type { PurchaseRequest, ShopService } from "../services/shop_service.ts";
import type {
  AdvanceResponse,
  SettlementService,
} from "../services/settlement_service.ts";
import {
  badRequest,
  notFound,
  statusCodeFor,
  TidekeepersError,
} from "../utils/errors.ts";
import {
  apiError,
  decodeJsonBody,
  principalFromContext,
} from "./middleware.ts";

const maxAuthRequestBytes = 8 << 10;

interface CredentialsRequest {
  email: string;
  password: string;
}

export class AuthHandler {
  #auth: AuthService;
  #cookieName: string;
  #cookieSecure: boolean;
  #cookieDomain: string;

  constructor(
    auth: AuthService,
    cookieName: string,
    cookieSecure: boolean,
    cookieDomain: string,
  ) {
    this.#auth = auth;
    this.#cookieName = cookieName;
    this.#cookieSecure = cookieSecure;
    this.#cookieDomain = cookieDomain;
  }

  register = async (c: Context): Promise<Response> => {
    const credentials = await decodeCredentials(c);
    if (!credentials) {
      return writeAuthError(
        c,
        400,
        "INVALID_AUTH_INPUT",
        "Enter a valid email and password.",
      );
    }
    try {
      const session = await this.#auth.register(
        credentials.email,
        credentials.password,
        new Date(),
      );
      this.#setSessionCookie(c, session);
      c.header("Cache-Control", "no-store");
      return c.body(null, 201);
    } catch (err) {
      return this.#respondError(c, err);
    }
  };

  login = async (c: Context): Promise<Response> => {
    const credentials = await decodeCredentials(c);
    if (!credentials) {
      return writeAuthError(
        c,
        400,
        "INVALID_AUTH_INPUT",
        "Enter a valid email and password.",
      );
    }
    try {
      const session = await this.#auth.login(
        credentials.email,
        credentials.password,
        new Date(),
      );
      this.#setSessionCookie(c, session);
      c.header("Cache-Control", "no-store");
      return c.body(null, 204);
    } catch (err) {
      return this.#respondError(c, err);
    }
  };

  #respondError(c: Context, err: unknown): Response {
    if (err instanceof TidekeepersError) {
      const status = statusCodeFor(err);
      const code = err.code;
      let message = "An unexpected error occurred.";
      switch (code) {
        case "INVALID_AUTH_INPUT":
          message = "Enter a valid email and password.";
          break;
        case "EMAIL_ALREADY_REGISTERED":
          message = "An account already exists for this email address.";
          break;
        case "INVALID_CREDENTIALS":
          message = "Invalid email or password.";
          break;
      }
      return writeAuthError(c, status, code, message);
    }
    return writeAuthError(
      c,
      500,
      "INTERNAL_ERROR",
      "An unexpected error occurred.",
    );
  }

  #setSessionCookie(c: Context, session: IssuedSession): void {
    const maxAge = Math.max(
      1,
      Math.floor((session.expiresAt.getTime() - Date.now()) / 1000),
    );
    let cookie = `${this.#cookieName}=${
      encodeURIComponent(session.token)
    }; Path=/; Expires=${session.expiresAt.toUTCString()}; Max-Age=${maxAge}; HttpOnly; SameSite=Lax`;
    if (this.#cookieSecure) {
      cookie += "; Secure";
    }
    if (this.#cookieDomain !== "") {
      cookie += `; Domain=${this.#cookieDomain}`;
    }
    c.header("Set-Cookie", cookie);
  }
}

export class MeHandler {
  #players: PlayerService;

  constructor(players: PlayerService) {
    this.#players = players;
  }

  get = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    const me = await this.#players.getMe(principal.playerId);
    c.header("Cache-Control", "private, no-store");
    return c.json({
      playerId: me.publicId,
      onboardingCompleted: me.onboardingCompleted,
      locale: me.locale,
      timezone: me.timezone,
      activeVoyageId: me.activeVoyageId,
    });
  };
}

export class KeeperUnlocksHandler {
  #players: PlayerService;

  constructor(players: PlayerService) {
    this.#players = players;
  }

  list = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    const unlocks = await this.#players.listUnlocks(principal.playerId);
    c.header("Cache-Control", "private, no-store");
    return c.json({
      unlocks: unlocks.map((unlock) => ({
        definitionKey: unlock.definitionKey,
        definitionVersion: unlock.definitionVersion,
        name: unlock.name,
        currentName: unlock.currentName,
        sector: unlock.sector,
        role: unlock.role,
        rarity: unlock.rarity,
        unlockSource: unlock.unlockSource,
        unlockedAt: unlock.unlockedAt,
      })),
    });
  };
}

export class VoyageHandler {
  #service: VoyageService;

  constructor(service: VoyageService) {
    this.#service = service;
  }

  create = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    const idempotencyKey = c.req.header("Idempotency-Key") ?? "";
    if (idempotencyKey === "") {
      return this.#writeError(
        c,
        badRequest("invalid idempotency key", {
          code: "INVALID_IDEMPOTENCY_KEY",
        }),
      );
    }
    try {
      const resp = await this.#service.create(
        principal.playerId,
        idempotencyKey,
        new Date(),
      );
      c.header("Cache-Control", "private, no-store");
      return c.json(resp, 201);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  getCurrent = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    try {
      const resp = await this.#service.getCurrent(principal.playerId);
      c.header("Cache-Control", "private, no-store");
      return c.json(resp, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  getById = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    const voyageId = c.req.param("voyageId") ?? "";
    if (voyageId === "") {
      return this.#writeError(
        c,
        notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" }),
      );
    }
    try {
      const resp = await this.#service.getById(principal.playerId, voyageId);
      c.header("Cache-Control", "private, no-store");
      return c.json(resp, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  abandon = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    const voyageId = c.req.param("voyageId") ?? "";
    if (voyageId === "") {
      return this.#writeError(
        c,
        notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" }),
      );
    }
    const idempotencyKey = c.req.header("Idempotency-Key") ?? "";
    if (idempotencyKey === "") {
      return this.#writeError(
        c,
        badRequest("invalid idempotency key", {
          code: "INVALID_IDEMPOTENCY_KEY",
        }),
      );
    }
    try {
      const resp = await this.#service.abandon(
        principal.playerId,
        voyageId,
        idempotencyKey,
        new Date(),
      );
      c.header("Cache-Control", "private, no-store");
      return c.json(resp, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  getHistory = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    const voyageId = c.req.param("voyageId") ?? "";
    if (voyageId === "") {
      return this.#writeError(
        c,
        notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" }),
      );
    }
    try {
      const resp = await this.#service.getHistory(principal.playerId, voyageId);
      c.header("Cache-Control", "private, no-store");
      return c.json(resp, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  #writeError(c: Context, err: unknown): Response {
    const error = err instanceof TidekeepersError
      ? err
      : new TidekeepersError("internal error", "internal");
    c.header("Cache-Control", "private, no-store");
    switch (error.code) {
      case "INVALID_IDEMPOTENCY_KEY":
        return apiError(
          c,
          400,
          "INVALID_IDEMPOTENCY_KEY",
          "Invalid idempotency key.",
        );
      case "IDEMPOTENCY_KEY_REUSED":
        return apiError(
          c,
          409,
          "IDEMPOTENCY_KEY_REUSED",
          "Idempotency key was reused with a different request.",
        );
      case "ACTIVE_VOYAGE_EXISTS":
        return apiError(
          c,
          409,
          "ACTIVE_VOYAGE_EXISTS",
          "An active voyage already exists.",
        );
      case "NO_ACTIVE_VOYAGE":
        return apiError(c, 404, "NO_ACTIVE_VOYAGE", "No current voyage.");
      case "VOYAGE_NOT_FOUND":
        return apiError(c, 404, "VOYAGE_NOT_FOUND", "Voyage not found.");
      case "VOYAGE_NOT_ACTIVE":
        return apiError(c, 409, "VOYAGE_NOT_ACTIVE", "Voyage is not active.");
      case "VOYAGE_INITIALIZATION_UNAVAILABLE":
        return apiError(
          c,
          503,
          "VOYAGE_INITIALIZATION_UNAVAILABLE",
          "Voyage initialization is currently unavailable.",
        );
      case "SERVICE_UNAVAILABLE":
        return apiError(
          c,
          503,
          "SERVICE_UNAVAILABLE",
          "Service is temporarily unavailable.",
        );
      default:
        return apiError(
          c,
          500,
          "INTERNAL_ERROR",
          "An unexpected error occurred.",
        );
    }
  }
}

export class VoyageKeepersHandler {
  #service: VoyageService;

  constructor(service: VoyageService) {
    this.#service = service;
  }

  list = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    try {
      const inventory = await this.#service.getActiveVoyageKeepers(
        principal.playerId,
      );
      c.header("Cache-Control", "private, no-store");
      return c.json({
        voyageId: inventory.voyageId,
        keepers: inventory.keepers.map((keeper) => ({
          id: keeper.id,
          definitionKey: keeper.definitionKey,
          definitionVersion: keeper.definitionVersion,
          name: keeper.name,
          currentName: keeper.currentName,
          sector: keeper.sector,
          role: keeper.role,
          rarity: keeper.rarity,
          upgradeNodeKey: keeper.upgradeNodeKey,
          level: keeper.level,
          acquiredDay: keeper.acquiredDay,
          acquiredSource: keeper.acquiredSource,
          acquiredAt: keeper.acquiredAt,
        })),
      });
    } catch (err) {
      return this.#writeVoyageError(c, err);
    }
  };

  #writeVoyageError(c: Context, err: unknown): Response {
    const error = err instanceof TidekeepersError
      ? err
      : new TidekeepersError("internal error", "internal");
    c.header("Cache-Control", "private, no-store");
    if (error.code === "NO_ACTIVE_VOYAGE") {
      return apiError(c, 404, "NO_ACTIVE_VOYAGE", "No current voyage.");
    }
    if (error.code === "VOYAGE_NOT_FOUND") {
      return apiError(c, 404, "VOYAGE_NOT_FOUND", "Voyage not found.");
    }
    return apiError(c, 500, "INTERNAL_ERROR", "An unexpected error occurred.");
  }
}

export class DailyContextHandler {
  #daily: DailyService;

  constructor(daily: DailyService) {
    this.#daily = daily;
  }

  getCurrent = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) {
      return writeAuthenticationRequired(c);
    }
    try {
      const ctx = await this.#daily.getCurrentDailyContext(
        principal.playerId,
        new Date(),
      );
      c.header("Cache-Control", "private, no-store");
      return c.json(ctx, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  #writeError(c: Context, err: unknown): Response {
    const error = err instanceof TidekeepersError
      ? err
      : new TidekeepersError("internal error", "internal");
    c.header("Cache-Control", "private, no-store");
    switch (error.code) {
      case "NO_ACTIVE_VOYAGE":
        return apiError(c, 404, "NO_ACTIVE_VOYAGE", "No active voyage.");
      case "VOYAGE_NOT_FOUND":
        return apiError(c, 404, "VOYAGE_NOT_FOUND", "Voyage not found.");
      case "SERVICE_UNAVAILABLE":
        return apiError(
          c,
          503,
          "SERVICE_UNAVAILABLE",
          "Service is temporarily unavailable.",
        );
      default:
        return apiError(
          c,
          500,
          "INTERNAL_ERROR",
          "An unexpected error occurred.",
        );
    }
  }
}

export class FleetHandler {
  #fleet: FleetService;

  constructor(fleet: FleetService) {
    this.#fleet = fleet;
  }

  updateLineup = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) return writeAuthenticationRequired(c);
    const request = await decodeLineupRequest(c);
    if (!request) {
      return apiError(c, 400, "LINEUP_INVALID", "Invalid lineup request.");
    }
    try {
      const result = await this.#fleet.updateLineup(
        principal.playerId,
        request,
        new Date(),
      );
      return this.#writeResult(c, result);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  lock = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) return writeAuthenticationRequired(c);
    const request = await decodeLineupRequest(c);
    if (!request) {
      return apiError(c, 400, "LINEUP_INVALID", "Invalid lineup request.");
    }
    try {
      const result = await this.#fleet.lockFleet(
        principal.playerId,
        request,
        new Date(),
      );
      return this.#writeResult(c, result);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  #writeResult(c: Context, result: FleetMutationResponse): Response {
    c.header("Cache-Control", "private, no-store");
    return c.json(result, 200);
  }

  #writeError(c: Context, err: unknown): Response {
    const error = err instanceof TidekeepersError
      ? err
      : new TidekeepersError("internal error", "internal");
    c.header("Cache-Control", "private, no-store");
    switch (error.code) {
      case "NO_ACTIVE_VOYAGE":
        return apiError(c, 404, error.code, "No current voyage.");
      case "VOYAGE_NOT_FOUND":
      case "KEEPER_NOT_FOUND":
        return apiError(
          c,
          404,
          error.code,
          error.code === "KEEPER_NOT_FOUND"
            ? "Keeper not found."
            : "Voyage not found.",
        );
      case "VERSION_CONFLICT":
        return apiError(c, 409, error.code, "The daily state has changed.");
      case "ALREADY_LOCKED":
        return apiError(c, 409, error.code, "The fleet is already locked.");
      case "INVALID_PHASE":
        return apiError(
          c,
          409,
          error.code,
          "The fleet cannot be changed in this phase.",
        );
      case "LINEUP_INVALID":
        return apiError(c, 422, error.code, error.message);
      default:
        return apiError(
          c,
          500,
          "INTERNAL_ERROR",
          "An unexpected error occurred.",
        );
    }
  }
}

export class ShopHandler {
  #shop: ShopService;

  constructor(shop: ShopService) {
    this.#shop = shop;
  }

  purchase = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) return writeAuthenticationRequired(c);
    const offerId = c.req.param("offerId") ?? "";
    const body = await decodeJsonBody<PurchaseRequest>(c, 8 << 10);
    if (!body || !Number.isInteger(body.expectedVersion)) {
      return apiError(c, 400, "SHOP_INVALID", "Invalid shop request.");
    }
    try {
      const result = await this.#shop.purchase(
        principal.playerId,
        offerId,
        body,
        new Date(),
      );
      c.header("Cache-Control", "private, no-store");
      return c.json(result, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  #writeError(c: Context, err: unknown): Response {
    const error = err instanceof TidekeepersError
      ? err
      : new TidekeepersError("internal error", "internal");
    c.header("Cache-Control", "private, no-store");
    switch (error.code) {
      case "NO_ACTIVE_VOYAGE":
      case "SHOP_OFFER_NOT_FOUND":
        return apiError(
          c,
          404,
          error.code,
          error.code === "NO_ACTIVE_VOYAGE"
            ? "No current voyage."
            : "Shop offer not found.",
        );
      case "VERSION_CONFLICT":
        return apiError(c, 409, error.code, "The daily state has changed.");
      case "INVALID_PHASE":
        return apiError(c, 409, error.code, "The shop is closed.");
      case "SHOP_OFFER_UNAVAILABLE":
        return apiError(
          c,
          409,
          error.code,
          "Shop offer is no longer available.",
        );
      case "INSUFFICIENT_CAPITAL":
        return apiError(c, 409, error.code, "Insufficient capital.");
      case "SHOP_INVALID":
        return apiError(c, 400, error.code, error.message);
      default:
        return apiError(
          c,
          500,
          "INTERNAL_ERROR",
          "An unexpected error occurred.",
        );
    }
  }
}

export class SettlementHandler {
  #settlement: SettlementService;

  constructor(settlement: SettlementService) {
    this.#settlement = settlement;
  }

  result = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) return writeAuthenticationRequired(c);
    try {
      const result = await this.#settlement.getCurrentResult(
        principal.playerId,
      );
      c.header("Cache-Control", "private, no-store");
      return c.json(result, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  advance = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) return writeAuthenticationRequired(c);
    const body = await decodeJsonBody<{ expectedVersion?: unknown }>(
      c,
      8 << 10,
    );
    if (!body || !Number.isInteger(body.expectedVersion)) {
      return apiError(c, 400, "VERSION_CONFLICT", "Invalid advance request.");
    }
    try {
      const result: AdvanceResponse = await this.#settlement.advanceDay(
        principal.playerId,
        body.expectedVersion as number,
        new Date(),
      );
      c.header("Cache-Control", "private, no-store");
      return c.json(result, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  claimReward = async (c: Context): Promise<Response> => {
    const principal = principalFromContext(c);
    if (!principal) return writeAuthenticationRequired(c);
    const body = await decodeJsonBody<{ expectedVersion?: unknown }>(
      c,
      8 << 10,
    );
    if (!body || !Number.isInteger(body.expectedVersion)) {
      return apiError(c, 400, "REWARD_INVALID", "Invalid reward request.");
    }
    try {
      const result = await this.#settlement.claimCurrentReward(
        principal.playerId,
        body.expectedVersion as number,
        new Date(),
      );
      c.header("Cache-Control", "private, no-store");
      return c.json(result, 200);
    } catch (err) {
      return this.#writeError(c, err);
    }
  };

  #writeError(c: Context, err: unknown): Response {
    const error = err instanceof TidekeepersError
      ? err
      : new TidekeepersError("internal error", "internal");
    c.header("Cache-Control", "private, no-store");
    switch (error.code) {
      case "NO_ACTIVE_VOYAGE":
        return apiError(c, 404, error.code, "No current voyage.");
      case "VOYAGE_NOT_FOUND":
        return apiError(c, 404, error.code, "Voyage not found.");
      case "VOYAGE_NOT_ACTIVE":
        return apiError(c, 409, error.code, "Voyage is not active.");
      case "RESULT_NOT_READY":
        return apiError(c, 404, error.code, "Result is not ready.");
      case "NEXT_TIDE_NOT_READY":
        return apiError(c, 503, error.code, "Next daily tide is not ready.");
      case "REWARD_NOT_CLAIMED":
        return apiError(c, 409, error.code, "Claim the pending reward first.");
      case "INVALID_PHASE":
        return apiError(c, 409, error.code, "The daily tide is not ready.");
      case "REWARD_ALREADY_CLAIMED":
        return apiError(c, 409, error.code, "Reward has already been claimed.");
      case "VERSION_CONFLICT":
        return apiError(c, 409, error.code, "The daily state has changed.");
      case "REWARD_INVALID":
        return apiError(c, 400, error.code, error.message);
      default:
        return apiError(
          c,
          500,
          "INTERNAL_ERROR",
          "An unexpected error occurred.",
        );
    }
  }
}

export class HealthHandler {
  ready: () => boolean;

  constructor(ready: () => boolean) {
    this.ready = ready;
  }

  live = (c: Context): Response => {
    c.header("Cache-Control", "no-store");
    return c.json({ status: "live" }, 200);
  };

  readyHandler = async (c: Context): Promise<Response> => {
    c.header("Cache-Control", "no-store");
    if (!this.ready()) {
      return c.json({ status: "unavailable" }, 503);
    }
    return c.json({ status: "ready" }, 200);
  };
}

async function decodeLineupRequest(
  c: Context,
): Promise<LineupMutationRequest | null> {
  const body = await decodeJsonBody<unknown>(c, 16 << 10);
  if (!body || typeof body !== "object") return null;
  const candidate = body as Record<string, unknown>;
  if (
    !Number.isInteger(candidate.expectedVersion) ||
    !Array.isArray(candidate.slots)
  ) {
    return null;
  }
  const slots: LineupMutationRequest["slots"] = [];
  for (const value of candidate.slots) {
    if (!value || typeof value !== "object") return null;
    const slot = value as Record<string, unknown>;
    if (!Number.isInteger(slot.index)) return null;
    if (slot.keeperId !== null && typeof slot.keeperId !== "string") {
      return null;
    }
    slots.push({
      index: slot.index as number,
      keeperId: slot.keeperId as string | null,
    });
  }
  return { expectedVersion: candidate.expectedVersion as number, slots };
}

async function decodeCredentials(
  c: Context,
): Promise<CredentialsRequest | null> {
  const body = await decodeJsonBody<CredentialsRequest>(c, maxAuthRequestBytes);
  if (
    !body || typeof body.email !== "string" || typeof body.password !== "string"
  ) {
    return null;
  }
  return body;
}

export function writeAuthenticationRequired(c: Context): Response {
  c.header("Cache-Control", "private, no-store");
  return apiError(c, 401, "AUTH_REQUIRED", "Authentication is required.");
}

export function writeAuthError(
  c: Context,
  status: number,
  code: string,
  message: string,
): Response {
  c.header("Cache-Control", "no-store");
  return apiError(c, status, code, message);
}
