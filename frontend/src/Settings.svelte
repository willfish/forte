<script lang="ts">
  import { getPreference, setPreference, onPreferenceChange, type ThemePreference } from './lib/theme';
  import { LibraryService } from "../bindings/github.com/willfish/forte";

  // Theme state
  let preference = $state<ThemePreference>(getPreference());

  $effect(() => {
    return onPreferenceChange((p) => { preference = p; });
  });

  function handleChange(pref: ThemePreference) {
    setPreference(pref);
    preference = pref;
  }

  const themeOptions: { value: ThemePreference; label: string; description: string }[] = [
    { value: 'dark', label: 'Dark', description: 'Dark background with light text' },
    { value: 'light', label: 'Light', description: 'Light background with dark text' },
    { value: 'system', label: 'System', description: 'Follow your desktop theme' },
  ];

  // Server state
  type ServerConfig = {
    id: string;
    name: string;
    type: string;
    url: string;
    username: string;
    password: string;
  };

  let servers = $state<ServerConfig[]>([]);
  let editing = $state<ServerConfig | null>(null);
  let testing = $state(false);
  let testResult = $state<{ ok: boolean; message: string } | null>(null);
  let showPassword = $state(false);
  let syncing = $state(false);
  let syncResult = $state<{ ok: boolean; message: string } | null>(null);

  async function loadServers() {
    servers = ((await LibraryService.GetServers()) || []).map((s: any) => ({
      id: s.id,
      name: s.name,
      type: s.type,
      url: s.url,
      username: s.username,
      password: s.password,
    }));
  }

  $effect(() => {
    loadServers();
  });

  function startAdd() {
    editing = { id: '', name: '', type: 'subsonic', url: '', username: '', password: '' };
    testResult = null;
    showPassword = false;
  }

  function startEdit(srv: ServerConfig) {
    editing = { ...srv };
    testResult = null;
    showPassword = false;
  }

  function cancelEdit() {
    editing = null;
    testResult = null;
    showPassword = false;
  }

  async function testConnection() {
    if (!editing) return;
    testing = true;
    testResult = null;
    try {
      await LibraryService.TestConnection(editing);
      testResult = { ok: true, message: 'Connection successful' };
    } catch (err: any) {
      testResult = { ok: false, message: err?.message || String(err) };
    } finally {
      testing = false;
    }
  }

  async function saveServer() {
    if (!editing) return;
    try {
      if (editing.id) {
        await LibraryService.UpdateServer(editing);
      } else {
        await LibraryService.AddServer(editing);
      }
      editing = null;
      testResult = null;
      showPassword = false;
      await loadServers();
    } catch (err: any) {
      testResult = { ok: false, message: err?.message || String(err) };
    }
  }

  async function deleteServer(id: string) {
    await LibraryService.DeleteServer(id);
    await loadServers();
  }

  async function syncServers() {
    syncing = true;
    syncResult = null;
    try {
      await LibraryService.SyncServers();
      syncResult = { ok: true, message: 'Sync completed' };
    } catch (err: any) {
      syncResult = { ok: false, message: err?.message || String(err) };
    } finally {
      syncing = false;
    }
  }

  function canSave(): boolean {
    if (!editing) return false;
    return editing.name.trim() !== '' && editing.url.trim() !== '' && editing.username.trim() !== '';
  }

  // Last.fm scrobble state
  type ScrobbleConfig = {
    apiKey: string;
    sessionKey: string;
    username: string;
    enabled: boolean;
  };

  let scrobbleConfig = $state<ScrobbleConfig | null>(null);
  let lfmApiKey = $state('');
  let lfmApiSecret = $state('');
  let lfmAuthToken = $state('');
  let lfmConnecting = $state(false);
  let lfmResult = $state<{ ok: boolean; message: string } | null>(null);

  async function loadScrobbleConfig() {
    try {
      const cfg = await LibraryService.GetScrobbleConfig();
      scrobbleConfig = cfg;
      lfmApiKey = cfg.apiKey || '';
    } catch {
      scrobbleConfig = null;
    }
  }

  $effect(() => {
    loadScrobbleConfig();
  });

  async function saveApiKeys() {
    lfmResult = null;
    try {
      await LibraryService.SaveScrobbleAPIKeys(lfmApiKey, lfmApiSecret);
      lfmApiSecret = '';
      lfmResult = { ok: true, message: 'API keys saved' };
      await loadScrobbleConfig();
    } catch (err: any) {
      lfmResult = { ok: false, message: err?.message || String(err) };
    }
  }

  async function startLastFmAuth() {
    lfmConnecting = true;
    lfmResult = null;
    try {
      const token = await LibraryService.StartLastFmAuth();
      lfmAuthToken = token;
      lfmResult = { ok: true, message: 'Browser opened - approve the request, then click "Complete authentication"' };
    } catch (err: any) {
      lfmResult = { ok: false, message: err?.message || String(err) };
    } finally {
      lfmConnecting = false;
    }
  }

  async function completeLastFmAuth() {
    lfmConnecting = true;
    lfmResult = null;
    try {
      await LibraryService.CompleteLastFmAuth(lfmAuthToken);
      lfmAuthToken = '';
      lfmResult = { ok: true, message: 'Connected to Last.fm' };
      await loadScrobbleConfig();
    } catch (err: any) {
      lfmResult = { ok: false, message: err?.message || String(err) };
    } finally {
      lfmConnecting = false;
    }
  }

  async function disconnectLastFm() {
    lfmResult = null;
    try {
      await LibraryService.DisconnectLastFm();
      lfmAuthToken = '';
      await loadScrobbleConfig();
    } catch (err: any) {
      lfmResult = { ok: false, message: err?.message || String(err) };
    }
  }

  async function toggleScrobbleEnabled() {
    if (!scrobbleConfig) return;
    try {
      await LibraryService.SetScrobbleEnabled(!scrobbleConfig.enabled);
      await loadScrobbleConfig();
    } catch (err: any) {
      lfmResult = { ok: false, message: err?.message || String(err) };
    }
  }

  // ListenBrainz state
  type LBConfig = {
    username: string;
    enabled: boolean;
  };

  let lbConfig = $state<LBConfig | null>(null);
  let lbToken = $state('');
  let lbConnecting = $state(false);
  let lbResult = $state<{ ok: boolean; message: string } | null>(null);

  async function loadLBConfig() {
    try {
      const cfg = await LibraryService.GetListenBrainzConfig();
      lbConfig = cfg;
    } catch {
      lbConfig = null;
    }
  }

  $effect(() => {
    loadLBConfig();
  });

  async function connectListenBrainz() {
    if (!lbToken.trim()) return;
    lbConnecting = true;
    lbResult = null;
    try {
      await LibraryService.ConnectListenBrainz(lbToken);
      lbToken = '';
      lbResult = { ok: true, message: 'Connected to ListenBrainz' };
      await loadLBConfig();
    } catch (err: any) {
      lbResult = { ok: false, message: err?.message || String(err) };
    } finally {
      lbConnecting = false;
    }
  }

  async function disconnectListenBrainz() {
    lbResult = null;
    try {
      await LibraryService.DisconnectListenBrainz();
      await loadLBConfig();
    } catch (err: any) {
      lbResult = { ok: false, message: err?.message || String(err) };
    }
  }

  async function toggleLBEnabled() {
    if (!lbConfig) return;
    try {
      await LibraryService.SetListenBrainzEnabled(!lbConfig.enabled);
      await loadLBConfig();
    } catch (err: any) {
      lbResult = { ok: false, message: err?.message || String(err) };
    }
  }

  // Scrobble queue state
  let queueSize = $state(0);

  async function loadQueueSize() {
    try {
      queueSize = await LibraryService.GetScrobbleQueueSize();
    } catch {
      queueSize = 0;
    }
  }

  $effect(() => {
    loadQueueSize();
  });
