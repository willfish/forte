<script lang="ts">
  import { fly, fade } from 'svelte/transition';
  import { PlayerService } from "../bindings/github.com/willfish/forte";

  type QueueTrack = {
    trackId: number;
    title: string;
    artist: string;
    album: string;
    durationMs: number;
    filePath: string;
  };

  const { open, onclose }: { open: boolean; onclose: () => void } = $props();

  let tracks = $state<QueueTrack[]>([]);
  let currentPosition = $state(-1);
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let listEl: HTMLDivElement | undefined = $state();

  async function refresh() {
    tracks = ((await PlayerService.GetQueue()) || []).map((t: any) => ({
      trackId: t.trackId,
      title: t.title,
      artist: t.artist,
      album: t.album,
      durationMs: t.durationMs,
      filePath: t.filePath,
    }));
    currentPosition = await PlayerService.GetQueuePosition();
  }

  $effect(() => {
    if (open) {
      refresh();
      pollTimer = setInterval(refresh, 1000);
      // Scroll to current track after initial load.
      setTimeout(() => {
        if (listEl && currentPosition >= 0) {
          const row = listEl.children[currentPosition] as HTMLElement;
          if (row) row.scrollIntoView({ block: 'center', behavior: 'smooth' });
        }
      }, 100);
    } else {
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    }
    return () => {
      if (pollTimer) {
        clearInterval(pollTimer);
        pollTimer = null;
      }
    };
  });

  async function removeTrack(index: number) {
    await PlayerService.QueueRemove(index);
    await refresh();
  }

  async function clearQueue() {
    await PlayerService.QueueClear();
    await refresh();
  }

  // Drag and drop reorder.
  let dragIndex = $state<number | null>(null);

  function handleDragStart(index: number, e: DragEvent) {
    dragIndex = index;
    if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move';
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
  }

  async function handleDrop(targetIndex: number) {
    if (dragIndex === null || dragIndex === targetIndex) return;
    await PlayerService.QueueMove(dragIndex, targetIndex);
    dragIndex = null;
    await refresh();
  }

  function formatDuration(ms: number): string {
    const totalSeconds = Math.floor(ms / 1000);
    const m = Math.floor(totalSeconds / 60);
    const s = totalSeconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div class="overlay" onclick={onclose} transition:fade={{ duration: 150 }}></div>
  <aside class="panel" aria-label="Play queue" transition:fly={{ x: 350, duration: 200 }}>
    <div class="panel-header">
      <h3>Queue</h3>
      <div class="header-actions">
        <button class="clear-btn" onclick={clearQueue} disabled={tracks.length === 0}>Clear</button>
        <button class="close-btn" onclick={onclose} aria-label="Close">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
            <path d="M19 6.41 17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
          </svg>
        </button>
      </div>
    </div>

    {#if tracks.length === 0}
      <p class="empty-msg">Queue is empty</p>
    {:else}
      <div class="track-list" bind:this={listEl}>
        {#each tracks as track, i (track.trackId + '-' + i)}
          <div
            class="track-row"
            class:current={i === currentPosition}
            draggable="true"
            ondragstart={(e) => handleDragStart(i, e)}
            ondragover={handleDragOver}
            ondrop={() => handleDrop(i)}
            role="listitem"
          >
            <span class="drag-handle">
              <svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor" opacity="0.3">
                <path d="M3 15h18v-2H3v2zm0 4h18v-2H3v2zm0-8h18V9H3v2zm0-6v2h18V5H3z"/>
              </svg>
            </span>
            <div class="track-info">
              <span class="track-title">{track.title}</span>
              <span class="track-meta">{track.artist}{track.album ? ` - ${track.album}` : ''}</span>
            </div>
            <span class="track-duration">{formatDuration(track.durationMs)}</span>
            <button class="remove-btn" onclick={() => removeTrack(i)} aria-label="Remove">
              <svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
                <path d="M19 6.41 17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </aside>
{/if}

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.3);
    z-index: 100;
  }

  .panel {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 72px; /* above the now-playing bar */
    width: 350px;
    background: var(--bg-sidebar);
    border-left: 1px solid var(--border);
    z-index: 101;
    display: flex;
    flex-direction: column;
  }

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 1rem;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }

  .panel-header h3 {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .clear-btn {
    padding: 0.3rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.8rem;
  }

  .clear-btn:hover:not(:disabled) {
    color: var(--text-primary);
    border-color: var(--text-secondary);
  }

  .clear-btn:disabled {
    opacity: 0.3;
    cursor: default;
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 4px;
    display: flex;
    border-radius: 4px;
  }

  .close-btn:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .empty-msg {
    padding: 2rem 1rem;
    text-align: center;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .track-list {
    flex: 1;
    overflow-y: auto;
    padding: 0.5rem;
  }

  .track-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.5rem;
    border-radius: 4px;
    cursor: default;
  }

  .track-row:hover {
    background: var(--bg-hover);
  }

  .track-row.current {
    background: var(--bg-active);
  }

  .track-row.current .track-title {
    color: var(--accent);
  }

  .drag-handle {
    cursor: grab;
    display: flex;
    align-items: center;
    flex-shrink: 0;
  }

  .track-info {
    flex: 1;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .track-title {
    font-size: 0.85rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-meta {
    font-size: 0.75rem;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-duration {
    font-size: 0.75rem;
    color: var(--text-secondary);
    font-variant-numeric: tabular-nums;
    flex-shrink: 0;
  }

  .remove-btn {
    width: 24px;
    height: 24px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    opacity: 0;
    flex-shrink: 0;
  }

  .track-row:hover .remove-btn {
    opacity: 1;
  }

  .remove-btn:hover {
    color: var(--error);
    background: var(--bg-hover);
  }
</style>
