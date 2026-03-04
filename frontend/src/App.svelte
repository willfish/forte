<script lang="ts">
  import Sidebar from './Sidebar.svelte';
  import Content from './Content.svelte';
  import NowPlayingBar from './NowPlayingBar.svelte';
  import NowPlayingView from './NowPlayingView.svelte';
  import ShortcutHelp from './ShortcutHelp.svelte';
  import QueuePanel from './QueuePanel.svelte';
  import Toast from './Toast.svelte';
  import { handleKeydown } from './lib/shortcuts';
  import { initTheme } from './lib/theme';

  initTheme();

  let showHelp = $state(false);
  let showQueue = $state(false);
  let showNowPlaying = $state(false);

  function onKeydown(e: KeyboardEvent) {
    if (e.key === '?' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      showHelp = !showHelp;
      return;
    }
    if (e.key === 'Escape') {
      if (showHelp) { showHelp = false; return; }
      if (showQueue) { showQueue = false; return; }
      if (showNowPlaying) { showNowPlaying = false; return; }
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
    <div class="content-area">
      <Content />
      {#if showNowPlaying}
        <NowPlayingView onclose={() => showNowPlaying = false} />
      {/if}
    </div>
  </div>
  <NowPlayingBar onqueuetoggle={() => showQueue = !showQueue} onexpand={() => showNowPlaying = true} />
</div>

<QueuePanel open={showQueue} onclose={() => showQueue = false} />

{#if showHelp}
  <ShortcutHelp onclose={() => showHelp = false} />
{/if}

<Toast />
