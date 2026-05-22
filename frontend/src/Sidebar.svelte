<script lang="ts">
  import {
    getCurrentView,
    isLibraryEnabled,
    onLibraryEnabledChange,
    onViewChange,
    setCurrentView,
    type View
  } from './lib/stores';
  import ForteMark from './ForteMark.svelte';
  import Icon from './lib/Icon.svelte';
  import type { IconName } from './lib/icons';

  let currentView = $state<View>(getCurrentView());
  let libraryEnabled = $state(isLibraryEnabled());

  $effect(() => {
    return onViewChange((v) => { currentView = v; });
  });

  $effect(() => {
    return onLibraryEnabledChange((enabled) => { libraryEnabled = enabled; });
  });

  function navigate(view: View) {
    setCurrentView(view);
    currentView = view;
  }

  const navItems: { view: View; label: string; icon: IconName }[] = [
    { view: 'radio', label: 'Radio', icon: 'radio' },
    { view: 'library', label: 'Library', icon: 'library' },
    { view: 'playlists', label: 'Playlists', icon: 'playlists' },
    { view: 'stats', label: 'Stats', icon: 'stats' },
    { view: 'settings', label: 'Settings', icon: 'settings' },
  ];

  const visibleItems = $derived(navItems.filter(item =>
    libraryEnabled || !['library', 'playlists', 'stats'].includes(item.view)
  ));
</script>

<nav class="sidebar">
  <div class="brand">
    <ForteMark class="brand-mark" size={24} />
    <span class="brand-text">Forte</span>
  </div>
  <ul>
    {#each visibleItems as item}
      <li>
        <button class="nav-btn" class:active={currentView === item.view} onclick={() => navigate(item.view)}>
          <Icon class="icon" name={item.icon} size={16} />
          <span class="label">{item.label}</span>
        </button>
      </li>
    {/each}
  </ul>
</nav>

<style>
  .sidebar {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 0;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 0.625rem;
    padding: 1.25rem 1rem;
    font-size: 1.1rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    color: var(--text-primary);
    border-bottom: 1px solid var(--border);
  }

  :global(.brand-mark) {
    flex-shrink: 0;
  }

  ul {
    list-style: none;
    margin: 0;
    padding: 0.5rem 0;
  }

  li {
    padding: 0 0.5rem;
  }

  .nav-btn {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    width: 100%;
    padding: 0.5rem 0.75rem;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.9rem;
    cursor: pointer;
    text-align: left;
    transition: background 0.15s ease, color 0.15s ease;
  }

  .nav-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .nav-btn.active {
    background: var(--bg-active);
    color: var(--accent);
  }


  .label {
    overflow: hidden;
    white-space: nowrap;
  }

  @media (max-width: 900px) {
    .brand {
      justify-content: center;
      padding: 1.25rem 0.25rem;
    }

    .brand-text {
      display: none;
    }

    li {
      padding: 0 0.25rem;
    }

    .nav-btn {
      justify-content: center;
      padding: 0.5rem;
      gap: 0;
    }

    .label {
      display: none;
    }
  }
</style>
