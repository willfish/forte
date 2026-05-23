export type ThemePreference =
  | 'green-dark'
  | 'green-light'
  | 'blue-dark'
  | 'blue-light'
  | 'financial-times-dark'
  | 'financial-times-light';
export type ThemeMode = 'dark' | 'light';
export type ThemeColour = 'green' | 'blue' | 'financial-times';

const STORAGE_KEY = 'forte-theme';
const THEME_PREFERENCES: ThemePreference[] = [
  'green-dark',
  'green-light',
  'blue-dark',
  'blue-light',
  'financial-times-dark',
  'financial-times-light',
];

let _preference: ThemePreference = loadPreference();
const _listeners: Array<(theme: ThemePreference) => void> = [];

function loadPreference(): ThemePreference {
  const stored = localStorage.getItem(STORAGE_KEY);
  if (THEME_PREFERENCES.includes(stored as ThemePreference)) return stored as ThemePreference;
  if (stored === 'light') return 'green-light';
  if (stored === 'system' && window.matchMedia('(prefers-color-scheme: light)').matches) return 'green-light';
  return 'green-dark';
}

function applyTheme(theme: ThemePreference) {
  document.documentElement.setAttribute('data-theme', theme);
}

export function getPreference(): ThemePreference {
  return _preference;
}

export function setPreference(pref: ThemePreference) {
  _preference = pref;
  localStorage.setItem(STORAGE_KEY, pref);
  applyTheme(pref);
  for (const fn of _listeners) fn(pref);
}

export function themeMode(pref: ThemePreference): ThemeMode {
  return pref.endsWith('-light') ? 'light' : 'dark';
}

export function themeColour(pref: ThemePreference): ThemeColour {
  if (pref.startsWith('financial-times')) return 'financial-times';
  if (pref.startsWith('blue')) return 'blue';
  return 'green';
}

export function themePreference(colour: ThemeColour, mode: ThemeMode): ThemePreference {
  return `${colour}-${mode}` as ThemePreference;
}

export function onPreferenceChange(fn: (pref: ThemePreference) => void): () => void {
  _listeners.push(fn);
  return () => {
    const idx = _listeners.indexOf(fn);
    if (idx >= 0) _listeners.splice(idx, 1);
  };
}

export function initTheme() {
  applyTheme(_preference);
}
