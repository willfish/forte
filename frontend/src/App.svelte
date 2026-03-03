<script lang="ts">
  import Sidebar from './Sidebar.svelte';
  import Content from './Content.svelte';
  import NowPlayingBar from './NowPlayingBar.svelte';
</script>

<div class="shell">
  <div class="top">
    <Sidebar />
    <Content />
  </div>
  <NowPlayingBar />
</div>

<style>
  :global(:root) {
    --bg-main: #1b2636;
    --bg-sidebar: #151e2b;
    --bg-bar: #151e2b;
    --bg-hover: rgba(255, 255, 255, 0.06);
    --bg-active: rgba(51, 102, 170, 0.15);
    --border: #2a3a4a;
    --text-primary: #e0e0e0;
    --text-secondary: #8899aa;
    --accent: #3366aa;
  }

  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: var(--bg-main);
    color: var(--text-primary);
    overflow: hidden;
  }

  .shell {
    display: flex;
    flex-direction: column;
    height: 100vh;
  }

  .top {
    display: flex;
    flex: 1;
    overflow: hidden;
  }
</style>
