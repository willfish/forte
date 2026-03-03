<script lang="ts">
  import { LibraryService, PlayerService } from "../bindings/github.com/willfish/forte";

  type PlaylistSummary = { id: number; name: string };
  type Track = {
    trackId: number;
    title: string;
    artist: string;
    album: string;
    durationMs: number;
    filePath: string;
    position: number;
  };

  let playlists = $state<PlaylistSummary[]>([]);
  let selectedId = $state<number | null>(null);
  let selectedName = $state('');
  let tracks = $state<Track[]>([]);
  let newName = $state('');
  let editingId = $state<number | null>(null);
  let editName = $state('');

  async function loadPlaylists() {
    playlists = ((await LibraryService.GetPlaylists()) || []).map((p: any) => ({
      id: p.id,
      name: p.name,
    }));
  }

  async function loadTracks(id: number) {
    tracks = ((await LibraryService.GetPlaylistTracks(id)) || []).map((t: any) => ({
      trackId: t.trackId,
      title: t.title,
      artist: t.artist,
      album: t.album,
      durationMs: t.durationMs,
      filePath: t.filePath,
      position: t.position,
    }));
  }

  $effect(() => {
    loadPlaylists();
  });

  async function selectPlaylist(p: PlaylistSummary) {
    selectedId = p.id;
    selectedName = p.name;
    await loadTracks(p.id);
  }

  async function createPlaylist() {
    const name = newName.trim();
    if (!name) return;
    await LibraryService.CreatePlaylist(name);
    newName = '';
    await loadPlaylists();
  }

  async function startRename(p: PlaylistSummary) {
    editingId = p.id;
    editName = p.name;
  }

  async function finishRename() {
    if (editingId === null) return;
    const name = editName.trim();
    if (name) {
      await LibraryService.RenamePlaylist(editingId, name);
      if (editingId === selectedId) selectedName = name;
      await loadPlaylists();
    }
    editingId = null;
  }

  async function deletePlaylist(id: number) {
    await LibraryService.DeletePlaylist(id);
    if (id === selectedId) {
      selectedId = null;
      tracks = [];
    }
    await loadPlaylists();
  }

  async function removeTrack(trackId: number) {
    if (selectedId === null) return;
    await LibraryService.RemoveTrackFromPlaylist(selectedId, trackId);
    await loadTracks(selectedId);
  }

  async function playFromTrack(index: number) {
    const queueTracks = tracks.map(t => ({
      trackId: t.trackId,
      title: t.title,
      artist: t.artist,
      album: t.album,
      durationMs: t.durationMs,
      filePath: t.filePath,
    }));
    if (queueTracks.length > 0) {
      await PlayerService.PlayQueue(queueTracks, index);
    }
  }

  // Drag and drop reorder.
  let dragIndex = $state<number | null>(null);

  function handleDragStart(index: number, e: DragEvent) {
    dragIndex = index;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
    }
  }

  function handleDragOver(e: DragEvent) {
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
  }

  async function handleDrop(targetIndex: number) {
    if (dragIndex === null || dragIndex === targetIndex || selectedId === null) return;
    await LibraryService.MoveTrackInPlaylist(selectedId, dragIndex, targetIndex);
    dragIndex = null;
    await loadTracks(selectedId);
  }

  function formatDuration(ms: number): string {
    const totalSeconds = Math.floor(ms / 1000);
    const m = Math.floor(totalSeconds / 60);
    const s = totalSeconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  }
</script>

