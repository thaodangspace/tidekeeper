/** The small, render-only projection of a Keeper used by the game board. */
export interface BoardKeeper {
  id: string;
  name: string;
  image?: string;
  sector: string;
  role: string;
  slotIndex: number;
}

/** Synergy membership is supplied by Svelte; the renderer does not infer it. */
export interface BoardSynergy {
  id: string;
  keeperIds: string[];
  label?: string;
  type?: string;
}

export type BoardTideState = 'CALM' | 'GROWTH' | 'DECLINE' | 'ROTATION' | 'RECOVERY' | 'SHOCK';

export interface GameBoardProps {
  keepers: BoardKeeper[];
  maxSlots: number;
  selectedKeeperId: string | null;
  locked: boolean;
  activeSynergies: BoardSynergy[];
  tideState?: BoardTideState;
  reducedMotion?: boolean;
  onKeeperSelect?: (keeperId: string) => void;
  onKeeperMove?: (keeperId: string, targetSlot: number) => void;
  onEmptySlotSelect?: (slotIndex: number) => void;
}

export interface BoardRendererCallbacks {
  onKeeperSelect: (keeperId: string) => void;
  onKeeperMove: (keeperId: string, targetSlot: number) => void;
  onEmptySlotSelect: (slotIndex: number) => void;
}
