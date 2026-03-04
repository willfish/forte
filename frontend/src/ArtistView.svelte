<script lang="ts">
  import { LibraryService } from "../bindings/github.com/willfish/forte";

  type ArtistInfo = {
    name: string;
    bio: string;
    imageUrl: string;
    area: string;
    type: string;
    activeYears: string;
    similar: { name: string; inLibrary: boolean }[];
    albums: { id: number; title: string; artist: string; year: number; trackCount: number; source: string; serverId: string }[];
    tags: string;
  };

  const { artistName, onback, onalbum, onartist }: {
    artistName: string;
    onback: () => void;
    onalbum: (albumId: number) => void;
    onartist: (name: string) => void;
  } = $props();

  let info = $state<ArtistInfo | null>(null);
  let loading = $state(true);
  let error = $state('');
  let artworkCache = $state<Record<number, string>>({});

  async function loadArtist() {
    loading = true;
    error = '';
    try {
      const result = await LibraryService.GetArtistInfo(artistName);
      info = result;

      // Load artwork for each album.
      const cache: Record<number, string> = {};
      if (result.albums) {
        for (const album of result.albums) {
          try {
            const art = await LibraryService.AlbumArtwork(album.id);
            if (art) cache[album.id] = art;
          } catch {
            // ignore missing artwork
          }
        }
      }
      artworkCache = cache;
    } catch (e: any) {
      error = e?.message || 'Failed to load artist info';
    } finally {
      loading = false;
    }
  }

  $effect(() => {
    loadArtist();
  });
</script>

<div class="artist-view">
  <button class="back-btn" onclick={onback}>
    <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
      <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/>
    </svg>
    Back
  </button>

  {#if loading}
    <div class="loading">Loading artist info...</div>
  {:else if error}
    <div class="error-state">{error}</div>
  {:else if info}
    <div class="artist-header">
      {#if info.imageUrl}
        <img class="artist-image" src={info.imageUrl} alt="{info.name}" />
      {:else}
        <div class="artist-image-placeholder">
          <svg viewBox="0 0 24 24" width="48" height="48" fill="currentColor" opacity="0.3">
            <path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/>
          </svg>
        </div>
      {/if}
      <div class="artist-meta">
        <h1 class="artist-name">{info.name}</h1>
        <div class="artist-details">
          {#if info.type}
            <span class="detail">{info.type}</span>
          {/if}
          {#if info.area}
            <span class="detail">{info.area}</span>
          {/if}
          {#if info.activeYears}
            <span class="detail">{info.activeYears}</span>
          {/if}
        </div>
        {#if info.tags}
          <div class="tags">
            {#each info.tags.split(', ') as tag}
              <span class="tag">{tag}</span>
            {/each}
          </div>
        {/if}
      </div>
    </div>

    {#if info.bio}
      <section class="bio-section">
        <h3>Biography</h3>
        <p class="bio-text">{info.bio}</p>
      </section>
    {/if}

    {#if info.albums && info.albums.length > 0}
      <section class="albums-section">
        <h3>Albums ({info.albums.length})</h3>
        <div class="albums-grid">
          {#each info.albums as album (album.id)}
            <button class="album-card" onclick={() => onalbum(album.id)}>
              {#if artworkCache[album.id]}
                <img class="album-art" src={artworkCache[album.id]} alt="{album.title}" />
              {:else}
                <div class="album-art-placeholder">
                  <svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor" opacity="0.3">
                    <path d="M12 3v10.55c-.59-.34-1.27-.55-2-.55C7.79 13 6 14.79 6 17s1.79 4 4 4 4-1.79 4-4V7h4V3h-6z"/>
                  </svg>
                </div>
              {/if}
              <span class="album-title">{album.title}</span>
              {#if album.year > 0}
                <span class="album-year">{album.year}</span>
              {/if}
            </button>
          {/each}
        </div>
      </section>
    {/if}

    {#if info.similar && info.similar.length > 0}
      <section class="similar-section">
        <h3>Similar Artists</h3>
        <div class="similar-list">
          {#each info.similar as sim}
            {#if sim.inLibrary}
              <button class="similar-link" onclick={() => onartist(sim.name)}>
                {sim.name}
              </button>
            {:else}
              <span class="similar-name">{sim.name}</span>
            {/if}
          {/each}
        </div>
      </section>
    {/if}

    {#if !info.bio && (!info.albums || info.albums.length === 0)}
      <div class="empty-state">No information available for this artist.</div>
    {/if}
  {/if}
</div>

<style>
  .artist-view {
    display: flex;
    flex-direction: column;
    height: 100%;
    gap: 1.5rem;
    animation: view-enter 0.2s ease-out;
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
  }

  .back-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .loading, .error-state, .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 1;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .artist-header {
    display: flex;
    gap: 1.5rem;
    flex-shrink: 0;
  }

  .artist-image {
    width: 200px;
    height: 200px;
    border-radius: 50%;
    object-fit: cover;
    flex-shrink: 0;
  }

  .artist-image-placeholder {
    width: 200px;
    height: 200px;
    border-radius: 50%;
    background: var(--bg-hover);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .artist-meta {
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    padding-bottom: 0.5rem;
    gap: 0.5rem;
  }

  .artist-name {
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
  }

  .artist-details {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .detail {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .detail + .detail::before {
    content: '\00B7';
    margin-right: 0.5rem;
  }

  .tags {
    display: flex;
    gap: 0.35rem;
    flex-wrap: wrap;
  }

  .tag {
    font-size: 0.7rem;
    padding: 0.15rem 0.5rem;
    border-radius: 10px;
    background: var(--bg-hover);
    color: var(--text-secondary);
  }

  section h3 {
    margin: 0 0 0.75rem;
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .bio-text {
    font-size: 0.9rem;
    color: var(--text-primary);
    line-height: 1.6;
    margin: 0;
  }

  .albums-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 1rem;
  }

  .album-card {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    border: none;
    background: transparent;
    cursor: pointer;
    text-align: left;
    padding: 0.5rem;
    border-radius: 6px;
    color: inherit;
  }

  .album-card:hover {
    background: var(--bg-hover);
  }

  .album-art {
    width: 100%;
    aspect-ratio: 1;
    border-radius: 6px;
    object-fit: cover;
  }

  .album-art-placeholder {
    width: 100%;
    aspect-ratio: 1;
    border-radius: 6px;
    background: var(--bg-hover);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--text-secondary);
  }

  .album-title {
    font-size: 0.8rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .album-year {
    font-size: 0.75rem;
    color: var(--text-secondary);
  }

  .similar-list {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .similar-link {
    font-size: 0.85rem;
    color: var(--accent);
    background: transparent;
    border: 1px solid var(--border);
    padding: 0.3rem 0.6rem;
    border-radius: 4px;
    cursor: pointer;
  }

  .similar-link:hover {
    background: var(--bg-hover);
  }

  .similar-name {
    font-size: 0.85rem;
    color: var(--text-secondary);
    padding: 0.3rem 0.6rem;
    border: 1px solid var(--border);
    border-radius: 4px;
  }
</style>
