<script lang="ts">
  import {GreetService} from "../bindings/github.com/willfish/forte";

  let name = $state('');
  let result = $state('Enter your name to test the Go bridge');

  async function greet() {
    const greeting = await GreetService.Greet(name || 'anonymous');
    result = greeting;
  }
</script>

<main>
  <h1>Forte</h1>
  <p class="subtitle">A modern music player</p>
  <div class="bridge-test">
    <p class="result">{result}</p>
    <div class="input-row">
      <input
        type="text"
        bind:value={name}
        placeholder="Your name"
        onkeydown={(e) => e.key === 'Enter' && greet()}
      />
      <button onclick={greet}>Greet</button>
    </div>
  </div>
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

  .bridge-test {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 12px;
    padding: 2rem;
    max-width: 400px;
    margin: 0 auto;
  }

  .result {
    margin: 0 0 1.5rem;
    min-height: 1.5em;
  }

  .input-row {
    display: flex;
    gap: 0.5rem;
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
