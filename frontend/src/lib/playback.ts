import { PlayerService } from "../../bindings/github.com/willfish/forte";
import { toPlaybackState, toRepeatMode, type PlaybackStatus } from './types';

const stoppedStatus: PlaybackStatus = {
  state: 'stopped',
  position: 0,
  duration: 0,
  volume: 100,
  title: '',
  artist: '',
  album: '',
  shuffle: false,
  repeat: 'off',
  mediaPath: '',
  radioMode: false,
  radioUuid: '',
  radioStation: '',
  radioArtwork: '',
  artworkSrc: '',
};

let current: PlaybackStatus = { ...stoppedStatus };
let statusTimer: ReturnType<typeof setTimeout> | null = null;
let artworkTimer: ReturnType<typeof setInterval> | null = null;
let statusInFlight = false;
let pollingActive = false;
let lastArtworkKey = '';
const listeners: Array<(status: PlaybackStatus) => void> = [];

export function getPlaybackStatus(): PlaybackStatus {
  return current;
}

export function onPlaybackStatusChange(fn: (status: PlaybackStatus) => void): () => void {
  listeners.push(fn);
  fn(current);
  startPolling();
  return () => {
    const idx = listeners.indexOf(fn);
    if (idx >= 0) listeners.splice(idx, 1);
    if (listeners.length === 0) stopPolling();
  };
}

export async function refreshPlaybackStatus() {
  if (statusInFlight) return;
  statusInFlight = true;
  try {
    const next = await PlayerService.GetPlaybackStatus();
    current = {
      ...current,
      state: toPlaybackState(next.state),
      position: next.position,
      duration: next.duration,
      volume: next.volume,
      title: next.title,
      artist: next.artist,
      album: next.album,
      shuffle: next.shuffle,
      repeat: toRepeatMode(next.repeat),
      mediaPath: next.state !== 'stopped' ? next.mediaPath : '',
      radioMode: next.radioMode,
      radioUuid: next.radioUuid || '',
      radioStation: next.radioStation,
      radioArtwork: next.radioArtwork,
    };
    notify();
  } catch {
    // Keep the last known state when the backend is briefly unavailable.
  } finally {
    statusInFlight = false;
  }
}

async function refreshArtwork() {
  const key = `${current.radioMode}|${current.title}|${current.artist}|${current.radioArtwork}`;
  if (key === lastArtworkKey) return;
  lastArtworkKey = key;

  if (current.radioMode && current.radioArtwork) {
    current = { ...current, artworkSrc: current.radioArtwork };
    notify();
    return;
  }
  if (current.state === 'stopped' || !current.title) {
    current = { ...current, artworkSrc: '' };
    notify();
    return;
  }

  try {
    current = { ...current, artworkSrc: await PlayerService.Artwork() };
    notify();
  } catch {
    current = { ...current, artworkSrc: '' };
    notify();
  }
}

function notify() {
  for (const fn of listeners) fn(current);
}

function startPolling() {
  if (pollingActive) return;
  pollingActive = true;
  pollPlaybackStatus();
  artworkTimer = setInterval(refreshArtwork, 1000);
}

async function pollPlaybackStatus() {
  await refreshPlaybackStatus();
  if (!pollingActive || listeners.length === 0) {
    statusTimer = null;
    return;
  }
  const delay = current.radioMode ? 2000 : 250;
  statusTimer = setTimeout(pollPlaybackStatus, delay);
}

function stopPolling() {
  pollingActive = false;
  if (statusTimer) {
    clearTimeout(statusTimer);
    statusTimer = null;
  }
  if (artworkTimer) {
    clearInterval(artworkTimer);
    artworkTimer = null;
  }
}
