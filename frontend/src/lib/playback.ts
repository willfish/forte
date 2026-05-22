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
  radioStation: '',
  radioArtwork: '',
  artworkSrc: '',
};

let current: PlaybackStatus = { ...stoppedStatus };
let statusTimer: ReturnType<typeof setInterval> | null = null;
let artworkTimer: ReturnType<typeof setInterval> | null = null;
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
      radioStation: next.radioStation,
      radioArtwork: next.radioArtwork,
    };
    notify();
  } catch {
    // Keep the last known state when the backend is briefly unavailable.
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
  if (statusTimer) return;
  void refreshPlaybackStatus();
  statusTimer = setInterval(refreshPlaybackStatus, 250);
  artworkTimer = setInterval(refreshArtwork, 1000);
}

function stopPolling() {
  if (statusTimer) {
    clearInterval(statusTimer);
    statusTimer = null;
  }
  if (artworkTimer) {
    clearInterval(artworkTimer);
    artworkTimer = null;
  }
}
