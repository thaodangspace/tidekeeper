<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api';

  let name = $state('');
  let error = $state('');
  let status = $state('');
  let isSubmitting = $state(false);

  onMount(async () => {
    try {
      if ((await apiFetch('/api/v1/me', { cache: 'no-store' })).ok) await goto('/play/prepare');
    } catch {
      // The entry form remains available while the API is unavailable.
    }
  });

  function validate(): boolean {
    const trimmed = name.trim();
    if (Array.from(trimmed).length < 2 || Array.from(trimmed).length > 32) {
      error = 'Enter a name between 2 and 32 characters.';
      return false;
    }
    error = '';
    return true;
  }

  async function submitForm() {
    status = '';
    if (!validate()) return;
    isSubmitting = true;
    try {
      const response = await apiFetch('/auth/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: name.trim() })
      });
      if (!response.ok) {
        status = response.status === 400
          ? 'Enter a name between 2 and 32 characters.'
          : 'We could not enter Tidekeepers. Please try again shortly.';
        return;
      }
      const me = await apiFetch('/api/v1/me', { cache: 'no-store' });
      if (me.ok) await goto('/play/prepare');
      else status = 'We could not load your voyage. Please try again shortly.';
    } catch {
      status = 'We could not reach Tidekeepers. Check your connection and try again.';
    } finally {
      isSubmitting = false;
    }
  }
</script>

<svelte:head>
  <title>Enter Tidekeepers</title>
  <meta name="description" content="Enter your name to begin or resume a Tidekeepers voyage." />
</svelte:head>

<main class="auth-shell">
  <div class="stars" aria-hidden="true"></div>
  <a class="wordmark" href="/" aria-label="Tidekeepers home">
    <span class="compass" aria-hidden="true">✦</span>
    <span><strong>Tidekeepers</strong><small>Seven days beyond the tide</small></span>
  </a>

  <section class="auth-card" aria-labelledby="auth-title">
    <div class="crest" aria-hidden="true">⚓</div>
    <p class="eyebrow">The sea remembers</p>
    <h1 id="auth-title">Enter your name</h1>
    <p class="promise">Your fleet is waiting beyond the tide.</p>

    <form onsubmit={(event) => { event.preventDefault(); submitForm(); }} novalidate>
      <div class="field">
        <label for="name">Your name</label>
        <input
          id="name"
          type="text"
          autocomplete="nickname"
          maxlength="32"
          bind:value={name}
          aria-invalid={error ? 'true' : undefined}
          aria-describedby={error ? 'name-error' : undefined}
          oninput={() => { error = ''; status = ''; }}
        />
        {#if error}<p class="field-error" id="name-error">{error}</p>{/if}
      </div>
      {#if status}<p class="status" role="status">{status}</p>{/if}
      <button class="submit" type="submit" disabled={isSubmitting}>
        {isSubmitting ? 'Charting a course…' : 'Enter Tidekeepers'}
      </button>
    </form>

    <p class="terms">By continuing, you agree to our <a href="/legal/terms">Terms</a>, <a href="/legal/privacy">Privacy Policy</a>, and <a href="/legal/gameplay-disclaimer">gameplay disclaimer</a>.</p>
  </section>
  <p class="footer-note">A daily strategy voyage for those who heed the signals.</p>
</main>

<style>
  .auth-shell { min-height: 100vh; position: relative; display: grid; place-items: center; overflow: hidden; padding: 42px 20px 58px; background: radial-gradient(ellipse at 50% 105%, rgba(12,95,96,.38), transparent 42%), linear-gradient(160deg, #020b12, #041822 50%, #020d14); }
  .stars { position: absolute; inset: 0; pointer-events: none; background-image: radial-gradient(circle at 14% 28%, #d6c0a0 0 1px, transparent 1.5px), radial-gradient(circle at 78% 14%, #82d9d4 0 1px, transparent 1.5px), radial-gradient(circle at 86% 67%, #d6c0a0 0 1px, transparent 1.5px); opacity: .55; }
  .wordmark { position: absolute; top: 28px; left: clamp(20px, 5vw, 72px); z-index: 1; display: flex; align-items: center; gap: 11px; color: var(--ivory); text-decoration: none; }
  .compass, .crest { display: grid; place-items: center; border: 1px solid rgba(64,217,209,.52); border-radius: 50%; color: var(--aqua); }
  .compass { width: 43px; height: 43px; font-size: 25px; } .wordmark strong { display: block; font: 600 23px/1 var(--display-font); letter-spacing: .06em; } .wordmark small { color: var(--aqua); font-size: 13px; }
  .auth-card { z-index: 1; width: min(100%, 480px); padding: 42px 48px 31px; text-align: center; border: 1px solid var(--line-soft); background: linear-gradient(145deg, rgba(7,32,43,.97), rgba(3,17,26,.96)); box-shadow: 0 24px 70px rgba(0,0,0,.52); clip-path: polygon(16px 0, calc(100% - 16px) 0, 100% 16px, 100% calc(100% - 16px), calc(100% - 16px) 100%, 16px 100%, 0 calc(100% - 16px), 0 16px); }
  .crest { width: 58px; height: 58px; margin: 0 auto 18px; color: var(--gold); font-size: 28px; } .eyebrow { margin: 0; color: var(--aqua); font-size: 13px; letter-spacing: .14em; text-transform: uppercase; } h1 { margin: 8px 0; color: var(--ivory); font: 500 clamp(27px, 6vw, 34px)/1.15 var(--display-font); } .promise { margin: 0 auto 26px; color: var(--sand); font-size: 18px; }
  form { display: grid; gap: 18px; text-align: left; } .field { display: grid; gap: 7px; } label { color: var(--sand); font-size: 17px; } input { width: 100%; border: 1px solid rgba(174,128,67,.5); padding: 11px 13px; color: var(--ivory); background: rgba(1,12,18,.62); font: 18px var(--body-font); } input:focus { border-color: var(--aqua); outline: 0; box-shadow: 0 0 0 2px rgba(64,217,209,.18); } input[aria-invalid='true'] { border-color: var(--coral); } .field-error { margin: 0; color: #ff9a8f; font-size: 15px; }
  .status { margin: -3px 0 0; padding: 10px 12px; border: 1px solid rgba(183,137,78,.58); color: var(--sand); background: rgba(102,74,33,.16); font-size: 16px; } .submit { min-height: 50px; border: 1px solid var(--gold); color: #daf8f2; background: linear-gradient(180deg, #0c6566, #084348); cursor: pointer; font: 500 17px var(--display-font); } .submit:disabled { cursor: wait; opacity: .7; } .terms { margin: 25px 0 0; color: var(--muted); font-size: 14px; line-height: 1.35; } .terms a { color: var(--sand); } .footer-note { position: absolute; bottom: 20px; margin: 0; color: rgba(214,192,160,.75); font-size: 15px; }
  @media (max-width: 580px) { .auth-shell { align-items: start; padding-top: 116px; } .wordmark { top: 22px; left: 50%; transform: translateX(-50%); white-space: nowrap; } .auth-card { padding: 34px 25px 27px; } .footer-note { display: none; } }
</style>
