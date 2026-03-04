<script lang="ts">
  import { fly } from 'svelte/transition';
  import { PlayerService } from "../bindings/github.com/willfish/forte";

  type ToastItem = {
    id: number;
    message: string;
    type: string;
    expiry: number;
  };

  let toasts = $state<ToastItem[]>([]);
  let nextId = 0;
  let pollTimer: ReturnType<typeof setInterval> | null = null;

  function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(async () => {
      try {
        const items = await PlayerService.GetToasts();
        if (items && items.length > 0) {
          const now = Date.now();
          for (const item of items) {
            toasts.push({
              id: nextId++,
              message: item.message,
              type: item.type || 'info',
              expiry: now + 4000,
            });
          }
        }
      } catch {
        // ignore polling errors
      }
      // Remove expired toasts.
      const now = Date.now();
      toasts = toasts.filter(t => t.expiry > now);
    }, 500);
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  $effect(() => {
    startPolling();
    return () => stopPolling();
  });

  function dismiss(id: number) {
    toasts = toasts.filter(t => t.id !== id);
  }
</script>

{#if toasts.length > 0}
  <div class="toast-container">
    {#each toasts as toast (toast.id)}
      <div class="toast toast-{toast.type}" role="alert" transition:fly={{ x: 50, duration: 200 }}>
        <span class="toast-message">{toast.message}</span>
        <button class="toast-close" onclick={() => dismiss(toast.id)} aria-label="Dismiss">
          <svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
          </svg>
        </button>
      </div>
    {/each}
  </div>
{/if}

<style>
  .toast-container {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    max-width: 360px;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
    border-radius: 6px;
    background: var(--bg-elevated, var(--bg-hover));
    border-left: 3px solid var(--accent);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  }

  .toast-warn {
    border-left-color: var(--warning);
  }

  .toast-error {
    border-left-color: var(--error);
  }

  .toast-info {
    border-left-color: var(--accent);
  }

  .toast-message {
    flex: 1;
    font-size: 0.85rem;
    color: var(--text-primary);
  }

  .toast-close {
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0.15rem;
    border-radius: 3px;
    flex-shrink: 0;
  }

  .toast-close:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

</style>
