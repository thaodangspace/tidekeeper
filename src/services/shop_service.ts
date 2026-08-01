/** Shop purchases backed by one Deno KV transaction. */

import type {
  KeeperInstance,
  PlayerDailyContextView,
  PlayerDailyState,
  PlayerKeeperUnlock,
  Voyage,
} from "../domain/types.ts";
import type { DailyRepository } from "../repositories/daily_repository.ts";
import type { VoyageRepository } from "../repositories/voyage_repository.ts";
import {
  playerUnlockKey,
  type Store,
  voyageDailyStateKey,
  voyageDailyViewKey,
  voyageKeeperKey,
  voyageKey,
} from "../repositories/kv.ts";
import type { Keeper, Shop } from "./daily_service.ts";
import { rootUpgradeNodeKey } from "../content/voyage_definition.ts";
import {
  conflict,
  internal,
  notFound,
  validationFailed,
} from "../utils/errors.ts";
import { newId } from "../utils/ids.ts";

export interface PurchaseRequest {
  expectedVersion: number;
}

export interface PurchaseResponse {
  voyageId: string;
  dayNumber: number;
  version: number;
  capital: number;
  keeper: Keeper;
}

interface Projection {
  shop: Shop;
  inventory: Keeper[];
  [key: string]: unknown;
}

export class ShopService {
  #store: Store;
  #daily: DailyRepository;
  #voyages: VoyageRepository;

  constructor(
    store: Store,
    daily: DailyRepository,
    voyages: VoyageRepository,
  ) {
    this.#store = store;
    this.#daily = daily;
    this.#voyages = voyages;
  }

