<script lang="ts">
  import { LibraryService, PlayerService } from "../bindings/github.com/willfish/forte";

  type Station = {
    uuid: string;
    name: string;
    streamUrl: string;
    favicon: string;
    country: string;
    tags: string;
    bitrate: number;
    codec: string;
    votes: number;
    clicks: number;
  };

  type Favourite = {
    stationUuid: string;
    name: string;
    streamUrl: string;
    faviconUrl: string;
    tags: string;
    addedAt: string;
  };

  let tab = $state<'featured' | 'favourites'>('featured');
  let searchQuery = $state('');
  let stations = $state<Station[]>([]);
  let favourites = $state<Favourite[]>([]);
  let favouriteUuids = $state<Set<string>>(new Set());
  let loading = $state(false);
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;

  const isSearchActive = $derived(searchQuery.trim().length > 0);

  async function loadFeatured() {
    loading = true;
    try {
      const result = await LibraryService.GetTopVotedRadioStations(50);
      stations = (result || []).map(mapStation);
    } catch {
      stations = [];
    } finally {
      loading = false;
    }
  }

  async function loadFavourites() {
    try {
      const result = await LibraryService.GetRadioFavourites();
      favourites = (result || []).map((f: any) => ({
        stationUuid: f.stationUuid,
        name: f.name,
        streamUrl: f.streamUrl,
        faviconUrl: f.faviconUrl,
        tags: f.tags,
        addedAt: f.addedAt,
      }));
      favouriteUuids = new Set(favourites.map(f => f.stationUuid));
    } catch {
      favourites = [];
      favouriteUuids = new Set();
    }
  }

  function mapStation(s: any): Station {
    return {
      uuid: s.uuid,
      name: s.name,
      streamUrl: s.streamUrl,
      favicon: s.favicon,
      country: s.country,
      tags: s.tags,
      bitrate: s.bitrate,
      codec: s.codec,
      votes: s.votes,
      clicks: s.clicks,
    };
  }

  function handleSearchInput(e: Event) {
    const value = (e.target as HTMLInputElement).value;
    searchQuery = value;

    if (debounceTimer) clearTimeout(debounceTimer);

    if (value.trim() === '') {
      loadFeatured();
      return;
    }

    loading = true;
    debounceTimer = setTimeout(async () => {
      try {
        const result = await LibraryService.SearchRadioStations(value.trim(), 50);
        stations = (result || []).map(mapStation);
      } catch {
        stations = [];
      } finally {
        loading = false;
      }
    }, 300);
  }

  function clearSearch() {
    searchQuery = '';
    if (debounceTimer) clearTimeout(debounceTimer);
    loadFeatured();
  }

  async function playStation(name: string, url: string, favicon: string) {
    try {
      await PlayerService.PlayRadio(name, url, favicon);
    } catch {
      // ignore play errors
    }
  }

  async function toggleFavourite(station: Station) {
    if (favouriteUuids.has(station.uuid)) {
      try {
        await LibraryService.RemoveRadioFavourite(station.uuid);
        favouriteUuids.delete(station.uuid);
        favouriteUuids = new Set(favouriteUuids);
        favourites = favourites.filter(f => f.stationUuid !== station.uuid);
      } catch { /* ignore */ }
    } else {
      try {
        await LibraryService.AddRadioFavourite(
          station.uuid, station.name, station.streamUrl, station.favicon, station.tags
        );
        favouriteUuids.add(station.uuid);
        favouriteUuids = new Set(favouriteUuids);
        await loadFavourites();
      } catch { /* ignore */ }
    }
  }

  async function removeFavourite(uuid: string) {
    try {
      await LibraryService.RemoveRadioFavourite(uuid);
      favouriteUuids.delete(uuid);
      favouriteUuids = new Set(favouriteUuids);
      favourites = favourites.filter(f => f.stationUuid !== uuid);
    } catch { /* ignore */ }
  }

  function formatTags(tags: string): string[] {
    if (!tags) return [];
    return tags.split(',').map(t => t.trim()).filter(Boolean).slice(0, 4);
  }

  // Load data on mount.
  $effect(() => {
    loadFeatured();
    loadFavourites();
  });
