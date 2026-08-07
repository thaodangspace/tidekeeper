/** Deno KV storage wrapper and key helpers. */

export type KvKey = Deno.KvKey;

export function playerKey(id: string): KvKey {
  return ["player", id];
}
export function playerPublicKey(publicId: string): KvKey {
  return ["player_public", publicId];
}
export function sessionKey(digestHex: string): KvKey {
  return ["session", digestHex];
}
export function voyageKey(id: string): KvKey {
  return ["voyage", id];
}
export function voyagePublicKey(publicId: string): KvKey {
  return ["voyage_public", publicId];
}
export function voyageActiveKey(playerId: string): KvKey {
  return ["voyage_active", playerId];
}
export function voyageKeeperKey(voyageId: string, instanceId: string): KvKey {
  return ["voyage_keeper", voyageId, instanceId];
}
export function voyageDailyStateKey(
  voyageId: string,
  dayNumber: number,
): KvKey {
  return ["voyage_daily_state", voyageId, dayNumber];
}
export function voyageDailyViewKey(voyageId: string, dayNumber: number): KvKey {
  return ["voyage_daily_view", voyageId, dayNumber];
}
export function dailyTideKey(dayKey: string): KvKey {
  return ["daily_tide", dayKey];
}
export function dailyTideIdKey(id: string): KvKey {
  return ["daily_tide_id", id];
}
export function dailyTideSequenceKey(sequenceNumber: number): KvKey {
  return ["daily_tide_seq", sequenceNumber];
}
export function idempotencyKey(
  playerId: string,
  scope: string,
  key: string,
): KvKey {
  return ["idempotency", playerId, scope, key];
}
export function playerUnlockKey(playerId: string, keeperKey: string): KvKey {
  return ["player_unlock", playerId, keeperKey];
}
export function settlementKey(voyageId: string, dayNumber: number): KvKey {
  return ["settlement", voyageId, dayNumber];
}
export function rewardClaimKey(voyageId: string, dayNumber: number): KvKey {
  return ["reward_claim", voyageId, dayNumber];
}
export function sequenceKey(name: string): KvKey {
  return ["sequence", name];
}
export function contentReleaseKey(version: number): KvKey {
  return ["content_release", version];
}
export function contentLatestKey(): KvKey {
  return ["content_release_latest"];
}
export function dailyTideStrategyKey(
  dailyTideId: string,
  strategyId: string,
): KvKey {
  return ["daily_tide_strategy", dailyTideId, strategyId];
}

export type DenoKvInstance = Deno.Kv;

export class Store {
  readonly kv: DenoKvInstance;
  constructor(kv: DenoKvInstance) {
    this.kv = kv;
  }
  async get<T>(key: KvKey): Promise<T | null> {
    const entry = await this.kv.get<T>(key);
    return entry.value ?? null;
  }
  async set(key: KvKey, value: unknown): Promise<void> {
    await this.kv.set(key, value);
  }
  async delete(key: KvKey): Promise<void> {
    await this.kv.delete(key);
  }
  async *list<T>(prefix: KvKey): AsyncIterable<{ key: KvKey; value: T }> {
    const iter = this.kv.list<T>({ prefix });
    for await (const entry of iter) {
      yield { key: entry.key, value: entry.value };
    }
  }
  async listValues<T>(prefix: KvKey): Promise<T[]> {
    const values: T[] = [];
    for await (const entry of this.kv.list<T>({ prefix })) {
      values.push(entry.value);
    }
    return values;
  }
  async versionstamp(key: KvKey): Promise<string | null> {
    const entry = await this.kv.get(key);
    return entry.versionstamp;
  }
  atomic(): Deno.AtomicOperation {
    return this.kv.atomic();
  }
}