  async purchase(
    playerId: string,
    offerId: string,
    request: PurchaseRequest,
    now: Date,
  ): Promise<PurchaseResponse> {
    if (
      !Number.isInteger(request.expectedVersion) || request.expectedVersion < 1
    ) {
      throw validationFailed(
        "expectedVersion must be a positive integer",
        "expectedVersion",
        {
          code: "SHOP_INVALID",
        },
      );
    }
    const activeId = await this.#voyages.getActiveVoyageId(playerId);
    if (activeId === null) {
      throw notFound("no active voyage", { code: "NO_ACTIVE_VOYAGE" });
    }
    const voyageEntry = await this.#store.kv.get<Voyage>(voyageKey(activeId));
    const voyage = voyageEntry.value;
    if (!voyage || voyage.playerId !== playerId || !voyageEntry.versionstamp) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    const stateEntry = await this.#daily.getPlayerDailyStateEntry(
      voyage.id,
      voyage.currentDayNumber,
    );
    const viewEntry = await this.#daily.getPlayerDailyViewEntry(
      voyage.id,
      voyage.currentDayNumber,
    );
    const state = stateEntry?.value;
    const view = viewEntry?.value;
    if (!state || !view) throw internal("missing daily shop state");
    if (state.phase !== "PREPARATION") {
      throw conflict("the shop is closed", { code: "INVALID_PHASE" });
    }
    if (state.version !== request.expectedVersion) {
      throw conflict("daily state version conflict", {
        code: "VERSION_CONFLICT",
      });
    }
    const projection = asProjection(view.projection);
    const offer = projection.shop.offers.find((candidate) =>
      candidate.id === offerId
    );
    if (!offer) {
      throw notFound("shop offer not found", { code: "SHOP_OFFER_NOT_FOUND" });
    }
    if (!offer.available) {
      throw conflict("shop offer is no longer available", {
        code: "SHOP_OFFER_UNAVAILABLE",
      });
    }
    if (voyage.capital < offer.cost) {
      throw conflict("insufficient capital", { code: "INSUFFICIENT_CAPITAL" });
    }

    const keeper = makeKeeper(voyage, offer.keeper, now);
    const updatedProjection: Projection = {
      ...projection,
      inventory: [...projection.inventory, keeper],
      shop: {
        ...projection.shop,
        offers: projection.shop.offers.map((candidate) =>
          candidate.id === offerId
            ? { ...candidate, available: false }
            : candidate
        ),
      },
    };
    const updatedState: PlayerDailyState = {
      ...state,
      version: state.version + 1,
      updatedAt: now.toISOString(),
    };
    const updatedView: PlayerDailyContextView = {
      ...view,
      projection: updatedProjection,
      validatedAt: now.toISOString(),
    };
    const updatedVoyage: Voyage = {
      ...voyage,
      capital: voyage.capital - offer.cost,
      rowVersion: voyage.rowVersion + 1,
      updatedAt: now.toISOString(),
    };
    const unlockKey = playerUnlockKey(playerId, offer.keeper.definitionId);
    const existingUnlock = await this.#store.get<PlayerKeeperUnlock>(unlockKey);
    const atomic = this.#store.atomic()
      .check({
        key: voyageKey(voyage.id),
        versionstamp: voyageEntry.versionstamp,
      })
      .check({
        key: voyageDailyStateKey(state.voyageId, state.dayNumber),
        versionstamp: stateEntry!.versionstamp,
      })
      .check({
        key: voyageDailyViewKey(view.voyageId, view.dayNumber),
        versionstamp: viewEntry!.versionstamp,
      })
      .set(voyageKey(voyage.id), updatedVoyage)
      .set(voyageDailyStateKey(state.voyageId, state.dayNumber), updatedState)
      .set(voyageDailyViewKey(view.voyageId, view.dayNumber), updatedView)
      .set(voyageKeeperKey(voyage.id, keeper.id), keeper);
    if (!existingUnlock) {
      const unlock: PlayerKeeperUnlock = {
        playerId,
        keeperKey: offer.keeper.definitionId,
        definitionVersion: 1,
        name: offer.keeper.name,
        currentName: offer.keeper.name,
        sector: offer.keeper.sector,
        role: offer.keeper.role,
        rarity: offer.keeper.rarity,
        unlockSource: "SHOP",
        unlockedAt: now.toISOString(),
      };
      atomic.check({ key: unlockKey, versionstamp: null }).set(
        unlockKey,
        unlock,
      );
    }
    const committed = await atomic.commit();
    if (!committed.ok) {
      throw conflict("shop state changed", { code: "VERSION_CONFLICT" });
    }
    return {
      voyageId: voyage.publicId,
      dayNumber: voyage.currentDayNumber,
      version: updatedState.version,
      capital: updatedVoyage.capital,
      keeper,
    };
  }
}

function makeKeeper(
  voyage: Voyage,
  preview: Keeper,
  now: Date,
): Keeper & KeeperInstance {
  const id = newId();
  const publicId = `kpr_${id}`;
  const nodeKey = rootUpgradeNodeKey(preview.definitionId) ?? "base";
  return {
    id: publicId,
    publicId,
    voyageId: voyage.id,
    definitionKey: preview.definitionId,
    definitionVersion: 1,
    name: preview.name,
    currentName: preview.name,
    sector: preview.sector,
    role: preview.role,
    rarity: preview.rarity,
    upgradeNodeKey: nodeKey,
    nodeDepth: 0,
    acquiredDay: voyage.currentDayNumber,
    acquiredSource: "SHOP",
    acquiredAt: now.toISOString(),
    definitionId: preview.definitionId,
    level: preview.level,
    passiveSummary: preview.passiveSummary,
    artworkUrl: preview.artworkUrl,
  };
}

function asProjection(value: unknown): Projection {
  if (!value || typeof value !== "object") {
    throw internal("invalid daily projection");
  }
  const projection = value as Partial<Projection>;
  if (
    !projection.shop || !Array.isArray(projection.shop.offers) ||
    !Array.isArray(projection.inventory)
  ) {
    throw internal("invalid daily shop projection");
  }
  return projection as Projection;
}
