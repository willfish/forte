<script lang="ts">
  import { LibraryService } from "../bindings/github.com/willfish/forte";
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
    <div class="loading">Loading albums...</div>
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
            {:else}
              <div class="artwork-placeholder">
                <svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor" opacity="0.3">
                  <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55C7.79 13 6 14.79 6 17s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
                </svg>
              </div>
            {/if}
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

  .loading, .empty {
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
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
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
  }

  .album-card:hover {
    background: var(--bg-hover);
  }

  .artwork-wrapper {
    position: relative;
    width: 100%;
  }

  .artwork {
    width: 100%;
    aspect-ratio: 1;
    border-radius: 4px;
    object-fit: cover;
  }

  .artwork-placeholder {
    width: 100%;
    aspect-ratio: 1;
    border-radius: 4px;
    background: var(--bg-hover);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
  }

  .source-badge {
    position: absolute;
    top: 4px;
    right: 4px;
    background: rgba(0, 0, 0, 0.6);
    color: #fff;
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
