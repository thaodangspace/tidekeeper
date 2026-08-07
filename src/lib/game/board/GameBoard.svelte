<script lang="ts">
  import { onMount } from 'svelte';
  import type { BoardKeeper, BoardSynergy, BoardTideState } from '$lib/game/state/board-types';

  let {
    keepers,
    maxSlots,
    selectedKeeperId,
    locked,
    activeSynergies,
    tideState = 'CALM',
    reducedMotion,
    onKeeperSelect,
    onKeeperMove,
    onEmptySlotSelect
  }: {
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
  } = $props();

  let canvasHost: HTMLDivElement;
  let renderer: import('./BoardRenderer').BoardRenderer | null = null;
  let rendererError = $state('');
  let prefersReducedMotion = $state(false);

  const keeperAt = (slotIndex: number) => keepers.find((keeper) => keeper.slotIndex === slotIndex);
  const selectedKeeper = () => keepers.find((keeper) => keeper.id === selectedKeeperId);

  $effect(() => {
    renderer?.setState({
      keepers,
      maxSlots,
      selectedKeeperId,
      locked,
      activeSynergies,
      tideState,
      reducedMotion: reducedMotion ?? prefersReducedMotion
    });
  });

  onMount(() => {
    let disposed = false;
    const media = window.matchMedia('(prefers-reduced-motion: reduce)');
    prefersReducedMotion = media.matches;
    const onMotionChange = (event: MediaQueryListEvent) => {
      prefersReducedMotion = event.matches;
      renderer?.setReducedMotion(event.matches);
    };
    media.addEventListener('change', onMotionChange);

    const resizeObserver = new ResizeObserver(([entry]) => {
      const { width, height } = entry.contentRect;
      renderer?.resize(width, height);
    });
    resizeObserver.observe(canvasHost);

    void import('./BoardRenderer').then(async ({ BoardRenderer }) => {
      if (disposed) return;
      const nextRenderer = new BoardRenderer({
        onKeeperSelect: (keeperId) => onKeeperSelect?.(keeperId),
        onKeeperMove: (keeperId, targetSlot) => onKeeperMove?.(keeperId, targetSlot),
        onEmptySlotSelect: (slotIndex) => onEmptySlotSelect?.(slotIndex)
      });
      try {
        const canvas = await nextRenderer.init(canvasHost, reducedMotion ?? prefersReducedMotion);
        if (disposed) {
          nextRenderer.destroy();
          return;
        }
        renderer = nextRenderer;
        canvas.setAttribute('aria-hidden', 'true');
        renderer.setState({
          keepers,
          maxSlots,
          selectedKeeperId,
          locked,
          activeSynergies,
          tideState,
          reducedMotion: reducedMotion ?? prefersReducedMotion
        });
      } catch {
        nextRenderer.destroy();
        rendererError = 'Không thể khởi động chế độ xem đội hình động. Bạn vẫn có thể chỉnh sửa đội hình bên dưới.';
      }
    }).catch(() => {
      rendererError = 'Không thể tải chế độ xem đội hình động. Bạn vẫn có thể chỉnh sửa đội hình bên dưới.';
    });

    const onVisibilityChange = () => renderer?.setVisibility(document.hidden);
    document.addEventListener('visibilitychange', onVisibilityChange);

    return () => {
      disposed = true;
      media.removeEventListener('change', onMotionChange);
      resizeObserver.disconnect();
      document.removeEventListener('visibilitychange', onVisibilityChange);
      renderer?.destroy();
      renderer = null;
    };
  });
</script>

