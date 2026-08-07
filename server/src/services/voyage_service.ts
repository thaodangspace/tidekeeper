/** Voyage lifecycle and inventory service (ported from voyage/). */

import type {
  IdempotencyKeyRecord,
  KeeperInstance,
  Player,
  Voyage,
  VoyageLedgerEntry,
  VoyageLifecycleEvent,
} from "../domain/types.ts";
import {
  buildInitialProjection,
  getPublishedVoyageDefinition,
  getVoyageDefinitionInitialOffers,
  getVoyageDefinitionStarterKeepers,
  rootUpgradeNodeKey,
} from "../content/voyage_definition.ts";
import {
  badRequest,
  conflict,
  notFound,
  serviceUnavailable,
  TidekeepersError,
} from "../utils/errors.ts";
import { newId } from "../utils/ids.ts";
import { base64UrlEncodeBytes, sha256Hex } from "../utils/crypto.ts";
import type { VoyageRepository } from "../repositories/voyage_repository.ts";
import type { DailyRepository } from "../repositories/daily_repository.ts";
import type { IdempotencyRepository } from "../repositories/idempotency_repository.ts";
import { type Store, voyageActiveKey } from "../repositories/kv.ts";
import { dayKeyInTimezone } from "../utils/time.ts";

export type VoyageStatus = "ACTIVE" | "COMPLETED" | "FAILED" | "ABANDONED";

const idempotencyScopeCreate = "voyage_create";
const idempotencyScopeAbandon = "voyage_abandon";
const ledgerEntryHull = "START_HULL";
const ledgerEntrySupplies = "START_SUPPLIES";
const lifecycleEventCreated = "CREATED";
const lifecycleEventAbandoned = "ABANDONED";
const acquisitionSourceStarter = "STARTER";

export interface VoyageResponse {
  id: string;
  definitionKey: string;
  definitionVersion: number;
  status: string;
  dayNumber: number;
  fundHealth: number;
  maxFundHealth: number;
  capital: number;
  score: string;
  rowVersion: number;
  startedAt: string;
  completedAt: string | null;
}

export interface LifecycleEvent {
  id: string;
  eventType: string;
  resultingStatus: string;
  occurredAt: string;
}

export interface VoyageHistoryResponse {
  voyage: VoyageResponse;
  events: LifecycleEvent[];
}

export interface VoyageKeeper {
  id: string;
  definitionKey: string;
  definitionVersion: number;
  name: string;
  currentName: string;
  sector: string;
  role: string;
  rarity: string;
  upgradeNodeKey: string;
  level: number;
  acquiredDay: number;
  acquiredSource: string;
  acquiredAt: string;
}

export interface VoyageKeepers {
  voyageId: string;
  keepers: VoyageKeeper[];
}

interface KeeperInstanceSeed {
  definitionVersionId: string;
  key: string;
  name: string;
  sector: string;
  role: string;
  rarity: string;
}

interface InitialOfferSeed {
  keeperKey: string;
  name: string;
  sector: string;
  role: string;
  rarity: string;
  cost: number;
}

export class VoyageService {
  #store: Store;
  #voyages: VoyageRepository;
  #daily: DailyRepository;
  #idempotency: IdempotencyRepository;
  #gameTimezone: string;

  constructor(
    store: Store,
    voyages: VoyageRepository,
    daily: DailyRepository,
    idempotency: IdempotencyRepository,
    gameTimezone = "UTC",
  ) {
    this.#store = store;
    this.#voyages = voyages;
    this.#daily = daily;
    this.#idempotency = idempotency;
    this.#gameTimezone = gameTimezone;
  }

