<script lang="ts">
  import { LibraryService, PlayerService } from "../bindings/github.com/willfish/forte";
  import { refreshPlaybackStatus, onPlaybackStatusChange } from './lib/playback';
  import type { RadioStationHint } from './lib/stores';
  import { browseRadioTag } from './lib/stores';

  type StationDetail = {
    uuid: string;
    name: string;
    streamUrl: string;
    homepage: string;
    favicon: string;
    country: string;
    tags: string;
    bitrate: number;
    codec: string;
    votes: number;
    clicks: number;
  };

  const {
    stationUuid,
    hint = null,
    onback,
  }: {
    stationUuid: string;
    hint?: RadioStationHint | null;
    onback: () => void;
  } = $props();

  let station = $state<StationDetail | null>(null);
  let loading = $state(true);
  let error = $state('');
  let favourite = $state(false);
  let proxiedIcon = $state('');
  let linkMessage = $state('');
  let playbackState = $state('stopped');
  let currentStationUuid = $state('');
  let radioMode = $state(false);

  $effect(() => {
    return onPlaybackStatusChange(status => {
      radioMode = status.radioMode;
      currentStationUuid = status.radioUuid;
      playbackState = status.state;
    });
  });

  $effect(() => {
    void loadStation(stationUuid, hint);
  });

  async function loadStation(uuid: string, initial: RadioStationHint | null) {
    loading = true;
    error = '';
    linkMessage = '';
    if (initial) {
      station = hintToDetail(initial);
    }

    try {
      const result = await LibraryService.GetRadioStationByUUID(uuid);
      if (result) {
        station = {
          uuid: result.uuid,
          name: result.name,
          streamUrl: result.streamUrl,
          homepage: result.homepage || '',
          favicon: result.favicon || '',
          country: result.country || '',
          tags: result.tags || '',
          bitrate: result.bitrate || 0,
          codec: result.codec || '',
          votes: result.votes || 0,
          clicks: result.clicks || 0,
        };
        favourite = await LibraryService.IsRadioFavourite(uuid);
        if (station.favicon) {
          try {
            proxiedIcon = await LibraryService.ProxyImageURL(station.favicon);
          } catch {
            proxiedIcon = '';
          }
        } else {
          proxiedIcon = '';
        }
      }
    } catch (err) {
      if (!station) {
        error = err instanceof Error ? err.message : String(err);
      }
    } finally {
      loading = false;
    }
  }

  function hintToDetail(initial: RadioStationHint): StationDetail {
    return {
      uuid: initial.uuid,
      name: initial.name || 'Radio station',
      streamUrl: initial.streamUrl || '',
      homepage: initial.homepage || '',
      favicon: initial.favicon || '',
      country: initial.country || '',
      tags: initial.tags || '',
      bitrate: initial.bitrate || 0,
      codec: initial.codec || '',
      votes: 0,
      clicks: 0,
    };
  }

  function formatTags(tags: string): string[] {
    if (!tags) return [];
    return tags.split(',').map(t => t.trim()).filter(Boolean);
  }

  function stationInitials(name: string): string {
    const cleaned = name.replace(/\([^)]*\)/g, ' ').replace(/[^a-zA-Z0-9]+/g, ' ').trim();
    if (!cleaned) return 'R';
    const words = cleaned.split(/\s+/).filter(word => !/^\d+$/.test(word));
    const source = words.length > 0 ? words : cleaned.split(/\s+/);
    if (source.length === 1) return source[0].slice(0, 2).toUpperCase();
    return `${source[0][0]}${source[1][0]}`.toUpperCase();
  }

  function isCurrentStation(): boolean {
    return radioMode && currentStationUuid === stationUuid;
  }

  function isPlaying(): boolean {
    return isCurrentStation() && playbackState === 'playing';
  }

  function isPaused(): boolean {
    return isCurrentStation() && playbackState === 'paused';
  }

  async function togglePlay() {
    if (!station) return;
    try {
      if (isCurrentStation()) {
        if (playbackState === 'playing') {
          await PlayerService.Pause();
        } else if (playbackState === 'paused') {
          await PlayerService.Resume();
        }
      } else {
        const art = station.favicon ? await LibraryService.ProxyImageURL(station.favicon) : '';
        await PlayerService.PlayRadioStation(
          station.uuid,
          station.name,
          station.streamUrl,
          art,
          station.homepage,
          station.tags
        );
      }
      await refreshPlaybackStatus();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function toggleFavourite() {
    if (!station) return;
    try {
      if (favourite) {
        await LibraryService.RemoveRadioFavourite(station.uuid);
        favourite = false;
      } else {
        await LibraryService.AddRadioFavourite(
          station.uuid,
          station.name,
          station.streamUrl,
          station.favicon,
          station.homepage,
          station.tags
        );
        favourite = true;
      }
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function openLink(url: string) {
    if (!url) return;
    try {
      await LibraryService.OpenURL(url);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function copyLink(url: string, label: string) {
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
      linkMessage = `${label} copied`;
      setTimeout(() => { linkMessage = ''; }, 2000);
    } catch {
      linkMessage = 'Could not copy link';
    }
  }
</script>

<div class="station-view">
  <button class="back-btn" type="button" onclick={onback}>
    <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
      <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/>
    </svg>
    Radio
  </button>

  {#if loading && !station}
    <div class="loading" role="status">Loading station...</div>
  {:else if error && !station}
    <div class="error-state" role="alert">{error}</div>
  {:else if station}
    {@const detail = station}
    <div class="station-header">
      {#if proxiedIcon}
        <img class="station-art" src={proxiedIcon} alt="" />
      {:else}
        <div class="station-art placeholder">
          <span>{stationInitials(station.name)}</span>
        </div>
      {/if}
      <div class="station-meta">
        <h1>{station.name}</h1>
        <div class="badges">
          {#if isPlaying()}<span class="badge playing">Playing</span>{/if}
          {#if isPaused()}<span class="badge">Paused</span>{/if}
          {#if station.country}<span>{station.country}</span>{/if}
          {#if station.codec}
            <span>{station.codec}{#if station.bitrate} · {station.bitrate} kbps{/if}</span>
          {/if}
        </div>
        {#if formatTags(station.tags).length > 0}
          <div class="tags">
            {#each formatTags(station.tags) as tag}
              <button class="tag" type="button" onclick={() => browseRadioTag(tag)}>{tag}</button>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    <div class="actions">
      <button class="primary-btn" type="button" onclick={togglePlay}>
        {isPlaying() ? 'Pause' : isPaused() ? 'Resume' : 'Play'}
      </button>
      <button class="secondary-btn" class:active={favourite} type="button" onclick={toggleFavourite}>
        {favourite ? 'Remove favourite' : 'Add favourite'}
      </button>
    </div>

    <section class="links">
      <h2>Links</h2>
      {#if linkMessage}
        <p class="link-message" role="status">{linkMessage}</p>
      {/if}
      {#if detail.homepage}
        <div class="link-row">
          <span class="link-label">Website</span>
          <a class="link-url" href={detail.homepage} onclick={(e) => { e.preventDefault(); void openLink(detail.homepage); }}>
            {detail.homepage}
          </a>
          <button class="link-action" type="button" onclick={() => copyLink(detail.homepage, 'Website')}>Copy</button>
        </div>
      {/if}
      {#if detail.streamUrl}
        <div class="link-row">
          <span class="link-label">Stream</span>
          <span class="link-url mono">{detail.streamUrl}</span>
          <button class="link-action" type="button" onclick={() => copyLink(detail.streamUrl, 'Stream URL')}>Copy</button>
        </div>
      {/if}
      {#if !detail.homepage && !detail.streamUrl}
        <p class="empty-links">No links available for this station.</p>
      {/if}
    </section>

    <section class="details">
      <h2>Details</h2>
      <dl>
        <div><dt>Station ID</dt><dd class="mono">{station.uuid}</dd></div>
        {#if station.votes > 0}<div><dt>Votes</dt><dd>{station.votes}</dd></div>{/if}
        {#if station.clicks > 0}<div><dt>Clicks</dt><dd>{station.clicks}</dd></div>{/if}
      </dl>
    </section>

    {#if error}
      <div class="error-state" role="alert">{error}</div>
    {/if}
  {/if}
</div>

<style>
  .station-view {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    animation: view-enter 0.2s ease-out;
  }

  .back-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0;
    font-size: 0.9rem;
  }

  .back-btn:hover {
    color: var(--text-primary);
  }

  .loading,
  .error-state,
  .empty-links {
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .error-state {
    color: var(--error);
  }

  .station-header {
    display: flex;
    gap: 1.25rem;
    align-items: flex-start;
  }

  .station-art {
    width: 120px;
    height: 120px;
    border-radius: 8px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .station-art.placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    background: linear-gradient(135deg, color-mix(in srgb, var(--accent) 18%, var(--bg-elevated)) 0%, var(--bg-hover) 100%);
    color: var(--accent);
    border: 1px solid color-mix(in srgb, var(--accent) 26%, var(--border));
    font-size: 1.4rem;
    font-weight: 700;
  }

  .station-meta {
    min-width: 0;
    flex: 1;
  }

  h1 {
    margin: 0;
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  h2 {
    margin: 0 0 0.6rem;
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .badges {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-top: 0.4rem;
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .badge.playing {
    color: var(--accent);
    font-weight: 600;
  }

  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
    margin-top: 0.6rem;
  }

  .tag {
    font-size: 0.75rem;
    padding: 0.15rem 0.45rem;
    border-radius: 3px;
    background: var(--bg-hover);
    color: var(--text-secondary);
    border: none;
    cursor: pointer;
  }

  .tag:hover {
    color: var(--text-primary);
    background: var(--bg-elevated, var(--bg-hover));
  }

  .actions {
    display: flex;
    gap: 0.6rem;
    flex-wrap: wrap;
  }

  .primary-btn,
  .secondary-btn,
  .link-action {
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9rem;
    padding: 0.45rem 0.8rem;
  }

  .primary-btn {
    border: none;
    background: var(--accent);
    color: white;
  }

  .secondary-btn {
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text-secondary);
  }

  .secondary-btn.active,
  .secondary-btn:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .secondary-btn.active {
    border-color: var(--accent);
    color: var(--accent);
  }

  .links,
  .details {
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 0.9rem 1rem;
  }

  .link-message {
    margin: 0 0 0.5rem;
    font-size: 0.8rem;
    color: var(--accent);
  }

  .link-row {
    display: grid;
    grid-template-columns: 5.5rem minmax(0, 1fr) auto;
    gap: 0.5rem 0.75rem;
    align-items: center;
    padding: 0.35rem 0;
  }

  .link-row + .link-row {
    border-top: 1px solid var(--border);
  }

  .link-label {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  .link-url {
    min-width: 0;
    font-size: 0.85rem;
    color: var(--accent);
    text-decoration: none;
    word-break: break-all;
  }

  .link-url:hover {
    text-decoration: underline;
  }

  .mono {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.8rem;
    word-break: break-all;
  }

  .link-action {
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text-secondary);
  }

  .link-action:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  dl {
    margin: 0;
    display: grid;
    gap: 0.5rem;
  }

  dl > div {
    display: grid;
    grid-template-columns: 6rem minmax(0, 1fr);
    gap: 0.75rem;
  }

  dt {
    margin: 0;
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  dd {
    margin: 0;
    font-size: 0.85rem;
    color: var(--text-primary);
  }

  @media (max-width: 640px) {
    .station-header {
      flex-direction: column;
    }

    .link-row {
      grid-template-columns: 1fr;
    }
  }
</style>