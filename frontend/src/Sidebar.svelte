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
    { view: 'radio', label: 'Radio', icon: '\u25CE' },
    { view: 'stats', label: 'Stats', icon: '\u2584' },
    { view: 'settings', label: 'Settings', icon: '\u2699' },
  ];
</script>

<nav class="sidebar">
  <div class="brand">Forte</div>
  <ul>
    {#each navItems as item}
      <li>
        <button class="nav-btn" class:active={currentView === item.view} onclick={() => navigate(item.view)}>
          <span class="icon">{item.icon}</span>
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
    padding: 1.25rem 1rem;
    font-size: 1.1rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    color: var(--text-primary);
    border-bottom: 1px solid var(--border);
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

  .icon {
    font-size: 1.1rem;
    width: 1.25rem;
    text-align: center;
    flex-shrink: 0;
  }

  .label {
    overflow: hidden;
    white-space: nowrap;
  }

  @media (max-width: 900px) {
    .brand {
      text-align: center;
      padding: 1.25rem 0.25rem;
      font-size: 0;
    }

    .brand::after {
      content: 'F';
      font-size: 1.1rem;
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
