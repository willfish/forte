<script lang="ts">
  import Sidebar from './Sidebar.svelte';
  import Content from './Content.svelte';
  import NowPlayingBar from './NowPlayingBar.svelte';
  import ShortcutHelp from './ShortcutHelp.svelte';
  import { handleKeydown } from './lib/shortcuts';
  import { initTheme } from './lib/theme';

  initTheme();

  let showHelp = $state(false);

  function onKeydown(e: KeyboardEvent) {
    if (e.key === '?' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      showHelp = !showHelp;
      return;
    }
    if (e.key === 'Escape' && showHelp) {
      showHelp = false;
      return;
    }
    handleKeydown(e);
  }
</script>

<svelte:window onkeydown={onKeydown} />

<div class="shell">
  <div class="top">
    <div class="sidebar-wrap">
      <Sidebar />
    </div>
    <Content />
  </div>
  <NowPlayingBar />
</div>

{#if showHelp}
  <ShortcutHelp onclose={() => showHelp = false} />
{/if}
