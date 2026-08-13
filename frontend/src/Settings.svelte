<script lang="ts">
  import {
    getPreference,
    getTransparencyPreference,
    onPreferenceChange,
    onTransparencyPreferenceChange,
    setPreference,
    setTransparencyPreference,
    themeColour,
    themeMode,
    themePreference,
    type ThemeColour,
    type ThemeMode,
    type ThemePreference,
    type ThemeTransparencyPreference
  } from './lib/theme';
  import { setLibraryEnabled, setTitlebarEnabled } from './lib/stores';
  import { Dialogs } from '@wailsio/runtime';
  import { LibraryService } from "../bindings/github.com/willfish/forte";
  import type { ServerConfig } from './lib/types';

  type ServerResponse = Omit<ServerConfig, 'password' | 'hasPassword'> & {
    password?: string;
    hasPassword?: boolean;
  };

  // Theme state
  let preference = $state<ThemePreference>(getPreference());
  let transparencyPreference = $state<ThemeTransparencyPreference>(getTransparencyPreference());
  let appPreferences = $state({
    libraryEnabled: false,
    startLastStation: true,
    autoReconnect: true,
    showTitlebar: false,
    logLevel: 'warn',
    logFilePath: '',
  });

  const logLevelOptions: { value: string; label: string; description: string }[] = [
    { value: 'error', label: 'Errors only', description: 'Log failures and critical issues' },
    { value: 'warn', label: 'Normal', description: 'Default — warnings, errors, and legacy diagnostic lines' },
    { value: 'info', label: 'Detailed', description: 'More library and sync messages' },
    { value: 'debug', label: 'Verbose', description: 'Everything, including playback and tray debug lines' },
    { value: 'off', label: 'Off', description: 'Disable file logging (crash.log is still written)' },
  ];

  $effect(() => {
    return onPreferenceChange((p) => { preference = p; });
  });

  $effect(() => {
    return onTransparencyPreferenceChange((p) => { transparencyPreference = p; });
  });

  function handleChange(pref: ThemePreference) {
    setPreference(pref);
    preference = pref;
  }

  function handleTransparencyEnabledChange(enabled: boolean) {
    setTransparencyPreference({ enabled });
    transparencyPreference = getTransparencyPreference();
  }

  function handleOpacityChange(opacity: number) {
    setTransparencyPreference({ opacity });
    transparencyPreference = getTransparencyPreference();
  }

  async function loadAppPreferences() {
    try {
      const prefs = await LibraryService.GetAppPreferences();
      appPreferences = {
        libraryEnabled: Boolean(prefs?.libraryEnabled),
        startLastStation: Boolean(prefs?.startLastStation),
        autoReconnect: Boolean(prefs?.autoReconnect),
        showTitlebar: Boolean(prefs?.showTitlebar),
        logLevel: prefs?.logLevel || 'warn',
        logFilePath: prefs?.logFilePath || '',
      };
      setLibraryEnabled(appPreferences.libraryEnabled);
      setTitlebarEnabled(appPreferences.showTitlebar);
    } catch {
      setLibraryEnabled(false);
      setTitlebarEnabled(false);
    }
  }

  $effect(() => {
    loadAppPreferences();
  });

  async function saveAppPreferences(next: typeof appPreferences) {
    appPreferences = next;
    if ('libraryEnabled' in next) {
      setLibraryEnabled(next.libraryEnabled);
    }
    if ('showTitlebar' in next) {
      setTitlebarEnabled(next.showTitlebar);
    }
    try {
      await LibraryService.SaveAppPreferences({
        libraryEnabled: next.libraryEnabled,
        startLastStation: next.startLastStation,
        autoReconnect: next.autoReconnect,
        showTitlebar: next.showTitlebar,
        logLevel: next.logLevel,
        logFilePath: next.logFilePath,
      });
      await loadAppPreferences();
    } catch {
      await loadAppPreferences();
    }
  }

  async function saveAppPreference(key: 'libraryEnabled' | 'startLastStation' | 'autoReconnect' | 'showTitlebar', value: boolean) {
    await saveAppPreferences({ ...appPreferences, [key]: value });
  }

  async function saveLogLevel(level: string) {
    await saveAppPreferences({ ...appPreferences, logLevel: level });
  }

  let userConfigPath = $state('');
  let userConfigStatus = $state<{ ok: boolean; message: string } | null>(null);
  let userConfigBusy = $state(false);

  async function loadUserConfigPath() {
    try {
      userConfigPath = await LibraryService.GetUserConfigPath() || '';
    } catch {
      userConfigPath = '';
    }
  }

  $effect(() => {
    loadUserConfigPath();
  });

  async function saveUserConfigHome() {
    userConfigBusy = true;
    userConfigStatus = null;
    try {
      const path = await LibraryService.ExportUserConfig();
      userConfigPath = path;
      userConfigStatus = {
        ok: true,
        message: `Saved to ${path}. Restart merges new stations from this file without overwriting favourites already in the database.`,
      };
    } catch (err) {
      userConfigStatus = { ok: false, message: err instanceof Error ? err.message : 'Save failed' };
    } finally {
      userConfigBusy = false;
    }
  }

  async function exportUserConfigCopy() {
    userConfigBusy = true;
    userConfigStatus = null;
    try {
      const savePath = await Dialogs.SaveFile({
        Title: 'Export Forte configuration copy',
        Filename: 'config.toml',
        Filters: [{ DisplayName: 'TOML configuration', Pattern: '*.toml' }],
      });
      const path = typeof savePath === 'string' ? savePath : '';
      if (!path) {
        userConfigStatus = { ok: false, message: 'Export cancelled' };
        return;
      }
      await LibraryService.ExportUserConfigToPath(path);
      userConfigStatus = { ok: true, message: `Exported copy to ${path}` };
    } catch (err) {
      userConfigStatus = { ok: false, message: err instanceof Error ? err.message : 'Export failed' };
    } finally {
      userConfigBusy = false;
    }
  }

  async function importUserConfig() {
    userConfigBusy = true;
    userConfigStatus = null;
    try {
      const openPath = await Dialogs.OpenFile({
        Title: 'Import Forte configuration',
        CanChooseFiles: true,
        CanChooseDirectories: false,
        Filters: [{ DisplayName: 'TOML configuration', Pattern: '*.toml' }],
      });
      const path = Array.isArray(openPath) ? openPath[0] : openPath;
      if (!path) {
        userConfigStatus = { ok: false, message: 'Import cancelled' };
        return;
      }
      const result = await LibraryService.ImportUserConfigFromPath(path);
      await loadAppPreferences();
      const sections = result?.sectionsApplied?.join(', ') || 'none';
      const warnings = result?.warnings?.length ? ` Warnings: ${result.warnings.join('; ')}` : '';
      userConfigStatus = { ok: true, message: `Imported from ${path}. Applied: ${sections}.${warnings}` };
    } catch (err) {
      userConfigStatus = { ok: false, message: err instanceof Error ? err.message : 'Import failed' };
    } finally {
      userConfigBusy = false;
    }
  }

  const themeModeOptions: { value: ThemeMode; label: string; description: string }[] = [
    { value: 'dark', label: 'Dark', description: 'Use darker surfaces and light text' },
    { value: 'light', label: 'Light', description: 'Use lighter surfaces and dark text' },
  ];

  const themeColourOptions: { value: ThemeColour; label: string; description: string }[] = [
    { value: 'green', label: 'Green', description: 'Forte green accents' },
    { value: 'blue', label: 'Blue', description: 'Cool blue accents' },
    { value: 'financial-times', label: 'Financial Times', description: 'FT-inspired pink and paper tones' },
  ];

  function handleThemeModeChange(mode: ThemeMode) {
    handleChange(themePreference(themeColour(preference), mode));
  }

  function handleThemeColourChange(colour: ThemeColour) {
    handleChange(themePreference(colour, themeMode(preference)));
  }

  // Server state
  let servers = $state<ServerConfig[]>([]);
  let editing = $state<ServerConfig | null>(null);
  let testing = $state(false);
  let testResult = $state<{ ok: boolean; message: string } | null>(null);
  let showPassword = $state(false);
  let syncing = $state(false);
  let syncResult = $state<{ ok: boolean; message: string } | null>(null);
  let musicDirectories = $state<string[]>([]);
  let musicDirectoryBusy = $state(false);
  let musicDirectoryStatus = $state<{ ok: boolean; message: string } | null>(null);

  async function loadServers() {
    servers = ((await LibraryService.GetServers()) || []).map((s: ServerResponse) => ({
      id: s.id,
      name: s.name,
      type: s.type,
      url: s.url,
      username: s.username,
      password: '',
      hasPassword: s.hasPassword ?? false,
    }));
  }

  $effect(() => {
    if (appPreferences.libraryEnabled) {
      loadServers();
    } else {
      servers = [];
    }
  });

  async function loadMusicDirectories() {
    try {
      musicDirectories = (await LibraryService.GetMusicDirectories()) || [];
    } catch {
      musicDirectories = [];
    }
  }

  $effect(() => {
    if (appPreferences.libraryEnabled) {
      loadMusicDirectories();
    } else {
      musicDirectories = [];
      musicDirectoryStatus = null;
    }
  });

  async function addMusicDirectory() {
    musicDirectoryBusy = true;
    musicDirectoryStatus = null;
    try {
      const selected = await Dialogs.OpenFile({
        Title: 'Add music directory',
        CanChooseFiles: false,
        CanChooseDirectories: true,
      });
      const path = Array.isArray(selected) ? selected[0] : selected;
      if (!path) {
        musicDirectoryStatus = { ok: false, message: 'Add directory cancelled' };
        return;
      }
      await LibraryService.AddMusicDirectory(path);
      await loadMusicDirectories();
      if (!musicDirectories.includes(path)) {
        musicDirectories = [...musicDirectories, path];
      }
      musicDirectoryStatus = { ok: true, message: 'Music directory added' };
    } catch (err) {
      musicDirectoryStatus = { ok: false, message: err instanceof Error ? err.message : 'Add directory failed' };
    } finally {
      musicDirectoryBusy = false;
    }
  }

  async function removeMusicDirectory(path: string) {
    musicDirectoryBusy = true;
    musicDirectoryStatus = null;
    try {
      await LibraryService.RemoveMusicDirectory(path);
      await loadMusicDirectories();
      musicDirectoryStatus = { ok: true, message: 'Music directory removed' };
    } catch (err) {
      musicDirectoryStatus = { ok: false, message: err instanceof Error ? err.message : 'Remove directory failed' };
    } finally {
      musicDirectoryBusy = false;
    }
  }

  async function scanMusicLibrary() {
    musicDirectoryBusy = true;
    musicDirectoryStatus = null;
    try {
      await LibraryService.ScanMusicLibrary();
      musicDirectoryStatus = { ok: true, message: 'Library scan completed' };
    } catch (err) {
      musicDirectoryStatus = { ok: false, message: err instanceof Error ? err.message : 'Library scan failed' };
    } finally {
      musicDirectoryBusy = false;
    }
  }

  function startAdd() {
    editing = { id: '', name: '', type: 'subsonic', url: '', username: '', password: '', hasPassword: false };
    testResult = null;
    showPassword = false;
  }

  function startEdit(srv: ServerConfig) {
    editing = { ...srv, password: '' };
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
    <div class="theme-controls">
      <div class="theme-control-group">
        <h4>Mode</h4>
        <div class="theme-options">
          {#each themeModeOptions as opt}
            <label class="theme-option" class:selected={themeMode(preference) === opt.value}>
              <input
                type="radio"
                name="theme-mode"
                value={opt.value}
                checked={themeMode(preference) === opt.value}
                onchange={() => handleThemeModeChange(opt.value)}
              />
              <div class="option-content">
                <span class="option-label">{opt.label}</span>
                <span class="option-desc">{opt.description}</span>
              </div>
            </label>
          {/each}
        </div>
      </div>

      <div class="theme-control-group">
        <h4>Colour</h4>
        <div class="theme-options">
          {#each themeColourOptions as opt}
            <label class="theme-option" class:selected={themeColour(preference) === opt.value}>
              <input
                type="radio"
                name="theme-colour"
                value={opt.value}
                checked={themeColour(preference) === opt.value}
                onchange={() => handleThemeColourChange(opt.value)}
              />
              <div class="option-content">
                <span class="option-label">{opt.label}</span>
                <span class="option-desc">{opt.description}</span>
              </div>
            </label>
          {/each}
        </div>
      </div>

      <div class="theme-control-group">
        <h4>Transparency</h4>
        <div class="preference-list">
          <label class="preference-row">
            <span>
              <span class="option-label">Transparent theme</span>
              <span class="option-desc">Let Forte's main surfaces show the desktop behind the window</span>
            </span>
            <input
              type="checkbox"
              checked={transparencyPreference.enabled}
              onchange={(e) => handleTransparencyEnabledChange((e.target as HTMLInputElement).checked)}
            />
          </label>
          <label class="opacity-row">
            <span>
              <span class="option-label">Theme opacity</span>
              <span class="option-desc">{transparencyPreference.opacity.toFixed(2)}</span>
            </span>
            <input
              type="range"
              min="0.2"
              max="1"
              step="0.05"
              value={transparencyPreference.opacity}
              disabled={!transparencyPreference.enabled}
              oninput={(e) => handleOpacityChange(Number((e.target as HTMLInputElement).value))}
            />
          </label>
        </div>
      </div>
    </div>
  </section>

  <section class="section">
    <h3>Application</h3>
    <div class="preference-list">
      <label class="preference-row">
        <span>
          <span class="option-label">Show title bar</span>
          <span class="option-desc">Use Forte's compact window bar when your desktop does not provide one</span>
        </span>
        <input
          type="checkbox"
          checked={appPreferences.showTitlebar}
          onchange={(e) => saveAppPreference('showTitlebar', (e.target as HTMLInputElement).checked)}
        />
      </label>
      <label class="preference-row">
        <span>
          <span class="option-label">Start last station</span>
          <span class="option-desc">Resume your most recent radio stream when Forte opens</span>
        </span>
        <input
          type="checkbox"
          checked={appPreferences.startLastStation}
          onchange={(e) => saveAppPreference('startLastStation', (e.target as HTMLInputElement).checked)}
        />
      </label>
      <label class="preference-row">
        <span>
          <span class="option-label">Reconnect streams</span>
          <span class="option-desc">Retry radio playback automatically when a stream drops, with backoff</span>
        </span>
        <input
          type="checkbox"
          checked={appPreferences.autoReconnect}
          onchange={(e) => saveAppPreference('autoReconnect', (e.target as HTMLInputElement).checked)}
        />
      </label>
      <label class="preference-row">
        <span>
          <span class="option-label">Library mode</span>
          <span class="option-desc">Show local/server library, playlists, and statistics</span>
        </span>
        <input
          type="checkbox"
          checked={appPreferences.libraryEnabled}
          onchange={(e) => saveAppPreference('libraryEnabled', (e.target as HTMLInputElement).checked)}
        />
      </label>
      <div class="preference-row log-level-row">
        <span>
          <span class="option-label">Diagnostic logging</span>
          <span class="option-desc">
            {#each logLevelOptions.filter((o) => o.value === appPreferences.logLevel) as opt}
              {opt.description}
            {/each}
          </span>
          {#if appPreferences.logFilePath}
            <span class="log-path">{appPreferences.logFilePath}</span>
          {/if}
        </span>
        <select
          class="log-level-select"
          value={appPreferences.logLevel}
          onchange={(e) => saveLogLevel((e.target as HTMLSelectElement).value)}
        >
          {#each logLevelOptions as opt}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>
      </div>
    </div>
  </section>

  <section class="section">
    <h3>Configuration file</h3>
    <p class="section-desc">
      Save favourites, custom stations, and app preferences to <code>config.toml</code> in your config directory.
      On startup Forte merges in stations from that file (adds missing UUIDs; does not overwrite favourites already in the database).
      Use Import to apply a chosen file and overwrite matching stations.
    </p>
    {#if userConfigPath}
      <p class="config-path">Default path: <span>{userConfigPath}</span></p>
    {/if}
    <div class="config-actions">
      <button class="btn-save" type="button" onclick={saveUserConfigHome} disabled={userConfigBusy}>
        Save to config directory
      </button>
      <button class="btn-cancel" type="button" onclick={exportUserConfigCopy} disabled={userConfigBusy}>
        Export copy…
      </button>
      <button class="btn-cancel" type="button" onclick={importUserConfig} disabled={userConfigBusy}>
        Import from file…
      </button>
    </div>
    {#if userConfigStatus}
      <p class="config-status" class:ok={userConfigStatus.ok} class:error={!userConfigStatus.ok}>
        {userConfigStatus.message}
      </p>
    {/if}
  </section>

  {#if appPreferences.libraryEnabled}
  <section class="section">
    <h3>Local Library</h3>
    {#if musicDirectories.length === 0}
      <p class="empty-msg">No music directories configured.</p>
    {:else}
      <ul class="directory-list">
        {#each musicDirectories as dir}
          <li class="directory-item">
            <span class="directory-path">{dir}</span>
            <button
              class="action-btn delete"
              type="button"
              onclick={() => removeMusicDirectory(dir)}
              disabled={musicDirectoryBusy}
              aria-label={`Remove ${dir}`}
            >
              <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
              </svg>
            </button>
          </li>
        {/each}
      </ul>
    {/if}
    <div class="library-actions">
      <button class="btn-add" type="button" onclick={addMusicDirectory} disabled={musicDirectoryBusy}>
        Add directory…
      </button>
      <button class="btn-sync" type="button" onclick={scanMusicLibrary} disabled={musicDirectoryBusy || musicDirectories.length === 0}>
        {musicDirectoryBusy ? 'Working...' : 'Scan now'}
      </button>
    </div>
    {#if musicDirectoryStatus}
      <p class="config-status" class:ok={musicDirectoryStatus.ok} class:error={!musicDirectoryStatus.ok}>
        {musicDirectoryStatus.message}
      </p>
    {/if}
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
              <input id="srv-pass" type="text" bind:value={editing.password} placeholder={editing.id && editing.hasPassword ? 'Leave blank to keep existing password' : ''} />
            {:else}
              <input id="srv-pass" type="password" bind:value={editing.password} placeholder={editing.id && editing.hasPassword ? 'Leave blank to keep existing password' : ''} />
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
  {/if}

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
    animation: view-enter 0.2s ease-out;
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
  .theme-controls {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .theme-control-group h4 {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
    margin: 0 0 0.5rem;
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

  .preference-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .preference-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    cursor: pointer;
  }

  .preference-row:hover {
    background: var(--bg-hover);
  }

  .preference-row > span {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .log-level-row {
    align-items: flex-start;
  }

  .log-path {
    display: block;
    margin-top: 0.35rem;
    font-size: 0.75rem;
    color: var(--text-muted, #888);
    font-family: ui-monospace, monospace;
    word-break: break-all;
  }

  .log-level-select {
    min-width: 9rem;
    padding: 0.35rem 0.5rem;
    border-radius: 6px;
    border: 1px solid var(--border-subtle, #333);
    background: var(--surface-raised, #1a1a1a);
    color: inherit;
  }

  .preference-row input {
    flex-shrink: 0;
    accent-color: var(--accent);
  }

  .opacity-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(10rem, 14rem);
    align-items: center;
    gap: 1rem;
    padding: 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
  }

  .opacity-row > span {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .opacity-row input {
    width: 100%;
    accent-color: var(--accent);
  }

  .opacity-row input:disabled {
    opacity: 0.45;
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

  .directory-list {
    list-style: none;
    margin: 0 0 0.75rem;
    padding: 0;
  }

  .directory-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 0;
  }

  .directory-path {
    flex: 1;
    min-width: 0;
    padding: 0.55rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    color: var(--text-primary);
    font-family: ui-monospace, monospace;
    font-size: 0.8rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .library-actions {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 0.5rem;
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
    color: var(--error);
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
    color: var(--success);
    background: color-mix(in srgb, var(--success) 10%, transparent);
  }

  .test-result.err {
    color: var(--error);
    background: color-mix(in srgb, var(--error) 10%, transparent);
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
    color: var(--text-on-accent);
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

  .section-desc {
    margin: 0 0 0.75rem;
    color: var(--text-secondary);
    font-size: 0.85rem;
    line-height: 1.45;
  }

  .section-desc code {
    font-size: 0.8rem;
  }

  .config-path {
    margin: 0 0 0.75rem;
    font-size: 0.8rem;
    color: var(--text-secondary);
    word-break: break-all;
  }

  .config-path span {
    color: var(--text-primary);
  }

  .config-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }

  .config-status {
    margin: 0.75rem 0 0;
    font-size: 0.8rem;
    color: var(--text-secondary);
    word-break: break-word;
  }

  .config-status.ok {
    color: var(--accent);
  }

  .config-status.error {
    color: #e57373;
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
    color: var(--success);
  }

  .sync-result.err {
    color: var(--error);
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
