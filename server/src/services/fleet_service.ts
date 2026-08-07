/** Fleet lineup editing and locking for the current daily tide. */

import type {
  PlayerDailyContextView,
  PlayerDailyState,
} from "../domain/types.ts";
import type { DailyRepository } from "../repositories/daily_repository.ts";
import type { VoyageRepository } from "../repositories/voyage_repository.ts";
import type { FleetSlot, Keeper, Lineup } from "./daily_service.ts";
import {
  conflict,
  internal,
  notFound,
  validationFailed,
} from "../utils/errors.ts";

export interface SlotInput {
  index: number;
  keeperId: string | null;
}

export interface LineupMutationRequest {
  expectedVersion: number;
  slots: SlotInput[];
}

export interface FleetMutationResponse {
  voyageId: string;
  dayNumber: number;
  version: number;
  lineup: Lineup;
}

interface Projection {
  lineup: Lineup;
  inventory: Keeper[];
  [key: string]: unknown;
}

export class FleetService {
  #daily: DailyRepository;
  #voyages: VoyageRepository;

  constructor(daily: DailyRepository, voyages: VoyageRepository) {
    this.#daily = daily;
    this.#voyages = voyages;
  }

  async updateLineup(
    playerId: string,
    request: LineupMutationRequest,
    now: Date,
  ): Promise<FleetMutationResponse> {
    return this.mutate(playerId, request, now, false);
  }

  async lockFleet(
    playerId: string,
    request: LineupMutationRequest,
    now: Date,
  ): Promise<FleetMutationResponse> {
    return this.mutate(playerId, request, now, true);
  }

  private async mutate(
    playerId: string,
    request: LineupMutationRequest,
    now: Date,
    lock: boolean,
  ): Promise<FleetMutationResponse> {
    if (
      !Number.isInteger(request.expectedVersion) || request.expectedVersion < 1
    ) {
      throw validationFailed(
        "expectedVersion must be a positive integer",
        "expectedVersion",
        {
          code: "LINEUP_INVALID",
        },
      );
    }
    const activeId = await this.#voyages.getActiveVoyageId(playerId);
    if (activeId === null) {
      throw notFound("no active voyage", { code: "NO_ACTIVE_VOYAGE" });
    }
    const voyage = await this.#voyages.getVoyageOrFail(activeId);
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
    if (!state || !view) {
      throw internal("missing daily fleet state");
    }
    if (state.playerId !== playerId) {
      throw notFound("voyage not found", { code: "VOYAGE_NOT_FOUND" });
    }
    if (state.version !== request.expectedVersion) {
      throw conflict("daily state version conflict", {
        code: "VERSION_CONFLICT",
      });
    }
    if (state.phase !== "PREPARATION") {
      throw conflict("the fleet can no longer be changed", {
        code: lock ? "ALREADY_LOCKED" : "INVALID_PHASE",
      });
    }

    const projection = asProjection(view.projection);
    const lineup = normalizeLineup(
      projection.lineup,
      projection.inventory,
      request.slots,
      lock,
      now,
    );
    const updatedProjection: Projection = {
      ...projection,
      lineup,
    };
    const updatedState: PlayerDailyState = {
      ...state,
      phase: lock ? "LOCKED" : state.phase,
      version: state.version + 1,
      updatedAt: now.toISOString(),
    };
    const updatedView: PlayerDailyContextView = {
      ...view,
      projection: updatedProjection,
      validatedAt: now.toISOString(),
    };
    const committed = await this.#daily.updatePlayerDaily(
      updatedState,
      updatedView,
      stateEntry!.versionstamp,
      viewEntry!.versionstamp,
    );
    if (!committed) {
      throw conflict("daily state version conflict", {
        code: "VERSION_CONFLICT",
      });
    }
    return {
      voyageId: voyage.publicId,
      dayNumber: voyage.currentDayNumber,
      version: updatedState.version,
      lineup,
    };
  }
}

function normalizeLineup(
  current: Lineup,
  inventory: Keeper[],
  inputs: SlotInput[],
  lock: boolean,
  now: Date,
): Lineup {
  if (!Array.isArray(inputs) || inputs.length !== current.maxSlots) {
    throw validationFailed(
      `lineup must contain exactly ${current.maxSlots} slots`,
      "slots",
      { code: "LINEUP_INVALID" },
    );
  }
  const byId = new Map(inventory.map((keeper) => [keeper.id, keeper]));
  const seen = new Set<string>();
  const slots: FleetSlot[] = [];
  for (let index = 0; index < current.maxSlots; index++) {
    const input = inputs.find((slot) => slot.index === index);
    if (!input || !Number.isInteger(input.index)) {
      throw validationFailed(
        "lineup slot indexes must be contiguous",
        "slots",
        {
          code: "LINEUP_INVALID",
        },
      );
    }
    let keeper: Keeper | null = null;
    if (input.keeperId !== null) {
      if (typeof input.keeperId !== "string" || input.keeperId === "") {
        throw validationFailed(
          "keeperId must be a non-empty string or null",
          "slots",
          {
            code: "LINEUP_INVALID",
          },
        );
      }
      keeper = byId.get(input.keeperId) ?? null;
      if (!keeper) {
        throw notFound("keeper not found", { code: "KEEPER_NOT_FOUND" });
      }
      if (seen.has(input.keeperId)) {
        throw validationFailed("a Keeper may only occupy one slot", "slots", {
          code: "LINEUP_INVALID",
        });
      }
      seen.add(input.keeperId);
    }
    slots.push({ index, keeper });
  }
  if (lock && seen.size === 0) {
    throw validationFailed(
      "at least one Keeper is required to lock the fleet",
      "slots",
      {
        code: "LINEUP_INVALID",
      },
    );
  }
  const hasEmptySlot = slots.some((slot) => slot.keeper === null);
  return {
    ...current,
    lockedAt: lock ? now.toISOString() : current.lockedAt,
    slots,
    warnings: hasEmptySlot
      ? [{
        code: "EMPTY_SLOT",
        severity: "WARNING",
        message: "One or more slots are empty.",
      }]
      : [],
  };
}

function asProjection(value: unknown): Projection {
  if (!value || typeof value !== "object") {
    throw internal("invalid daily projection");
  }
  const projection = value as Partial<Projection>;
  if (!projection.lineup || !Array.isArray(projection.inventory)) {
    throw internal("invalid daily fleet projection");
  }
  return projection as Projection;
}
