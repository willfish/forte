<script lang="ts">
  import { PlayerService } from "../bindings/github.com/willfish/forte";

  let playbackState = $state('stopped');
  let position = $state(0);
  let duration = $state(0);
  let volume = $state(100);
  let pollTimer: ReturnType<typeof setInterval> | null = null;

  function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(async () => {
      playbackState = await PlayerService.State();
      position = await PlayerService.Position();
      duration = await PlayerService.Duration();
      volume = await PlayerService.Volume();
    }, 250);
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

  function formatTime(seconds: number): string {
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
  }

  async function togglePlayPause() {
    if (playbackState === 'playing') {
      await PlayerService.Pause();
    } else if (playbackState === 'paused') {
      await PlayerService.Resume();
    }
  }

  async function stop() {
    await PlayerService.Stop();
  }

  async function handleSeek(e: Event) {
    const target = e.target as HTMLInputElement;
    await PlayerService.Seek(parseFloat(target.value));
  }

  async function handleVolume(e: Event) {
    const target = e.target as HTMLInputElement;
    await PlayerService.SetVolume(parseInt(target.value));
  }
</script>

<footer class="bar">
  <div class="track-info">
    <div class="artwork-placeholder"></div>
    <div class="meta">
      {#if playbackState !== 'stopped'}
        <span class="title">Now Playing</span>
        <span class="artist">-</span>
      {:else}
        <span class="title idle">No track selected</span>
      {/if}
    </div>
  </div>

  <div class="controls">
    <div class="transport">
      <button onclick={togglePlayPause} disabled={playbackState === 'stopped'}>
        {playbackState === 'playing' ? '\u23F8' : '\u25B6'}
      </button>
      <button onclick={stop} disabled={playbackState === 'stopped'}>
        \u23F9
      </button>
    </div>

    <div class="seek">
      <span class="time">{formatTime(position)}</span>
      <input
        type="range"
        min="0"
        max={duration || 1}
        value={position}
        step="0.5"
        oninput={handleSeek}
        disabled={playbackState === 'stopped'}
      />
      <span class="time">{formatTime(duration)}</span>
    </div>
  </div>

  <div class="volume-section">
    <span class="vol-icon">{volume === 0 ? '\u{1F507}' : '\u{1F50A}'}</span>
    <input
      type="range"
      min="0"
      max="100"
      value={volume}
      oninput={handleVolume}
    />
  </div>
</footer>

<style>
  .bar {
    height: 72px;
    background: var(--bg-bar);
    border-top: 1px solid var(--border);
    display: grid;
    grid-template-columns: 250px 1fr 150px;
    align-items: center;
    padding: 0 1rem;
    gap: 1rem;
  }

  .track-info {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    overflow: hidden;
  }

  .artwork-placeholder {
    width: 48px;
    height: 48px;
    border-radius: 4px;
    background: var(--bg-hover);
    flex-shrink: 0;
  }

  .meta {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .title {
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .title.idle {
    color: var(--text-secondary);
  }

  .artist {
    font-size: 0.75rem;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .controls {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.25rem;
  }

  .transport {
    display: flex;
    gap: 0.5rem;
  }

  .transport button {
    width: 32px;
    height: 32px;
    border-radius: 50%;
    border: none;
    background: var(--accent);
    color: #fff;
    font-size: 0.85rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
  }

  .transport button:hover:not(:disabled) {
    filter: brightness(1.15);
  }

  .transport button:disabled {
    opacity: 0.4;
    cursor: default;
  }

  .seek {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    max-width: 500px;
  }

  .seek input {
    flex: 1;
    accent-color: var(--accent);
    height: 4px;
  }

  .time {
    font-size: 0.7rem;
    color: var(--text-secondary);
    min-width: 2.5em;
    font-variant-numeric: tabular-nums;
  }

  .time:last-child {
    text-align: right;
  }

  .volume-section {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    justify-content: flex-end;
  }

  .vol-icon {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .volume-section input {
    width: 80px;
    accent-color: var(--accent);
    height: 4px;
  }
</style>
