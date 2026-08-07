/** Guest player and opaque session persistence over Deno KV. */

import type { Player, Principal, Session } from "../domain/types.ts";
import { playerKey, playerPublicKey, sessionKey, type Store } from "./kv.ts";

export class AuthRepository {
  #store: Store;
  constructor(store: Store) {
    this.#store = store;
  }

  async findPlayer(id: string): Promise<Player | null> {
    return this.#store.get<Player>(playerKey(id));
  }

  /** Atomically creates the player, public index, and first session. */
  async createPlayerAndSession(
    player: Player,
    session: Session,
  ): Promise<boolean> {
    const result = await this.#store.atomic()
      .check({ key: playerKey(player.id), versionstamp: null })
      .check({ key: playerPublicKey(player.publicId), versionstamp: null })
      .check({ key: sessionKey(session.tokenDigest), versionstamp: null })
      .set(playerKey(player.id), player)
      .set(playerPublicKey(player.publicId), player.id)
      .set(sessionKey(session.tokenDigest), session)
      .commit();
    return result.ok;
  }

  async createSession(session: Session): Promise<void> {
    await this.#store.set(sessionKey(session.tokenDigest), session);
  }

  /** Resolves an active session directly to its player. */
  async findAuthenticatedSession(
    digestHex: string,
    now: Date,
  ): Promise<Principal | null> {
    const session = await this.#store.get<Session>(sessionKey(digestHex));
    if (
      !session || session.revokedAt || session.rotatedAt ||
      session.replacedBySessionId
    ) return null;
    if (new Date(session.expiresAt).getTime() <= now.getTime()) return null;
    const player = await this.findPlayer(session.playerId);
    return player ? { playerId: player.id } : null;
  }
}