</script>

<div class="radio-view">
  <h2>Radio</h2>

  <div class="tabs">
    <button class="tab" class:active={tab === 'featured'} onclick={() => tab = 'featured'}>
      Browse
    </button>
    <button class="tab" class:active={tab === 'favourites'} onclick={() => tab = 'favourites'}>
      Favourites ({favourites.length})
    </button>
  </div>

  {#if tab === 'featured'}
    <div class="search-bar">
      <svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
        <path d="M15.5 14h-.79l-.28-.27A6.47 6.47 0 0 0 16 9.5 6.5 6.5 0 1 0 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
      </svg>
      <input
        type="text"
        class="search-input"
        placeholder="Search stations by name or genre..."
        value={searchQuery}
        oninput={handleSearchInput}
      />
      {#if isSearchActive}
        <button class="search-clear" onclick={clearSearch} aria-label="Clear search">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
          </svg>
        </button>
      {/if}
    </div>

    {#if loading}
      <div class="empty">Loading stations...</div>
    {:else if stations.length === 0}
      <div class="empty">
        {#if isSearchActive}
          No stations found for "{searchQuery.trim()}"
        {:else}
          No stations available
        {/if}
      </div>
    {:else}
      <div class="station-list">
        {#each stations as station (station.uuid)}
          <div class="station-card">
            <button class="station-play" onclick={() => playStation(station.name, station.streamUrl, station.favicon)} aria-label="Play {station.name}">
              <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                <path d="M8 5v14l11-7z"/>
              </svg>
            </button>
            {#if station.favicon}
              <img class="station-icon" src={station.favicon} alt="" loading="lazy" />
            {:else}
              <div class="station-icon placeholder">
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M3.24 6.15C2.51 6.43 2 7.17 2 8v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8c0-.83-.49-1.57-1.24-1.85L12 2 3.24 6.15zM12 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3z"/>
                </svg>
              </div>
            {/if}
            <div class="station-info">
              <div class="station-name">{station.name}</div>
              <div class="station-meta">
                {#if station.country}
                  <span class="station-country">{station.country}</span>
                {/if}
                {#if station.codec}
                  <span class="station-codec">{station.codec}{#if station.bitrate} {station.bitrate}kbps{/if}</span>
                {/if}
              </div>
              {#if formatTags(station.tags).length > 0}
                <div class="station-tags">
                  {#each formatTags(station.tags) as tag}
                    <span class="tag">{tag}</span>
                  {/each}
                </div>
              {/if}
            </div>
            <button
              class="fav-btn"
              class:active={favouriteUuids.has(station.uuid)}
              onclick={() => toggleFavourite(station)}
              aria-label={favouriteUuids.has(station.uuid) ? 'Remove from favourites' : 'Add to favourites'}
            >
              {#if favouriteUuids.has(station.uuid)}
                <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                  <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
                </svg>
              {:else}
                <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                  <path d="M16.5 3c-1.74 0-3.41.81-4.5 2.09C10.91 3.81 9.24 3 7.5 3 4.42 3 2 5.42 2 8.5c0 3.78 3.4 6.86 8.55 11.54L12 21.35l1.45-1.32C18.6 15.36 22 12.28 22 8.5 22 5.42 19.58 3 16.5 3zm-4.4 15.55l-.1.1-.1-.1C7.14 14.24 4 11.39 4 8.5 4 6.5 5.5 5 7.5 5c1.54 0 3.04.99 3.57 2.36h1.87C13.46 5.99 14.96 5 16.5 5c2 0 3.5 1.5 3.5 3.5 0 2.89-3.14 5.74-7.9 10.05z"/>
                </svg>
              {/if}
            </button>
          </div>
        {/each}
      </div>
    {/if}
  {:else}
    {#if favourites.length === 0}
      <div class="empty">No favourite stations yet. Browse and add some!</div>
    {:else}
      <div class="station-list">
        {#each favourites as fav (fav.stationUuid)}
          <div class="station-card">
            <button class="station-play" onclick={() => playStation(fav.name, fav.streamUrl, fav.faviconUrl)} aria-label="Play {fav.name}">
              <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                <path d="M8 5v14l11-7z"/>
              </svg>
            </button>
            {#if fav.faviconUrl}
              <img class="station-icon" src={fav.faviconUrl} alt="" loading="lazy" />
            {:else}
              <div class="station-icon placeholder">
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M3.24 6.15C2.51 6.43 2 7.17 2 8v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8c0-.83-.49-1.57-1.24-1.85L12 2 3.24 6.15zM12 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3z"/>
                </svg>
              </div>
            {/if}
            <div class="station-info">
              <div class="station-name">{fav.name}</div>
              {#if formatTags(fav.tags).length > 0}
                <div class="station-tags">
                  {#each formatTags(fav.tags) as tag}
                    <span class="tag">{tag}</span>
                  {/each}
                </div>
              {/if}
            </div>
            <button
              class="fav-btn active"
              onclick={() => removeFavourite(fav.stationUuid)}
              aria-label="Remove from favourites"
            >
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M12 21.35l-1.45-1.32C5.4 15.36 2 12.28 2 8.5 2 5.42 4.42 3 7.5 3c1.74 0 3.41.81 4.5 2.09C13.09 3.81 14.76 3 16.5 3 19.58 3 22 5.42 22 8.5c0 3.78-3.4 6.86-8.55 11.54L12 21.35z"/>
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .radio-view {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    animation: view-enter 0.2s ease-out;
  }

  h2 {
    margin: 0;
    font-size: 1.3rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .tabs {
    display: flex;
    gap: 0.25rem;
    border-bottom: 1px solid var(--border);
    padding-bottom: 0;
  }

  .tab {
    padding: 0.5rem 1rem;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.9rem;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
  }

  .tab:hover {
    color: var(--text-primary);
  }

  .tab.active {
    color: var(--accent);
    border-bottom-color: var(--accent);
  }

  .search-bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-secondary, var(--bg-hover));
  }

  .search-bar:focus-within {
    border-color: var(--accent);
  }

  .search-icon {
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .search-input {
    flex: 1;
    border: none;
    background: transparent;
    color: var(--text-primary);
    font-size: 0.9rem;
    outline: none;
    padding: 0.2rem 0;
  }

  .search-input::placeholder {
    color: var(--text-secondary);
    opacity: 0.6;
  }

  .search-clear {
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0.15rem;
    border-radius: 3px;
    flex-shrink: 0;
  }

  .search-clear:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .station-list {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .station-card {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 0.75rem;
    border-radius: 6px;
    background: transparent;
  }

  .station-card:hover {
    background: var(--bg-hover);
  }

  .station-play {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: none;
    border-radius: 50%;
    background: var(--accent);
    color: white;
    cursor: pointer;
    flex-shrink: 0;
    opacity: 0.8;
  }

  .station-play:hover {
    opacity: 1;
  }

  .station-icon {
    width: 40px;
    height: 40px;
    border-radius: 4px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .station-icon.placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-hover);
    color: var(--text-secondary);
  }

  .station-info {
    flex: 1;
    min-width: 0;
  }

  .station-name {
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .station-meta {
    display: flex;
    gap: 0.5rem;
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin-top: 0.1rem;
  }

  .station-tags {
    display: flex;
    gap: 0.25rem;
    margin-top: 0.25rem;
    flex-wrap: wrap;
  }

  .tag {
    font-size: 0.7rem;
    padding: 0.1rem 0.4rem;
    border-radius: 3px;
    background: var(--bg-hover);
    color: var(--text-secondary);
  }

  .fav-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0.25rem;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .fav-btn:hover {
    color: var(--text-primary);
  }

  .fav-btn.active {
    color: #e74c3c;
  }

  .empty {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 3rem;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }
</style>