</script>

<div class="settings">
  <h2>Settings</h2>

  <section class="section">
    <h3>Theme</h3>
    <div class="theme-options">
      {#each themeOptions as opt}
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

  <section class="section servers-section">
    <h3>Servers</h3>

    {#if editing}
      <div class="server-form">
        <div class="form-field">
          <label for="srv-name">Name</label>
          <input id="srv-name" type="text" bind:value={editing.name} placeholder="My server" />
        </div>

        <div class="form-field">
          <!-- svelte-ignore a11y_label_has_associated_control -->
          <label>Type</label>
          <div class="type-radios">
            <label class="type-option" class:selected={editing.type === 'subsonic'}>
              <input type="radio" name="server-type" value="subsonic" bind:group={editing.type} />
              Subsonic
            </label>
            <label class="type-option" class:selected={editing.type === 'jellyfin'}>
              <input type="radio" name="server-type" value="jellyfin" bind:group={editing.type} />
              Jellyfin
            </label>
          </div>
        </div>

        <div class="form-field">
          <label for="srv-url">URL</label>
          <input id="srv-url" type="text" bind:value={editing.url} placeholder="https://music.example.com" />
        </div>

        <div class="form-field">
          <label for="srv-user">Username</label>
          <input id="srv-user" type="text" bind:value={editing.username} />
        </div>

        <div class="form-field">
          <label for="srv-pass">Password</label>
          <div class="password-field">
            {#if showPassword}
              <input id="srv-pass" type="text" bind:value={editing.password} />
            {:else}
              <input id="srv-pass" type="password" bind:value={editing.password} />
            {/if}
            <button class="toggle-pw" type="button" onclick={() => { showPassword = !showPassword; }}>
              {showPassword ? 'Hide' : 'Show'}
            </button>
          </div>
        </div>

        {#if testResult}
          <div class="test-result" class:ok={testResult.ok} class:err={!testResult.ok}>
            {testResult.message}
          </div>
        {/if}

        <div class="form-actions">
          <button class="btn-test" onclick={testConnection} disabled={testing || !canSave()}>
            {testing ? 'Testing...' : 'Test Connection'}
          </button>
          <div class="form-actions-right">
            <button class="btn-cancel" onclick={cancelEdit}>Cancel</button>
            <button class="btn-save" onclick={saveServer} disabled={!canSave()}>Save</button>
          </div>
        </div>
      </div>
    {:else}
      {#if servers.length === 0}
        <p class="empty-msg">No servers configured.</p>
      {:else}
        <ul class="server-list">
          {#each servers as srv (srv.id)}
            <li class="server-item">
              <button class="server-btn" onclick={() => startEdit(srv)}>
                <span class="server-name">{srv.name}</span>
                <span class="server-type-badge">{srv.type}</span>
                <span class="server-url">{srv.url}</span>
              </button>
              <div class="server-actions">
                <button class="action-btn" onclick={() => startEdit(srv)} aria-label="Edit">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                    <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04a.996.996 0 0 0 0-1.41l-2.34-2.34a.996.996 0 0 0-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
                  </svg>
                </button>
                <button class="action-btn delete" onclick={() => deleteServer(srv.id)} aria-label="Delete">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                    <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                  </svg>
                </button>
              </div>
            </li>
          {/each}
        </ul>
      {/if}

      <button class="btn-add" onclick={startAdd}>Add server</button>

      {#if servers.length > 0}
        <div class="sync-row">
          <button class="btn-sync" onclick={syncServers} disabled={syncing}>
            {syncing ? 'Syncing...' : 'Sync Now'}
          </button>
          {#if syncResult}
            <span class="sync-result" class:ok={syncResult.ok} class:err={!syncResult.ok}>
              {syncResult.message}
            </span>
          {/if}
        </div>
      {/if}
    {/if}
  </section>

  <section class="section">
    <h3>Last.fm</h3>

    {#if scrobbleConfig?.sessionKey}
      <div class="lfm-connected">
        <p class="lfm-status">Connected as <strong>{scrobbleConfig.username}</strong></p>
        <label class="lfm-toggle">
          <input type="checkbox" checked={scrobbleConfig.enabled} onchange={toggleScrobbleEnabled} />
          Scrobbling {scrobbleConfig.enabled ? 'enabled' : 'disabled'}
        </label>
        <button class="btn-cancel" onclick={disconnectLastFm}>Disconnect</button>
      </div>
    {:else if scrobbleConfig?.apiKey && !lfmAuthToken}
      <div class="lfm-auth">
        <p class="lfm-status">API key configured. Connect your Last.fm account to start scrobbling.</p>
        <button class="btn-save" onclick={startLastFmAuth} disabled={lfmConnecting}>
          {lfmConnecting ? 'Opening browser...' : 'Connect to Last.fm'}
        </button>
      </div>
    {:else if lfmAuthToken}
      <div class="lfm-auth">
        <p class="lfm-status">Approve the request in your browser, then click below.</p>
        <button class="btn-save" onclick={completeLastFmAuth} disabled={lfmConnecting}>
          {lfmConnecting ? 'Verifying...' : 'Complete authentication'}
        </button>
      </div>
    {:else}
      <div class="server-form">
        <div class="form-field">
          <label for="lfm-key">API Key</label>
          <input id="lfm-key" type="text" bind:value={lfmApiKey} placeholder="Your Last.fm API key" />
        </div>
        <div class="form-field">
          <label for="lfm-secret">API Secret</label>
          <input id="lfm-secret" type="password" bind:value={lfmApiSecret} placeholder="Your Last.fm API secret" />
        </div>
        <div class="form-actions">
          <button class="btn-save" onclick={saveApiKeys} disabled={!lfmApiKey.trim() || !lfmApiSecret.trim()}>
            Save
          </button>
        </div>
      </div>
    {/if}

    {#if lfmResult}
      <div class="test-result" class:ok={lfmResult.ok} class:err={!lfmResult.ok}>
        {lfmResult.message}
      </div>
    {/if}
  </section>

  <section class="section">
    <h3>ListenBrainz</h3>

    {#if lbConfig?.username}
      <div class="lfm-connected">
        <p class="lfm-status">Connected as <strong>{lbConfig.username}</strong></p>
        <label class="lfm-toggle">
          <input type="checkbox" checked={lbConfig.enabled} onchange={toggleLBEnabled} />
          Scrobbling {lbConfig.enabled ? 'enabled' : 'disabled'}
        </label>
        <button class="btn-cancel" onclick={disconnectListenBrainz}>Disconnect</button>
      </div>
    {:else}
      <div class="server-form">
        <p class="lfm-status">
          Paste your user token from
          <a href="https://listenbrainz.org/settings/" target="_blank" rel="noopener">listenbrainz.org/settings</a>.
        </p>
        <div class="form-field">
          <label for="lb-token">User Token</label>
          <input id="lb-token" type="password" bind:value={lbToken} placeholder="Your ListenBrainz user token" />
        </div>
        <div class="form-actions">
          <button class="btn-save" onclick={connectListenBrainz} disabled={!lbToken.trim() || lbConnecting}>
            {lbConnecting ? 'Connecting...' : 'Connect'}
          </button>
        </div>
      </div>
    {/if}

    {#if lbResult}
      <div class="test-result" class:ok={lbResult.ok} class:err={!lbResult.ok}>
        {lbResult.message}
      </div>
    {/if}
  </section>

  {#if queueSize > 0}
    <section class="section">
      <h3>Scrobble Queue</h3>
      <p class="queue-info">{queueSize} scrobble{queueSize === 1 ? '' : 's'} pending retry. These will be submitted automatically.</p>
    </section>
  {/if}
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

  .section + .section {
    margin-top: 2rem;
  }

  /* Theme options */
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

  /* Server list */
  .empty-msg {
    color: var(--text-secondary);
    font-size: 0.9rem;
    margin: 0 0 0.75rem;
  }

  .server-list {
    list-style: none;
    margin: 0 0 0.75rem;
    padding: 0;
  }

  .server-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.25rem 0;
  }

  .server-btn {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-primary);
    cursor: pointer;
    font-size: 0.9rem;
    text-align: left;
    min-width: 0;
  }

  .server-btn:hover {
    background: var(--bg-hover);
  }

  .server-name {
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .server-type-badge {
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    padding: 0.15rem 0.4rem;
    border-radius: 3px;
    background: var(--bg-hover);
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .server-url {
    font-size: 0.8rem;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    min-width: 0;
  }

  .server-actions {
    display: flex;
    gap: 0.25rem;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .server-item:hover .server-actions {
    opacity: 1;
  }

  .action-btn {
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
  }

  .action-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .action-btn.delete:hover {
    color: #e55;
  }

  .btn-add {
    padding: 0.5rem 1rem;
    border: 1px dashed var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.9rem;
    width: 100%;
  }

  .btn-add:hover {
    border-color: var(--accent);
    color: var(--text-primary);
    background: var(--bg-hover);
  }

  /* Server form */
  .server-form {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .form-field > label {
    font-size: 0.8rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .form-field input[type="text"],
  .form-field input[type="password"] {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-primary);
    font-size: 0.9rem;
  }

  .form-field input::placeholder {
    color: var(--text-secondary);
  }

  .form-field input:focus {
    outline: none;
    border-color: var(--accent);
  }

  .type-radios {
    display: flex;
    gap: 0.5rem;
  }

  .type-option {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    cursor: pointer;
    font-size: 0.9rem;
    color: var(--text-primary);
  }

  .type-option:hover {
    background: var(--bg-hover);
  }

  .type-option.selected {
    border-color: var(--accent);
    background: var(--bg-active);
  }

  .type-option input {
    accent-color: var(--accent);
  }

  .password-field {
    display: flex;
    gap: 0.5rem;
  }

  .password-field input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-primary);
    font-size: 0.9rem;
  }

  .password-field input:focus {
    outline: none;
    border-color: var(--accent);
  }

  .toggle-pw {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.8rem;
    white-space: nowrap;
  }

  .toggle-pw:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .test-result {
    font-size: 0.85rem;
    padding: 0.5rem 0.75rem;
    border-radius: 6px;
  }

  .test-result.ok {
    color: #4c8;
    background: rgba(68, 204, 136, 0.1);
  }

  .test-result.err {
    color: #e55;
    background: rgba(238, 85, 85, 0.1);
  }

  .form-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.25rem;
  }

  .form-actions-right {
    display: flex;
    gap: 0.5rem;
    margin-left: auto;
  }

  .btn-test {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.85rem;
  }

  .btn-test:hover:not(:disabled) {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .btn-test:disabled {
    opacity: 0.4;
    cursor: default;
  }

  .btn-cancel {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.85rem;
  }

  .btn-cancel:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .btn-save {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 6px;
    background: var(--accent);
    color: #fff;
    cursor: pointer;
    font-size: 0.85rem;
  }

  .btn-save:disabled {
    opacity: 0.4;
    cursor: default;
  }

  .btn-save:hover:not(:disabled) {
    filter: brightness(1.15);
  }

  /* Sync */
  .sync-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-top: 0.75rem;
  }

  .btn-sync {
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.85rem;
  }

  .btn-sync:hover:not(:disabled) {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .btn-sync:disabled {
    opacity: 0.4;
    cursor: default;
  }

  .sync-result {
    font-size: 0.8rem;
  }

  .sync-result.ok {
    color: #4c8;
  }

  .sync-result.err {
    color: #e55;
  }

  /* Last.fm */
  .lfm-connected {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .lfm-status {
    font-size: 0.9rem;
    color: var(--text-primary);
    margin: 0;
  }

  .lfm-toggle {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.9rem;
    color: var(--text-primary);
    cursor: pointer;
  }

  .lfm-toggle input {
    accent-color: var(--accent);
  }

  .lfm-auth {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .queue-info {
    font-size: 0.9rem;
    color: var(--text-secondary);
    margin: 0;
  }
</style>
