<script lang="ts">
  import { getCurrentView, onViewChange, type View } from './lib/stores';
  import AlbumGrid from './AlbumGrid.svelte';
  import AlbumView from './AlbumView.svelte';
  import Settings from './Settings.svelte';

  let currentView = $state<View>(getCurrentView());
  let selectedAlbumId = $state<number | null>(null);

  $effect(() => {
    return onViewChange((v) => {
      currentView = v;
      selectedAlbumId = null;
    });
  });

  function handleAlbumSelect(albumId: number) {
    selectedAlbumId = albumId;
  }
</script>

<main class="content">
  {#if currentView === 'library'}
    {#if selectedAlbumId !== null}
      <AlbumView albumId={selectedAlbumId} onback={() => selectedAlbumId = null} />
    {:else}
      <AlbumGrid onselect={handleAlbumSelect} />
    {/if}
  {:else if currentView === 'playlists'}
    <div class="placeholder">
      <h2>Playlists</h2>
      <p>Your playlists will appear here.</p>
    </div>
  {:else if currentView === 'settings'}
    <Settings />
  {/if}
</main>

<style>
  .content {
    flex: 1;
    overflow-y: auto;
    padding: 1.5rem;
  }

  .placeholder {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
  }

  .placeholder h2 {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 0.5rem;
  }

  .placeholder p {
    margin: 0;
  }
</style>
