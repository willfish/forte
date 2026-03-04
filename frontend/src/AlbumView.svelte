<script lang="ts">
  import { LibraryService, PlayerService } from "../bindings/github.com/willfish/forte";
  import { isServerOnline, onServerStatusChange } from './lib/stores';

  type Track = {
    trackId: number;
    title: string;
    artist: string;
    trackNumber: number;
    discNumber: number;
    durationMs: number;
    filePath: string;
    source: string;
    serverId: string;
  };

  type AlbumInfo = {
    title: string;
    artist: string;
    year: number;
    trackCount: number;
    artworkSrc: string;
    totalDurationMs: number;
  };

  const { albumId, onback, onartist }: { albumId: number; onback: () => void; onartist: (name: string) => void } = $props();

  let albumInfo = $state<AlbumInfo>({ title: '', artist: '', year: 0, trackCount: 0, artworkSrc: '', totalDurationMs: 0 });
  let tracks = $state<Track[]>([]);
  let currentFilePath = $state('');
  let pollTimer: ReturnType<typeof setInterval> | null = null;
  let statusVersion = $state(0);

  $effect(() => {
    return onServerStatusChange(() => { statusVersion++; });
  });

  async function loadAlbum() {
    const [trackList, artworkSrc] = await Promise.all([
      LibraryService.GetAlbumTracks(albumId),
      LibraryService.AlbumArtwork(albumId),
    ]);

    tracks = (trackList || []).map((t: any) => ({
      trackId: t.trackId,
      title: t.title,
      artist: t.artist,
      trackNumber: t.trackNumber,
      discNumber: t.discNumber,
      durationMs: t.durationMs,
      filePath: t.filePath,
      source: t.source || 'local',
      serverId: t.serverId || '',
    }));

    const totalMs = tracks.reduce((sum, t) => sum + t.durationMs, 0);
    const first = tracks[0];
    albumInfo = {
      title: first?.title ? '' : '',
      artist: first?.artist || '',
      year: 0,
      trackCount: tracks.length,
      artworkSrc: artworkSrc || '',
      totalDurationMs: totalMs,
    };

    // Get album metadata from the album list
    const albums = await LibraryService.GetAlbums('title', 'asc', '');
    const match = (albums || []).find((a: any) => a.id === albumId);
    if (match) {
      albumInfo.title = match.title;
      albumInfo.artist = match.artist;
      albumInfo.year = match.year;
      albumInfo.trackCount = match.trackCount;
    }
  }

  function startPolling() {
    if (pollTimer) return;
    pollTimer = setInterval(async () => {
      const state = await PlayerService.State();
      if (state !== 'stopped') {
        currentFilePath = await PlayerService.MediaPath();
      } else {
        currentFilePath = '';
      }
    }, 500);
  }

  function stopPolling() {
    if (pollTimer) {
      clearInterval(pollTimer);
      pollTimer = null;
    }
  }

  $effect(() => {
    loadAlbum();
    startPolling();
    return () => stopPolling();
  });

  async function playFromTrack(index: number) {
    const queueTracks = tracks.map(t => ({
      trackId: t.trackId,
      title: t.title,
      artist: t.artist,
      album: albumInfo.title,
      durationMs: t.durationMs,
      filePath: t.filePath,
    }));
    if (queueTracks.length > 0) {
      await PlayerService.PlayQueue(queueTracks, index);
    }
  }

  function formatDuration(ms: number): string {
    const totalSeconds = Math.floor(ms / 1000);
    const m = Math.floor(totalSeconds / 60);
    const s = totalSeconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  }

  function formatTotalDuration(ms: number): string {
    const totalMinutes = Math.floor(ms / 60000);
    if (totalMinutes < 60) return `${totalMinutes} min`;
    const h = Math.floor(totalMinutes / 60);
    const m = totalMinutes % 60;
    return `${h} hr ${m} min`;
  }

  // Group tracks by disc number for multi-disc display.
  const discs = $derived(() => {
    const map = new Map<number, Track[]>();
    for (const t of tracks) {
      const disc = t.discNumber || 1;
      if (!map.has(disc)) map.set(disc, []);
      map.get(disc)!.push(t);
    }
    return map;
  });

  const isMultiDisc = $derived(() => discs().size > 1);
</script>

