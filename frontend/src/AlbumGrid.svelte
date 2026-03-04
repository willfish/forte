<script lang="ts">
  import { LibraryService, PlayerService } from "../bindings/github.com/willfish/forte";
  import { isServerOnline, onServerStatusChange } from './lib/stores';

  type AlbumItem = {
    id: number;
    title: string;
    artist: string;
    year: number;
    trackCount: number;
    source: string;
    serverId: string;
    artworkSrc?: string;
  };

  let albums = $state<AlbumItem[]>([]);
  let sortField = $state('artist');
  let sortOrder = $state('asc');
  let sourceFilter = $state('');
  let loading = $state(false);
  let statusVersion = $state(0);

  $effect(() => {
    return onServerStatusChange(() => { statusVersion++; });
  });

  const { onselect }: { onselect?: (albumId: number) => void } = $props();

  async function loadAlbums() {
    loading = true;
    try {
      const result = await LibraryService.GetAlbums(sortField, sortOrder, sourceFilter);
      albums = (result || []).map((a: any) => ({
        id: a.id,
        title: a.title,
        artist: a.artist,
        year: a.year,
        trackCount: a.trackCount,
        source: a.source || 'local',
        serverId: a.serverId || '',
      }));
      // Load artwork lazily after albums are rendered.
      for (const album of albums) {
        LibraryService.AlbumArtwork(album.id).then((src: string) => {
          album.artworkSrc = src;
        });
      }
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    loadAlbums();
  });

  function handleSort(field: string) {
    if (sortField === field) {
      sortOrder = sortOrder === 'asc' ? 'desc' : 'asc';
    } else {
      sortField = field;
      sortOrder = 'asc';
    }
    loadAlbums();
  }

  function handleSourceChange(src: string) {
    sourceFilter = src;
    loadAlbums();
  }

  function handleAlbumClick(albumId: number) {
    if (onselect) onselect(albumId);
  }

  async function playAlbum(e: Event, albumId: number, albumTitle: string) {
    e.stopPropagation();
    const trackList = await LibraryService.GetAlbumTracks(albumId);
    const queueTracks = (trackList || []).map((t: any) => ({
      trackId: t.trackId,
      title: t.title,
      artist: t.artist,
      album: albumTitle,
      durationMs: t.durationMs,
      filePath: t.filePath,
    }));
    if (queueTracks.length > 0) {
      await PlayerService.PlayQueue(queueTracks, 0);
    }
  }

  function formatYear(year: number): string {
    return year > 0 ? String(year) : '';
  }
</script>

