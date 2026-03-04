<script lang="ts">
  import { LibraryService } from "../bindings/github.com/willfish/forte";

  const { onartist }: { onartist: (name: string) => void } = $props();

  type StatEntry = {
    name: string;
    secondLine: string;
    playCount: number;
    totalMs: number;
  };

  type RecentPlay = {
    trackId: number;
    title: string;
    artist: string;
    album: string;
    durationMs: number;
    playedAt: string;
  };

  const periods = [
    { value: '7d', label: '7 days' },
    { value: '30d', label: '30 days' },
    { value: '12m', label: '12 months' },
    { value: 'all', label: 'All time' },
  ];

  let period = $state('30d');
  let topArtists = $state<StatEntry[]>([]);
  let topAlbums = $state<StatEntry[]>([]);
  let topTracks = $state<StatEntry[]>([]);
  let recentPlays = $state<RecentPlay[]>([]);
  let loading = $state(false);

  async function loadStats() {
    loading = true;
    try {
      const [artists, albums, tracks, recent] = await Promise.all([
        LibraryService.GetTopArtists(period, 10),
        LibraryService.GetTopAlbums(period, 10),
        LibraryService.GetTopTracks(period, 10),
        LibraryService.GetRecentlyPlayed(50),
      ]);
      topArtists = artists || [];
      topAlbums = albums || [];
      topTracks = tracks || [];
      recentPlays = recent || [];
    } catch {
      topArtists = [];
      topAlbums = [];
      topTracks = [];
      recentPlays = [];
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    // Re-run when period changes.
    period;
    loadStats();
  });

  function formatDuration(ms: number): string {
    const totalSec = Math.floor(ms / 1000);
    const hours = Math.floor(totalSec / 3600);
    const mins = Math.floor((totalSec % 3600) / 60);
    if (hours > 0) return `${hours}h ${mins}m`;
    return `${mins}m`;
  }

  function formatTrackDuration(ms: number): string {
    const totalSec = Math.floor(ms / 1000);
    const mins = Math.floor(totalSec / 60);
    const secs = totalSec % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  function relativeTime(dateStr: string): string {
    const date = new Date(dateStr + 'Z');
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins}m ago`;
    const diffHours = Math.floor(diffMins / 60);
    if (diffHours < 24) return `${diffHours}h ago`;
    const diffDays = Math.floor(diffHours / 24);
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  }

  const hasData = $derived(
    topArtists.length > 0 || topAlbums.length > 0 || topTracks.length > 0 || recentPlays.length > 0
  );
</script>

<div class="stats-view">
  <div class="toolbar">
    <h2>Listening Stats</h2>
    <div class="period-tabs">
      {#each periods as p}
        <button
          class="period-tab"
          class:active={period === p.value}
          onclick={() => { period = p.value; }}
        >
          {p.label}
        </button>
      {/each}
    </div>
  </div>

  {#if loading}
    <div class="empty-state">Loading...</div>
  {:else if !hasData}
    <div class="empty-state">No listening history yet</div>
  {:else}
    <div class="stats-grid">
      {#if topArtists.length > 0}
        <section class="stat-section">
          <h3>Top Artists</h3>
          <ol class="stat-list">
            {#each topArtists as entry, i}
              <li class="stat-row">
                <span class="rank">{i + 1}</span>
                <div class="stat-info">
                  <button class="stat-name artist-link" onclick={() => onartist(entry.name)}>{entry.name}</button>
                </div>
                <span class="stat-meta">{entry.playCount} plays</span>
                <span class="stat-duration">{formatDuration(entry.totalMs)}</span>
              </li>
            {/each}
          </ol>
        </section>
      {/if}

      {#if topAlbums.length > 0}
        <section class="stat-section">
          <h3>Top Albums</h3>
          <ol class="stat-list">
            {#each topAlbums as entry, i}
              <li class="stat-row">
                <span class="rank">{i + 1}</span>
                <div class="stat-info">
                  <span class="stat-name">{entry.name}</span>
                  {#if entry.secondLine}
                    <button class="stat-secondary artist-link" onclick={() => onartist(entry.secondLine)}>{entry.secondLine}</button>
                  {/if}
                </div>
                <span class="stat-meta">{entry.playCount} plays</span>
                <span class="stat-duration">{formatDuration(entry.totalMs)}</span>
              </li>
            {/each}
          </ol>
        </section>
      {/if}

      {#if topTracks.length > 0}
        <section class="stat-section">
          <h3>Top Tracks</h3>
          <ol class="stat-list">
            {#each topTracks as entry, i}
              <li class="stat-row">
                <span class="rank">{i + 1}</span>
                <div class="stat-info">
                  <span class="stat-name">{entry.name}</span>
                  {#if entry.secondLine}
                    <button class="stat-secondary artist-link" onclick={() => onartist(entry.secondLine)}>{entry.secondLine}</button>
                  {/if}
                </div>
                <span class="stat-meta">{entry.playCount} plays</span>
                <span class="stat-duration">{formatDuration(entry.totalMs)}</span>
              </li>
            {/each}
          </ol>
        </section>
      {/if}
    </div>

    {#if recentPlays.length > 0}
      <section class="stat-section recent-section">
        <h3>Recently Played</h3>
        <ol class="stat-list">
          {#each recentPlays as play}
            <li class="stat-row">
              <div class="stat-info">
                <span class="stat-name">{play.title}</span>
                <span class="stat-secondary">
                  <button class="artist-link inline-link" onclick={() => onartist(play.artist)}>{play.artist}</button>{play.album ? ` - ${play.album}` : ''}
                </span>
              </div>
              <span class="stat-duration">{formatTrackDuration(play.durationMs)}</span>
              <span class="stat-meta">{relativeTime(play.playedAt)}</span>
            </li>
          {/each}
        </ol>
      </section>
    {/if}
  {/if}
</div>

<style>
  .stats-view {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-shrink: 0;
  }

  .toolbar h2 {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .period-tabs {
    display: flex;
    gap: 0.25rem;
    background: var(--bg-secondary, var(--bg-hover));
    border-radius: 6px;
    padding: 0.2rem;
  }

  .period-tab {
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.8rem;
    padding: 0.3rem 0.6rem;
    border-radius: 4px;
    cursor: pointer;
  }

  .period-tab:hover {
    color: var(--text-primary);
  }

  .period-tab.active {
    background: var(--bg-active);
    color: var(--accent);
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-secondary);
    font-size: 0.9rem;
    min-height: 200px;
  }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1.5rem;
  }

  .stat-section {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .stat-section h3 {
    margin: 0;
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .recent-section {
    margin-top: 0.5rem;
  }

  .stat-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .stat-row {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.4rem 0;
    border-bottom: 1px solid var(--border);
  }

  .stat-row:last-child {
    border-bottom: none;
  }

  .rank {
    width: 1.5rem;
    text-align: right;
    font-size: 0.8rem;
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .stat-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0.1rem;
  }

  .stat-name {
    font-size: 0.85rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stat-secondary {
    font-size: 0.75rem;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .stat-meta {
    font-size: 0.75rem;
    color: var(--text-secondary);
    flex-shrink: 0;
    white-space: nowrap;
  }

  .stat-duration {
    font-size: 0.75rem;
    color: var(--text-secondary);
    flex-shrink: 0;
    white-space: nowrap;
  }

  .artist-link {
    background: transparent;
    border: none;
    padding: 0;
    cursor: pointer;
    text-align: left;
    font-size: inherit;
    color: inherit;
  }

  .artist-link:hover {
    color: var(--accent);
    text-decoration: underline;
  }

  .inline-link {
    display: inline;
  }
</style>
