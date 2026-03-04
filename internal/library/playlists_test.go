package library

import "testing"

func TestCreateAndGetPlaylists(t *testing.T) {
	db := openTestDB(t)

	id, err := db.CreatePlaylist("My Playlist")
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	playlists, err := db.GetPlaylists()
	if err != nil {
		t.Fatalf("GetPlaylists: %v", err)
	}
	if len(playlists) != 1 {
		t.Fatalf("got %d playlists, want 1", len(playlists))
	}
	if playlists[0].Name != "My Playlist" {
		t.Errorf("name = %q, want 'My Playlist'", playlists[0].Name)
	}
}

func TestGetPlaylistsOrdering(t *testing.T) {
	db := openTestDB(t)
	_, _ = db.CreatePlaylist("Zebra")
	_, _ = db.CreatePlaylist("Alpha")

	playlists, err := db.GetPlaylists()
	if err != nil {
		t.Fatalf("GetPlaylists: %v", err)
	}
	if len(playlists) != 2 {
		t.Fatalf("got %d playlists", len(playlists))
	}
	if playlists[0].Name != "Alpha" {
		t.Errorf("first = %q, want 'Alpha'", playlists[0].Name)
	}
}

func TestGetPlaylistsEmpty(t *testing.T) {
	db := openTestDB(t)
	playlists, err := db.GetPlaylists()
	if err != nil {
		t.Fatalf("GetPlaylists: %v", err)
	}
	if len(playlists) != 0 {
		t.Errorf("expected empty, got %d", len(playlists))
	}
}

func TestRenamePlaylist(t *testing.T) {
	db := openTestDB(t)
	id, _ := db.CreatePlaylist("Old Name")

	if err := db.RenamePlaylist(id, "New Name"); err != nil {
		t.Fatalf("RenamePlaylist: %v", err)
	}

	playlists, _ := db.GetPlaylists()
	if playlists[0].Name != "New Name" {
		t.Errorf("name = %q, want 'New Name'", playlists[0].Name)
	}
}

func TestDeletePlaylist(t *testing.T) {
	db := openTestDB(t)
	id, _ := db.CreatePlaylist("To Delete")

	if err := db.DeletePlaylist(id); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}

	playlists, _ := db.GetPlaylists()
	if len(playlists) != 0 {
		t.Errorf("expected empty after delete, got %d", len(playlists))
	}
}

