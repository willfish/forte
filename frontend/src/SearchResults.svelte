<script lang="ts">
  import { PlayerService } from "../bindings/github.com/willfish/forte";
  import { isServerOnline, onServerStatusChange } from './lib/stores';

  type Result = {
    trackId: number;
    title: string;
    artist: string;
    album: string;
    genre: string;
    durationMs: number;
    filePath: string;
    source: string;
    serverId: string;
  };

  const { results, query }: { results: Result[]; query: string } = $props();

  let currentFilePath = $state('');
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let statusVersion = $state(0);

  $effect(() => {
    return onServerStatusChange(() => { statusVersion++; });
  });

  function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(async () => {
      const state = await PlayerService.State();
      if (state !== 'stopped') {
        currentFilePath = await PlayerService.MediaPath();
      } else {
        currentFilePath = '';
      }
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

  async function playFromResult(index: number) {
    const queueTracks = results.map(r => ({
      trackId: r.trackId,
      title: r.title,
      artist: r.artist,
      album: r.album,
      durationMs: r.durationMs,
      filePath: r.filePath,
    }));
    if (queueTracks.length > 0) {
      await PlayerService.PlayQueue(queueTracks, index);
    }
  }

  function formatDuration(ms: number): string {
    const totalSeconds = Math.floor(ms / 1000);
    const m = Math.floor(totalSeconds / 60);
    const s = totalSeconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  }
</script>

<div class="search-results">
  <p class="result-count">
    {results.length} result{results.length !== 1 ? 's' : ''} for "{query}"
  </p>

  {#if results.length === 0}
    <div class="empty">
      <p>No tracks found.</p>
    </div>
  {:else}
    <div class="result-header">
      <span class="col-num">#</span>
      <span class="col-title">Title</span>
      <span class="col-artist">Artist</span>
      <span class="col-album">Album</span>
      <span class="col-duration">Duration</span>
    </div>
    {#each results as result, i (result.trackId)}
      {@const resultUnavailable = statusVersion >= 0 && result.serverId && !isServerOnline(result.serverId)}
      <button
        class="result-row"
        class:playing={result.filePath === currentFilePath}
        class:unavailable={resultUnavailable}
        ondblclick={() => playFromResult(i)}
      >
        <span class="col-num">
          {#if result.filePath === currentFilePath}
            <svg viewBox="0 0 24 24" width="14" height="14" fill="var(--accent)">
              <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3A4.5 4.5 0 0 0 14 7.97v8.05c1.48-.73 2.5-2.25 2.5-4.02z"/>
            </svg>
          {:else}
            {i + 1}
          {/if}
        </span>
        <span class="col-title">
          {result.title}
          {#if result.source === 'server'}
            <svg class="server-icon" viewBox="0 0 24 24" width="10" height="10" fill="currentColor">
              <path d="M19.35 10.04A7.49 7.49 0 0 0 12 4C9.11 4 6.6 5.64 5.35 8.04A5.994 5.994 0 0 0 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/>
            </svg>
          {/if}
        </span>
        <span class="col-artist">{result.artist}</span>
        <span class="col-album">{result.album}</span>
        <span class="col-duration">{formatDuration(result.durationMs)}</span>
      </button>
    {/each}
  {/if}
</div>

<style>
  .search-results {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .result-count {
    font-size: 0.85rem;
    color: var(--text-secondary);
    margin: 0 0 0.75rem;
    flex-shrink: 0;
  }

  .empty {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .result-header {
    display: grid;
    grid-template-columns: 2.5rem 1fr 1fr 1fr 3.5rem;
    gap: 0.5rem;
    padding: 0.25rem 0.5rem;
    border-bottom: 1px solid var(--border);
    margin-bottom: 0.25rem;
    flex-shrink: 0;
  }

  .result-header span {
    font-size: 0.75rem;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .result-row {
    display: grid;
    grid-template-columns: 2.5rem 1fr 1fr 1fr 3.5rem;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.5rem;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: inherit;
    cursor: pointer;
    text-align: left;
    width: 100%;
  }

  .result-row:hover {
    background: var(--bg-hover);
  }

  .result-row.playing {
    background: var(--bg-active);
  }

  .result-row.playing .col-title {
    color: var(--accent);
  }

  .result-row.unavailable {
    opacity: 0.45;
  }

  .col-num {
    font-size: 0.85rem;
    color: var(--text-secondary);
    text-align: center;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .col-title {
    font-size: 0.9rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
  }

  .col-artist, .col-album {
    font-size: 0.85rem;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .col-duration {
    font-size: 0.8rem;
    color: var(--text-secondary);
    text-align: right;
    font-variant-numeric: tabular-nums;
  }

  .server-icon {
    opacity: 0.4;
    flex-shrink: 0;
  }
</style>
