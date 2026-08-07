/** Voyage lifecycle persistence over Deno KV. */

import type {
  KeeperInstance,
  Principal,
  Voyage,
  VoyageLedgerEntry,
  VoyageLifecycleEvent,
} from "../domain/types.ts";
import { notFound, TidekeepersError } from "../utils/errors.ts";
import {
  type Store,
  voyageActiveKey,
  voyageKey,
  voyagePublicKey,
} from "./kv.ts";

export class VoyageRepository {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async getVoyage(id: string): Promise<Voyage | null> {
    return this.#store.get<Voyage>(voyageKey(id));
  }

  async getVoyageByPublicId(
    publicId: string,
  ): Promise<{ voyageId: string; playerId: string } | null> {
    return this.#store.get<{ voyageId: string; playerId: string }>(
      voyagePublicKey(publicId),
    );
  }

  /** The player's active voyage id, if any. */
  async getActiveVoyageId(playerId: string): Promise<string | null> {
    return this.#store.get<string>(voyageActiveKey(playerId));
  }

  async getVoyageOrFail(id: string): Promise<Voyage> {
    const voyage = await this.getVoyage(id);
    if (!voyage) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    return voyage;
  }

  async assertVoyageOwnership(
    voyage: Voyage,
    principal: Principal,
  ): Promise<void> {
    if (voyage.playerId !== principal.playerId) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
  }

  async listKeepers(voyageId: string): Promise<KeeperInstance[]> {
    const keepers = await this.#store.listValues<KeeperInstance>(
      voyageKeeperPrefix(voyageId),
    );
    keepers.sort((a, b) => {
      const at = new Date(a.acquiredAt).getTime() -
        new Date(b.acquiredAt).getTime();
      if (at !== 0) {
        return at;
      }
      return a.definitionKey < b.definitionKey
        ? -1
        : a.definitionKey > b.definitionKey
        ? 1
        : 0;
    });
    return keepers;
  }

  async listEvents(voyageId: string): Promise<VoyageLifecycleEvent[]> {
    const events = await this.#store.listValues<VoyageLifecycleEvent>(
      voyageEventPrefix(voyageId),
    );
    events.sort((a, b) => {
      const at = new Date(a.occurredAt).getTime() -
        new Date(b.occurredAt).getTime();
      if (at !== 0) {
        return at;
      }
      return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
    });
    return events;
  }

  async listLedger(voyageId: string): Promise<VoyageLedgerEntry[]> {
    const entries = await this.#store.listValues<VoyageLedgerEntry>(
      voyageLedgerPrefix(voyageId),
    );
    entries.sort((a, b) => {
      const at = new Date(a.occurredAt).getTime() -
        new Date(b.occurredAt).getTime();
      if (at !== 0) {
        return at;
      }
      return a.id < b.id ? -1 : a.id > b.id ? 1 : 0;
    });
    return entries;
  }
}

export function voyageKeeperPrefix(voyageId: string): Deno.KvKey {
  return ["voyage_keeper", voyageId];
}

export function voyageEventPrefix(voyageId: string): Deno.KvKey {
  return ["voyage_event", voyageId];
}

export function voyageLedgerPrefix(voyageId: string): Deno.KvKey {
  return ["voyage_ledger", voyageId];
}

export function assertKeeperBelongsToVoyage(
  keeper: KeeperInstance,
  voyageId: string,
): void {
  if (keeper.voyageId !== voyageId) {
    throw new TidekeepersError(
      "keeper does not belong to voyage",
      "not_found",
      { code: "KEEPER_NOT_FOUND" },
    );
  }
}
