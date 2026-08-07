<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api';

  type Mode = 'sign-in' | 'create-account';
  type FormErrors = Partial<Record<'email' | 'password' | 'confirmPassword', string>>;

  let mode = $state<Mode>('sign-in');
  let email = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let errors = $state<FormErrors>({});
  let isSubmitting = $state(false);
  let status = $state('');

  onMount(async () => {
    try {
      if ((await apiFetch('/api/v1/me', { cache: 'no-store' })).ok) {
        await goto('/play/prepare');
      }
    } catch {
      // The form remains available so the player can retry once connectivity returns.
    }
  });

  function switchMode(nextMode: Mode) {
    mode = nextMode;
    errors = {};
    status = '';
    password = '';
    confirmPassword = '';
  }

  function validate(): boolean {
    const nextErrors: FormErrors = {};
    const normalizedEmail = email.trim();

    if (!normalizedEmail) {
      nextErrors.email = 'Enter your email address.';
    } else if (!/^\S+@\S+\.\S+$/.test(normalizedEmail)) {
      nextErrors.email = 'Enter a valid email address.';
    }

    if (!password) {
      nextErrors.password = 'Enter your password.';
    } else if (mode === 'create-account' && password.length < 8) {
      nextErrors.password = 'Use at least 8 characters.';
    }

    if (mode === 'create-account') {
      if (!confirmPassword) {
        nextErrors.confirmPassword = 'Confirm your password.';
      } else if (confirmPassword !== password) {
        nextErrors.confirmPassword = 'Passwords do not match.';
      }
    }

    errors = nextErrors;
    return Object.keys(nextErrors).length === 0;
  }

  function authErrorMessage(response: Response): string {
    switch (response.status) {
      case 400:
        return 'Enter a valid email and password.';
      case 401:
        return 'Invalid email or password.';
      case 409:
        return 'An account already exists for this email address.';
      default:
        return 'We could not reach your account. Please try again shortly.';
    }
  }

  async function submitForm() {
    status = '';

    if (!validate()) return;

    isSubmitting = true;
    try {
      const response = await apiFetch(mode === 'create-account' ? '/auth/register' : '/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email.trim(), password })
      });

      if (!response.ok) {
        status = authErrorMessage(response);
        return;
      }

      await goto('/play/prepare');
    } catch {
      status = 'We could not reach your account. Check your connection and try again.';
    } finally {
      isSubmitting = false;
    }
  }
</script>

<svelte:head>
  <title>Enter Tidekeepers</title>
  <meta name="description" content="Sign in or create a Tidekeepers account." />
</svelte:head>

