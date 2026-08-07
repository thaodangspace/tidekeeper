/** Account and session persistence over Deno KV. */

import type { Account, Player, Principal, Session } from "../domain/types.ts";
import {
  accountEmailKey,
  accountKey,
  accountPlayerKey,
  playerKey,
  playerPublicKey,
  sessionKey,
  type Store,
} from "./kv.ts";

export class AuthRepository {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async findAccountByEmail(email: string): Promise<Account | null> {
    return this.#store.get<Account>(accountEmailKey(email));
  }

  async findAccountById(id: string): Promise<Account | null> {
    return this.#store.get<Account>(accountKey(id));
  }

  /**
   * Persists a complete registration as one conditional KV transaction.
   *
   * Registration creates several indexes as well as the entities themselves;
   * writing them independently can leave an unusable account when a second
   * request races the first one. Every key is checked so a failed registration
   * has no partial side effects.
   */
  async createRegistration(
    account: Account,
    player: Player,
    session: Session,
  ): Promise<boolean> {
    const result = await this.#store.atomic()
      .check({ key: accountEmailKey(account.email), versionstamp: null })
      .check({ key: accountKey(account.id), versionstamp: null })
      .check({ key: playerKey(player.id), versionstamp: null })
      .check({ key: playerPublicKey(player.publicId), versionstamp: null })
      .check({ key: accountPlayerKey(account.id), versionstamp: null })
      .check({ key: sessionKey(session.tokenDigest), versionstamp: null })
      .set(accountEmailKey(account.email), account)
      .set(accountKey(account.id), account)
      .set(playerKey(player.id), player)
      .set(playerPublicKey(player.publicId), player.id)
      .set(accountPlayerKey(account.id), player.id)
      .set(sessionKey(session.tokenDigest), session)
      .commit();
    return result.ok;
  }

  async findPlayerByPublicId(publicId: string): Promise<string | null> {
    return this.#store.get<string>(playerPublicKey(publicId));
  }

  async createPlayer(player: Player): Promise<void> {
    await this.#store.set(playerKey(player.id), player);
    await this.#store.set(playerPublicKey(player.publicId), player.id);
  }

  /** Links a player to its owning account for login resolution. */
  async createPlayerForAccount(
    playerId: string,
    accountId: string,
  ): Promise<void> {
    await this.#store.set(accountPlayerKey(accountId), playerId);
  }

  async findPlayerIdByAccountId(accountId: string): Promise<string | null> {
    return this.#store.get<string>(accountPlayerKey(accountId));
  }

  async createSession(session: Session): Promise<void> {
    await this.#store.set(sessionKey(session.tokenDigest), session);
  }

  /** Resolves an active, non-expired, non-revoked session digest to a principal. */
  async findAuthenticatedSession(
    digestHex: string,
    now: Date,
  ): Promise<Principal | null> {
    const session = await this.#store.get<Session>(sessionKey(digestHex));
    if (!session) {
      return null;
    }
    if (session.revokedAt !== null && session.revokedAt !== undefined) {
      return null;
    }
    if (session.rotatedAt !== null && session.rotatedAt !== undefined) {
      return null;
    }
    if (
      session.replacedBySessionId !== null &&
      session.replacedBySessionId !== undefined
    ) {
      return null;
    }
    if (new Date(session.expiresAt).getTime() <= now.getTime()) {
      return null;
    }
    const account = await this.#store.get<Account>(
      accountKey(session.accountId),
    );
    if (!account || account.status !== "ACTIVE") {
      return null;
    }
    return { accountId: session.accountId, playerId: session.playerId };
  }
}
