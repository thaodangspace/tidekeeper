<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import KeeperCard from '$lib/components/KeeperCard.svelte';
  import { asKeeper, getMyKeepers, PlayerKeepersRequestError } from '$lib/keepers';
  import { fleet, initialOffers, type Keeper, type Offer } from '$lib/data';

  let supplies = 11;
  let secondsLeft = 6 * 3600 + 42 * 60 + 18;
  let selectedStrategy = 'shelter';
  let selectedKeeper = '';
  let lockOpen = false;
  let locked = false;
  let offers: Offer[] = [...initialOffers];
  let owned: Keeper[] = [];
  let inventoryLoading = true;
  let inventoryUnavailable = false;
  let notice = '';

  const strategies = [
    { id: 'shelter', rune: '≋', name: 'Phòng hộ', upside: 'Giảm Tăng trưởng 50%', downside: 'Giảm Áp lực 50%' },
    { id: 'flow', rune: '≈', name: 'Theo Dòng Chảy' },
    { id: 'counter', rune: '♒', name: 'Nghịch Dòng' }
  ];

  const pad = (value: number) => String(value).padStart(2, '0');
  const countdown = () => `${pad(Math.floor(secondsLeft / 3600))}:${pad(Math.floor(secondsLeft % 3600 / 60))}:${pad(secondsLeft % 60)}`;

  onMount(() => {
    const timer = window.setInterval(() => secondsLeft = Math.max(0, secondsLeft - 1), 1000);
    void loadOwnedKeepers();
    return () => window.clearInterval(timer);
  });

  async function loadOwnedKeepers() {
    try {
      owned = (await getMyKeepers()).map(asKeeper);
    } catch (error) {
      if (error instanceof PlayerKeepersRequestError && error.status === 401) {
        await goto('/login');
        return;
      }
      inventoryUnavailable = true;
    } finally {
      inventoryLoading = false;
    }
  }

  function recruit(_offer: Offer) {
    notice = 'Tuyển mộ chưa khả dụng.';
    window.setTimeout(() => notice = '', 2600);
  }

  function confirmLock() {
    locked = true;
    lockOpen = false;
    notice = 'Đội hình đã được khóa cho Ngày 3.';
  }
</script>

<svelte:head>
  <title>Tidekeepers — Chuẩn bị hôm nay</title>
</svelte:head>

