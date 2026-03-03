<script lang="ts">
  import { getCurrentView, onViewChange, type View } from './lib/stores';
  import AlbumGrid from './AlbumGrid.svelte';

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
      <div class="placeholder">
        <h2>Album tracks</h2>
        <p>Track list view coming soon.</p>
        <button class="back-btn" onclick={() => selectedAlbumId = null}>Back to albums</button>
      </div>
    {:else}
      <AlbumGrid onselect={handleAlbumSelect} />
    {/if}
  {:else if currentView === 'playlists'}
    <div class="placeholder">
      <h2>Playlists</h2>
      <p>Your playlists will appear here.</p>
    </div>
  {:else if currentView === 'settings'}
    <div class="placeholder">
      <h2>Settings</h2>
      <p>Application settings will appear here.</p>
    </div>
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

  .back-btn {
    margin-top: 1rem;
    padding: 0.4rem 1rem;
    border: 1px solid var(--border);
    border-radius: 4px;
    background: transparent;
    color: var(--text-primary);
    cursor: pointer;
    font-size: 0.85rem;
  }

  .back-btn:hover {
    background: var(--bg-hover);
  }
</style>
