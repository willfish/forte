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

<div class="now-playing">
  <div class="transport">
    <button onclick={togglePlayPause} disabled={playbackState === 'stopped'}>
      {playbackState === 'playing' ? 'Pause' : 'Play'}
    </button>
    <button onclick={stop} disabled={playbackState === 'stopped'}>Stop</button>
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

  <div class="volume">
    <span class="label">Vol</span>
    <input
      type="range"
      min="0"
      max="100"
      value={volume}
      oninput={handleVolume}
    />
    <span class="value">{volume}%</span>
  </div>
</div>

<style>
  .now-playing {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 500px;
    margin: 0 auto;
  }

  .transport {
    display: flex;
    gap: 0.5rem;
    justify-content: center;
  }

  .seek, .volume {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .seek input, .volume input {
    flex: 1;
    accent-color: #3366aa;
  }

  .time, .label, .value {
    font-size: 0.85rem;
    color: #8899aa;
    min-width: 3em;
    font-variant-numeric: tabular-nums;
  }

  .time:last-child, .value {
    text-align: right;
  }

  button {
    padding: 0.5rem 1.25rem;
    border-radius: 8px;
    border: none;
    background: #3366aa;
    color: #fff;
    font-size: 0.9rem;
    cursor: pointer;
  }

  button:hover:not(:disabled) {
    background: #4477bb;
  }

  button:disabled {
    opacity: 0.4;
    cursor: default;
  }
</style>
