<script lang="ts">
  import NowPlaying from './NowPlaying.svelte';
  import { PlayerService } from "../bindings/github.com/willfish/forte";

  let filePath = $state('');

  async function loadFile() {
    if (!filePath.trim()) return;
    await PlayerService.Play(filePath.trim());
  }
</script>

<main>
  <h1>Forte</h1>
  <p class="subtitle">A modern music player</p>

  <div class="file-input">
    <input
      type="text"
      bind:value={filePath}
      placeholder="Path to audio file"
      onkeydown={(e) => e.key === 'Enter' && loadFile()}
    />
    <button onclick={loadFile}>Load</button>
  </div>

  <NowPlaying />
</main>

<style>
  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    background: #1b2636;
    color: #e0e0e0;
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
  }

  main {
    text-align: center;
    padding: 2rem;
    width: 100%;
    max-width: 600px;
  }

  h1 {
    font-size: 3rem;
    font-weight: 700;
    margin: 0;
    color: #fff;
  }

  .subtitle {
    color: #8899aa;
    margin: 0.5rem 0 2rem;
  }

  .file-input {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
  }

  input {
    flex: 1;
    padding: 0.6rem 1rem;
    border-radius: 8px;
    border: 1px solid #334;
    background: #0d1520;
    color: #e0e0e0;
    font-size: 1rem;
  }

  input:focus {
    outline: none;
    border-color: #5588cc;
  }

  button {
    padding: 0.6rem 1.5rem;
    border-radius: 8px;
    border: none;
    background: #3366aa;
    color: #fff;
    font-size: 1rem;
    cursor: pointer;
  }

  button:hover {
    background: #4477bb;
  }
</style>
