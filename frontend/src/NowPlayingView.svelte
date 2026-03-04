<script lang="ts">
  import { PlayerService } from "../bindings/github.com/willfish/forte";

  const { onclose }: { onclose: () => void } = $props();

  let playbackState = $state('stopped');
  let position = $state(0);
  let duration = $state(0);
  let title = $state('');
  let artist = $state('');
  let album = $state('');
  let artworkSrc = $state('');
  let shuffleOn = $state(false);
  let repeatMode = $state('off');
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

  async function toggleShuffle() {
    await PlayerService.SetShuffle(!shuffleOn);
    shuffleOn = !shuffleOn;
  }

  async function cycleRepeat() {
    const n = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
    await PlayerService.SetRepeat(n);
    repeatMode = n;
  }

  const isStopped = $derived(playbackState === 'stopped');
  const displayArt = $derived(radioMode && radioArtwork ? radioArtwork : artworkSrc);
  const displayTitle = $derived(radioMode && radioStation ? (title || radioStation) : title);
  const displayArtist = $derived(radioMode && radioStation ? (title ? radioStation : 'Radio') : artist);
  const displayAlbum = $derived(radioMode ? '' : album);

  function handleBackgroundClick(e: MouseEvent) {
    if ((e.target as HTMLElement).classList.contains('npv-backdrop')) {
      onclose();
    }
  }
</script>