<div class="album-grid-container">
  <div class="toolbar">
    <span class="count">{albums.length} album{albums.length !== 1 ? 's' : ''}</span>
    <div class="toolbar-right">
      <div class="source-filter">
        {#each [['', 'All'], ['local', 'Local'], ['server', 'Server']] as [value, label]}
          <button
            class:active={sourceFilter === value}
            onclick={() => handleSourceChange(value)}
          >
            {label}
          </button>
        {/each}
      </div>
      <div class="sort-buttons">
        <span class="sort-label">Sort:</span>
        {#each [['title', 'Title'], ['artist', 'Artist'], ['year', 'Year'], ['created_at', 'Added']] as [field, label]}
          <button
            class:active={sortField === field}
            onclick={() => handleSort(field)}
          >
            {label}
            {#if sortField === field}
              <span class="arrow">{sortOrder === 'asc' ? '\u2191' : '\u2193'}</span>
            {/if}
          </button>
        {/each}
      </div>
    </div>
  </div>

  {#if loading}
    <div class="grid">
      {#each Array(12) as _}
        <div class="album-card skeleton-card">
          <div class="artwork-wrapper">
            <div class="artwork-skeleton"></div>
          </div>
          <div class="album-info">
            <span class="skeleton-line skeleton-title"></span>
            <span class="skeleton-line skeleton-artist"></span>
          </div>
        </div>
      {/each}
    </div>
  {:else if albums.length === 0}
    <div class="empty">
      <svg class="empty-icon" viewBox="0 0 24 24" width="64" height="64" fill="currentColor">
        <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55C7.79 13 6 14.79 6 17s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
      </svg>
      <p class="empty-title">No albums yet</p>
      <p class="hint">Add music folders in Settings to get started.</p>
    </div>
  {:else}
    <div class="grid">
      {#each albums as album (album.id)}
        {@const unavailable = statusVersion >= 0 && album.serverId && !isServerOnline(album.serverId)}
        <button class="album-card" class:unavailable={unavailable} onclick={() => handleAlbumClick(album.id)}>
          <div class="artwork-wrapper">
            {#if album.artworkSrc}
              <img class="artwork" src={album.artworkSrc} alt="{album.title} cover" loading="lazy" />
            {:else if album.artworkSrc === undefined}
              <div class="artwork-skeleton"></div>
            {:else}
              <div class="artwork-placeholder">
                <svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor" opacity="0.3">
                  <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55C7.79 13 6 14.79 6 17s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
                </svg>
              </div>
            {/if}
            <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
            <div class="artwork-overlay" onclick={(e) => playAlbum(e, album.id, album.title)}>
              <div class="play-btn" aria-label="Play {album.title}">
                <svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
              </div>
            </div>
            {#if album.source === 'server'}
              <span class="source-badge" class:source-badge-offline={unavailable} title={unavailable ? 'Server offline' : 'Server'}>
                <svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
                  <path d="M19.35 10.04A7.49 7.49 0 0 0 12 4C9.11 4 6.6 5.64 5.35 8.04A5.994 5.994 0 0 0 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/>
                </svg>
              </span>
            {/if}
          </div>
          <div class="album-info">
            <span class="album-title">{album.title}</span>
            <span class="album-artist">{album.artist}{formatYear(album.year) ? ` (${formatYear(album.year)})` : ''}</span>
          </div>
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .album-grid-container {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 0 1rem;
    flex-shrink: 0;
  }

  .count {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .source-filter {
    display: flex;
    align-items: center;
    gap: 0.15rem;
    border: 1px solid var(--border);
    border-radius: 4px;
    overflow: hidden;
  }

  .source-filter button {
    padding: 0.25rem 0.5rem;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.75rem;
    cursor: pointer;
  }

  .source-filter button:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .source-filter button.active {
    background: var(--bg-active);
    color: var(--accent);
  }

  .sort-buttons {
    display: flex;
    align-items: center;
    gap: 0.25rem;
  }

  .sort-label {
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin-right: 0.25rem;
  }

  .sort-buttons button {
    padding: 0.3rem 0.6rem;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.8rem;
    cursor: pointer;
  }

  .sort-buttons button:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .sort-buttons button.active {
    background: var(--bg-active);
    color: var(--accent);
  }

  .arrow {
    font-size: 0.7rem;
  }

  .empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-secondary);
  }

  .empty-icon {
    opacity: 0.3;
    margin-bottom: 1rem;
  }

  .empty-title {
    font-size: 1.25rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 0.5rem;
  }

  .hint {
    font-size: 0.9rem;
    color: var(--text-secondary);
    opacity: 0.7;
    margin: 0;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 1rem;
    overflow-y: auto;
    flex: 1;
    padding-bottom: 1rem;
  }

  .album-card {
    display: flex;
    flex-direction: column;
    background: transparent;
    border: none;
    border-radius: 8px;
    padding: 0.5rem;
    cursor: pointer;
    text-align: left;
    color: inherit;
    transition: transform 0.15s ease, background 0.15s ease;
  }

  .album-card:hover {
    background: var(--bg-hover);
    transform: translateY(-2px);
  }

  .skeleton-card {
    pointer-events: none;
  }

  .artwork-wrapper {
    position: relative;
    width: 100%;
  }

  .artwork {
    width: 100%;
    aspect-ratio: 1;
    border-radius: 6px;
    object-fit: cover;
    display: block;
  }

  .artwork-placeholder {
    width: 100%;
    aspect-ratio: 1;
    border-radius: 6px;
    background: var(--bg-hover);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
  }

  .artwork-skeleton {
    width: 100%;
    aspect-ratio: 1;
    border-radius: 6px;
    background: var(--bg-hover);
    animation: pulse 1.5s ease-in-out infinite;
  }

  .skeleton-line {
    display: block;
    border-radius: 3px;
    background: var(--bg-hover);
    animation: pulse 1.5s ease-in-out infinite;
  }

  .skeleton-title {
    height: 0.85rem;
    width: 80%;
  }

  .skeleton-artist {
    height: 0.75rem;
    width: 60%;
    margin-top: 0.25rem;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }

  .artwork-overlay {
    position: absolute;
    inset: 0;
    border-radius: 6px;
    background: rgba(0, 0, 0, 0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0;
    transition: opacity 0.15s ease;
  }

  .album-card:hover .artwork-overlay {
    opacity: 1;
  }

  .play-btn {
    width: 44px;
    height: 44px;
    border-radius: 50%;
    border: none;
    background: var(--accent);
    color: var(--text-on-accent);
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: transform 0.15s ease;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.3);
  }

  .play-btn:hover {
    transform: scale(1.1);
  }

  .source-badge {
    position: absolute;
    top: 4px;
    right: 4px;
    background: rgba(0, 0, 0, 0.6);
    color: var(--text-on-accent);
    border-radius: 3px;
    padding: 2px 4px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .source-badge-offline {
    background: rgba(239, 68, 68, 0.7);
  }

  .album-card.unavailable {
    opacity: 0.45;
  }

  .album-info {
    display: flex;
    flex-direction: column;
    padding: 0.4rem 0 0;
    overflow: hidden;
  }

  .album-title {
    font-size: 0.85rem;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .album-artist {
    font-size: 0.75rem;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
