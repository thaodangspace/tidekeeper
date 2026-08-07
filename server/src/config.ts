/** Application configuration loaded from the environment and validated at startup. */

export type Environment = "local" | "test" | "staging" | "production";

export interface Config {
  environment: Environment;
  http: {
    address: string;
    readTimeoutMs: number;
    writeTimeoutMs: number;
    shutdownTimeoutMs: number;
  };
  session: {
    cookieName: string;
    cookieSecure: boolean;
    cookieDomain: string;
    ttlMs: number;
  };
  rateLimit: {
    requests: number;
    windowMs: number;
  };
  logLevel: "debug" | "info" | "warn" | "error";
  serviceVersion: string;
  /** IANA timezone the game day boundary and lock time are computed in. */
  gameTimezone: string;
  /** Local time (HH:mm) at which a Daily Tide locks, in gameTimezone. */
  lockTime: string;
  /** Content release version used when creating Daily Tides. */
  contentVersion: number;
  market: {
    provider: string;
    apiKey: string;
  };
  /** HMAC secret used to derive opaque deterministic guest player ids. */
  playerIdSecret: string;
  /** Forwarding headers are read only when the deployment explicitly trusts its proxy. */
  proxy: {
    trusted: boolean;
    header: string;
  };
}

const ENVIRONMENTS: readonly string[] = [
  "local",
  "test",
  "staging",
  "production",
];
const LOG_LEVELS: readonly string[] = ["debug", "info", "warn", "error"];

function required(
  lookup: (name: string) => string | undefined,
  name: string,
): string {
  const value = lookup(name);
  if (value === undefined || value.trim() === "") {
    throw new Error(`${name} is required`);
  }
  return value.trim();
}

function value(
  lookup: (name: string) => string | undefined,
  name: string,
  fallback: string,
): string {
  const raw = lookup(name);
  if (raw === undefined || raw.trim() === "") {
    return fallback;
  }
  return raw.trim();
}

function integer(
  lookup: (name: string) => string | undefined,
  name: string,
  fallback: number,
): number {
  const raw = lookup(name);
  if (raw === undefined || raw.trim() === "") {
    return fallback;
  }
  const parsed = Number.parseInt(raw.trim(), 10);
  if (!Number.isInteger(parsed)) {
    throw new Error(`${name} must be an integer`);
  }
  return parsed;
}

function boolean(
  lookup: (name: string) => string | undefined,
  name: string,
  fallback: boolean,
): boolean {
  const raw = lookup(name);
  if (raw === undefined || raw.trim() === "") {
    return fallback;
  }
  const parsed = raw.trim().toLowerCase();
  if (parsed === "true" || parsed === "1") return true;
  if (parsed === "false" || parsed === "0") return false;
  throw new Error(`${name} must be a boolean`);
}

function parseClock(value: string): number {
  const match = /^(\d{1,2}):(\d{2})$/.exec(value);
  if (!match) {
    throw new Error(`lock time must use HH:mm format, got ${value}`);
  }
  const hours = Number.parseInt(match[1]!, 10);
  const minutes = Number.parseInt(match[2]!, 10);
  if (hours < 0 || hours > 23 || minutes < 0 || minutes > 59) {
    throw new Error(`lock time is out of range: ${value}`);
  }
  return hours * 60 + minutes;
}

/** Loads and validates configuration from process.env. */
export function loadConfig(
  lookup: (name: string) => string | undefined = (name) => Deno.env.get(name),
): Config {
  const environment = required(lookup, "APP_ENV").toLowerCase();
  if (!ENVIRONMENTS.includes(environment)) {
    throw new Error("APP_ENV must be one of local, test, staging, production");
  }

  const logLevel = value(lookup, "LOG_LEVEL", "info").toLowerCase();
  if (!LOG_LEVELS.includes(logLevel)) {
    throw new Error("LOG_LEVEL must be one of debug, info, warn, error");
  }

  const cookieSecure = boolean(lookup, "SESSION_COOKIE_SECURE", true);
  const playerIdSecret = environment === "local" || environment === "test"
    ? value(lookup, "PLAYER_ID_SECRET", "local-development-player-id-secret")
    : required(lookup, "PLAYER_ID_SECRET");
  const trustedProxy = boolean(lookup, "TRUSTED_PROXY", false);
  const trustedProxyHeader = value(
    lookup,
    "TRUSTED_PROXY_HEADER",
    "x-forwarded-for",
  );
  if (environment === "production" && !cookieSecure) {
    throw new Error("SESSION_COOKIE_SECURE must be true in production");
  }

  const ttlMs = integer(lookup, "SESSION_TTL_MS", 30 * 24 * 60 * 60 * 1000);
  if (ttlMs <= 0) {
    throw new Error("SESSION_TTL_MS must be positive");
  }

  const contentVersion = integer(lookup, "CONTENT_VERSION", 4);
  if (contentVersion <= 0) {
    throw new Error("CONTENT_VERSION must be positive");
  }

  const lockTime = value(lookup, "LOCK_TIME", "23:55");
  parseClock(lockTime);
  const gameTimezone = value(lookup, "GAME_TIMEZONE", "UTC");
  try {
    new Intl.DateTimeFormat("en-US", { timeZone: gameTimezone }).format();
  } catch {
    throw new Error(
      `GAME_TIMEZONE must be a valid IANA timezone: ${gameTimezone}`,
    );
  }

  const rateRequests = integer(lookup, "RATE_LIMIT_REQUESTS", 120);
  const rateWindowMs = integer(lookup, "RATE_LIMIT_WINDOW_MS", 60_000);
  if (rateRequests < 1) {
    throw new Error("RATE_LIMIT_REQUESTS must be positive");
  }
  if (rateWindowMs <= 0) {
    throw new Error("RATE_LIMIT_WINDOW_MS must be positive");
  }

  return {
    environment: environment as Environment,
    http: {
      address: value(
        lookup,
        "HTTP_ADDR",
        `:${value(lookup, "PORT", "8000")}`,
      ),
      readTimeoutMs: integer(lookup, "HTTP_READ_TIMEOUT_MS", 10_000),
      writeTimeoutMs: integer(lookup, "HTTP_WRITE_TIMEOUT_MS", 15_000),
      shutdownTimeoutMs: integer(lookup, "HTTP_SHUTDOWN_TIMEOUT_MS", 15_000),
    },
    session: {
      cookieName: value(lookup, "SESSION_COOKIE_NAME", "tidekeepers_session"),
      cookieSecure,
      cookieDomain: value(lookup, "SESSION_COOKIE_DOMAIN", ""),
      ttlMs,
    },
    rateLimit: { requests: rateRequests, windowMs: rateWindowMs },
    logLevel: logLevel as Config["logLevel"],
    serviceVersion: value(lookup, "SERVICE_VERSION", "development"),
    gameTimezone,
    lockTime,
    contentVersion,
    market: {
      provider: value(lookup, "MARKET_PROVIDER", "static"),
      apiKey: value(lookup, "MARKET_API_KEY", ""),
    },
    playerIdSecret,
    proxy: { trusted: trustedProxy, header: trustedProxyHeader },
  };
}

/** Minutes past midnight for the configured lock time (0-1439). */
export function lockMinutesOfDay(config: Config): number {
  const match = /^(\d{1,2}):(\d{2})$/.exec(config.lockTime)!;
  return Number.parseInt(match[1]!, 10) * 60 + Number.parseInt(match[2]!, 10);
}