<main class="auth-shell">
  <div class="stars" aria-hidden="true"></div>

  <a class="wordmark" href="/" aria-label="Tidekeepers home">
    <span class="compass" aria-hidden="true">✦</span>
    <span>
      <strong>Tidekeepers</strong>
      <small>Seven days beyond the tide</small>
    </span>
  </a>

  <section class="auth-card" aria-labelledby="auth-title">
    <div class="crest" aria-hidden="true">⚓</div>
    <p class="eyebrow">The sea remembers</p>
    <h1 id="auth-title">{mode === 'sign-in' ? 'Return to your voyage' : 'Claim your compass'}</h1>
    <p class="promise">
      {mode === 'sign-in'
        ? 'Your fleet is waiting beyond the tide.'
        : 'Build today. Face tomorrow.'}
    </p>

    <div class="mode-toggle" aria-label="Authentication option">
      <button
        type="button"
        class:active={mode === 'sign-in'}
        aria-pressed={mode === 'sign-in'}
        onclick={() => switchMode('sign-in')}
      >Sign in</button>
      <button
        type="button"
        class:active={mode === 'create-account'}
        aria-pressed={mode === 'create-account'}
        onclick={() => switchMode('create-account')}
      >Create account</button>
    </div>

    <form onsubmit={(event) => { event.preventDefault(); submitForm(); }} novalidate>
      <div class="field">
        <label for="email">Email address</label>
        <input
          id="email"
          type="email"
          autocomplete="email"
          bind:value={email}
          aria-invalid={errors.email ? 'true' : undefined}
          aria-describedby={errors.email ? 'email-error' : undefined}
          oninput={() => { errors.email = undefined; status = ''; }}
        />
        {#if errors.email}<p class="field-error" id="email-error">{errors.email}</p>{/if}
      </div>

      <div class="field">
        <div class="label-row">
          <label for="password">Password</label>
          {#if mode === 'sign-in'}<a href="/help">Forgot password?</a>{/if}
        </div>
        <input
          id="password"
          type="password"
          autocomplete={mode === 'sign-in' ? 'current-password' : 'new-password'}
          bind:value={password}
          aria-invalid={errors.password ? 'true' : undefined}
          aria-describedby={errors.password ? 'password-error' : undefined}
          oninput={() => { errors.password = undefined; status = ''; }}
        />
        {#if errors.password}<p class="field-error" id="password-error">{errors.password}</p>{/if}
      </div>

      {#if mode === 'create-account'}
        <div class="field">
          <label for="confirm-password">Confirm password</label>
          <input
            id="confirm-password"
            type="password"
            autocomplete="new-password"
            bind:value={confirmPassword}
            aria-invalid={errors.confirmPassword ? 'true' : undefined}
            aria-describedby={errors.confirmPassword ? 'confirm-password-error' : undefined}
            oninput={() => { errors.confirmPassword = undefined; status = ''; }}
          />
          {#if errors.confirmPassword}<p class="field-error" id="confirm-password-error">{errors.confirmPassword}</p>{/if}
        </div>
      {/if}

      {#if status}
        <p class="status" role="status">{status}</p>
      {/if}

      <button class="submit" type="submit" disabled={isSubmitting}>
        {isSubmitting ? 'Charting a course…' : mode === 'sign-in' ? 'Enter Tidekeepers' : 'Create account'}
      </button>
    </form>

    <p class="terms">
      By continuing, you agree to our <a href="/legal/terms">Terms</a>, <a href="/legal/privacy">Privacy Policy</a>,
      and <a href="/legal/gameplay-disclaimer">gameplay disclaimer</a>.
    </p>
  </section>

  <p class="footer-note">A daily strategy voyage for those who heed the signals.</p>
</main>

<style>
  .auth-shell {
    min-height: 100vh;
    position: relative;
    display: grid;
    place-items: center;
    overflow: hidden;
    padding: 42px 20px 58px;
    background:
      radial-gradient(ellipse at 50% 105%, rgba(12, 95, 96, .38), transparent 42%),
      radial-gradient(ellipse at 8% 20%, rgba(130, 89, 40, .17), transparent 25%),
      linear-gradient(160deg, #020b12 0%, #041822 50%, #020d14 100%);
  }

  .stars, .stars::after {
    position: absolute;
    inset: 0;
    pointer-events: none;
    content: '';
    background-image:
      radial-gradient(circle at 14% 28%, #d6c0a0 0 1px, transparent 1.5px),
      radial-gradient(circle at 78% 14%, #82d9d4 0 1px, transparent 1.5px),
      radial-gradient(circle at 86% 67%, #d6c0a0 0 1px, transparent 1.5px),
      radial-gradient(circle at 23% 84%, #82d9d4 0 1px, transparent 1.5px),
      radial-gradient(circle at 61% 42%, #d6c0a0 0 1px, transparent 1.5px);
    opacity: .52;
  }

  .stars::after { transform: rotate(27deg) scale(1.2); opacity: .42; }

  .wordmark {
    position: absolute;
    top: 28px;
    left: clamp(20px, 5vw, 72px);
    z-index: 1;
    display: flex;
    align-items: center;
    gap: 11px;
    color: var(--ivory);
    text-decoration: none;
  }

  .compass, .crest {
    display: grid;
    place-items: center;
    border: 1px solid rgba(64, 217, 209, .52);
    border-radius: 50%;
    color: var(--aqua);
    box-shadow: inset 0 0 16px rgba(64, 217, 209, .16), 0 0 20px rgba(64, 217, 209, .1);
  }

  .compass { width: 43px; height: 43px; font-size: 25px; }
  .wordmark strong { display: block; font: 600 23px/1 var(--display-font); letter-spacing: .06em; }
  .wordmark small { color: var(--aqua); font-size: 13px; letter-spacing: .04em; }

  .auth-card {
    z-index: 1;
    width: min(100%, 480px);
    padding: 42px 48px 31px;
    text-align: center;
    border: 1px solid var(--line-soft);
    background: linear-gradient(145deg, rgba(7, 32, 43, .97), rgba(3, 17, 26, .96));
    box-shadow: inset 0 0 50px rgba(0, 0, 0, .26), 0 24px 70px rgba(0, 0, 0, .52);
    clip-path: polygon(16px 0, calc(100% - 16px) 0, 100% 16px, 100% calc(100% - 16px), calc(100% - 16px) 100%, 16px 100%, 0 calc(100% - 16px), 0 16px);
  }

  .crest { width: 58px; height: 58px; margin: 0 auto 18px; color: var(--gold); border-color: var(--line); font-size: 28px; }
  .eyebrow { margin: 0; color: var(--aqua); font-size: 13px; letter-spacing: .14em; text-transform: uppercase; }
  h1 { margin: 8px 0; color: var(--ivory); font: 500 clamp(27px, 6vw, 34px)/1.15 var(--display-font); }
  .promise { margin: 0 auto 26px; max-width: 270px; color: var(--sand); font-size: 18px; line-height: 1.25; }

  .mode-toggle { display: grid; grid-template-columns: 1fr 1fr; margin-bottom: 25px; border-bottom: 1px solid rgba(174, 128, 67, .32); }
  .mode-toggle button { border: 0; border-bottom: 2px solid transparent; padding: 10px 6px; background: transparent; color: var(--muted); cursor: pointer; font-size: 17px; }
  .mode-toggle button.active { border-color: var(--aqua); color: var(--ivory); }

  form { display: grid; gap: 18px; text-align: left; }
  .field { display: grid; gap: 7px; }
  label { color: var(--sand); font-size: 17px; }
  .label-row { display: flex; justify-content: space-between; align-items: baseline; gap: 12px; }
  .label-row a { color: var(--aqua); font-size: 14px; }
  input { width: 100%; border: 1px solid rgba(174, 128, 67, .5); border-radius: 0; outline: 0; padding: 11px 13px; color: var(--ivory); background: rgba(1, 12, 18, .62); font: 18px var(--body-font); }
  input:focus { border-color: var(--aqua); box-shadow: 0 0 0 2px rgba(64, 217, 209, .18); }
  input[aria-invalid='true'] { border-color: var(--coral); }
  .field-error { margin: 0; color: #ff9a8f; font-size: 15px; }

  .status { margin: -3px 0 0; padding: 10px 12px; border: 1px solid rgba(183, 137, 78, .58); color: var(--sand); background: rgba(102, 74, 33, .16); font-size: 16px; line-height: 1.2; }
  .submit { min-height: 50px; border: 1px solid var(--gold); color: #daf8f2; background: linear-gradient(180deg, #0c6566, #084348); cursor: pointer; font: 500 17px var(--display-font); letter-spacing: .04em; box-shadow: inset 0 0 17px rgba(135, 231, 220, .16); }
  .submit:hover:not(:disabled) { filter: brightness(1.14); }
  .submit:disabled { cursor: wait; opacity: .7; }
  .terms { margin: 25px 0 0; color: var(--muted); font-size: 14px; line-height: 1.35; }
  .terms a { color: var(--sand); }
  a:focus-visible { outline: 2px solid var(--aqua); outline-offset: 3px; }
  .footer-note { position: absolute; z-index: 1; bottom: 20px; margin: 0; color: rgba(214, 192, 160, .75); font-size: 15px; letter-spacing: .03em; }

  @media (max-width: 580px) {
    .auth-shell { align-items: start; padding-top: 116px; }
    .wordmark { top: 22px; left: 50%; transform: translateX(-50%); white-space: nowrap; }
    .auth-card { padding: 34px 25px 27px; }
    .footer-note { display: none; }
  }

  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after { transition-duration: .01ms !important; animation-duration: .01ms !important; }
  }
</style>