{#if selectedId !== null}
  <div class="playlist-detail">
    <button class="back-btn" onclick={() => { selectedId = null; tracks = []; }}>
      <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor">
        <path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/>
      </svg>
      Playlists
    </button>

    <h2 class="playlist-title">{selectedName}</h2>
    <p class="track-count">{tracks.length} track{tracks.length !== 1 ? 's' : ''}</p>

    {#if tracks.length === 0}
      <p class="empty-msg">No tracks in this playlist yet.</p>
    {:else}
      <div class="track-list">
        {#each tracks as track, i (track.trackId)}
          <div
            class="track-row"
            draggable="true"
            ondragstart={(e) => handleDragStart(i, e)}
            ondragover={handleDragOver}
            ondrop={() => handleDrop(i)}
            role="listitem"
          >
            <span class="drag-handle">
              <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor" opacity="0.4">
                <path d="M3 15h18v-2H3v2zm0 4h18v-2H3v2zm0-8h18V9H3v2zm0-6v2h18V5H3z"/>
              </svg>
            </span>
            <button class="track-btn" ondblclick={() => playFromTrack(i)}>
              <span class="track-title">{track.title}</span>
              <span class="track-artist">{track.artist}</span>
              <span class="track-duration">{formatDuration(track.durationMs)}</span>
            </button>
            <button class="remove-btn" onclick={() => removeTrack(track.trackId)} aria-label="Remove">
              <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                <path d="M19 6.41 17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/>
              </svg>
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </div>
{:else}
  <div class="playlist-list">
    <h2>Playlists</h2>

    <div class="create-form">
      <input
        type="text"
        bind:value={newName}
        placeholder="New playlist name..."
        onkeydown={(e) => { if (e.key === 'Enter') createPlaylist(); }}
      />
      <button class="create-btn" onclick={createPlaylist} disabled={!newName.trim()}>Create</button>
    </div>

    {#if playlists.length === 0}
      <p class="empty-msg">No playlists yet. Create one above.</p>
    {:else}
      <ul class="playlists">
        {#each playlists as p (p.id)}
          <li class="playlist-item">
            {#if editingId === p.id}
              <input
                class="rename-input"
                type="text"
                bind:value={editName}
                onkeydown={(e) => { if (e.key === 'Enter') finishRename(); if (e.key === 'Escape') editingId = null; }}
                onblur={finishRename}
              />
            {:else}
              <button class="playlist-btn" onclick={() => selectPlaylist(p)}>
                <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor" opacity="0.5">
                  <path d="M15 6H3v2h12V6zm0 4H3v2h12v-2zM3 16h8v-2H3v2zM17 6v8.18c-.31-.11-.65-.18-1-.18-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3V8h3V6h-5z"/>
                </svg>
                {p.name}
              </button>
              <div class="playlist-actions">
                <button class="action-btn" onclick={() => startRename(p)} aria-label="Rename">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                    <path d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04a.996.996 0 0 0 0-1.41l-2.34-2.34a.996.996 0 0 0-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z"/>
                  </svg>
                </button>
                <button class="action-btn delete" onclick={() => deletePlaylist(p.id)} aria-label="Delete">
                  <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                    <path d="M6 19c0 1.1.9 2 2 2h8c1.1 0 2-.9 2-2V7H6v12zM19 4h-3.5l-1-1h-5l-1 1H5v2h14V4z"/>
                  </svg>
                </button>
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
{/if}

<style>
  h2 {
    font-size: 1.5rem;
    font-weight: 600;
    color: var(--text-primary);
    margin: 0 0 1rem;
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
    margin-bottom: 1rem;
  }

  .back-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .playlist-title {
    margin-bottom: 0.25rem;
  }

  .track-count {
    font-size: 0.85rem;
    color: var(--text-secondary);
    margin: 0 0 1rem;
  }

  .empty-msg {
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .create-form {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.5rem;
  }

  .create-form input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: transparent;
    color: var(--text-primary);
    font-size: 0.9rem;
  }

  .create-form input::placeholder {
    color: var(--text-secondary);
  }

  .create-btn {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 6px;
    background: var(--accent);
    color: #fff;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .create-btn:disabled {
    opacity: 0.4;
    cursor: default;
  }

  .create-btn:hover:not(:disabled) {
    filter: brightness(1.15);
  }

  .playlists {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .playlist-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.25rem 0;
  }

  .playlist-btn {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.6rem 0.75rem;
    border: none;
    border-radius: 6px;
    background: transparent;
    color: var(--text-primary);
    cursor: pointer;
    font-size: 0.9rem;
    text-align: left;
  }

  .playlist-btn:hover {
    background: var(--bg-hover);
  }

  .playlist-actions {
    display: flex;
    gap: 0.25rem;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .playlist-item:hover .playlist-actions {
    opacity: 1;
  }

  .action-btn {
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
  }

  .action-btn:hover {
    background: var(--bg-hover);
    color: var(--text-primary);
  }

  .action-btn.delete:hover {
    color: #e55;
  }

  .rename-input {
    flex: 1;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--accent);
    border-radius: 6px;
    background: transparent;
    color: var(--text-primary);
    font-size: 0.9rem;
  }

  .track-list {
    display: flex;
    flex-direction: column;
  }

  .track-row {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    border-radius: 4px;
  }

  .track-row:hover {
    background: var(--bg-hover);
  }

  .drag-handle {
    cursor: grab;
    padding: 0.5rem 0.25rem;
    display: flex;
    align-items: center;
  }

  .track-btn {
    flex: 1;
    display: grid;
    grid-template-columns: 1fr auto 3.5rem;
    align-items: center;
    gap: 0.5rem;
    padding: 0.5rem 0.5rem;
    border: none;
    background: transparent;
    color: inherit;
    cursor: pointer;
    text-align: left;
  }

  .track-title {
    font-size: 0.9rem;
    color: var(--text-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
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

  .remove-btn {
    width: 28px;
    height: 28px;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--text-secondary);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 0;
    opacity: 0;
    transition: opacity 0.15s;
  }

  .track-row:hover .remove-btn {
    opacity: 1;
  }

  .remove-btn:hover {
    color: #e55;
    background: var(--bg-hover);
  }
</style>