<div class="game-board-shell" data-tide={tideState}>
  <div class="pixi-board" bind:this={canvasHost} aria-hidden="true"></div>
  {#if rendererError}
    <p class="board-error" role="status">{rendererError}</p>
  {/if}
</div>

<!-- This is intentionally real DOM rather than a visually hidden canvas label.
     It is the keyboard/touch and assistive-technology representation of the
     same Fleet projection supplied to Pixi. -->
<div class="board-controls" aria-label="Điều khiển đội hình">
  <p class="board-instructions">
    {#if locked}
      Đội hình đã khóa. Bạn vẫn có thể xem Keeper.
    {:else if selectedKeeper()}
      Đã chọn {selectedKeeper()?.name}. Chọn ô đích để di chuyển, hoặc kéo Keeper trên bảng.
    {:else}
      Chọn Keeper rồi chọn ô đích, hoặc kéo Keeper trên bảng.
    {/if}
  </p>
  <div class="semantic-slots">
    {#each Array.from({ length: maxSlots }, (_, slotIndex) => slotIndex) as slotIndex}
      {@const keeper = keeperAt(slotIndex)}
      <div class:selected={keeper?.id === selectedKeeperId} class="semantic-slot">
        <span class="slot-label">Ô {slotIndex + 1}</span>
        {#if keeper}
          <button
            class="semantic-keeper"
            type="button"
            aria-pressed={keeper.id === selectedKeeperId}
            onclick={() => onKeeperSelect?.(keeper.id)}
          >
            <span>{keeper.name}</span>
            <small>{keeper.sector} · {keeper.role}</small>
          </button>
        {:else}
          <button
            class="semantic-keeper empty"
            type="button"
            disabled={locked || !selectedKeeperId}
            aria-label={selectedKeeperId ? `Đặt Keeper vào ô ${slotIndex + 1}` : `Ô ${slotIndex + 1} đang trống`}
            onclick={() => onEmptySlotSelect?.(slotIndex)}
          >
            Trống
          </button>
        {/if}
        {#if selectedKeeperId && !locked && keeper?.id !== selectedKeeperId}
          <button class="move-to-slot" type="button" onclick={() => onKeeperMove?.(selectedKeeperId!, slotIndex)}>
            Đặt vào ô này
          </button>
        {/if}
      </div>
    {/each}
  </div>
</div>

<style>
  .game-board-shell { position: relative; min-width: 0; }
  .pixi-board { width: 100%; min-height: 330px; aspect-ratio: 1.72; overflow: hidden; border: 1px solid rgba(64, 217, 209, .18); background: #041923; }
  :global(.pixi-board canvas) { display: block; width: 100%; height: 100%; touch-action: none; }
  .board-error { margin: 0; padding: 10px 12px; border: 1px solid rgba(242, 85, 67, .55); color: #f3a388; background: rgba(70, 24, 25, .35); font-size: 16px; }
  .board-controls { margin-top: 8px; padding: 8px 10px; border: 1px solid rgba(174, 128, 67, .31); background: rgba(2, 13, 20, .42); }
  .board-instructions { margin: 0 0 8px; color: var(--muted, #aa9e8c); font-size: 15px; }
  .semantic-slots { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 7px; }
  .semantic-slot { min-width: 0; display: grid; gap: 4px; align-content: start; padding: 6px; border: 1px solid rgba(174, 128, 67, .28); }
  .semantic-slot.selected { border-color: var(--aqua, #40d9d1); box-shadow: inset 0 0 12px rgba(64, 217, 209, .12); }
  .slot-label { color: var(--gold, #b7894e); font-size: 12px; }
  .semantic-keeper, .move-to-slot { min-width: 0; border: 1px solid rgba(64, 217, 209, .38); background: rgba(7, 31, 42, .78); color: var(--ivory, #ece4d1); padding: 5px; text-align: left; cursor: pointer; }
  .semantic-keeper span, .semantic-keeper small { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .semantic-keeper small { color: var(--sand, #d6c0a0); font-size: 12px; }
  .semantic-keeper.empty { color: var(--muted, #aa9e8c); border-color: rgba(174, 128, 67, .35); text-align: center; }
  .semantic-keeper:disabled { opacity: .55; cursor: not-allowed; }
  .move-to-slot { color: var(--aqua, #40d9d1); font-size: 12px; text-align: center; }
  @media (max-width: 800px) {
    .pixi-board { min-height: 275px; aspect-ratio: 1.45; }
    .semantic-slots { grid-template-columns: repeat(5, minmax(100px, 1fr)); overflow-x: auto; }
    .semantic-slot { min-width: 100px; }
    .board-controls { overflow: hidden; }
  }
  @media (prefers-reduced-motion: reduce) {
    .pixi-board { scroll-behavior: auto; }
  }
</style>
