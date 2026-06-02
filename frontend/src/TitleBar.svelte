<script lang="ts">
  import { Window } from '@wailsio/runtime';
  import ForteMark from './ForteMark.svelte';
  import Icon from './lib/Icon.svelte';

  function minimise() {
    void Window.Minimise();
  }

  function toggleMaximise() {
    void Window.ToggleMaximise();
  }

  function close() {
    void Window.Close();
  }
</script>

<header class="titlebar" aria-label="Forte window controls">
  <div class="titlebar-brand">
    <ForteMark size={18} />
    <span>Forte</span>
  </div>
  <div class="window-controls">
    <button type="button" class="window-btn" onclick={minimise} aria-label="Minimise">
      <Icon name="minimize" size={14} />
    </button>
    <button type="button" class="window-btn" onclick={toggleMaximise} aria-label="Maximise">
      <Icon name="maximize" size={13} />
    </button>
    <button type="button" class="window-btn close" onclick={close} aria-label="Close">
      <Icon name="close" size={14} />
    </button>
  </div>
</header>

<style>
  .titlebar {
    height: 34px;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: color-mix(in srgb, var(--bg-sidebar) var(--theme-surface-opacity), transparent);
    border-bottom: 1px solid var(--border);
    color: var(--text-secondary);
    --wails-draggable: drag;
  }

  .titlebar-brand {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    min-width: 0;
    padding-left: 0.75rem;
    font-size: var(--text-sm);
    font-weight: 650;
    color: var(--text-primary);
  }

  .window-controls {
    display: flex;
    align-self: stretch;
    --wails-draggable: no-drag;
  }

  .window-btn {
    width: 42px;
    height: 100%;
    display: grid;
    place-items: center;
    border: none;
    border-radius: 0;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    transition: background 0.15s ease, color 0.15s ease;
  }

  .window-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .window-btn.close:hover {
    background: var(--error);
    color: var(--text-on-accent);
  }
</style>