<!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
<div class="npv-backdrop" onclick={handleBackgroundClick}>
  {#if displayArt}
    <img class="npv-bg" src={displayArt} alt="" aria-hidden="true" />
  {/if}
  <div class="npv-content">
    <button class="npv-close" onclick={onclose} aria-label="Close">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
        <path d="M19 6.41 17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
      </svg>
    </button>

    {#if displayArt}
      <img class="npv-artwork" src={displayArt} alt="Album art" />
    {:else}
      <div class="npv-artwork-placeholder">
        <svg viewBox="0 0 24 24" width="64" height="64" fill="currentColor" opacity="0.3">
          <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55C7.79 13 6 14.79 6 17s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
        </svg>
      </div>
    {/if}

    <div class="npv-info">
      <span class="npv-title">{displayTitle || 'No track selected'}</span>
      {#if displayArtist}
        <span class="npv-artist">{displayArtist}</span>
      {/if}
      {#if displayAlbum}
        <span class="npv-album">{displayAlbum}</span>
      {/if}
    </div>

    {#if !radioMode}
      <div class="npv-seek">
        <span class="npv-time">{formatTime(position)}</span>
        <input
          type="range"
          min="0"
          max={duration || 1}
          value={position}
          step="0.5"
          oninput={handleSeek}
          disabled={isStopped}
        />
        <span class="npv-time">{formatTime(duration)}</span>
      </div>
    {:else}
      <div class="npv-seek npv-radio-label">
        <span class="npv-radio-indicator">LIVE</span>
      </div>
    {/if}

    <div class="npv-transport">
      {#if !radioMode}
        <button class="npv-mode-btn" class:active={shuffleOn} onclick={toggleShuffle} aria-label="Shuffle">
          <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
            <path d="M10.59 9.17 5.41 4 4 5.41l5.17 5.17 1.42-1.41zM14.5 4l2.04 2.04L4 18.59 5.41 20 17.96 7.46 20 9.5V4h-5.5zm.33 9.41-1.41 1.41 3.13 3.13L14.5 20H20v-5.5l-2.04 2.04-3.13-3.13z"/>
          </svg>
        </button>
        <button onclick={previous} disabled={isStopped} aria-label="Previous">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
            <path d="M6 6h2v12H6zm3.5 6 8.5 6V6z"/>
          </svg>
        </button>
      {/if}
      <button class="npv-play-btn" onclick={togglePlayPause} disabled={isStopped} aria-label={playbackState === 'playing' ? 'Pause' : 'Play'}>
        {#if playbackState === 'playing'}
          <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
            <path d="M6 19h4V5H6zm8-14v14h4V5z"/>
          </svg>
        {:else}
          <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
            <path d="M8 5v14l11-7z"/>
          </svg>
        {/if}
      </button>
      {#if !radioMode}
        <button onclick={next} disabled={isStopped} aria-label="Next">
          <svg viewBox="0 0 24 24" width="22" height="22" fill="currentColor">
            <path d="M6 18l8.5-6L6 6zm10-12v12h2V6z"/>
          </svg>
        </button>
        <button class="npv-mode-btn" class:active={repeatMode !== 'off'} onclick={cycleRepeat} aria-label="Repeat: {repeatMode}">
          {#if repeatMode === 'one'}
            <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
              <path d="M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4zm-4-2V9h-1l-2 1v1h1.5v4H13z"/>
            </svg>
          {:else}
            <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
              <path d="M7 7h10v3l4-4-4-4v3H5v6h2V7zm10 10H7v-3l-4 4 4 4v-3h12v-6h-2v4z"/>
            </svg>
          {/if}
        </button>
      {/if}
    </div>
  </div>
</div>

<style>
  .npv-backdrop {
    position: absolute;
    inset: 0;
    background: var(--bg-main);
    overflow: hidden;
    display: flex;
    align-items: center;
    justify-content: center;
    animation: npv-fade-in 0.25s ease-out;
    z-index: 50;
  }

  .npv-bg {
    position: absolute;
    inset: -40px;
    width: calc(100% + 80px);
    height: calc(100% + 80px);
    object-fit: cover;
    filter: blur(60px) brightness(0.3);
    opacity: 0.6;
    pointer-events: none;
  }

  .npv-content {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.25rem;
    max-width: 400px;
    width: 100%;
    padding: 2rem;
  }

  .npv-close {
    position: absolute;
    top: 0;
    right: 0;
    background: transparent;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0.5rem;
    border-radius: 4px;
    display: flex;
  }

  .npv-close:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .npv-artwork {
    width: 280px;
    height: 280px;
    border-radius: 8px;
    object-fit: cover;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .npv-artwork-placeholder {
    width: 280px;
    height: 280px;
    border-radius: 8px;
    background: var(--bg-hover);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
  }

  .npv-info {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.25rem;
    text-align: center;
    width: 100%;
  }

  .npv-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 100%;
  }

  .npv-artist {
    font-size: 0.9rem;
    color: var(--text-secondary);
  }

  .npv-album {
    font-size: 0.8rem;
    color: var(--text-secondary);
    opacity: 0.7;
  }

  .npv-seek {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
  }

  .npv-seek input {
    flex: 1;
    accent-color: var(--accent);
    height: 6px;
  }

  .npv-time {
    font-size: 0.75rem;
    color: var(--text-secondary);
    min-width: 2.5em;
    font-variant-numeric: tabular-nums;
  }

  .npv-time:last-child {
    text-align: right;
  }

  .npv-radio-label {
    justify-content: center;
  }

  .npv-radio-indicator {
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    color: var(--error);
    padding: 0.15rem 0.5rem;
    border: 1px solid var(--error);
    border-radius: 3px;
  }

  .npv-transport {
    display: flex;
    align-items: center;
    gap: 1rem;
  }

  .npv-transport button {
    width: 36px;
    height: 36px;
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

  .npv-transport button:hover:not(:disabled) {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .npv-transport button:disabled {
    opacity: 0.3;
    cursor: default;
  }

  .npv-play-btn {
    width: 52px !important;
    height: 52px !important;
    background: var(--accent) !important;
    color: var(--text-on-accent) !important;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  }

  .npv-play-btn:hover:not(:disabled) {
    filter: brightness(1.15);
    background: var(--accent) !important;
    color: var(--text-on-accent) !important;
  }

  .npv-mode-btn {
    opacity: 0.5;
  }

  .npv-mode-btn:hover {
    opacity: 0.8;
  }

  .npv-mode-btn.active {
    color: var(--accent) !important;
    opacity: 1;
  }

  @keyframes npv-fade-in {
    from {
      opacity: 0;
      transform: translateY(20px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
</style>
