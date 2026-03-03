<script lang="ts">
  import { shortcuts, formatKey } from './lib/shortcuts';

  let { onclose }: { onclose: () => void } = $props();
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="overlay" onclick={onclose} role="presentation">
  <!-- svelte-ignore a11y_click_events_have_key_events a11y_no_static_element_interactions -->
  <div class="panel" onclick={(e) => e.stopPropagation()} role="dialog" aria-label="Keyboard shortcuts" tabindex="-1">
    <div class="header">
      <h2>Keyboard Shortcuts</h2>
      <button class="close-btn" onclick={onclose} aria-label="Close">x</button>
    </div>
    <table>
      <tbody>
        {#each shortcuts as shortcut}
          <tr>
            <td class="key"><kbd>{formatKey(shortcut)}</kbd></td>
            <td class="desc">{shortcut.description}</td>
          </tr>
        {/each}
        <tr>
          <td class="key"><kbd>Ctrl + ?</kbd></td>
          <td class="desc">Show this help</td>
        </tr>
      </tbody>
    </table>
  </div>
</div>

<style>
  .overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .panel {
    background: var(--bg-sidebar);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 1.5rem;
    min-width: 320px;
    max-width: 420px;
  }

  .header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  h2 {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .close-btn {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    font-size: 1rem;
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 4px;
  }

  .close-btn:hover {
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  table {
    width: 100%;
    border-collapse: collapse;
  }

  tr {
    border-bottom: 1px solid var(--border);
  }

  tr:last-child {
    border-bottom: none;
  }

  td {
    padding: 0.5rem 0;
  }

  .key {
    width: 120px;
  }

  kbd {
    background: var(--bg-hover);
    border: 1px solid var(--border);
    border-radius: 4px;
    padding: 2px 6px;
    font-family: inherit;
    font-size: 0.8rem;
    color: var(--text-primary);
  }

  .desc {
    color: var(--text-secondary);
    font-size: 0.85rem;
  }
</style>