<div class="game-shell">
  <header class="status-header">
    <div class="brand">
      <div class="compass" aria-hidden="true"><span>✦</span></div>
      <div>
        <h1>TIDEKEEPERS</h1>
      </div>
    </div>

    <div class="status voyage-status"><span class="status-icon">☼</span><span>Hành trình 01 <b>· Ngày 3/7</b></span></div>
    <div class="status hull-status">
      <span class="status-icon">♢</span>
      <div><span>Thân tàu <b class="sand">82/100</b></span><div class="hull-track"><i></i></div></div>
    </div>
    <div class="status supply-status"><span class="status-icon">▧</span><span>Tiếp tế</span><strong>{supplies}</strong></div>
    <div class="status deadline-status"><span class="status-icon">⌛</span><span>Khóa sau <b>{countdown()}</b></span></div>
    <button class="prepare-button" class:locked type="button" onclick={() => !locked && (lockOpen = true)}>
      {locked ? 'ĐÃ KHÓA' : 'CHUẨN BỊ'}
    </button>
  </header>

  <main>
    <section class="workspace">
      <aside class="panel tide-panel ornamental">
        <h2><span>THỦY TRIỀU HÔM NAY</span></h2>
        <article class="rule-block">
          <div class="round-icon shield">♜</div>
          <div><span class="eyebrow">ĐIỀU KIỆN</span><h3>Trở về Cảng</h3><p>Warden và Tiên phong nhận thêm phòng thủ.<br />Trickster nhận thêm Áp lực.</p></div>
        </article>
        <article class="rule-block">
          <div class="round-icon wave">≋</div>
          <div><span class="eyebrow">MỤC TIÊU</span><h3>Bảo toàn</h3><p>Giữ Độ Sâu của đội dưới ngưỡng giới hạn.</p></div>
        </article>
        <div class="signals">
          <span class="eyebrow">TÍN HIỆU</span>
          <p><i class="down">↓</i> Động lực toàn cục đang suy yếu</p>
          <p><i class="fire">♨</i> Nhiễu động Ember đang tăng</p>
          <p><i class="anchor">⚓</i> Harbor ổn định hơn<br /><span>Thủy triều Thế giới</span></p>
        </div>
      </aside>

      <section class="panel fleet-panel ornamental">
        <h2><span>ĐỘI HÌNH CỦA BẠN</span></h2>
        <div class="fleet-grid">
          {#each fleet as keeper}
            <KeeperCard {keeper} selected={selectedKeeper === keeper.id} onclick={() => selectedKeeper = keeper.id} />
          {/each}
          <button class="empty-slot" type="button" aria-label="Thêm Keeper vào ô trống">
            <span>＋</span><small>Trống</small>
          </button>
        </div>
        <div class="synergy-bar">
          <div class="synergy-group"><span class="eyebrow">SYNERGY ĐANG KÍCH HOẠT</span><div><button>♜ &nbsp; Crest II</button><button>♟ &nbsp; Đội hình đa dạng</button></div></div>
          <div class="synergy-group pending"><span class="eyebrow">SẮP KÍCH HOẠT</span><button>≋ &nbsp; Balanced Current 3/4</button></div>
          <span class="saved">✓ &nbsp; Đã lưu</span>
        </div>
      </section>

      <aside class="panel plan-panel ornamental">
        <h2><span>KẾ HOẠCH HÔM NAY</span></h2>
        <div class="strategy-list" role="radiogroup" aria-label="Chọn kế hoạch">
          {#each strategies as strategy}
            <button
              class:active={selectedStrategy === strategy.id}
              type="button"
              role="radio"
              aria-checked={selectedStrategy === strategy.id}
              onclick={() => selectedStrategy = strategy.id}
            >
              <i>{strategy.rune}</i>
              <span><strong>{strategy.name}</strong>{#if strategy.upside}<small class="up">↓ &nbsp;{strategy.upside}</small><small class="downside">↓ &nbsp;{strategy.downside}</small>{/if}</span>
              <b>{selectedStrategy === strategy.id ? '✓' : ''}</b>
            </button>
          {/each}
        </div>
        <div class="warnings">
          <h3>⚠ &nbsp; CẢNH BÁO</h3>
          <p>⌑ &nbsp; Còn 1 ô đội hình đang trống</p>
          <p>⚓ &nbsp; Khả năng tạo Tăng trưởng thấp</p>
        </div>
        <button class="lock-button" class:locked type="button" onclick={() => !locked && (lockOpen = true)}>{locked ? '✓ ĐÃ KHÓA ĐỘI HÌNH' : 'KHÓA ĐỘI HÌNH'}</button>
      </aside>
    </section>

    <section class="lower-deck panel ornamental">
      <div class="collection-section">
        <h2 class="section-title">♙ &nbsp; BỘ SƯU TẬP</h2>
        <div class="collection-grid" aria-busy={inventoryLoading}>
          {#if inventoryLoading}
            <p class="empty-shop">Đang tải Bộ sưu tập…</p>
          {:else if inventoryUnavailable}
            <p class="empty-shop" role="alert">Không thể tải Bộ sưu tập. Vui lòng thử lại sau.</p>
          {:else if owned.length === 0}
            <p class="empty-shop">Bạn chưa sở hữu Keeper nào.</p>
          {:else}
            {#each owned as keeper}
              <KeeperCard {keeper} compact addable selected={selectedKeeper === keeper.id} onclick={() => selectedKeeper = keeper.id} />
            {/each}
          {/if}
        </div>
      </div>
      <div class="divider-anchor" aria-hidden="true"><span>⚓</span></div>
      <div class="shop-section">
        <div class="shop-heading"><h2 class="section-title">▧ &nbsp; CỬA HÀNG</h2><span>⟳ &nbsp; Làm mới sau <b>18:42:18</b></span></div>
        <div class="offer-grid">
          {#each offers as offer}
            <article class="offer-card {offer.affinity}">
              <img src={offer.image} alt="" />
              <div class="offer-copy">
                <h3>{offer.name}</h3><strong>{offer.title}</strong><p>{offer.passive}</p>
                <small>GIÁ <b>▧ &nbsp;{offer.cost}</b></small>
              </div>
              <button type="button" disabled={supplies < offer.cost} onclick={() => recruit(offer)}>CHIÊU MỘ</button>
            </article>
          {:else}
            <p class="empty-shop">Mọi Keeper đã được chiêu mộ.</p>
          {/each}
        </div>
      </div>
    </section>

    <nav class="voyage-track" aria-label="Tiến trình hành trình">
      <span class="ship" aria-hidden="true">⛵</span>
      {#each [1,2,3,4,5,6,7] as day}
        <div class:done={day < 3} class:current={day === 3} class:boss={day === 7}>
          <i>{day < 3 ? '✓' : day === 3 ? '✦' : day === 7 ? '☼' : ''}</i><span>Ngày {day}</span>
        </div>
      {/each}
    </nav>
  </main>

  {#if notice}<div class="toast" role="status">✓ &nbsp; {notice}</div>{/if}

  {#if lockOpen}
    <div class="modal-backdrop" role="presentation" onclick={(event) => event.currentTarget === event.target && (lockOpen = false)}>
      <div class="confirm-modal ornamental" role="dialog" aria-modal="true" aria-labelledby="confirm-title">
        <button class="close" aria-label="Đóng" onclick={() => lockOpen = false}>×</button>
        <div class="modal-rune">⚓</div>
        <h2 id="confirm-title">Khóa đội hình?</h2>
        <p>Đội hình Ngày 3 sẽ đối mặt với Thủy triều ngày mai. Bạn sẽ không thể thay đổi Keeper hoặc kế hoạch sau khi khóa.</p>
        <div class="lock-summary"><span><b>4/5</b> Keeper</span><span><b>2</b> Synergy</span><span><b>{strategies.find((s) => s.id === selectedStrategy)?.name}</b> Kế hoạch</span></div>
        <div class="modal-warning">⚠ Còn 1 ô đội hình đang trống</div>
        <div class="modal-actions"><button onclick={() => lockOpen = false}>QUAY LẠI</button><button class="confirm" onclick={confirmLock}>KHÓA ĐỘI HÌNH</button></div>
      </div>
    </div>
  {/if}
</div>