  async create(
    playerId: string,
    idempotencyKey: string,
    now: Date,
  ): Promise<VoyageResponse> {
    if (!validKeyFormat(idempotencyKey)) {
      throw badRequest("invalid idempotency key", {
        code: "INVALID_IDEMPOTENCY_KEY",
      });
    }
    const requestHash = await sha256Hex(
      new TextEncoder().encode(idempotencyScopeCreate),
    );

    const claim = await this.claimKey(
      playerId,
      idempotencyScopeCreate,
      idempotencyKey,
      requestHash,
    );
    if (!claim.isNew) {
      return replayVoyageResponse(claim.record!);
    }

    const existingActive = await this.#voyages.getActiveVoyageId(playerId);
    if (existingActive !== null) {
      throw conflict("an active voyage already exists", {
        code: "ACTIVE_VOYAGE_EXISTS",
      });
    }

    const def = getPublishedVoyageDefinition();
    const tide = await this.currentDailyTide(now);
    if (!tide) {
      throw serviceUnavailable(
        "voyage initialization is currently unavailable",
        {
          code: "VOYAGE_INITIALIZATION_UNAVAILABLE",
        },
      );
    }

    const starters = getVoyageDefinitionStarterKeepers(def.id);
    const offers = getVoyageDefinitionInitialOffers(def.id);

    const voyageId = newId();
    const publicId = newPublicId("voy_");
    const nowIso = now.toISOString();
    const voyage: Voyage = {
      id: voyageId,
      publicId,
      playerId,
      status: "ACTIVE",
      definitionKey: def.definitionKey,
      definitionVersion: def.version,
      currentDayNumber: 1,
      fundHealth: def.startingHull,
      maxFundHealth: def.startingHull,
      capital: def.startingSupplies,
      score: "0.0000",
      rowVersion: 1,
      startedAt: nowIso,
      completedAt: null,
      createdAt: nowIso,
      updatedAt: nowIso,
    };

    const instances: KeeperInstanceSeed[] = starters.map((starter) => ({
      definitionVersionId: starter.keeperDefinitionVersionId,
      key: starter.keeperKey,
      name: starter.name,
      sector: starter.sectorKey,
      role: starter.roleKey,
      rarity: starter.rarityKey,
    }));
    const offerSeeds: InitialOfferSeed[] = offers.map((offer) => ({
      keeperKey: offer.keeperKey,
      name: offer.name,
      sector: offer.sectorKey,
      role: offer.roleKey,
      rarity: offer.rarityKey,
      cost: offer.cost,
    }));

    const atomic = this.#store.atomic()
      .check({ key: voyageActiveKey(playerId), versionstamp: null })
      .set(voyageActiveKey(playerId), voyageId);
    atomic.set(this.voyageKey(voyageId), voyage);
    atomic.set(this.voyagePublicKey(publicId), { voyageId, playerId });

    const ledger: VoyageLedgerEntry[] = [];
    if (def.startingHull > 0) {
      ledger.push(
        this.ledgerEntry(
          voyageId,
          publicId,
          ledgerEntryHull,
          def.startingHull,
          nowIso,
        ),
      );
    }
    if (def.startingSupplies > 0) {
      ledger.push(
        this.ledgerEntry(
          voyageId,
          publicId,
          ledgerEntrySupplies,
          def.startingSupplies,
          nowIso,
        ),
      );
    }
    for (const entry of ledger) {
      atomic.set(this.ledgerKey(entry.voyageId, entry.id), entry);
    }

    const keeperInstances: KeeperInstance[] = instances.map((seed) =>
      this.keeperInstance(voyageId, seed, 1, acquisitionSourceStarter, nowIso)
    );
    for (const keeper of keeperInstances) {
      atomic.set(this.keeperKey(keeper.voyageId, keeper.id), keeper);
    }

    const stateId = newId();
    const projection = buildInitialProjection({
      fleetSlotCount: def.fleetSlotCount,
      instances: keeperInstances.map((k) => ({
        publicId: k.publicId,
        key: k.definitionKey,
        name: k.name,
        sector: k.sector,
        role: k.role,
        rarity: k.rarity,
      })),
      offers: offerSeeds,
    });
    atomic.set(this.dailyStateKey(voyageId, 1), {
      id: stateId,
      playerId,
      voyageId,
      dailyTideId: tide.id,
      dayNumber: 1,
      phase: "PREPARATION",
      version: 1,
      selectedStrategyId: null,
      pendingRewardCount: 0,
      createdAt: nowIso,
      updatedAt: nowIso,
    });
    atomic.set(this.dailyViewKey(voyageId, 1), {
      voyageId,
      dayNumber: 1,
      schemaVersion: 1,
      projection,
      validatedAt: nowIso,
    });

    const eventId = newId();
    atomic.set(
      this.eventKey(voyageId, eventId),
      {
        id: eventId,
        publicId: newPublicId("evt_"),
        voyageId,
        eventType: lifecycleEventCreated,
        resultingStatus: "ACTIVE",
        occurredAt: nowIso,
      } satisfies VoyageLifecycleEvent,
    );

    const player = await this.#store.get<Player>(["player", playerId]);
    if (player) {
      const updated: Player = {
        ...player,
        currentVoyageId: voyageId,
        onboardingCompleted: true,
        updatedAt: nowIso,
      };
      atomic.set(["player", playerId], updated);
    }

    const resp = makeResponse(voyage);
    const responseJson = JSON.stringify(resp);
    atomic.set(
      this.idempotencyKey(playerId, idempotencyScopeCreate, idempotencyKey),
      {
        playerId,
        scope: idempotencyScopeCreate,
        key: idempotencyKey,
        state: "COMPLETED",
        requestHash,
        resultVoyageId: voyageId,
        responseStatus: 201,
        responseJson,
        createdAt: nowIso,
        completedAt: nowIso,
      } satisfies IdempotencyKeyRecord,
    );

    const result = await atomic.commit();
    if (!result.ok) {
      throw conflict("an active voyage already exists", {
        code: "ACTIVE_VOYAGE_EXISTS",
      });
    }
    return resp;
  }

