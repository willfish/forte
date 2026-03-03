<script lang="ts">
  import { getCurrentView, onViewChange, type View } from './lib/stores';

  let currentView = $state<View>(getCurrentView());

  $effect(() => {
    return onViewChange((v) => { currentView = v; });
  });
</script>

<main class="content">
  {#if currentView === 'library'}
    <div class="placeholder">
      <h2>Library</h2>
      <p>Your music collection will appear here.</p>
    </div>
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
</style>
