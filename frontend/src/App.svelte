<script lang="ts">
  import Sidebar from './Sidebar.svelte';
  import TitleBar from './TitleBar.svelte';
  import Content from './Content.svelte';
  import NowPlayingBar from './NowPlayingBar.svelte';
  import NowPlayingView from './NowPlayingView.svelte';
  import ShortcutHelp from './ShortcutHelp.svelte';
  import QueuePanel from './QueuePanel.svelte';
  import Toast from './Toast.svelte';
  import { LibraryService } from "../bindings/github.com/willfish/forte";
  import { handleKeydown } from './lib/shortcuts';
  import { isTitlebarEnabled, onTitlebarEnabledChange, setTitlebarEnabled } from './lib/stores';
  import { initTheme } from './lib/theme';

  initTheme();

  let showHelp = $state(false);
  let showQueue = $state(false);
  let showNowPlaying = $state(false);
  let showTitlebar = $state(isTitlebarEnabled());

  $effect(() => {
    return onTitlebarEnabledChange((enabled) => { showTitlebar = enabled; });
  });

  $effect(() => {
    async function loadTitlebarPreference() {
      try {
        const prefs = await LibraryService.GetAppPreferences();
        setTitlebarEnabled(Boolean(prefs?.showTitlebar));
      } catch {
        setTitlebarEnabled(false);
      }
    }

    loadTitlebarPreference();
  });

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
  {#if showTitlebar}
    <TitleBar />
  {/if}
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
