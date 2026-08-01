/** Read-only gameplay content repository backed by Deno KV. */

import type {
  GameRuleSet,
  Modifier,
  Objective,
  Release,
  Relic,
  Strategy,
  Synergy,
} from "../content/model.ts";
import type { ContentReleaseRecord } from "../content/publisher.ts";
import { contentReleaseKey, type Store } from "./kv.ts";
import { notFound } from "../utils/errors.ts";

export class ContentRepository {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async loadRelease(version: number): Promise<Release> {
    const record = await this.#store.get<ContentReleaseRecord>(
      contentReleaseKey(version),
    );
    if (!record || record.status !== "PUBLISHED") {
      throw notFound(`content release ${version} not found`, {
        code: "CONTENT_RELEASE_NOT_FOUND",
      });
    }
    return record.release;
  }

  async getPublishedRelease(version: number): Promise<Release | null> {
    const record = await this.#store.get<ContentReleaseRecord>(
      contentReleaseKey(version),
    );
    return record?.status === "PUBLISHED" ? record.release : null;
  }

  async getStrategy(version: number, key: string): Promise<Strategy | null> {
    return this.find(version, (release) => release.strategies, key);
  }

  async getRelic(version: number, key: string): Promise<Relic | null> {
    return this.find(version, (release) => release.relics, key);
  }

  async getSynergy(version: number, key: string): Promise<Synergy | null> {
    return this.find(version, (release) => release.synergies, key);
  }

  async getModifier(version: number, key: string): Promise<Modifier | null> {
    return this.find(version, (release) => release.modifiers, key);
  }

  async getObjective(version: number, key: string): Promise<Objective | null> {
    return this.find(version, (release) => release.objectives, key);
  }

  async getGameRuleSet(
    version: number,
    key: string,
  ): Promise<GameRuleSet | null> {
    return this.find(version, (release) => release.gameRuleSets, key);
  }

  async getKeeperCatalog(version: number) {
    const release = await this.getPublishedRelease(version);
    return release?.keeper ?? null;
  }

  async getKeeperDefinition(version: number, key: string) {
    const catalog = await this.getKeeperCatalog(version);
    return catalog?.definitions.find((definition) => definition.key === key) ??
      null;
  }

  private async find<T extends { key: string }>(
    version: number,
    select: (release: Release) => T[],
    key: string,
  ): Promise<T | null> {
    const release = await this.getPublishedRelease(version);
    return release
      ? select(release).find((definition) => definition.key === key) ?? null
      : null;
  }
}
