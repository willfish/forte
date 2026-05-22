export type Source = 'local' | 'server';
export type PlaybackState = 'stopped' | 'playing' | 'paused';
export type RepeatMode = 'off' | 'all' | 'one';

export type Album = {
  id: number;
  title: string;
  artist: string;
  year: number;
  trackCount: number;
  source: Source;
  serverId: string;
  artworkSrc?: string;
};

export type AlbumTrack = {
  trackId: number;
  title: string;
  artist: string;
  trackNumber: number;
  discNumber: number;
  durationMs: number;
  filePath: string;
  source: Source;
  serverId: string;
};

export type SearchResult = {
  trackId: number;
  title: string;
  artist: string;
  album: string;
  genre: string;
  durationMs: number;
  filePath: string;
  source: Source;
  serverId: string;
};

export type QueueTrack = {
  trackId: number;
  title: string;
  artist: string;
  album: string;
  durationMs: number;
  filePath: string;
};

export type PlaybackStatus = {
  state: PlaybackState;
  position: number;
  duration: number;
  volume: number;
  title: string;
  artist: string;
  album: string;
  shuffle: boolean;
  repeat: RepeatMode;
  mediaPath: string;
  radioMode: boolean;
  radioStation: string;
  radioArtwork: string;
  artworkSrc: string;
};

export type ServerConfig = {
  id: string;
  name: string;
  type: string;
  url: string;
  username: string;
  password: string;
  hasPassword: boolean;
};

export function toSource(value: string | undefined): Source {
  return value === 'server' ? 'server' : 'local';
}

export function toPlaybackState(value: string | undefined): PlaybackState {
  return value === 'playing' || value === 'paused' ? value : 'stopped';
}

export function toRepeatMode(value: string | undefined): RepeatMode {
  return value === 'all' || value === 'one' ? value : 'off';
}
