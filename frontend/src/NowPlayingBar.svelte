<script lang="ts">
  import { PlayerService } from "../bindings/github.com/willfish/forte";
  import { onPlaybackStatusChange, refreshPlaybackStatus } from './lib/playback';
  import { openRadioStation } from './lib/stores';
  import type { PlaybackStatus, RepeatMode } from './lib/types';
  import Icon from './lib/Icon.svelte';

  const { onqueuetoggle, onexpand }: { onqueuetoggle: () => void; onexpand: () => void } = $props();

  let playbackState = $state('stopped');
  let position = $state(0);
  let duration = $state(0);
  let volume = $state(100);
  let title = $state('');
  let artist = $state('');
  let album = $state('');
  let artworkSrc = $state('');
  let shuffleOn = $state(false);
  let repeatMode = $state<RepeatMode>('off');
  let muted = $state(false);
  let volumeBeforeMute = $state(100);
  let radioMode = $state(false);
  let radioStation = $state('');
  let radioUuid = $state('');
  let radioArtwork = $state('');

  $effect(() => {
    return onPlaybackStatusChange(applyStatus);
  });

  function applyStatus(s: PlaybackStatus) {
    playbackState = s.state;
    position = s.position;
    duration = s.duration;
    volume = s.volume;
    title = s.title;
    artist = s.artist;
    album = s.album;
    artworkSrc = s.artworkSrc;
    shuffleOn = s.shuffle;
    repeatMode = s.repeat;
    radioMode = s.radioMode;
    radioStation = s.radioStation;
    radioUuid = s.radioUuid;
    radioArtwork = s.radioArtwork;
  }

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
    await refreshPlaybackStatus();
  }

  async function previous() {
    await PlayerService.Previous();
    await refreshPlaybackStatus();
  }

  async function next() {
    await PlayerService.Next();
    await refreshPlaybackStatus();
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
    await refreshPlaybackStatus();
  }

  async function cycleRepeat() {
    const next: RepeatMode = repeatMode === 'off' ? 'all' : repeatMode === 'all' ? 'one' : 'off';
    await PlayerService.SetRepeat(next);
    await refreshPlaybackStatus();
  }

  const isStopped = $derived(playbackState === 'stopped');
</script>

<footer class="bar">
  <div class="track-info">
    <button class="artwork-btn" type="button" onclick={onexpand} aria-label="Open now playing">
      {#if radioMode && radioArtwork}
        <img class="artwork" src={radioArtwork} alt="Station art" />
      {:else if artworkSrc}
        <img class="artwork" src={artworkSrc} alt="Album art" />
      {:else}
        <div class="artwork-placeholder"></div>
      {/if}
    </button>
    <div class="meta">
      {#if radioMode && radioStation}
        <span class="title">{title || radioStation}</span>
        {#if radioUuid}
          <button type="button" class="artist station-link" onclick={() => openRadioStation(radioUuid, { name: radioStation })}>
            {title ? radioStation : 'View station'}
          </button>
        {:else}
          <span class="artist">Radio</span>
        {/if}
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
          <Icon name="shuffle" size={14} />
        </button>
        <button onclick={previous} disabled={isStopped} aria-label="Previous">
          <Icon name="prev" size={14} />
        </button>
      {/if}
      <button class="play-btn" onclick={togglePlayPause} disabled={isStopped} aria-label={playbackState === 'playing' ? 'Pause' : 'Play'}>
        <Icon name={playbackState === 'playing' ? 'pause' : 'play'} size={16} />
      </button>
      {#if !radioMode}
        <button onclick={next} disabled={isStopped} aria-label="Next">
          <Icon name="next" size={14} />
        </button>
      {/if}
      <button onclick={stop} disabled={isStopped} aria-label="Stop">
        <Icon name="stop" size={14} />
      </button>
      {#if !radioMode}
        <button class="mode-btn" class:active={repeatMode !== 'off'} onclick={cycleRepeat} aria-label="Repeat: {repeatMode}">
          <Icon name={repeatMode === 'one' ? 'repeatOne' : 'repeat'} size={14} />
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
      <Icon name="queue" size={16} />
    </button>
    <button class="vol-btn" onclick={toggleMute} aria-label={muted || volume === 0 ? 'Unmute' : 'Mute'}>
      {#if muted || volume === 0}
        <Icon name="volumeMute" size={16} />
      {:else if volume < 50}
        <Icon name="volumeLow" size={16} />
      {:else}
        <Icon name="volume" size={16} />
      {/if}
    </button>
    <input
      type="range"
      min="0"
      max="100"
      value={muted ? 0 : volume}
      oninput={handleVolume}
      aria-label="Volume"
    />
  </div>
</footer>

<style>
  .bar {
    height: 72px;
    flex-shrink: 0;
    background: color-mix(in srgb, var(--bg-bar) var(--theme-surface-opacity), transparent);
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

  .artwork-btn {
    cursor: pointer;
    flex-shrink: 0;
    border-radius: 4px;
    transition: opacity 0.15s ease;
  }

  .artwork-btn:hover {
    opacity: 0.8;
  }

  .artwork {
    width: 48px;
    height: 48px;
    border-radius: 4px;
    object-fit: cover;
    display: block;
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

  .station-link {
    padding: 0;
    border: none;
    background: transparent;
    text-align: left;
    cursor: pointer;
  }

  .station-link:hover {
    color: var(--accent);
    text-decoration: underline;
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
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    transition: color 0.15s ease, background 0.15s ease, opacity 0.15s ease;
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
    width: 40px;
    height: 40px;
    background: var(--accent);
    color: var(--text-on-accent);
    transition: transform 0.1s ease;
  }

  .transport .play-btn:hover:not(:disabled) {
    background: var(--accent);
    color: var(--text-on-accent);
    filter: brightness(1.15);
  }

  .transport .play-btn:active:not(:disabled) {
    transform: scale(0.9);
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
    color: var(--error);
    padding: 0.15rem 0.5rem;
    border: 1px solid var(--error);
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
    width: 32px;
    height: 32px;
    padding: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
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
    width: 32px;
    height: 32px;
    padding: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
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

  @media (max-width: 900px) {
    .bar {
      grid-template-columns: 1fr auto auto;
      height: auto;
      padding: 0.5rem 0.75rem;
      gap: 0.5rem;
    }

    .controls {
      flex-direction: row;
      gap: 0.5rem;
    }

    .seek {
      display: none;
    }

    .volume-section input {
      display: none;
    }
  }
</style>
