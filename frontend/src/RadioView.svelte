<script lang="ts">
  import { LibraryService, PlayerService } from "../bindings/github.com/willfish/forte";
  import { onPlaybackStatusChange, refreshPlaybackStatus } from './lib/playback';

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
    pinned: boolean;
  };

  type CustomStation = {
    stationUuid: string;
    name: string;
    streamUrl: string;
    faviconUrl: string;
    tags: string;
    createdAt: string;
  };

  type HistoryEntry = {
    stationUuid: string;
    name: string;
    streamUrl: string;
    faviconUrl: string;
    tags: string;
    lastTitle: string;
    lastError: string;
    playCount: number;
    lastPlayedAt: string;
  };

  const {
    initialTab = 'featured',
  }: { initialTab?: 'featured' | 'favourites' | 'custom' | 'history' } = $props();

  let tab = $state<'featured' | 'favourites' | 'custom' | 'history'>('featured');
  let searchQuery = $state('');
  let stations = $state<Station[]>([]);
  let favourites = $state<Favourite[]>([]);
  let customStations = $state<CustomStation[]>([]);
  let history = $state<HistoryEntry[]>([]);
  let favouriteUuids = $state<Set<string>>(new Set());
  let loading = $state(false);
  let customSaving = $state(false);
  let customError = $state('');
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let customName = $state('');
  let customStreamUrl = $state('');
  let customFaviconUrl = $state('');
  let customTags = $state('');
  let radioMode = $state(false);
  let currentStationUuid = $state('');

  // Active filters.
  let activeTag = $state('');
  let activeSource = $state<'all' | 'somafm'>('all');
  let activeCountry = $state('');
  let activeCodec = $state('');

  const countries = [
    { code: 'The United States Of America', label: 'US' },
    { code: 'United Kingdom', label: 'UK' },
    { code: 'Germany', label: 'DE' },
    { code: 'France', label: 'FR' },
    { code: 'Canada', label: 'CA' },
    { code: 'Australia', label: 'AU' },
  ];
  const codecs = ['MP3', 'AAC', 'OGG'];

  // Proxied image cache: external URL -> data URI.
  const iconCache = new Map<string, string>();
  let proxiedIcons = $state<Record<string, string>>({});

  // Proxy all favicon URLs for a list of stations in parallel.
  // Returns the set of URLs that resolved successfully.
  async function proxyStationIcons(urls: string[]): Promise<Set<string>> {
    const resolved = new Set<string>();
    const toFetch = urls.filter(u => u && !u.startsWith('data:') && !iconCache.has(u));

    // Already-cached or data URIs count as resolved.
    for (const u of urls) {
      if (!u) continue;
      if (u.startsWith('data:')) { resolved.add(u); continue; }
      const cached = iconCache.get(u);
      if (cached) resolved.add(u);
    }

    // Mark pending immediately so we don't re-request.
    for (const url of toFetch) {
      iconCache.set(url, '');
    }

    await Promise.all(toFetch.map(async (url) => {
      try {
        const dataUri = await LibraryService.ProxyImageURL(url);
        iconCache.set(url, dataUri || '');
        if (dataUri) resolved.add(url);
      } catch {
        // Failed to proxy - leave as empty.
      }
    }));

    proxiedIcons = Object.fromEntries(iconCache);
    return resolved;
  }

  function resolvedIcon(url: string): string {
    if (!url) return '';
    if (url.startsWith('data:')) return url;
    return proxiedIcons[url] || '';
  }

  const isSearchActive = $derived(searchQuery.trim().length > 0);
  const hasFilter = $derived(
    activeTag !== '' || activeSource !== 'all' ||
    activeCountry !== '' || activeCodec !== ''
  );

  $effect(() => {
    return onPlaybackStatusChange(status => {
      radioMode = status.radioMode;
      currentStationUuid = status.radioUuid;
    });
  });

  // Proxy favicons without hiding stations that do not publish artwork.
  async function proxyAndFilter(raw: Station[], limit: number): Promise<Station[]> {
    await proxyStationIcons(raw.map(s => s.favicon));
    return raw.slice(0, limit);
  }

  async function loadFeatured() {
    loading = true;
    try {
      // Fetch top-voted stations from curated countries in parallel,
      // then merge and deduplicate for a mainstream default view.
      const perCountry = await Promise.all(
        countries.map(c =>
          LibraryService.SearchRadioStationsFiltered(c.code, '', 20)
            .then(r => (r || []).map(mapStation))
            .catch(() => [] as Station[])
        )
      );
      const seen = new Set<string>();
      const merged: Station[] = [];
      for (const batch of perCountry) {
        for (const s of batch) {
          if (!seen.has(s.uuid)) {
            seen.add(s.uuid);
            merged.push(s);
          }
        }
      }
      merged.sort((a, b) => b.votes - a.votes);
      stations = await proxyAndFilter(merged, 50);
    } catch {
      stations = [];
    } finally {
      loading = false;
    }
  }

  async function loadByTag(tag: string) {
    loading = true;
    try {
      const result = await LibraryService.GetRadioStationsByTag(tag, 100);
      stations = await proxyAndFilter((result || []).map(mapStation), 50);
    } catch {
      stations = [];
    } finally {
      loading = false;
    }
  }

  async function loadSomaFM() {
    loading = true;
    try {
      const result = await LibraryService.GetSomaFMStations();
      const mapped = (result || []).map(mapStation);
      await proxyStationIcons(mapped.map(s => s.favicon));
      stations = mapped;
    } catch {
      stations = [];
    } finally {
      loading = false;
    }
  }

  async function loadFiltered() {
    loading = true;
    try {
      const result = await LibraryService.SearchRadioStationsFiltered(
        activeCountry, activeCodec, 100
      );
      stations = await proxyAndFilter((result || []).map(mapStation), 50);
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
        pinned: Boolean(f.pinned),
      }));
      favouriteUuids = new Set(favourites.map(f => f.stationUuid));
      await proxyStationIcons(favourites.map(f => f.faviconUrl));
    } catch {
      favourites = [];
      favouriteUuids = new Set();
    }
  }

  async function loadCustomStations() {
    try {
      const result = await LibraryService.GetCustomRadioStations();
      customStations = (result || []).map((s: any) => ({
        stationUuid: s.stationUuid,
        name: s.name,
        streamUrl: s.streamUrl,
        faviconUrl: s.faviconUrl,
        tags: s.tags,
        createdAt: s.createdAt,
      }));
      await proxyStationIcons(customStations.map(s => s.faviconUrl));
    } catch {
      customStations = [];
    }
  }

  async function loadHistory() {
    try {
      const result = await LibraryService.GetRadioHistory(50);
      history = (result || []).map((h: any) => ({
        stationUuid: h.stationUuid,
        name: h.name,
        streamUrl: h.streamUrl,
        faviconUrl: h.faviconUrl,
        tags: h.tags,
        lastTitle: h.lastTitle,
        lastError: h.lastError,
        playCount: h.playCount,
        lastPlayedAt: h.lastPlayedAt,
      }));
      await proxyStationIcons(history.map(h => h.faviconUrl));
    } catch {
      history = [];
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
    activeTag = '';
    activeSource = 'all';
    activeCountry = '';
    activeCodec = '';

    if (debounceTimer) clearTimeout(debounceTimer);

    if (value.trim() === '') {
      loadFeatured();
      return;
    }

    loading = true;
    debounceTimer = setTimeout(async () => {
      try {
        const result = await LibraryService.SearchRadioStations(value.trim(), 100);
        stations = await proxyAndFilter((result || []).map(mapStation), 50);
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

  function clearFilters() {
    activeTag = '';
    activeSource = 'all';
    activeCountry = '';
    activeCodec = '';
    searchQuery = '';
    if (debounceTimer) clearTimeout(debounceTimer);
    loadFeatured();
  }

  function filterByTag(tag: string) {
    searchQuery = '';
    if (debounceTimer) clearTimeout(debounceTimer);
    activeSource = 'all';
    activeCountry = '';
    activeCodec = '';
    activeTag = tag;
    loadByTag(tag);
  }

  function filterBySource(source: 'all' | 'somafm') {
    searchQuery = '';
    if (debounceTimer) clearTimeout(debounceTimer);
    activeTag = '';
    activeCountry = '';
    activeCodec = '';
    activeSource = source;
    if (source === 'somafm') {
      loadSomaFM();
    } else {
      loadFeatured();
    }
  }

  function filterByCountry(country: string) {
    searchQuery = '';
    if (debounceTimer) clearTimeout(debounceTimer);
    activeTag = '';
    activeSource = 'all';
    activeCountry = activeCountry === country ? '' : country;
    if (activeCountry === '' && activeCodec === '') {
      loadFeatured();
    } else {
      loadFiltered();
    }
  }

  function filterByCodec(codec: string) {
    searchQuery = '';
    if (debounceTimer) clearTimeout(debounceTimer);
    activeTag = '';
    activeSource = 'all';
    activeCodec = activeCodec === codec ? '' : codec;
    if (activeCountry === '' && activeCodec === '') {
      loadFeatured();
    } else {
      loadFiltered();
    }
  }

  async function playStation(stationUuid: string, name: string, url: string, favicon: string, tags: string) {
    try {
      if (isPlayingStation(stationUuid)) {
        await PlayerService.Stop();
        await refreshPlaybackStatus();
        await loadHistory();
        return;
      }

      // Proxy artwork so the webview can display it.
      const art = favicon ? await LibraryService.ProxyImageURL(favicon) : '';
      await PlayerService.PlayRadioStation(stationUuid, name, url, art, tags);
      await refreshPlaybackStatus();
      await loadHistory();
    } catch {
      // ignore play errors
    }
  }

  function isPlayingStation(stationUuid: string): boolean {
    return radioMode && stationUuid !== '' && currentStationUuid === stationUuid;
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

  async function togglePinned(fav: Favourite) {
    try {
      await LibraryService.SetRadioFavouritePinned(fav.stationUuid, !fav.pinned);
      await loadFavourites();
    } catch { /* ignore */ }
  }

  async function saveCustomStation() {
    customError = '';
    const name = customName.trim();
    const streamUrl = customStreamUrl.trim();
    if (!name || !streamUrl) return;

    customSaving = true;
    try {
      const saved = await LibraryService.AddCustomRadioStation(
        name,
        streamUrl,
        customFaviconUrl.trim(),
        customTags.trim()
      );
      customName = '';
      customStreamUrl = '';
      customFaviconUrl = '';
      customTags = '';
      await loadCustomStations();
      await playStation(saved.stationUuid, saved.name, saved.streamUrl, saved.faviconUrl, saved.tags);
    } catch (err: any) {
      customError = err?.message || String(err);
    } finally {
      customSaving = false;
    }
  }

  async function deleteCustomStation(uuid: string) {
    try {
      await LibraryService.DeleteCustomRadioStation(uuid);
      await loadCustomStations();
    } catch { /* ignore */ }
  }

  async function clearHistory() {
    try {
      await LibraryService.ClearRadioHistory();
      history = [];
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
    loadCustomStations();
    loadHistory();
  });

  $effect(() => {
    tab = initialTab;
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
    <button class="tab" class:active={tab === 'custom'} onclick={() => tab = 'custom'}>
      Custom ({customStations.length})
    </button>
    <button class="tab" class:active={tab === 'history'} onclick={() => tab = 'history'}>
      History
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
        placeholder="Search stations by name..."
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

    <div class="filter-bar">
      <div class="filter-group">
        <button
          class="filter-pill"
          class:active={activeSource === 'all' && activeTag === '' && activeCountry === '' && activeCodec === ''}
          onclick={() => filterBySource('all')}
        >All</button>
        <button
          class="filter-pill"
          class:active={activeSource === 'somafm'}
          onclick={() => filterBySource('somafm')}
        >SomaFM</button>
      </div>
      <div class="filter-group">
        {#each countries as c}
          <button
            class="filter-pill"
            class:active={activeCountry === c.code}
            onclick={() => filterByCountry(c.code)}
          >{c.label}</button>
        {/each}
      </div>
      <div class="filter-group">
        {#each codecs as codec}
          <button
            class="filter-pill"
            class:active={activeCodec === codec}
            onclick={() => filterByCodec(codec)}
          >{codec}</button>
        {/each}
      </div>
      {#if hasFilter}
        <div class="active-filter">
          {#if activeTag}<span class="filter-label">Tag: {activeTag}</span>{/if}
          {#if activeSource === 'somafm'}<span class="filter-label">Source: SomaFM</span>{/if}
          {#if activeCountry}<span class="filter-label">Country: {countries.find(c => c.code === activeCountry)?.label}</span>{/if}
          {#if activeCodec}<span class="filter-label">Codec: {activeCodec}</span>{/if}
          <button class="filter-clear" onclick={clearFilters} aria-label="Clear filter">
            <svg viewBox="0 0 24 24" width="12" height="12" fill="currentColor">
              <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
            </svg>
          </button>
        </div>
      {/if}
    </div>

    {#if loading}
      <div class="empty">Loading stations...</div>
    {:else if stations.length === 0}
      <div class="empty">
        {#if isSearchActive}
          No stations found for "{searchQuery.trim()}"
        {:else if activeTag}
          No stations found for tag "{activeTag}"
        {:else if activeCountry || activeCodec}
          No stations found for this filter
        {:else}
          No stations available
        {/if}
      </div>
    {:else}
      <div class="station-list">
        {#each stations as station (station.uuid)}
          <div class="station-card" class:playing={isPlayingStation(station.uuid)}>
            <button class="station-play" class:playing={isPlayingStation(station.uuid)} onclick={() => playStation(station.uuid, station.name, station.streamUrl, station.favicon, station.tags)} aria-label={isPlayingStation(station.uuid) ? `Stop ${station.name}` : `Play ${station.name}`}>
              {#if isPlayingStation(station.uuid)}
                <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                  <path d="M6 19h4V5H6zm8-14v14h4V5z"/>
                </svg>
              {:else}
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
              {/if}
            </button>
            {#if resolvedIcon(station.favicon)}
              <img class="station-icon" src={resolvedIcon(station.favicon)} alt="" />
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
                {#if isPlayingStation(station.uuid)}<span class="playing-badge">Playing</span>{/if}
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
                    <button class="tag" class:active={activeTag === tag} onclick={() => filterByTag(tag)}>{tag}</button>
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
  {:else if tab === 'favourites'}
    {#if favourites.length === 0}
      <div class="empty">No favourite stations yet. Browse and add some!</div>
    {:else}
      <div class="station-list">
        {#each favourites as fav (fav.stationUuid)}
          <div class="station-card" class:playing={isPlayingStation(fav.stationUuid)}>
            <button class="station-play" class:playing={isPlayingStation(fav.stationUuid)} onclick={() => playStation(fav.stationUuid, fav.name, fav.streamUrl, fav.faviconUrl, fav.tags)} aria-label={isPlayingStation(fav.stationUuid) ? `Stop ${fav.name}` : `Play ${fav.name}`}>
              {#if isPlayingStation(fav.stationUuid)}
                <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                  <path d="M6 19h4V5H6zm8-14v14h4V5z"/>
                </svg>
              {:else}
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
              {/if}
            </button>
            {#if resolvedIcon(fav.faviconUrl)}
              <img class="station-icon" src={resolvedIcon(fav.faviconUrl)} alt="" />
            {:else}
              <div class="station-icon placeholder">
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M3.24 6.15C2.51 6.43 2 7.17 2 8v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8c0-.83-.49-1.57-1.24-1.85L12 2 3.24 6.15zM12 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3z"/>
                </svg>
              </div>
            {/if}
            <div class="station-info">
              <div class="station-name">{fav.name}</div>
              <div class="station-meta">
                {#if isPlayingStation(fav.stationUuid)}<span class="playing-badge">Playing</span>{/if}
                {#if fav.pinned}<span>Pinned</span>{/if}
              </div>
              {#if formatTags(fav.tags).length > 0}
                <div class="station-tags">
                  {#each formatTags(fav.tags) as tag}
                    <span class="tag">{tag}</span>
                  {/each}
                </div>
              {/if}
            </div>
            <button
              class="pin-btn"
              class:active={fav.pinned}
              onclick={() => togglePinned(fav)}
              aria-label={fav.pinned ? 'Unpin favourite' : 'Pin favourite'}
            >
              <svg viewBox="0 0 24 24" width="17" height="17" fill="currentColor">
                <path d="M16 3l5 5-4 1-4 4v6l-2 2-2-2v-6L5 9 1 8l5-5h10z"/>
              </svg>
            </button>
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
  {:else if tab === 'custom'}
    <div class="custom-form">
      <div class="form-row">
        <input type="text" bind:value={customName} placeholder="Station name" aria-label="Station name" />
        <input type="url" bind:value={customStreamUrl} placeholder="Stream URL" aria-label="Stream URL" />
      </div>
      <div class="form-row">
        <input type="url" bind:value={customFaviconUrl} placeholder="Artwork URL" aria-label="Artwork URL" />
        <input type="text" bind:value={customTags} placeholder="Tags" aria-label="Tags" />
      </div>
      <div class="form-actions">
        <button class="primary-btn" onclick={saveCustomStation} disabled={customSaving || !customName.trim() || !customStreamUrl.trim()}>
          {customSaving ? 'Saving...' : 'Add Station'}
        </button>
        {#if customError}
          <span class="form-error">{customError}</span>
        {/if}
      </div>
    </div>

    {#if customStations.length === 0}
      <div class="empty">No custom stations yet.</div>
    {:else}
      <div class="station-list">
        {#each customStations as station (station.stationUuid)}
          <div class="station-card" class:playing={isPlayingStation(station.stationUuid)}>
            <button class="station-play" class:playing={isPlayingStation(station.stationUuid)} onclick={() => playStation(station.stationUuid, station.name, station.streamUrl, station.faviconUrl, station.tags)} aria-label={isPlayingStation(station.stationUuid) ? `Stop ${station.name}` : `Play ${station.name}`}>
              {#if isPlayingStation(station.stationUuid)}
                <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                  <path d="M6 19h4V5H6zm8-14v14h4V5z"/>
                </svg>
              {:else}
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
              {/if}
            </button>
            {#if resolvedIcon(station.faviconUrl)}
              <img class="station-icon" src={resolvedIcon(station.faviconUrl)} alt="" />
            {:else}
              <div class="station-icon placeholder">
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M3.24 6.15C2.51 6.43 2 7.17 2 8v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8c0-.83-.49-1.57-1.24-1.85L12 2 3.24 6.15zM12 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3z"/>
                </svg>
              </div>
            {/if}
            <div class="station-info">
              <div class="station-name">{station.name}</div>
              {#if isPlayingStation(station.stationUuid)}
                <div class="station-meta"><span class="playing-badge">Playing</span></div>
              {/if}
              {#if formatTags(station.tags).length > 0}
                <div class="station-tags">
                  {#each formatTags(station.tags) as tag}
                    <span class="tag">{tag}</span>
                  {/each}
                </div>
              {/if}
            </div>
            <button class="fav-btn" onclick={() => deleteCustomStation(station.stationUuid)} aria-label="Delete custom station">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}
  {:else if tab === 'history'}
    {#if history.length > 0}
      <div class="history-actions">
        <button class="secondary-btn" onclick={clearHistory}>Clear History</button>
      </div>
    {/if}
    {#if history.length === 0}
      <div class="empty">No radio history yet.</div>
    {:else}
      <div class="station-list">
        {#each history as item (item.stationUuid)}
          <div class="station-card" class:playing={isPlayingStation(item.stationUuid)}>
            <button class="station-play" class:playing={isPlayingStation(item.stationUuid)} onclick={() => playStation(item.stationUuid, item.name, item.streamUrl, item.faviconUrl, item.tags)} aria-label={isPlayingStation(item.stationUuid) ? `Stop ${item.name}` : `Play ${item.name}`}>
              {#if isPlayingStation(item.stationUuid)}
                <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                  <path d="M6 19h4V5H6zm8-14v14h4V5z"/>
                </svg>
              {:else}
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M8 5v14l11-7z"/>
                </svg>
              {/if}
            </button>
            {#if resolvedIcon(item.faviconUrl)}
              <img class="station-icon" src={resolvedIcon(item.faviconUrl)} alt="" />
            {:else}
              <div class="station-icon placeholder">
                <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                  <path d="M3.24 6.15C2.51 6.43 2 7.17 2 8v12a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V8c0-.83-.49-1.57-1.24-1.85L12 2 3.24 6.15zM12 16c-1.66 0-3-1.34-3-3s1.34-3 3-3 3 1.34 3 3-1.34 3-3 3z"/>
                </svg>
              </div>
            {/if}
            <div class="station-info">
              <div class="station-name">{item.name}</div>
              <div class="station-meta">
                {#if isPlayingStation(item.stationUuid)}<span class="playing-badge">Playing</span>{/if}
                <span>Played {item.playCount} {item.playCount === 1 ? 'time' : 'times'}</span>
                {#if item.lastTitle}<span>{item.lastTitle}</span>{/if}
                {#if item.lastError}<span class="station-error">{item.lastError}</span>{/if}
              </div>
              {#if formatTags(item.tags).length > 0}
                <div class="station-tags">
                  {#each formatTags(item.tags) as tag}
                    <span class="tag">{tag}</span>
                  {/each}
                </div>
              {/if}
            </div>
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

  .filter-bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .filter-group {
    display: flex;
    gap: 0.25rem;
  }

  .filter-pill {
    padding: 0.25rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: 12px;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.8rem;
    cursor: pointer;
  }

  .filter-pill:hover {
    color: var(--text-primary);
    border-color: var(--text-secondary);
  }

  .filter-pill.active {
    background: var(--accent);
    color: white;
    border-color: var(--accent);
  }

  .active-filter {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.2rem 0.5rem;
    border-radius: 12px;
    background: var(--bg-hover);
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .filter-label {
    white-space: nowrap;
  }

  .filter-clear {
    display: flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0.1rem;
    border-radius: 50%;
  }

  .filter-clear:hover {
    background: var(--bg-elevated, var(--bg-hover));
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

  .station-card.playing {
    background: var(--bg-active);
    box-shadow: inset 3px 0 0 var(--accent);
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

  .station-play.playing {
    opacity: 1;
    background: var(--bg-main);
    color: var(--accent);
    border: 1px solid var(--accent);
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
    flex-wrap: wrap;
  }

  .station-error {
    color: var(--error);
  }

  .playing-badge {
    color: var(--accent);
    font-weight: 600;
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
    border: none;
    cursor: pointer;
  }

  .tag:hover {
    color: var(--text-primary);
    background: var(--bg-elevated, var(--bg-hover));
  }

  .tag.active {
    background: var(--accent);
    color: white;
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
    color: var(--error);
  }

  .pin-btn {
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

  .pin-btn:hover,
  .pin-btn.active {
    color: var(--accent);
  }

  .custom-form {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    padding: 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
  }

  .form-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .form-row input {
    min-width: 0;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-primary);
    font-size: 0.9rem;
  }

  .form-row input:focus {
    outline: none;
    border-color: var(--accent);
  }

  .form-actions,
  .history-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .primary-btn,
  .secondary-btn {
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

  .primary-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .secondary-btn {
    border: 1px solid var(--border);
    background: transparent;
    color: var(--text-secondary);
  }

  .secondary-btn:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  .form-error {
    color: var(--error);
    font-size: 0.8rem;
  }

  .empty {
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 3rem;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  @media (max-width: 640px) {
    .form-row {
      grid-template-columns: 1fr;
    }
  }
</style>