<div class="album-view">
  <button class="back-btn" onclick={onback}>
    <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
      <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/>
    </svg>
    Albums
  </button>

  <div class="album-header">
    {#if albumInfo.artworkSrc}
      <img class="artwork" src={albumInfo.artworkSrc} alt="{albumInfo.title} cover" />
    {:else}
      <div class="artwork-placeholder">
        <svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor" opacity="0.3">
          <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55C7.79 13 6 14.79 6 17s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
        </svg>
      </div>
    {/if}
    <div class="album-meta">
      <h1 class="album-title">{albumInfo.title}</h1>
      <button class="album-artist artist-link" onclick={() => onartist(albumInfo.artist)}>{albumInfo.artist}</button>
      <p class="album-details">
        {#if albumInfo.year > 0}{albumInfo.year} &middot; {/if}
        {albumInfo.trackCount} track{albumInfo.trackCount !== 1 ? 's' : ''} &middot;
        {formatTotalDuration(albumInfo.totalDurationMs)}
      </p>
    </div>
  </div>

  <div class="track-list">
    {#each [...discs().entries()] as [discNum, discTracks] (discNum)}
      {#if isMultiDisc()}
        <div class="disc-separator">Disc {discNum}</div>
      {/if}
      {#each discTracks as track, i (track.trackId)}
        {@const globalIndex = tracks.indexOf(track)}
        {@const trackUnavailable = statusVersion >= 0 && track.serverId && !isServerOnline(track.serverId)}
        <button
          class="track-row"
          class:playing={track.filePath === currentFilePath}
          class:unavailable={trackUnavailable}
          ondblclick={() => playFromTrack(globalIndex)}
        >
          <span class="track-num">
            {#if track.filePath === currentFilePath}
              <svg viewBox="0 0 24 24" width="14" height="14" fill="var(--accent)">
                <path d="M3 9v6h4l5 5V4L7 9H3zm13.5 3A4.5 4.5 0 0 0 14 7.97v8.05c1.48-.73 2.5-2.25 2.5-4.02z"/>
              </svg>
            {:else}
              {track.trackNumber || i + 1}
            {/if}
          </span>
          <span class="track-title">
            {track.title}
            {#if track.source === 'server'}
              <svg class="server-icon" viewBox="0 0 24 24" width="10" height="10" fill="currentColor">
                <path d="M19.35 10.04A7.49 7.49 0 0 0 12 4C9.11 4 6.6 5.64 5.35 8.04A5.994 5.994 0 0 0 0 14c0 3.31 2.69 6 6 6h13c2.76 0 5-2.24 5-5 0-2.64-2.05-4.78-4.65-4.96z"/>
              </svg>
            {/if}
          </span>
          {#if track.artist !== albumInfo.artist}
            <span class="track-artist">{track.artist}</span>
          {:else}
            <span class="track-artist"></span>
          {/if}
          <span class="track-duration">{formatDuration(track.durationMs)}</span>
        </button>
      {/each}
    {/each}
  </div>
</div>

<style>
  .album-view {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .back-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.4rem 0.6rem;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.85rem;
    align-self: flex-start;
    margin-bottom: 1rem;
  }

  .back-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .album-header {
    display: flex;
    gap: 1.5rem;
    margin-bottom: 1.5rem;
    flex-shrink: 0;
  }

  .artwork {
    width: 200px;
    height: 200px;
    border-radius: 8px;
    object-fit: cover;
    flex-shrink: 0;
  }

  .artwork-placeholder {
    width: 200px;
    height: 200px;
    border-radius: 8px;
    background: var(--bg-hover);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .album-meta {
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    padding-bottom: 0.5rem;
  }

  .album-title {
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0 0 0.25rem;
  }

  .album-artist {
    font-size: 1rem;
    color: var(--text-secondary);
    margin: 0 0 0.5rem;
  }

  .artist-link {
    background: transparent;
    border: none;
    padding: 0;
    cursor: pointer;
    text-align: left;
  }

  .artist-link:hover {
    color: var(--accent);
    text-decoration: underline;
  }

  .album-details {
    font-size: 0.85rem;
    color: var(--text-secondary);
    margin: 0;
  }

  .track-list {
    flex: 1;
    overflow-y: auto;
  }

  .disc-separator {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-secondary);
    padding: 0.75rem 0.5rem 0.25rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .track-row {
    display: grid;
    grid-template-columns: 2.5rem 1fr auto 3.5rem;
    align-items: center;
    padding: 0.5rem;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: inherit;
    cursor: pointer;
    text-align: left;
    width: 100%;
    gap: 0.5rem;
  }

  .track-row:hover {
    background: var(--bg-hover);
  }

  .track-row.playing {
    background: var(--bg-active);
  }

  .track-row.playing .track-title {
    color: var(--accent);
  }

  .track-row.unavailable {
    opacity: 0.45;
  }

  .track-num {
    font-size: 0.85rem;
    color: var(--text-secondary);
    text-align: center;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .track-title {
    font-size: 0.9rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
  }

  .server-icon {
    opacity: 0.4;
    flex-shrink: 0;
  }

  .track-artist {
    font-size: 0.8rem;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .track-duration {
    font-size: 0.8rem;
    color: var(--text-secondary);
    text-align: right;
    font-variant-numeric: tabular-nums;
  }
</style>
