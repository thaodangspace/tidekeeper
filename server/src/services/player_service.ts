/** Player identity and meta-progression reads (ported from player/). */

import type { PlayerRepository } from "../repositories/player_repository.ts";

export interface Me {
  publicId: string;
  displayName: string;
  username: string;
  onboardingCompleted: boolean;
  locale: string;
  timezone: string;
  activeVoyageId: string | null;
}

export interface KeeperUnlock {
  definitionKey: string;
  definitionVersion: number;
  name: string;
  currentName: string;
  sector: string;
  role: string;
  rarity: string;
  unlockSource: string;
  unlockedAt: string;
}

export class PlayerService {
  #players: PlayerRepository;

  constructor(players: PlayerRepository) {
    this.#players = players;
  }

  async getMe(playerId: string): Promise<Me> {
    const me = await this.#players.getPlayerMe(playerId);
    if (!me) {
      throw new Error("player not found");
    }
    return {
      publicId: me.publicId,
      displayName: me.displayName,
      username: me.username,
      onboardingCompleted: me.onboardingCompleted,
      locale: me.locale,
      timezone: me.timezone,
      activeVoyageId: me.currentVoyageId,
    };
  }

  async listUnlocks(playerId: string): Promise<KeeperUnlock[]> {
    const unlocks = await this.#players.listUnlocks(playerId);
    return unlocks.map((unlock) => ({
      definitionKey: unlock.keeperKey,
      definitionVersion: unlock.definitionVersion,
      name: unlock.name,
      currentName: unlock.currentName,
      sector: unlock.sector,
      role: unlock.role,
      rarity: unlock.rarity,
      unlockSource: unlock.unlockSource,
      unlockedAt: unlock.unlockedAt,
    }));
  }
}
