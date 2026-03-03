<script lang="ts">
  import { getPreference, setPreference, onPreferenceChange, type ThemePreference } from './lib/theme';

  let preference = $state<ThemePreference>(getPreference());

  $effect(() => {
    return onPreferenceChange((p) => { preference = p; });
  });

  function handleChange(pref: ThemePreference) {
    setPreference(pref);
    preference = pref;
  }

  const options: { value: ThemePreference; label: string; description: string }[] = [
    { value: 'dark', label: 'Dark', description: 'Dark background with light text' },
    { value: 'light', label: 'Light', description: 'Light background with dark text' },
    { value: 'system', label: 'System', description: 'Follow your desktop theme' },
  ];
</script>

<div class="settings">
  <h2>Settings</h2>

  <section class="section">
    <h3>Theme</h3>
    <div class="theme-options">
      {#each options as opt}
        <label class="theme-option" class:selected={preference === opt.value}>
          <input
            type="radio"
            name="theme"
            value={opt.value}
            checked={preference === opt.value}
            onchange={() => handleChange(opt.value)}
          />
          <div class="option-content">
            <span class="option-label">{opt.label}</span>
            <span class="option-desc">{opt.description}</span>
          </div>
        </label>
      {/each}
    </div>
  </section>
</div>

<style>
  .settings {
    max-width: 500px;
  }

  h2 {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 1.5rem;
  }

  h3 {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin: 0 0 0.75rem;
  }

  .theme-options {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .theme-option {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.75rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: transparent;
    cursor: pointer;
  }

  .theme-option:hover {
    background: var(--bg-hover);
  }

  .theme-option.selected {
    border-color: var(--accent);
    background: var(--bg-active);
  }

  .theme-option input {
    accent-color: var(--accent);
  }

  .option-content {
    display: flex;
    flex-direction: column;
  }

  .option-label {
    font-size: 0.9rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .option-desc {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }
</style>
