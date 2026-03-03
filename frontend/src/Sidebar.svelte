<script lang="ts">
  import { getCurrentView, setCurrentView, onViewChange, type View } from './lib/stores';

  let currentView = $state<View>(getCurrentView());

  $effect(() => {
    return onViewChange((v) => { currentView = v; });
  });

  function navigate(view: View) {
    setCurrentView(view);
    currentView = view;
  }

  type NavItem = { view: View; label: string };

  const navItems: NavItem[] = [
    { view: 'library', label: 'Library' },
    { view: 'playlists', label: 'Playlists' },
    { view: 'settings', label: 'Settings' },
  ];
</script>

<nav class="sidebar">
  <div class="brand">Forte</div>
  <ul>
    {#each navItems as item}
      <li>
        <button
          class:active={currentView === item.view}
          onclick={() => navigate(item.view)}
        >
          <span class="icon">
            {#if item.view === 'library'}
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55C7.79 13 6 14.79 6 17s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
              </svg>
            {:else if item.view === 'playlists'}
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M15 6H3v2h12V6zm0 4H3v2h12v-2zM3 16h8v-2H3v2zM17 6v8.18c-.31-.11-.65-.18-1-.18-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3V8h3V6h-5z"/>
              </svg>
            {:else if item.view === 'settings'}
              <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
                <path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.484.484 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96c-.22-.08-.47 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.07.62-.07.94s.02.64.07.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2z"/>
              </svg>
            {/if}
          </span>
          {item.label}
        </button>
      </li>
    {/each}
  </ul>
</nav>

<style>
  .sidebar {
    width: 200px;
    background: var(--bg-sidebar);
    border-right: 1px solid var(--border);
    display: flex;
    flex-direction: column;
    overflow-y: auto;
  }

  .brand {
    font-size: 1.25rem;
    font-weight: 700;
    padding: 1.25rem 1rem;
    color: var(--text-primary);
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  li button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.6rem 1rem;
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.9rem;
    cursor: pointer;
    text-align: left;
  }

  li button:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  li button.active {
    background: var(--bg-active);
    color: var(--accent);
  }

  .icon {
    width: 1.5rem;
    display: flex;
    align-items: center;
    justify-content: center;
  }
</style>
