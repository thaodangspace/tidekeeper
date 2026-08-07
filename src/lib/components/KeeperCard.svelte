<script lang="ts">
  import type { Keeper } from '$lib/data';

  let {
    keeper,
    compact = false,
    selected = false,
    addable = false,
    onclick
  }: {
    keeper: Keeper;
    compact?: boolean;
    selected?: boolean;
    addable?: boolean;
    onclick?: () => void;
  } = $props();

  const sigils: Record<string, string> = {
    crest: '♜', harbor: '⚓', echo: '◉', current: '≋', ember: '♨', bloom: '✦'
  };
</script>

<button
  class:compact
  class:selected
  class="keeper-card {keeper.affinity}"
  type="button"
  aria-pressed={selected}
  aria-label={`Xem ${keeper.name}`}
  {onclick}
>
  {#if keeper.image}<img src={keeper.image} alt="" />{/if}
  <span class="shade"></span>
  <span class="sigil" aria-hidden="true">{sigils[keeper.affinity] ?? '✦'}</span>
  {#if addable}<span class="add" aria-hidden="true">+</span>{/if}
  <span class="copy">
    <strong>{keeper.name}</strong>
    <em>{keeper.title}</em>
    {#if !compact}<small>{keeper.passive}</small>{/if}
  </span>
  <span class="gem" aria-hidden="true">◆</span>
</button>

<style>
  .keeper-card {
    --accent: var(--aqua);
    position: relative;
    min-width: 0;
    min-height: 286px;
    border: 1px solid color-mix(in srgb, var(--accent) 42%, var(--gold));
    border-radius: 2px;
    padding: 0;
    overflow: hidden;
    color: var(--ivory);
    background: #071721;
    box-shadow: inset 0 0 0 3px rgba(2, 14, 21, .72), 0 8px 20px rgba(0,0,0,.28);
    isolation: isolate;
    cursor: pointer;
    font-family: var(--body-font);
    text-align: center;
    clip-path: polygon(8px 0, calc(100% - 8px) 0, 100% 8px, 100% calc(100% - 8px), calc(100% - 8px) 100%, 8px 100%, 0 calc(100% - 8px), 0 8px);
    transition: transform .18s ease, filter .18s ease, box-shadow .18s ease;
  }
  .keeper-card:hover, .keeper-card:focus-visible { transform: translateY(-3px); filter: brightness(1.12); outline: none; box-shadow: 0 0 0 1px var(--accent), 0 10px 24px rgba(0,0,0,.45); }
  .keeper-card.selected { box-shadow: 0 0 0 2px var(--aqua), 0 0 22px rgba(54, 220, 211, .28); }
  img { position: absolute; inset: 0; width: 100%; height: 70%; object-fit: cover; object-position: 50% 18%; z-index: -2; filter: saturate(.83) contrast(1.05); }
  .shade { position: absolute; inset: 0; z-index: -1; background: linear-gradient(180deg, transparent 28%, rgba(3,14,20,.35) 48%, #06141c 69%); }
  .sigil, .add { position: absolute; top: 8px; display: grid; place-items: center; width: 35px; height: 35px; border: 1px solid color-mix(in srgb, var(--accent) 65%, var(--gold)); border-radius: 50%; background: rgba(2, 17, 24, .9); color: var(--accent); font: 600 18px var(--display-font); }
  .sigil { left: 8px; }
  .add { right: 8px; border-radius: 0; border: 0; background: transparent; color: var(--gold); font-size: 34px; text-shadow: 0 2px #001017; }
  .copy { position: absolute; left: 7px; right: 7px; bottom: 20px; display: grid; gap: 3px; }
  strong { font-size: clamp(15px, 1.05vw, 19px); font-weight: 600; white-space: nowrap; }
  em { color: var(--accent); font-size: 15px; font-style: normal; white-space: nowrap; }
  small { color: var(--sand); font-size: 14px; line-height: 1.22; min-height: 34px; }
  .gem { position: absolute; bottom: 5px; left: 50%; translate: -50%; color: var(--accent); font-size: 10px; }
  .ember { --accent: var(--coral); }
  .bloom { --accent: var(--green); }
  .echo { --accent: #a6bfe9; }
  .crest { --accent: #8fc8c8; }
  .compact { min-height: 168px; }
  .compact img { height: 76%; }
  .compact .shade { background: linear-gradient(180deg, transparent 32%, rgba(2,13,19,.38) 49%, #06141c 73%); }
  .compact .sigil { width: 28px; height: 28px; font-size: 14px; }
  .compact .copy { bottom: 13px; gap: 0; }
  .compact strong { font-size: 14px; }
  .compact em { font-size: 13px; }
  .compact .gem { bottom: 2px; }

  @media (max-width: 700px) {
    .keeper-card { min-height: 238px; }
    .keeper-card small { display: none; }
    .compact { min-height: 154px; }
  }
</style>
