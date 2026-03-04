<script lang="ts">
  import { getCurrentView, onViewChange, setServerStatuses, type View } from './lib/stores';
  import { LibraryService } from "../bindings/github.com/willfish/forte";
  import AlbumGrid from './AlbumGrid.svelte';
  import AlbumView from './AlbumView.svelte';
  import PlaylistView from './PlaylistView.svelte';
  import StatsView from './StatsView.svelte';
  import Settings from './Settings.svelte';
  import SearchResults from './SearchResults.svelte';

  let currentView = $state<View>(getCurrentView());
  let selectedAlbumId = $state<number | null>(null);

  // Search state.
  let searchQuery = $state('');
  let searchResults = $state<any[]>([]);
  let searching = $state(false);
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let searchInputRef: HTMLInputElement | undefined = $state();

  const isSearchActive = $derived(searchQuery.trim().length > 0);

  $effect(() => {
    return onViewChange((v) => {
      currentView = v;
      selectedAlbumId = null;
      clearSearch();
    });
  });

  // Poll server statuses every 5 seconds.
  $effect(() => {
    async function poll() {
      try {
        const statuses = await LibraryService.GetServerStatuses();
        if (statuses) {
          const map: Record<string, boolean> = {};
          for (const s of statuses) {
            map[s.serverId] = s.online;
          }
          setServerStatuses(map);
        }
      } catch {
        // ignore polling errors
      }
    }
    poll();
    const timer = setInterval(poll, 5000);
    return () => clearInterval(timer);
  });

  function handleAlbumSelect(albumId: number) {
    selectedAlbumId = albumId;
  }

  function handleSearchInput(e: Event) {
    const value = (e.target as HTMLInputElement).value;
    searchQuery = value;

    if (debounceTimer) clearTimeout(debounceTimer);

    if (value.trim() === '') {
      searchResults = [];
      searching = false;
      return;
    }

    searching = true;
    debounceTimer = setTimeout(async () => {
      try {
        const results = await LibraryService.Search(value.trim(), 100);
        searchResults = (results || []).map((r: any) => ({
          trackId: r.trackId,
          title: r.title,
          artist: r.artist,
          album: r.album,
          genre: r.genre,
          durationMs: r.durationMs,
          filePath: r.filePath,
          source: r.source || 'local',
          serverId: r.serverId || '',
        }));
      } catch {
        searchResults = [];
      } finally {
        searching = false;
      }
    }, 300);
  }

  function clearSearch() {
    searchQuery = '';
    searchResults = [];
    searching = false;
    if (debounceTimer) clearTimeout(debounceTimer);
  }

  function handleSearchKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      clearSearch();
      searchInputRef?.blur();
    }
  }

  function handleGlobalKeydown(e: KeyboardEvent) {
    // Ctrl+F / Cmd+F focuses the search bar when in library view.
    if ((e.ctrlKey || e.metaKey) && e.key === 'f' && currentView === 'library') {
      e.preventDefault();
      searchInputRef?.focus();
    }
  }
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<main class="content">
  {#if currentView === 'library'}
    <div class="search-bar">
      <svg class="search-icon" viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
        <path d="M15.5 14h-.79l-.28-.27A6.47 6.47 0 0 0 16 9.5 6.5 6.5 0 1 0 9.5 16c1.61 0 3.09-.59 4.23-1.57l.27.28v.79l5 4.99L20.49 19l-4.99-5zm-6 0C7.01 14 5 11.99 5 9.5S7.01 5 9.5 5 14 7.01 14 9.5 11.99 14 9.5 14z"/>
      </svg>
      <input
        bind:this={searchInputRef}
        type="text"
        class="search-input"
        placeholder="Search tracks..."
        value={searchQuery}
        oninput={handleSearchInput}
        onkeydown={handleSearchKeydown}
      />
      {#if isSearchActive}
        <button class="search-clear" onclick={clearSearch} aria-label="Clear search">
          <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
            <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
          </svg>
        </button>
      {/if}
    </div>

    {#if isSearchActive}
      {#if searching}
        <div class="searching">Searching...</div>
      {:else}
        <SearchResults results={searchResults} query={searchQuery.trim()} />
      {/if}
    {:else if selectedAlbumId !== null}
      <AlbumView albumId={selectedAlbumId} onback={() => selectedAlbumId = null} />
    {:else}
      <AlbumGrid onselect={handleAlbumSelect} />
    {/if}
  {:else if currentView === 'playlists'}
    <PlaylistView />
  {:else if currentView === 'stats'}
    <StatsView />
  {:else if currentView === 'settings'}
    <Settings />
  {/if}
</main>

<style>
  .content {
    flex: 1;
    overflow-y: auto;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
  }

  .search-bar {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--bg-secondary, var(--bg-hover));
    margin-bottom: 1rem;
    flex-shrink: 0;
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

  .searching {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }
</style>