  async getCurrent(playerId: string): Promise<VoyageResponse> {
    const activeId = await this.#voyages.getActiveVoyageId(playerId);
    if (activeId === null) {
      throw notFound("no current voyage", { code: "NO_ACTIVE_VOYAGE" });
    }
    const voyage = await this.#voyages.getVoyageOrFail(activeId);
    return makeResponse(voyage);
  }

  async getById(playerId: string, publicId: string): Promise<VoyageResponse> {
    const resolved = await this.#voyages.getVoyageByPublicId(publicId);
    if (!resolved || resolved.playerId !== playerId) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    const voyage = await this.#voyages.getVoyageOrFail(resolved.voyageId);
    return makeResponse(voyage);
  }

  async abandon(
    playerId: string,
    voyageId: string,
    idempotencyKey: string,
    now: Date,
  ): Promise<VoyageResponse> {
    if (!validKeyFormat(idempotencyKey)) {
      throw badRequest("invalid idempotency key", {
        code: "INVALID_IDEMPOTENCY_KEY",
      });
    }
    const requestHash = await sha256Hex(
      new TextEncoder().encode(`${idempotencyScopeAbandon}:${voyageId}`),
    );

    const claim = await this.claimKey(
      playerId,
      idempotencyScopeAbandon,
      idempotencyKey,
      requestHash,
    );
    if (!claim.isNew) {
      return replayVoyageResponse(claim.record!);
    }

    const resolved = await this.#voyages.getVoyageByPublicId(voyageId);
    if (!resolved || resolved.playerId !== playerId) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    const voyage = await this.#voyages.getVoyageOrFail(resolved.voyageId);

    if (voyage.status === "ABANDONED") {
      const resp = makeResponse(voyage);
      await this.completeKey(
        playerId,
        idempotencyScopeAbandon,
        idempotencyKey,
        requestHash,
        voyage.id,
        200,
        resp,
        now,
      );
      return resp;
    }
    if (voyage.status !== "ACTIVE") {
      throw conflict("voyage is not active", { code: "VOYAGE_NOT_ACTIVE" });
    }

    const updated: Voyage = {
      ...voyage,
      status: "ABANDONED",
      completedAt: now.toISOString(),
      rowVersion: voyage.rowVersion + 1,
      updatedAt: now.toISOString(),
    };
    const eventId = newId();
    const event: VoyageLifecycleEvent = {
      id: eventId,
      publicId: newPublicId("evt_"),
      voyageId: voyage.id,
      eventType: lifecycleEventAbandoned,
      resultingStatus: "ABANDONED",
      occurredAt: now.toISOString(),
    };

    const atomic = this.#store.atomic()
      .check({
        key: this.voyageKey(voyage.id),
        versionstamp: await this.#store.versionstamp(this.voyageKey(voyage.id)),
      })
      .set(this.voyageKey(voyage.id), updated)
      .set(this.eventKey(voyage.id, event.id), event);
    atomic.delete(voyageActiveKey(playerId));
    const result = await atomic.commit();
    if (!result.ok) {
      throw conflict("voyage is not active", { code: "VOYAGE_NOT_ACTIVE" });
    }

    const resp = makeResponse(updated);
    await this.completeKey(
      playerId,
      idempotencyScopeAbandon,
      idempotencyKey,
      requestHash,
      voyage.id,
      200,
      resp,
      now,
    );
    return resp;
  }

