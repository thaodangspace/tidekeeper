/** Player identity and meta-progression reads over Deno KV. */

import type { Player, PlayerKeeperUnlock, Voyage } from "../domain/types.ts";
import { playerKey, playerUnlockKey, type Store, voyageKey } from "./kv.ts";

export class PlayerRepository {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async getPlayer(id: string): Promise<Player | null> {
    return this.#store.get<Player>(playerKey(id));
  }

  async getPlayerMe(id: string): Promise<
    {
      publicId: string;
      onboardingCompleted: boolean;
      locale: string;
      timezone: string;
      currentVoyageId: string | null;
    } | null
  > {
    const player = await this.#store.get<Player>(playerKey(id));
    if (!player) {
      return null;
    }
    let currentVoyageId: string | null = null;
    if (player.currentVoyageId !== null) {
      const voyage = await this.#store.get<Voyage>(
        voyageKey(player.currentVoyageId),
      );
      currentVoyageId = voyage?.publicId ?? null;
    }
    return {
      publicId: player.publicId,
      onboardingCompleted: player.onboardingCompleted,
      locale: player.locale,
      timezone: player.timezone,
      currentVoyageId,
    };
  }

  async listUnlocks(playerId: string): Promise<PlayerKeeperUnlock[]> {
    const unlocks = await this.#store.listValues<PlayerKeeperUnlock>(
      playerUnlockKey(playerId, ""),
    );
    unlocks.sort((a, b) => {
      const at = new Date(a.unlockedAt).getTime() -
        new Date(b.unlockedAt).getTime();
      if (at !== 0) {
        return at;
      }
      return a.keeperKey < b.keeperKey ? -1 : a.keeperKey > b.keeperKey ? 1 : 0;
    });
    return unlocks;
  }
}
