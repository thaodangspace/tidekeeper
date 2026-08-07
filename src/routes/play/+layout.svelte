<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { apiFetch } from '$lib/api';

  let { children } = $props();
  let checkingSession = $state(true);
  let unavailable = $state(false);

  onMount(async () => {
    try {
      const response = await apiFetch('/api/v1/me', { cache: 'no-store' });
      if (response.status === 401) {
        await goto('/login');
        return;
      }
      unavailable = !response.ok;
    } catch {
      unavailable = true;
    } finally {
      checkingSession = false;
    }
  });
</script>

{#if checkingSession}
  <main class="session-status" aria-busy="true">Restoring your voyage…</main>
{:else if unavailable}
  <main class="session-status" role="alert">
    Tidekeepers is unavailable. Please try again shortly.
  </main>
{:else}
  {@render children()}
{/if}

<style>
  .session-status {
    display: grid;
    min-height: 100vh;
    place-items: center;
    margin: 0;
    padding: 2rem;
    color: var(--ivory);
    background: #020d14;
    font-size: 1.2rem;
  }
</style>
