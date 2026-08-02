/** Immutable gameplay-content publication over Deno KV. */

import { contentChecksum } from "./checksum.ts";
import type { Release } from "./model.ts";
import { parseUpgradeTree } from "./upgrade_tree.ts";
import { contentReleaseKey, type Store } from "../repositories/kv.ts";
import { conflict } from "../utils/errors.ts";

export interface ContentRowCounts {
  basketMappingCount: number;
  keeperDefinitionCount: number;
  componentCount: number;
  nodeCount: number;
  strategyCount: number;
  relicCount: number;
  synergyCount: number;
  modifierCount: number;
  objectiveCount: number;
  gameRuleSetCount: number;
}

export interface ContentReleaseRecord {
  version: number;
  checksum: string;
  checksumSchemaVersion: 2;
  status: "PUBLISHED";
  release: Release;
  counts: ContentRowCounts;
  publishedAt: string;
}

export interface PublishResult extends ContentRowCounts {
  version: number;
  checksum: string;
  idempotent: boolean;
}

export class ContentPublisher {
  #store: Store;

  constructor(store: Store) {
    this.#store = store;
  }

  async publish(release: Release, now = new Date()): Promise<PublishResult> {
    const checksum = await contentChecksum(release);
    const counts = countReleaseRows(release);
    const key = contentReleaseKey(release.version);
    const existing = await this.#store.get<ContentReleaseRecord>(key);
    if (existing) {
      assertCompatible(existing, checksum, counts, release.version);
      return result(existing, true);
    }
    const record: ContentReleaseRecord = {
      version: release.version,
      checksum,
      checksumSchemaVersion: 2,
      status: "PUBLISHED",
      release,
      counts,
      publishedAt: now.toISOString(),
    };
    const committed = await this.#store.atomic()
      .check({ key, versionstamp: null })
      .set(key, record)
      .commit();
    if (!committed.ok) {
      const winner = await this.#store.get<ContentReleaseRecord>(key);
      if (winner) {
        assertCompatible(winner, checksum, counts, release.version);
        return result(winner, true);
      }
      throw conflict("content release publication conflicted", {
        code: "CONTENT_RELEASE_CONFLICT",
      });
    }
    return {
      version: release.version,
      checksum,
      ...counts,
      idempotent: false,
    };
  }

  async get(version: number): Promise<ContentReleaseRecord | null> {
    return this.#store.get<ContentReleaseRecord>(contentReleaseKey(version));
  }
}

function countReleaseRows(release: Release): ContentRowCounts {
  let componentCount = 0;
  let nodeCount = 0;
  for (const basket of release.keeper.baskets) {
    componentCount += basket.components.length;
  }
  for (const definition of release.keeper.definitions) {
    nodeCount += parseUpgradeTree(definition.upgradeTree).length;
  }
  return {
    basketMappingCount: release.keeper.baskets.length,
    keeperDefinitionCount: release.keeper.definitions.length,
    componentCount,
    nodeCount,
    strategyCount: release.strategies.length,
    relicCount: release.relics.length,
    synergyCount: release.synergies.length,
    modifierCount: release.modifiers.length,
    objectiveCount: release.objectives.length,
    gameRuleSetCount: release.gameRuleSets.length,
  };
}

function assertCompatible(
  existing: ContentReleaseRecord,
  checksum: string,
  counts: ContentRowCounts,
  version: number,
): void {
  if (
    existing.status !== "PUBLISHED" ||
    existing.checksumSchemaVersion !== 2 ||
    existing.checksum !== checksum ||
    JSON.stringify(existing.counts) !== JSON.stringify(counts)
  ) {
    throw conflict(
      `content release ${version} conflicts with persisted content`,
      {
        code: "CONTENT_RELEASE_CONFLICT",
      },
    );
  }
}

function result(
  record: ContentReleaseRecord,
  idempotent: boolean,
): PublishResult {
  return {
    version: record.version,
    checksum: record.checksum,
    ...record.counts,
    idempotent,
  };
}
