export type ThemePreference = 'dark' | 'light' | 'system';

const STORAGE_KEY = 'forte-theme';

let _preference: ThemePreference = loadPreference();
const _listeners: Array<(theme: ThemePreference) => void> = [];

function loadPreference(): ThemePreference {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === 'dark' || stored === 'light' || stored === 'system') return stored;
  return 'dark';
}

function resolveTheme(pref: ThemePreference): 'dark' | 'light' {
  if (pref !== 'system') return pref;
  return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark';
}

function applyTheme(theme: 'dark' | 'light') {
  document.documentElement.setAttribute('data-theme', theme);
}

export function getPreference(): ThemePreference {
  return _preference;
}

export function setPreference(pref: ThemePreference) {
  _preference = pref;
  localStorage.setItem(STORAGE_KEY, pref);
  applyTheme(resolveTheme(pref));
  for (const fn of _listeners) fn(pref);
}

export function onPreferenceChange(fn: (pref: ThemePreference) => void): () => void {
  _listeners.push(fn);
  return () => {
    const idx = _listeners.indexOf(fn);
    if (idx >= 0) _listeners.splice(idx, 1);
  };
}

export function initTheme() {
  applyTheme(resolveTheme(_preference));

  // Listen for OS theme changes when preference is 'system'.
  window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', () => {
    if (_preference === 'system') {
      applyTheme(resolveTheme('system'));
    }
  });
}