  async getHistory(
    playerId: string,
    voyageId: string,
  ): Promise<VoyageHistoryResponse> {
    const resolved = await this.#voyages.getVoyageByPublicId(voyageId);
    if (!resolved || resolved.playerId !== playerId) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    const voyage = await this.#voyages.getVoyageOrFail(resolved.voyageId);
    const events = await this.#voyages.listEvents(voyage.id);
    return {
      voyage: makeResponse(voyage),
      events: events.map((event) => ({
        id: event.publicId,
        eventType: event.eventType,
        resultingStatus: event.resultingStatus,
        occurredAt: event.occurredAt,
      })),
    };
  }

  async getActiveVoyageKeepers(playerId: string): Promise<VoyageKeepers> {
    const activeId = await this.#voyages.getActiveVoyageId(playerId);
    if (activeId === null) {
      throw notFound("no current voyage", { code: "NO_ACTIVE_VOYAGE" });
    }
    const voyage = await this.#voyages.getVoyageOrFail(activeId);
    const keepers = await this.#voyages.listKeepers(voyage.id);
    return {
      voyageId: voyage.publicId,
      keepers: keepers.map((keeper) => ({
        id: keeper.publicId,
        definitionKey: keeper.definitionKey,
        definitionVersion: keeper.definitionVersion,
        name: keeper.name,
        currentName: keeper.currentName,
        sector: keeper.sector,
        role: keeper.role,
        rarity: keeper.rarity,
        upgradeNodeKey: keeper.upgradeNodeKey,
        level: keeper.nodeDepth + 1,
        acquiredDay: keeper.acquiredDay,
        acquiredSource: keeper.acquiredSource,
        acquiredAt: keeper.acquiredAt,
      })),
    };
  }

  // ── Internals ──────────────────────────────────────────────────────────────

  async currentDailyTide(now: Date): Promise<{ id: string } | null> {
    const dayKey = dayKeyInTimezone(now, this.#gameTimezone);
    return this.#daily.getDailyTideByDayKey(dayKey);
  }

  async claimKey(
    playerId: string,
    scope: string,
    key: string,
    requestHash: string,
  ): Promise<{ isNew: boolean; record: IdempotencyKeyRecord | null }> {
    const existing = await this.#idempotency.get(playerId, scope, key);
    if (!existing) {
      return { isNew: true, record: null };
    }
    if (existing.state !== "COMPLETED") {
      throw new TidekeepersError(
        "stale in-progress idempotency key",
        "internal",
        { code: "INTERNAL_ERROR" },
      );
    }
    if (existing.requestHash !== requestHash) {
      throw conflict("idempotency key was reused with a different request", {
        code: "IDEMPOTENCY_KEY_REUSED",
      });
    }
    return { isNew: false, record: existing };
  }

  async completeKey(
    playerId: string,
    scope: string,
    key: string,
    requestHash: string,
    resultVoyageId: string,
    responseStatus: number,
    responseJson: unknown,
    now: Date,
  ): Promise<void> {
    await this.#idempotency.create(
      {
        playerId,
        scope,
        key,
        state: "COMPLETED",
        requestHash,
        resultVoyageId,
        responseStatus,
        responseJson: JSON.stringify(responseJson),
        createdAt: now.toISOString(),
        completedAt: now.toISOString(),
      } satisfies IdempotencyKeyRecord,
    );
  }

