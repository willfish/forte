<script lang="ts">
  import { PlayerService } from "../bindings/github.com/willfish/forte";

  const { onqueuetoggle }: { onqueuetoggle: () => void } = $props();

  let playbackState = $state('stopped');
  let position = $state(0);
  let duration = $state(0);
  let volume = $state(100);
  let title = $state('');
  let artist = $state('');
  let album = $state('');
  let artworkSrc = $state('');
  let shuffleOn = $state(false);
  let repeatMode = $state('off');
  let muted = $state(false);
  let volumeBeforeMute = $state(100);
  let radioMode = $state(false);
  let radioStation = $state('');
  let radioArtwork = $state('');
  let pollTimer: ReturnType<typeof setInterval> | null = null;

  function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(async () => {
      playbackState = await PlayerService.State();
      position = await PlayerService.Position();
      duration = await PlayerService.Duration();
      volume = await PlayerService.Volume();
      title = await PlayerService.MediaTitle();
      artist = await PlayerService.MediaArtist();
      album = await PlayerService.MediaAlbum();
      shuffleOn = await PlayerService.GetShuffle();
      repeatMode = await PlayerService.GetRepeat();
      radioMode = await PlayerService.IsRadioMode();
      if (radioMode) {
        radioStation = await PlayerService.RadioStationName();
        radioArtwork = await PlayerService.RadioArtworkURL();
      }
    }, 250);
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  // Fetch artwork less frequently (only when track changes).
  let lastArtworkKey = '';
  async function refreshArtwork() {
    const key = title + '|' + artist;
    if (key === lastArtworkKey) return;
    lastArtworkKey = key;
    if (playbackState === 'stopped' || !title) {
      artworkSrc = '';
      return;
    }
    artworkSrc = await PlayerService.Artwork();
  }

  $effect(() => {
    startPolling();
    const artworkTimer = setInterval(refreshArtwork, 1000);
    return () => {
      stopPolling();
      clearInterval(artworkTimer);
    };
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
    if (radioMode) {
      await PlayerService.StopRadio();
      radioMode = false;
      radioStation = '';
      radioArtwork = '';
    } else {
      await PlayerService.Stop();
    }
    artworkSrc = '';
    lastArtworkKey = '';
  }

  async function previous() {
    await PlayerService.Previous();
    lastArtworkKey = '';
  }

  async function next() {
    await PlayerService.Next();
    lastArtworkKey = '';
  }

  async function handleSeek(e: Event) {
    const target = e.target as HTMLInputElement;
    await PlayerService.Seek(parseFloat(target.value));
  }

  async function handleVolume(e: Event) {
    const target = e.target as HTMLInputElement;
    const v = parseInt(target.value);
    muted = false;
    await PlayerService.SetVolume(v);
  }

  async function toggleMute() {
    if (muted) {
      muted = false;
      await PlayerService.SetVolume(volumeBeforeMute);
    } else {
      volumeBeforeMute = volume;
      muted = true;
      await PlayerService.SetVolume(0);
    }
  }

  async function toggleShuffle() {
    await PlayerService.SetShuffle(!shuffleOn);
    shuffleOn = !shuffleOn;
  }

  async function cycleRepeat() {
    const next = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
    await PlayerService.SetRepeat(next);
    repeatMode = next;
  }

  const isStopped = $derived(playbackState === 'stopped');
</script>

<footer class="bar">
  <div class="track-info">
    {#if radioMode && radioArtwork}
      <img class="artwork" src={radioArtwork} alt="Station art" />
    {:else if artworkSrc}
      <img class="artwork" src={artworkSrc} alt="Album art" />
    {:else}
      <div class="artwork-placeholder"></div>
    {/if}
    <div class="meta">
      {#if radioMode && radioStation}
        <span class="title">{title || radioStation}</span>
        <span class="artist">{title ? radioStation : 'Radio'}</span>
      {:else if !isStopped && title}
        <span class="title">{title}</span>
        <span class="artist">{artist}{album ? ` - ${album}` : ''}</span>
      {:else if !isStopped}
        <span class="title">Playing</span>
        <span class="artist">Unknown track</span>
      {:else}
        <span class="title idle">No track selected</span>
      {/if}
    </div>
  </div>

  <div class="controls">
    <div class="transport">
      {#if !radioMode}
        <button class="mode-btn" class:active={shuffleOn} onclick={toggleShuffle} aria-label="Shuffle">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
            <path d="M10.59 9.17 5.41 4 4 5.41l5.17 5.17 1.42-1.41zM14.5 4l2.04 2.04L4 18.59 5.41 20 17.96 7.46 20 9.5V4h-5.5zm.33 9.41-1.41 1.41 3.13 3.13L14.5 20H20v-5.5l-2.04 2.04-3.13-3.13z"/>
          </svg>
        </button>
        <button onclick={previous} disabled={isStopped} aria-label="Previous">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
            <path d="M6 6h2v12H6zm3.5 6 8.5 6V6z"/>
          </svg>
        </button>
      {/if}
      <button class="play-btn" onclick={togglePlayPause} disabled={isStopped} aria-label={playbackState === 'playing' ? 'Pause' : 'Play'}>
        {#if playbackState === 'playing'}
          <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
            <path d="M6 19h4V5H6zm8-14v14h4V5z"/>
          </svg>
        {:else}
          <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
          </svg>
        {/if}
      </button>
      {#if !radioMode}
        <button onclick={next} disabled={isStopped} aria-label="Next">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
            <path d="M6 18l8.5-6L6 6zm10-12v12h2V6z"/>
          </svg>
        </button>
      {/if}
      <button onclick={stop} disabled={isStopped} aria-label="Stop">
        <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
          <path d="M6 6h12v12H6z"/>
        </svg>
      </button>
      {#if !radioMode}
        <button class="mode-btn" class:active={repeatMode !== 'off'} onclick={cycleRepeat} aria-label="Repeat: {repeatMode}">
          {#if repeatMode === 'one'}
            <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
              <path d="M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4zm-4-2V9h-1l-2 1v1h1.5v4H13z"/>
            </svg>
          {:else}
            <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
              <path d="M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4z"/>
            </svg>
          {/if}
        </button>
      {/if}
    </div>

    {#if !radioMode}
      <div class="seek">
        <span class="time">{formatTime(position)}</span>
        <input
          type="range"
          min="0"
          max={duration || 1}
          value={position}
          step="0.5"
          oninput={handleSeek}
          disabled={isStopped}
        />
        <span class="time">{formatTime(duration)}</span>
      </div>
    {:else}
      <div class="seek radio-label">
        <span class="radio-indicator">LIVE</span>
      </div>
    {/if}
  </div>

  <div class="volume-section">
    <button class="queue-btn" onclick={onqueuetoggle} aria-label="Queue">
      <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
        <path d="M15 6H3v2h12V6zm0 4H3v2h12v-2zM3 16h8v-2H3v2zM17 6v8.18c-.31-.11-.65-.18-1-.18-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3V8h3V6h-5z"/>
      </svg>
    </button>
    <button class="vol-btn" onclick={toggleMute} aria-label={muted || volume === 0 ? 'Unmute' : 'Mute'}>
      {#if muted || volume === 0}
        <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
          <path d="M16.5 12A4.5 4.5 0 0 0 14 7.97v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51A8.796 8.796 0 0 0 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3 3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06a8.99 8.99 0 0 0 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4 9.91 6.09 12 8.18V4z"/>
        </svg>
      {:else if volume < 50}
        <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
          <path d="M18.5 12A4.5 4.5 0 0 0 16 7.97v8.05c1.48-.73 2.5-2.25 2.5-4.02zM5 9v6h4l5 5V4L9 9H5z"/>
        </svg>
      {:else}
        <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
          <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3A4.5 4.5 0 0 0 14 7.97v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z"/>
        </svg>
      {/if}
    </button>
    <input
      type="range"
      min="0"
      max="100"
      value={muted ? 0 : volume}
      oninput={handleVolume}
    />
  </div>
</footer>

<style>
  .bar {
    height: 72px;
    flex-shrink: 0;
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

  .artwork {
    width: 48px;
    height: 48px;
    border-radius: 4px;
    object-fit: cover;
    flex-shrink: 0;
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
    width: 28px;
    height: 28px;
    border-radius: 50%;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
  }

  .transport button:hover:not(:disabled) {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .transport button:disabled {
    opacity: 0.3;
    cursor: default;
  }

  .transport .play-btn {
    width: 32px;
    height: 32px;
    background: var(--accent);
    color: #fff;
  }

  .transport .play-btn:hover:not(:disabled) {
    background: var(--accent);
    color: #fff;
    filter: brightness(1.15);
  }

  .transport .mode-btn {
    color: var(--text-secondary);
    opacity: 0.5;
  }

  .transport .mode-btn:hover {
    opacity: 0.8;
  }

  .transport .mode-btn.active {
    color: var(--accent);
    opacity: 1;
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

  .radio-label {
    justify-content: center;
  }

  .radio-indicator {
    font-size: 0.7rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    color: #e74c3c;
    padding: 0.15rem 0.5rem;
    border: 1px solid #e74c3c;
    border-radius: 3px;
  }

  .volume-section {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    justify-content: flex-end;
  }

  .queue-btn {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 4px;
    display: flex;
    align-items: center;
    border-radius: 4px;
  }

  .queue-btn:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .vol-btn {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 4px;
    display: flex;
    align-items: center;
    border-radius: 4px;
  }

  .vol-btn:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .volume-section input {
    width: 80px;
    accent-color: var(--accent);
    height: 4px;
  }
</style>
