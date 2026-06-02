<script lang="ts">
  import { onMount } from 'svelte';
  import { LibraryService, PlayerService } from '../bindings/github.com/willfish/forte';
  import { RadioStationJSON } from '../bindings/github.com/willfish/forte/models';
  import { onPlaybackStatusChange } from './lib/playback';
  import type { PlaybackStatus } from './lib/types';
  import type { RadioStationHint } from './lib/stores';

  type Props = {
    stationUuid: string;
    hint?: RadioStationHint | null;
    onback: () => void;
  };

  let { stationUuid, hint = null, onback }: Props = $props();

  let station = $state<RadioStationJSON | null>(null);
  let loading = $state(true);
  let error = $state('');
  let isFavourite = $state(false);
  let isPinned = $state(false);
  let playbackState = $state<'stopped' | 'playing' | 'paused'>('stopped');
  let currentStationUuid = $state('');
  let playbackTitle = $state('');
  let actionError = $state('');

  onMount(() => {
    void loadStation();
    return onPlaybackStatusChange((status: PlaybackStatus) => {
      playbackState = status.state;
      currentStationUuid = status.radioUuid;
      playbackTitle = status.title;
    });
  });

  async function loadStation() {
    loading = true;
    error = '';
    actionError = '';
    try {
      station = await LibraryService.GetRadioStationByUUID(stationUuid);
      isFavourite = await LibraryService.IsRadioFavourite(stationUuid);
      if (isFavourite) {
        isPinned = await LibraryService.GetRadioFavouritePinned(stationUuid);
      } else {
        isPinned = false;
      }
    } catch (err) {
      if (hint?.name) {
        station = RadioStationJSON.createFrom({
          uuid: stationUuid,
          name: hint.name,
          streamUrl: hint.streamUrl ?? '',
          favicon: hint.favicon ?? '',
          homepage: hint.homepage ?? '',
          country: hint.country ?? '',
          tags: hint.tags ?? '',
          codec: hint.codec ?? '',
          bitrate: hint.bitrate ?? 0,
        });
        isFavourite = await LibraryService.IsRadioFavourite(stationUuid).catch(() => false);
        if (isFavourite) {
          isPinned = await LibraryService.GetRadioFavouritePinned(stationUuid).catch(() => false);
        }
        error = '';
      } else {
        station = null;
        error = err instanceof Error ? err.message : 'Could not load station details.';
      }
    } finally {
      loading = false;
    }
  }

  function isThisStationPlaying(): boolean {
    return playbackState === 'playing' && currentStationUuid === stationUuid;
  }

  function isThisStationPaused(): boolean {
    return playbackState === 'paused' && currentStationUuid === stationUuid;
  }

  function isThisStationActive(): boolean {
    return currentStationUuid === stationUuid && playbackState !== 'stopped';
  }

  function nowPlayingTrack(): string {
    if (!isThisStationActive() || !playbackTitle) {
      return '';
    }
    const name = station?.name ?? '';
    if (playbackTitle === name) {
      return '';
    }
    return playbackTitle;
  }

  function playLabel(): string {
    if (isThisStationPlaying()) return 'Pause';
    if (isThisStationPaused()) return 'Resume';
    return 'Play';
  }

  async function togglePlay() {
    if (!station) return;
    actionError = '';
    try {
      if (isThisStationActive()) {
        if (playbackState === 'playing') {
          await PlayerService.Pause();
        } else {
          await PlayerService.Resume();
        }
        return;
      }

      const art = station.favicon ? await LibraryService.ProxyImageURL(station.favicon) : '';
      await PlayerService.PlayRadioStation(
        station.uuid,
        station.name,
        station.streamUrl,
        art,
        station.homepage,
        station.tags,
        station.country,
        station.codec,
        station.bitrate
      );
    } catch (err) {
      actionError = err instanceof Error ? err.message : 'Could not change playback.';
    }
  }

  async function toggleFavourite() {
    if (!station) return;
    actionError = '';
    try {
      if (isFavourite) {
        await LibraryService.RemoveRadioFavourite(station.uuid);
        isFavourite = false;
        isPinned = false;
      } else {
        await LibraryService.AddRadioFavourite(
          station.uuid,
          station.name,
          station.streamUrl,
          station.favicon,
          station.homepage,
          station.tags,
          station.country,
          station.codec,
          station.bitrate
        );
        isFavourite = true;
        isPinned = false;
      }
    } catch (err) {
      actionError = err instanceof Error ? err.message : 'Could not update favourite.';
    }
  }

  async function togglePin() {
    if (!station || !isFavourite) return;
    actionError = '';
    try {
      const next = !isPinned;
      await LibraryService.SetRadioFavouritePinned(station.uuid, next);
      isPinned = next;
    } catch (err) {
      actionError = err instanceof Error ? err.message : 'Could not update pin.';
    }
  }

  async function openExternalURL(url: string) {
    if (!url) return;
    try {
      await LibraryService.OpenURL(url);
    } catch (err) {
      actionError = err instanceof Error ? err.message : 'Could not open link.';
    }
  }

  function handleExternalLink(event: MouseEvent, url: string) {
    event.preventDefault();
    void openExternalURL(url);
  }

  async function copyToClipboard(value: string, label: string) {
    if (!value) return;
    try {
      await navigator.clipboard.writeText(value);
    } catch {
      actionError = `Could not copy ${label}.`;
    }
  }
