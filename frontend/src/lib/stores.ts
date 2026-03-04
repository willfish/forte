// Reactive stores for application state.

// Current view shown in the content area.
export type View = 'library' | 'playlists' | 'stats' | 'settings';

// Simple store using Svelte 5 module-level state is not possible
// in a .ts file, so we use a plain object with getter/setter callbacks.
// Components will use $state() locally and subscribe via these helpers.

let _currentView: View = 'library';
const _listeners: Array<(view: View) => void> = [];

// Server status tracking: maps serverId to online status.
let _serverStatuses: Record<string, boolean> = {};
const _statusListeners: Array<(statuses: Record<string, boolean>) => void> = [];

export function getCurrentView(): View {
  return _currentView;
}

export function setCurrentView(view: View) {
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
