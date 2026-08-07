import { apiFetch } from '$lib/api';
import type { Affinity, Keeper } from '$lib/data';

export interface PlayerKeeper {
  id: string;
  definitionKey: string;
  name: string;
  currentName: string;
  sector: string;
  role: string;
  rarity: string;
  level: number;
}

interface PlayerKeepersResponse {
  keepers: PlayerKeeper[];
}

const imagesByDefinitionKey: Record<string, string> = {
  bloom_keeper: '/art/bloom-keeper.webp',
  crest_guardian: '/art/crest-guardian.webp',
  current_weaver: '/art/current-weaver.webp',
  echo_seer: '/art/echo-seer.webp',
  ember_channeler: '/art/ember-channeler.webp',
  ember_trickster: '/art/ember-trickster.webp',
  harbor_scout: '/art/harbor-scout.webp',
  harbor_warden: '/art/harbor-warden.webp',
  tide_runner: '/art/tide-runner.webp'
};

const affinitiesBySector: Record<string, Affinity> = {
  BLOOM: 'bloom',
  CREST: 'crest',
  CURRENT: 'current',
  ECHO: 'echo',
  EMBER: 'ember',
  HARBOR: 'harbor'
};

export class PlayerKeepersRequestError extends Error {
  constructor(public readonly status: number) {
    super(`Unable to load player Keeper inventory (${status}).`);
  }
}

export async function getMyKeepers(): Promise<PlayerKeeper[]> {
  const response = await apiFetch('/api/v1/me/keepers', { cache: 'no-store' });
  if (!response.ok) throw new PlayerKeepersRequestError(response.status);

  const payload: PlayerKeepersResponse = await response.json();
  return payload.keepers;
}

export function asKeeper(keeper: PlayerKeeper): Keeper {
  const sector = keeper.sector.toUpperCase();
  const role = keeper.role.replaceAll('_', ' ');

  return {
    id: keeper.id,
    name: keeper.currentName,
    title: `${sector} · ${role} · Lv. ${keeper.level}`,
    role,
    passive: `${keeper.rarity} · Lv. ${keeper.level}`,
    affinity: affinitiesBySector[sector] ?? 'unknown',
    image: imagesByDefinitionKey[keeper.definitionKey]
  };
}
