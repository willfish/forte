// Reactive stores for application state.

// Current view shown in the content area.
export type View = 'library' | 'playlists' | 'settings';

// Simple store using Svelte 5 module-level state is not possible
// in a .ts file, so we use a plain object with getter/setter callbacks.
// Components will use $state() locally and subscribe via these helpers.

let _currentView: View = 'library';
const _listeners: Array<(view: View) => void> = [];

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
