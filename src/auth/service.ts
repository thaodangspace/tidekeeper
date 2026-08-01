/** Account registration, login, and session issuance (ported from auth/service.go). */

import type { Account, Player, Principal } from "../domain/types.ts";
import { conflict, TidekeepersError } from "../utils/errors.ts";
import { base64UrlEncodeBytes } from "../utils/crypto.ts";
import { newId } from "../utils/ids.ts";
import { digestToken, newToken } from "./session.ts";
import {
  hashPassword,
  invalidCredentials,
  minPasswordLength,
  verifyPassword,
} from "./password.ts";
import type { AuthRepository } from "../repositories/auth_repository.ts";

const maxEmailLength = 254;

export interface IssuedSession {
  token: string;
  expiresAt: Date;
}

export class AuthService {
  #sessions: AuthRepository;
  #sessionTTL: number;

  constructor(sessions: AuthRepository, sessionTTLMs: number) {
    this.#sessions = sessions;
    this.#sessionTTL = sessionTTLMs;
  }

  async register(
    email: string,
    password: string,
    now: Date,
  ): Promise<IssuedSession> {
    const normalized = normalizeCredentials(email, password);
    // Avoid doing the expensive password hash twice for the common duplicate
    // request path. The conditional write below remains the source of truth
    // for concurrent registrations.
    if (await this.#sessions.findAccountByEmail(normalized)) {
      throw conflict("an account already exists for this email", {
        code: "EMAIL_ALREADY_REGISTERED",
      });
    }
    const passwordHash = await hashPassword(password);
    const issued = await newToken();
    const expiresAt = new Date(now.getTime() + this.#sessionTTL);

    const account: Account = {
      id: newId(),
      email: normalized,
      passwordHash,
      status: "ACTIVE",
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
    const player: Player = {
      id: newId(),
      accountId: account.id,
      publicId: newPlayerPublicId(),
      onboardingCompleted: false,
      locale: "en-US",
      timezone: "UTC",
      currentVoyageId: null,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
    const session = {
      id: newId(),
      accountId: account.id,
      playerId: player.id,
      tokenDigest: issued.digestHex,
      expiresAt: expiresAt.toISOString(),
      lastSeenAt: now.toISOString(),
      rotatedAt: null,
      replacedBySessionId: null,
      revokedAt: null,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };

    const created = await this.#sessions.createRegistration(
      account,
      player,
      session,
    );
    if (!created) {
      throw conflict("an account already exists for this email", {
        code: "EMAIL_ALREADY_REGISTERED",
      });
    }
    return { token: issued.token, expiresAt };
  }

  async login(
    email: string,
    password: string,
    now: Date,
  ): Promise<IssuedSession> {
    const normalized = normalizeCredentials(email, password);
    const account = await this.#sessions.findAccountByEmail(normalized);
    if (!account || !(await verifyPassword(account.passwordHash, password))) {
      throw invalidCredentials();
    }
    const playerId = await this.#sessions.findPlayerIdByAccountId(account.id);
    if (!playerId) {
      throw invalidCredentials();
    }
    const issued = await newToken();
    const expiresAt = new Date(now.getTime() + this.#sessionTTL);
    const session = {
      id: newId(),
      accountId: account.id,
      playerId,
      tokenDigest: issued.digestHex,
      expiresAt: expiresAt.toISOString(),
      lastSeenAt: now.toISOString(),
      rotatedAt: null,
      replacedBySessionId: null,
      revokedAt: null,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
    await this.#sessions.createSession(session);
    return { token: issued.token, expiresAt };
  }

  async authenticate(rawToken: string, now: Date): Promise<Principal | null> {
    const digest = await digestToken(rawToken);
    if (!digest) {
      return null;
    }
    return this.#sessions.findAuthenticatedSession(digest, now);
  }
}

function normalizeCredentials(email: string, password: string): string {
  const normalized = email.toLowerCase().trim();
  if (
    normalized.length > maxEmailLength || !validEmail(normalized) ||
    password.length < minPasswordLength
  ) {
    throw new TidekeepersError(
      "invalid authentication input",
      "invalid_request",
      { code: "INVALID_AUTH_INPUT" },
    );
  }
  return normalized;
}

function validEmail(email: string): boolean {
  const at = email.lastIndexOf("@");
  return at > 0 && at < email.length - 3 &&
    email.split("@").length === 2 &&
    !/[\s\t\r\n]/.test(email) &&
    email.slice(at + 1).includes(".");
}

function newPlayerPublicId(): string {
  for (;;) {
    const bytes = crypto.getRandomValues(new Uint8Array(12));
    const id = "plr_" + base64UrlEncodeBytes(bytes);
    if (id[4] !== "-" && id[4] !== "_") {
      return id;
    }
  }
}
