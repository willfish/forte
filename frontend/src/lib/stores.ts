// Reactive stores for application state.

// Current view shown in the content area.
export type View = 'library' | 'playlists' | 'radio' | 'stats' | 'settings';

// Simple store using Svelte 5 module-level state is not possible
// in a .ts file, so we use a plain object with getter/setter callbacks.
// Components will use $state() locally and subscribe via these helpers.

let _currentView: View = 'radio';
const _listeners: Array<(view: View) => void> = [];

// Server status tracking: maps serverId to online status.
let _serverStatuses: Record<string, boolean> = {};
const _statusListeners: Array<(statuses: Record<string, boolean>) => void> = [];

let _libraryEnabled = false;
const _libraryPreferenceListeners: Array<(enabled: boolean) => void> = [];
let _titlebarEnabled = false;
const _titlebarPreferenceListeners: Array<(enabled: boolean) => void> = [];
let _radioTagFilter = '';
const _radioTagFilterListeners: Array<(tag: string) => void> = [];

export type RadioStationHint = {
  uuid: string;
  name?: string;
  streamUrl?: string;
  favicon?: string;
  homepage?: string;
  tags?: string;
  country?: string;
  bitrate?: number;
  codec?: string;
};

let _radioStationDetail: string | null = null;
let _radioStationHint: RadioStationHint | null = null;
const _radioStationDetailListeners: Array<(uuid: string | null, hint: RadioStationHint | null) => void> = [];

export function getCurrentView(): View {
  return _currentView;
}

export function setCurrentView(view: View) {
  if (!_libraryEnabled && ['library', 'playlists', 'stats'].includes(view)) {
    view = 'radio';
  }
  _currentView = view;
  for (const fn of _listeners) {
    fn(view);
  }
}

export function onViewChange(fn: (view: View) => void): () => void {
  _listeners.push(fn);
  return () => {
    const idx = _listeners.indexOf(fn);
    if (idx >= 0) _listeners.splice(idx, 1);
  };
}

export function getServerStatuses(): Record<string, boolean> {
  return _serverStatuses;
}

export function setServerStatuses(statuses: Record<string, boolean>) {
  _serverStatuses = statuses;
  for (const fn of _statusListeners) {
    fn(statuses);
  }
}

export function onServerStatusChange(fn: (statuses: Record<string, boolean>) => void): () => void {
  _statusListeners.push(fn);
  return () => {
    const idx = _statusListeners.indexOf(fn);
    if (idx >= 0) _statusListeners.splice(idx, 1);
  };
}

export function isServerOnline(serverId: string): boolean {
  if (!serverId) return true;
  if (!(serverId in _serverStatuses)) return true;
  return _serverStatuses[serverId];
}

export function isLibraryEnabled(): boolean {
  return _libraryEnabled;
}

export function setLibraryEnabled(enabled: boolean) {
  _libraryEnabled = enabled;
  if (!enabled && ['library', 'playlists', 'stats'].includes(_currentView)) {
    setCurrentView('radio');
  }
  for (const fn of _libraryPreferenceListeners) {
    fn(enabled);
  }
}

export function onLibraryEnabledChange(fn: (enabled: boolean) => void): () => void {
  _libraryPreferenceListeners.push(fn);
  return () => {
    const idx = _libraryPreferenceListeners.indexOf(fn);
    if (idx >= 0) _libraryPreferenceListeners.splice(idx, 1);
  };
}

export function isTitlebarEnabled(): boolean {
  return _titlebarEnabled;
}

export function setTitlebarEnabled(enabled: boolean) {
  _titlebarEnabled = enabled;
  for (const fn of _titlebarPreferenceListeners) {
    fn(enabled);
  }
}

export function onTitlebarEnabledChange(fn: (enabled: boolean) => void): () => void {
  _titlebarPreferenceListeners.push(fn);
  return () => {
    const idx = _titlebarPreferenceListeners.indexOf(fn);
    if (idx >= 0) _titlebarPreferenceListeners.splice(idx, 1);
  };
}

export function getRadioTagFilter(): string {
  return _radioTagFilter;
}

export function browseRadioTag(tag: string) {
  _radioTagFilter = tag;
  setCurrentView('radio');
  for (const fn of _radioTagFilterListeners) {
    fn(tag);
  }
}

export function onRadioTagFilterChange(fn: (tag: string) => void): () => void {
  _radioTagFilterListeners.push(fn);
  return () => {
    const idx = _radioTagFilterListeners.indexOf(fn);
    if (idx >= 0) _radioTagFilterListeners.splice(idx, 1);
  };
}

export function getRadioStationDetail(): string | null {
  return _radioStationDetail;
}

export function getRadioStationHint(): RadioStationHint | null {
  return _radioStationHint;
}

export function openRadioStation(uuid: string, hint: Omit<RadioStationHint, 'uuid'> = {}) {
  _radioStationDetail = uuid;
  _radioStationHint = { uuid, ...hint };
  setCurrentView('radio');
  for (const fn of _radioStationDetailListeners) {
    fn(uuid, _radioStationHint);
  }
}

export function clearRadioStationDetail() {
  _radioStationDetail = null;
  _radioStationHint = null;
  for (const fn of _radioStationDetailListeners) {
    fn(null, null);
  }
}

export function onRadioStationDetailChange(
  fn: (uuid: string | null, hint: RadioStationHint | null) => void
): () => void {
  _radioStationDetailListeners.push(fn);
  return () => {
    const idx = _radioStationDetailListeners.indexOf(fn);
    if (idx >= 0) _radioStationDetailListeners.splice(idx, 1);
  };
}
