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

  const navItems: { view: View; label: string; icon: string }[] = [
    { view: 'library', label: 'Library', icon: '\u266B' },
    { view: 'playlists', label: 'Playlists', icon: '\u2630' },
    { view: 'settings', label: 'Settings', icon: '\u2699' },
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
          <span class="icon">{item.icon}</span>
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
    font-size: 1.1rem;
    width: 1.5rem;
    text-align: center;
  }
</style>
