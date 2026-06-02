export type ThemePreference =
  | 'green-dark'
  | 'green-light'
  | 'blue-dark'
  | 'blue-light'
  | 'financial-times-dark'
  | 'financial-times-light';
export type ThemeMode = 'dark' | 'light';
export type ThemeColour = 'green' | 'blue' | 'financial-times';
export type ThemeTransparencyPreference = {
  enabled: boolean;
  opacity: number;
};

const STORAGE_KEY = 'forte-theme';
const TRANSPARENCY_STORAGE_KEY = 'forte-theme-transparency';
const DEFAULT_TRANSPARENCY: ThemeTransparencyPreference = {
  enabled: false,
  opacity: 0.8,
};
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
let _transparencyPreference: ThemeTransparencyPreference = loadTransparencyPreference();
const _transparencyListeners: Array<(preference: ThemeTransparencyPreference) => void> = [];

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

function clampOpacity(opacity: number): number {
  if (!Number.isFinite(opacity)) return DEFAULT_TRANSPARENCY.opacity;
  return Math.min(1, Math.max(0.2, opacity));
}

function normaliseTransparencyPreference(pref: Partial<ThemeTransparencyPreference> | null): ThemeTransparencyPreference {
  return {
    enabled: Boolean(pref?.enabled),
    opacity: clampOpacity(Number(pref?.opacity ?? DEFAULT_TRANSPARENCY.opacity)),
  };
}

function loadTransparencyPreference(): ThemeTransparencyPreference {
  const stored = localStorage.getItem(TRANSPARENCY_STORAGE_KEY);
  if (!stored) return DEFAULT_TRANSPARENCY;

  try {
    return normaliseTransparencyPreference(JSON.parse(stored));
  } catch {
    return DEFAULT_TRANSPARENCY;
  }
}

function applyTransparencyPreference(pref: ThemeTransparencyPreference) {
  const root = document.documentElement;
  const activeOpacity = pref.enabled ? pref.opacity : 1;
  root.setAttribute('data-theme-transparency', pref.enabled ? 'on' : 'off');
  root.style.setProperty('--theme-opacity', String(activeOpacity));
  root.style.setProperty('--theme-surface-opacity', `${activeOpacity * 100}%`);
}

function saveTransparencyPreference(pref: ThemeTransparencyPreference) {
  localStorage.setItem(TRANSPARENCY_STORAGE_KEY, JSON.stringify(pref));
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

export function getTransparencyPreference(): ThemeTransparencyPreference {
  return { ..._transparencyPreference };
}

export function setTransparencyPreference(pref: Partial<ThemeTransparencyPreference>) {
  _transparencyPreference = normaliseTransparencyPreference({
    ..._transparencyPreference,
    ...pref,
  });
  saveTransparencyPreference(_transparencyPreference);
  applyTransparencyPreference(_transparencyPreference);
  for (const fn of _transparencyListeners) fn(getTransparencyPreference());
}

export function onPreferenceChange(fn: (pref: ThemePreference) => void): () => void {
  _listeners.push(fn);
  return () => {
    const idx = _listeners.indexOf(fn);
    if (idx >= 0) _listeners.splice(idx, 1);
  };
}

export function onTransparencyPreferenceChange(fn: (pref: ThemeTransparencyPreference) => void): () => void {
  _transparencyListeners.push(fn);
  return () => {
    const idx = _transparencyListeners.indexOf(fn);
    if (idx >= 0) _transparencyListeners.splice(idx, 1);
  };
}

export function initTheme() {
  applyTheme(_preference);
  applyTransparencyPreference(_transparencyPreference);
}
