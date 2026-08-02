/** Idempotency-key record persistence over Deno KV. */

import type { IdempotencyKeyRecord } from "../domain/types.ts";
import { idempotencyKey, type Store } from "./kv.ts";

export class IdempotencyRepository {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async get(
    playerId: string,
    scope: string,
    key: string,
  ): Promise<IdempotencyKeyRecord | null> {
    return this.#store.get<IdempotencyKeyRecord>(
      idempotencyKey(playerId, scope, key),
    );
  }

  async create(record: IdempotencyKeyRecord): Promise<void> {
    await this.#store.set(
      idempotencyKey(record.playerId, record.scope, record.key),
      record,
    );
  }

  async delete(playerId: string, scope: string, key: string): Promise<void> {
    await this.#store.delete(idempotencyKey(playerId, scope, key));
  }
}
