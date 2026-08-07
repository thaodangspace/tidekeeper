/** HTTP middleware: request ID, security headers, logging, authentication. */

import type { Context, Next } from "hono";
import type { Principal } from "../domain/types.ts";
import { newId } from "../utils/ids.ts";
import type { AuthService } from "../auth/service.ts";

const requestIdHeader = "X-Request-ID";
const validRequestId = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$/;

const principalKey = "principal";
const requestIdKey = "requestId";

function newRequestId(): string {
  return "req_" + newId();
}

export function withRequestId(
  c: Context,
  next: Next,
): Promise<Response | void> {
  const incoming = c.req.header(requestIdHeader) ?? "";
  const requestId = validRequestId.test(incoming) ? incoming : newRequestId();
  c.header(requestIdHeader, requestId);
  c.set(requestIdKey, requestId);
  return next();
}

export function requestIdFromContext(c: Context): string {
  return c.get(requestIdKey) ?? "";
}

/** Connection metadata set by the trusted Deno.serve boundary. */
export function clientIpFromContext(c: Context): string | null {
  const env = c.env as { clientIp?: string | null } | undefined;
  return env?.clientIp ?? null;
}

export function securityHeaders(
  c: Context,
  next: Next,
): Promise<Response | void> {
  c.header("X-Content-Type-Options", "nosniff");
  c.header("Content-Security-Policy", "default-src 'none'");
  c.header("X-Frame-Options", "DENY");
  c.header("Referrer-Policy", "no-referrer");
  c.header("Permissions-Policy", "camera=(), geolocation=(), microphone=()");
  return next();
}

export function requestLogger(
  log: (fields: Record<string, unknown>) => void,
  service: string,
) {
  return async (c: Context, next: Next): Promise<Response | void> => {
    const started = performance.now();
    const response = await next();
    const durationMs = performance.now() - started;
    const status = typeof response === "object" && response !== null
      ? (response as Response).status
      : undefined;
    log({
      msg: "request completed",
      service,
      requestId: requestIdFromContext(c),
      method: c.req.method,
      path: new URL(c.req.url).pathname,
      status,
      durationMs: Math.round(durationMs * 1000) / 1000,
    });
    return response;
  };
}

export function attachPrincipal(c: Context, principal: Principal): void {
  c.set(principalKey, principal);
}

export function principalFromContext(c: Context): Principal | undefined {
  return c.get(principalKey);
}

/** Requires a valid session cookie and attaches the authenticated principal. */
export function requireAuthentication(
  authenticator: AuthService,
  cookieName: string,
) {
  return async (c: Context, next: Next): Promise<Response | void> => {
    // The protected router is mounted at root; do not turn unknown public
    // paths (including removed auth endpoints) into authentication failures.
    if (!new URL(c.req.url).pathname.startsWith("/api/")) return next();
    const cookieValue = c.req.header("cookie")?.match(
      new RegExp(`(?:^|;\\s*)${escapeRegExp(cookieName)}=([^;]+)`),
    )?.[1];
    if (cookieValue === undefined) {
      return writeUnauthorized(
        c,
        "AUTH_REQUIRED",
        "Authentication is required.",
        false,
        cookieName,
      );
    }
    const principal = await authenticator.authenticate(cookieValue, new Date());
    if (!principal) {
      return writeUnauthorized(
        c,
        "SESSION_EXPIRED",
        "The session has expired.",
        true,
        cookieName,
      );
    }
    attachPrincipal(c, principal);
    return next();
  };
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function writeUnauthorized(
  c: Context,
  code: string,
  message: string,
  clearCookie: boolean,
  cookieName: string,
): Response {
  c.header("Cache-Control", "private, no-store");
  if (clearCookie) {
    c.header(
      "Set-Cookie",
      `${cookieName}=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; Max-Age=0; HttpOnly; SameSite=Lax`,
    );
  }
  return c.json({ code, message, requestId: requestIdFromContext(c) }, 401);
}

export interface ApiErrorResponse {
  code: string;
  message: string;
  requestId: string;
}

export function apiError(
  c: Context,
  status: number,
  code: string,
  message: string,
): Response {
  return c.json(
    {
      code,
      message,
      requestId: requestIdFromContext(c),
    } satisfies ApiErrorResponse,
    status as never,
  );
}

/** Parses an auth JSON body with a size limit and strict single-object decode. */
export async function decodeJsonBody<T>(
  c: Context,
  maxBytes: number,
): Promise<T | null> {
  const text = await c.req.text();
  if (text.length > maxBytes) {
    return null;
  }
  try {
    return JSON.parse(text) as T;
  } catch {
    return null;
  }
}
