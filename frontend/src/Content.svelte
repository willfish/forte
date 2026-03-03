<script lang="ts">
  import { getCurrentView, onViewChange, type View } from './lib/stores';
  import AlbumGrid from './AlbumGrid.svelte';
  import AlbumView from './AlbumView.svelte';
  import PlaylistView from './PlaylistView.svelte';
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
    <PlaylistView />
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

</style>
