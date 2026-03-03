<script lang="ts">
  import Sidebar from './Sidebar.svelte';
  import Content from './Content.svelte';
  import NowPlayingBar from './NowPlayingBar.svelte';
  import { initTheme } from './lib/theme';

  initTheme();
</script>

<div class="shell">
  <div class="top">
    <div class="sidebar-wrap">
      <Sidebar />
    </div>
    <Content />
  </div>
  <NowPlayingBar />
</div>