  voyageKey(id: string): Deno.KvKey {
    return ["voyage", id];
  }

  voyagePublicKey(publicId: string): Deno.KvKey {
    return ["voyage_public", publicId];
  }

  ledgerKey(voyageId: string, entryId: string): Deno.KvKey {
    return ["voyage_ledger", voyageId, entryId];
  }

  keeperKey(voyageId: string, id: string): Deno.KvKey {
    return ["voyage_keeper", voyageId, id];
  }

  eventKey(voyageId: string, id: string): Deno.KvKey {
    return ["voyage_event", voyageId, id];
  }

  dailyStateKey(voyageId: string, dayNumber: number): Deno.KvKey {
    return ["voyage_daily_state", voyageId, dayNumber];
  }

  dailyViewKey(voyageId: string, dayNumber: number): Deno.KvKey {
    return ["voyage_daily_view", voyageId, dayNumber];
  }

  idempotencyKey(playerId: string, scope: string, key: string): Deno.KvKey {
    return ["idempotency", playerId, scope, key];
  }

  ledgerEntry(
    voyageId: string,
    sourceId: string,
    entryType: string,
    amount: number,
    nowIso: string,
  ): VoyageLedgerEntry {
    return {
      id: newId(),
      voyageId,
      entryType,
      amount,
      balanceAfter: amount,
      sourceType: "VOYAGE_START",
      sourceId,
      reasonKey: `voyage.create.${entryType.toLowerCase()}`,
      occurredAt: nowIso,
    };
  }

  keeperInstance(
    voyageId: string,
    seed: KeeperInstanceSeed,
    day: number,
    source: string,
    nowIso: string,
  ): KeeperInstance {
    const nodeKey = rootUpgradeNodeKey(seed.key) ?? "base";
    return {
      id: newId(),
      publicId: newPublicId("kpr_"),
      voyageId,
      definitionKey: seed.key,
      definitionVersion: 1,
      name: seed.name,
      currentName: seed.name,
      sector: seed.sector,
      role: seed.role,
      rarity: seed.rarity,
      upgradeNodeKey: nodeKey,
      nodeDepth: 0,
      acquiredDay: day,
      acquiredSource: source,
      acquiredAt: nowIso,
    };
  }
}

function makeResponse(voyage: Voyage): VoyageResponse {
  return {
    id: voyage.publicId,
    definitionKey: voyage.definitionKey,
    definitionVersion: voyage.definitionVersion,
    status: voyage.status,
    dayNumber: voyage.currentDayNumber,
    fundHealth: voyage.fundHealth,
    maxFundHealth: voyage.maxFundHealth,
    capital: voyage.capital,
    score: voyage.score,
    rowVersion: voyage.rowVersion,
    startedAt: voyage.startedAt,
    completedAt: voyage.completedAt,
  };
}

function replayVoyageResponse(record: IdempotencyKeyRecord): VoyageResponse {
  if (record.responseStatus !== 201 && record.responseStatus !== 200) {
    throw new TidekeepersError("unexpected replayed status", "internal", {
      code: "INTERNAL_ERROR",
    });
  }
  if (record.responseJson === null) {
    throw new TidekeepersError("missing replayed response", "internal", {
      code: "INTERNAL_ERROR",
    });
  }
  return JSON.parse(record.responseJson) as VoyageResponse;
}

function validKeyFormat(key: string): boolean {
  if (key.length < 1 || key.length > 255) {
    return false;
  }
  if (!/[A-Za-z0-9]/.test(key[0]!)) {
    return false;
  }
  for (const char of key) {
    if (!/[A-Za-z0-9._:-]/.test(char)) {
      return false;
    }
  }
  return true;
}

function newPublicId(prefix: string): string {
  for (;;) {
    const bytes = crypto.getRandomValues(new Uint8Array(12));
    const id = prefix + base64UrlEncodeBytes(bytes);
    if (id[prefix.length] !== "-" && id[prefix.length] !== "_") {
      return id;
    }
  }
}
