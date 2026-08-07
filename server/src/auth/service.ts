/** Username/IP guest identity bootstrap and opaque session issuance. */

import type { Player, Principal, Session } from "../domain/types.ts";
import { internal } from "../utils/errors.ts";
import { base64UrlEncodeBytes } from "../utils/crypto.ts";
import { newId } from "../utils/ids.ts";
import { digestToken, newToken } from "./session.ts";
import {
  derivePlayerId,
  normalizeClientIp,
  normalizeUsername,
} from "./identity.ts";
import type { AuthRepository } from "../repositories/auth_repository.ts";

export interface IssuedSession {
  token: string;
  expiresAt: Date;
  created: boolean;
  playerId: string;
}

export class AuthService {
  #sessions: AuthRepository;
  #sessionTTL: number;
  #playerIdSecret: string;

  constructor(
    sessions: AuthRepository,
    sessionTTLMs: number,
    playerIdSecret = "test-player-id-secret",
  ) {
    this.#sessions = sessions;
    this.#sessionTTL = sessionTTLMs;
    this.#playerIdSecret = playerIdSecret;
  }

  async createOrResumeSession(
    displayName: string,
    clientIp: string,
    now: Date,
  ): Promise<IssuedSession> {
    const username = normalizeUsername(displayName);
    const normalizedIp = normalizeClientIp(clientIp);
    if (!normalizedIp) {
      throw internal("Client IP is unavailable.", {
        code: "CLIENT_IP_UNAVAILABLE",
      });
    }
    if (this.#playerIdSecret === "") {
      throw internal("Player identity secret is unavailable.", {
        code: "INTERNAL_ERROR",
      });
    }
    const playerId = await derivePlayerId(
      username,
      normalizedIp,
      this.#playerIdSecret,
    );
    const issued = await newToken();
    const expiresAt = new Date(now.getTime() + this.#sessionTTL);
    const session = this.#newSession(
      issued.digestHex,
      playerId,
      now,
      expiresAt,
    );
    const player: Player = {
      id: playerId,
      publicId: newPlayerPublicId(),
      displayName: displayName.trim(),
      username,
      onboardingCompleted: false,
      locale: "en-US",
      timezone: "UTC",
      currentVoyageId: null,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };

    // The conditional transaction is the concurrency boundary. If another
    // request won, its player is reused and only this request's session is new.
    const created = await this.#sessions.createPlayerAndSession(
      player,
      session,
    );
    if (!created) {
      const existing = await this.#sessions.findPlayer(playerId);
      if (!existing) {
        // A public-id collision is extraordinarily unlikely; retrying avoids
        // turning a transient KV conflict into a duplicate player.
        return this.createOrResumeSession(displayName, normalizedIp, now);
      }
      session.playerId = existing.id;
      await this.#sessions.createSession(session);
    }
    return { token: issued.token, expiresAt, created, playerId };
  }

  async authenticate(rawToken: string, now: Date): Promise<Principal | null> {
    const digest = await digestToken(rawToken);
    return digest ? this.#sessions.findAuthenticatedSession(digest, now) : null;
  }

  #newSession(
    tokenDigest: string,
    playerId: string,
    now: Date,
    expiresAt: Date,
  ): Session {
    return {
      id: newId(),
      playerId,
      tokenDigest,
      expiresAt: expiresAt.toISOString(),
      lastSeenAt: now.toISOString(),
      rotatedAt: null,
      replacedBySessionId: null,
      revokedAt: null,
      createdAt: now.toISOString(),
      updatedAt: now.toISOString(),
    };
  }
}

function newPlayerPublicId(): string {
  for (;;) {
    const bytes = crypto.getRandomValues(new Uint8Array(12));
    const id = "plr_" + base64UrlEncodeBytes(bytes);
    if (id[4] !== "-" && id[4] !== "_") return id;
  }
}