func TestDeletePlaylistCascade(t *testing.T) {
	db := openTestDB(t)
	id, _ := db.CreatePlaylist("Playlist")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExec(t, db, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (1, 1, 'T', '/a.flac')`)
	_ = db.AddTrackToPlaylist(id, 1)

	if err := db.DeletePlaylist(id); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}

	var count int
	_ = db.QueryRow("SELECT COUNT(*) FROM playlist_tracks WHERE playlist_id = ?", id).Scan(&count)
	if count != 0 {
		t.Errorf("playlist_tracks count = %d after delete, want 0", count)
	}
}

func TestPlaylistTracksCRUD(t *testing.T) {
	db := openTestDB(t)
	plID, _ := db.CreatePlaylist("Playlist")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title) VALUES (1, 1, 'Album')")
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, duration_ms, file_path)
		VALUES (1, 1, 1, 'Track 1', 180000, '/a.flac')`)
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, duration_ms, file_path)
		VALUES (2, 1, 1, 'Track 2', 200000, '/b.flac')`)

	// Add tracks.
	if err := db.AddTrackToPlaylist(plID, 1); err != nil {
		t.Fatalf("AddTrackToPlaylist: %v", err)
	}
	if err := db.AddTrackToPlaylist(plID, 2); err != nil {
		t.Fatalf("AddTrackToPlaylist: %v", err)
	}

	tracks, err := db.GetPlaylistTracks(plID)
	if err != nil {
		t.Fatalf("GetPlaylistTracks: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
	if tracks[0].Title != "Track 1" || tracks[0].Position != 0 {
		t.Errorf("track[0] = %q pos %d", tracks[0].Title, tracks[0].Position)
	}
	if tracks[1].Title != "Track 2" || tracks[1].Position != 1 {
		t.Errorf("track[1] = %q pos %d", tracks[1].Title, tracks[1].Position)
	}

	// Remove first track.
	if err := db.RemoveTrackFromPlaylist(plID, 1); err != nil {
		t.Fatalf("RemoveTrackFromPlaylist: %v", err)
	}

	tracks, _ = db.GetPlaylistTracks(plID)
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks after remove, want 1", len(tracks))
	}
	if tracks[0].Title != "Track 2" {
		t.Errorf("remaining track = %q", tracks[0].Title)
	}
	if tracks[0].Position != 0 {
		t.Errorf("position = %d after reorder, want 0", tracks[0].Position)
	}
}

func TestGetPlaylistTracksEmpty(t *testing.T) {
	db := openTestDB(t)
	plID, _ := db.CreatePlaylist("Empty")

	tracks, err := db.GetPlaylistTracks(plID)
	if err != nil {
		t.Fatalf("GetPlaylistTracks: %v", err)
	}
	if len(tracks) != 0 {
		t.Errorf("expected empty, got %d", len(tracks))
	}
}

func TestMoveTrackInPlaylistDown(t *testing.T) {
	db := openTestDB(t)
	plID, _ := db.CreatePlaylist("Playlist")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExec(t, db, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (1, 1, 'T1', '/1.flac')`)
	mustExec(t, db, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (2, 1, 'T2', '/2.flac')`)
	mustExec(t, db, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (3, 1, 'T3', '/3.flac')`)
	_ = db.AddTrackToPlaylist(plID, 1)
	_ = db.AddTrackToPlaylist(plID, 2)
	_ = db.AddTrackToPlaylist(plID, 3)

	// Move first track to last position.
	if err := db.MoveTrackInPlaylist(plID, 0, 2); err != nil {
		t.Fatalf("MoveTrackInPlaylist: %v", err)
	}

	tracks, _ := db.GetPlaylistTracks(plID)
	if len(tracks) != 3 {
		t.Fatalf("got %d tracks", len(tracks))
	}
	// Expected order: T2, T3, T1
	if tracks[0].Title != "T2" {
		t.Errorf("pos 0 = %q, want T2", tracks[0].Title)
	}
	if tracks[1].Title != "T3" {
		t.Errorf("pos 1 = %q, want T3", tracks[1].Title)
	}
	if tracks[2].Title != "T1" {
		t.Errorf("pos 2 = %q, want T1", tracks[2].Title)
	}
}

func TestMoveTrackInPlaylistUp(t *testing.T) {
	db := openTestDB(t)
	plID, _ := db.CreatePlaylist("Playlist")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExec(t, db, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (1, 1, 'T1', '/1.flac')`)
	mustExec(t, db, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (2, 1, 'T2', '/2.flac')`)
	mustExec(t, db, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (3, 1, 'T3', '/3.flac')`)
	_ = db.AddTrackToPlaylist(plID, 1)
	_ = db.AddTrackToPlaylist(plID, 2)
	_ = db.AddTrackToPlaylist(plID, 3)

	// Move last track to first position.
	if err := db.MoveTrackInPlaylist(plID, 2, 0); err != nil {
		t.Fatalf("MoveTrackInPlaylist: %v", err)
	}

	tracks, _ := db.GetPlaylistTracks(plID)
	// Expected order: T3, T1, T2
	if tracks[0].Title != "T3" {
		t.Errorf("pos 0 = %q, want T3", tracks[0].Title)
	}
	if tracks[1].Title != "T1" {
		t.Errorf("pos 1 = %q, want T1", tracks[1].Title)
	}
	if tracks[2].Title != "T2" {
		t.Errorf("pos 2 = %q, want T2", tracks[2].Title)
	}
}

func TestAddDuplicateTrackToPlaylist(t *testing.T) {
	db := openTestDB(t)
	plID, _ := db.CreatePlaylist("Playlist")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExec(t, db, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (1, 1, 'T', '/a.flac')`)

	_ = db.AddTrackToPlaylist(plID, 1)
	// Adding same track again should be ignored (INSERT OR IGNORE).
	_ = db.AddTrackToPlaylist(plID, 1)

	tracks, _ := db.GetPlaylistTracks(plID)
	if len(tracks) != 1 {
		t.Errorf("expected 1 track (no duplicate), got %d", len(tracks))
	}
}