</script>

<div class="station-view">
  <div class="station-toolbar">
    <button type="button" class="back-btn" onclick={onback}>Radio</button>
  </div>

  {#if loading}
    <p class="status">Loading station…</p>
  {:else if error}
    <p class="status error" role="alert">{error}</p>
  {:else if station}
    {@const s = station}
    <header class="station-header">
      {#if s.favicon}
        <img class="station-art" src={s.favicon} alt="" />
      {:else}
        <div class="station-art placeholder" aria-hidden="true">📻</div>
      {/if}
      <div class="station-meta">
        <h1>{s.name}</h1>
        {#if s.country}
          <p class="country">{s.country}</p>
        {/if}
        {#if s.codec || s.bitrate}
          <p class="technical">{s.codec}{#if s.codec && s.bitrate} · {/if}{#if s.bitrate}{s.bitrate} kbps{/if}</p>
        {/if}
        {#if s.tags}
          <p class="tags">{s.tags}</p>
        {/if}
        {#if nowPlayingTrack()}
          <p class="now-playing-track" aria-live="polite">Now playing: {nowPlayingTrack()}</p>
        {/if}
      </div>
    </header>

    <div class="station-actions">
      <button type="button" class="primary" onclick={togglePlay}>{playLabel()}</button>
      <button type="button" class="secondary" onclick={toggleFavourite}>
        {isFavourite ? 'Remove favourite' : 'Add favourite'}
      </button>
      {#if isFavourite}
        <button type="button" class="secondary" onclick={togglePin}>
          {isPinned ? 'Unpin' : 'Pin'}
        </button>
      {/if}
    </div>

    <section class="links">
      <h2>Links</h2>
      {#if s.homepage}
        <div class="link-row">
          <span class="label">Website</span>
          <a
            class="url"
            href={s.homepage}
            rel="noopener noreferrer"
            target="_blank"
            onclick={(e) => handleExternalLink(e, s.homepage)}
          >{s.homepage}</a>
          <button type="button" class="ghost" onclick={() => openExternalURL(s.homepage)}>Open</button>
          <button type="button" class="ghost" onclick={() => copyToClipboard(s.homepage, 'website URL')}>Copy</button>
        </div>
      {:else}
        <p class="muted">No website listed for this station.</p>
      {/if}

      {#if s.streamUrl}
        <div class="link-row">
          <span class="label">Stream URL</span>
          <a
            class="url"
            href={s.streamUrl}
            rel="noopener noreferrer"
            target="_blank"
            onclick={(e) => handleExternalLink(e, s.streamUrl)}
          >{s.streamUrl}</a>
          <button type="button" class="ghost" onclick={() => openExternalURL(s.streamUrl)}>Open</button>
          <button type="button" class="ghost" onclick={() => copyToClipboard(s.streamUrl, 'stream URL')}>Copy</button>
        </div>
      {/if}
    </section>

    {#if actionError}
      <p class="status error" role="alert">{actionError}</p>
    {/if}
  {/if}
</div>

<style>
  .station-view {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    min-height: 0;
  }

  .station-toolbar {
    display: flex;
    align-items: center;
  }

  .back-btn {
    border: none;
    background: transparent;
    color: var(--accent);
    cursor: pointer;
    font: inherit;
    padding: 0;
  }

  .station-header {
    display: flex;
    gap: 1rem;
    align-items: flex-start;
  }

  .station-art {
    width: 96px;
    height: 96px;
    border-radius: 12px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .station-art.placeholder {
    display: grid;
    place-items: center;
    background: var(--surface-2);
    font-size: 2rem;
  }

  .station-meta h1 {
    margin: 0 0 0.35rem;
    font-size: 1.5rem;
  }

  .country,
  .technical,
  .tags,
  .now-playing-track {
    margin: 0.15rem 0;
    color: var(--text-muted);
  }

  .now-playing-track {
    color: var(--text);
    font-weight: 500;
  }

  .station-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .primary,
  .secondary,
  .ghost {
    border-radius: 8px;
    padding: 0.45rem 0.75rem;
    font: inherit;
    cursor: pointer;
  }

  .primary {
    border: 1px solid var(--accent);
    background: var(--accent);
    color: var(--accent-contrast, #fff);
  }

  .secondary,
  .ghost {
    border: 1px solid var(--border);
    background: var(--surface);
    color: var(--text);
  }

  .links h2 {
    margin: 0 0 0.5rem;
    font-size: 1rem;
  }

  .link-row {
    display: grid;
    grid-template-columns: auto 1fr auto auto;
    gap: 0.5rem;
    align-items: center;
    margin-bottom: 0.65rem;
  }

  .label {
    color: var(--text-muted);
    font-size: 0.85rem;
  }

  .url {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--accent);
    font-size: 0.85rem;
  }

  .muted,
  .status {
    color: var(--text-muted);
  }

  .status.error {
    color: var(--danger, #c0392b);
  }
</style>