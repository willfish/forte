package library

import (
	"testing"
)

func TestGetAlbumsDefaultSort(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (2, 'Bjork')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (1, 1, 'OK Computer', 1997)")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (2, 2, 'Homogenic', 1997)")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (3, 1, 'Kid A', 2000)")

	albums, err := db.GetAlbums("title", "asc", "")
	if err != nil {
		t.Fatalf("GetAlbums: %v", err)
	}
	if len(albums) != 3 {
		t.Fatalf("got %d albums, want 3", len(albums))
	}
	if albums[0].Title != "Homogenic" {
		t.Errorf("first album = %q, want 'Homogenic'", albums[0].Title)
	}
}

func TestGetAlbumsSortByYear(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (1, 1, 'Kid A', 2000)")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (2, 1, 'OK Computer', 1997)")

	albums, err := db.GetAlbums("year", "desc", "")
	if err != nil {
		t.Fatalf("GetAlbums: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("got %d albums, want 2", len(albums))
	}
	if albums[0].Title != "Kid A" {
		t.Errorf("first album = %q, want 'Kid A'", albums[0].Title)
	}
}

func TestGetAlbumsSortByArtist(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (2, 'Bjork')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (1, 1, 'OK Computer', 1997)")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (2, 2, 'Homogenic', 1997)")

	albums, err := db.GetAlbums("artist", "asc", "")
	if err != nil {
		t.Fatalf("GetAlbums: %v", err)
	}
	if albums[0].Artist != "Bjork" {
		t.Errorf("first artist = %q, want 'Bjork'", albums[0].Artist)
	}
}

func TestGetAlbumsSourceFilter(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year, server_id) VALUES (1, 1, 'OK Computer', 1997, '')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year, server_id) VALUES (2, 1, 'Kid A', 2000, 'srv-1')")

	local, err := db.GetAlbums("title", "asc", "local")
	if err != nil {
		t.Fatalf("GetAlbums local: %v", err)
	}
	if len(local) != 1 {
		t.Fatalf("local: got %d albums, want 1", len(local))
	}
	if local[0].Title != "OK Computer" {
		t.Errorf("local album = %q", local[0].Title)
	}

	server, err := db.GetAlbums("title", "asc", "server")
	if err != nil {
		t.Fatalf("GetAlbums server: %v", err)
	}
	if len(server) != 1 {
		t.Fatalf("server: got %d albums, want 1", len(server))
	}
	if server[0].Title != "Kid A" {
		t.Errorf("server album = %q", server[0].Title)
	}
}

func TestGetAlbumsDedup(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	// Same album locally and from server - local should win.
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year, server_id) VALUES (1, 1, 'OK Computer', 1997, '')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year, server_id) VALUES (2, 1, 'OK Computer', 1997, 'srv-1')")

	albums, err := db.GetAlbums("title", "asc", "")
	if err != nil {
		t.Fatalf("GetAlbums: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("got %d albums, want 1 (deduped)", len(albums))
	}
	if albums[0].ServerID != "" {
		t.Error("expected local album to win dedup, got server album")
	}
}

func TestGetAlbumsEmpty(t *testing.T) {
	db := openTestDB(t)
	albums, err := db.GetAlbums("title", "asc", "")
	if err != nil {
		t.Fatalf("GetAlbums: %v", err)
	}
	if len(albums) != 0 {
		t.Errorf("expected empty, got %d albums", len(albums))
	}
}

func TestAlbumArtwork(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title) VALUES (1, 1, 'Album')")

	// No artwork yet.
	art, err := db.AlbumArtwork(1)
	if err != nil {
		t.Fatalf("AlbumArtwork: %v", err)
	}
	if art != "" {
		t.Errorf("expected empty, got %q", art)
	}

	// Set artwork.
	mustExec(t, db, "UPDATE albums SET artwork_blob = X'89504E47' WHERE id = 1")
	art, err = db.AlbumArtwork(1)
	if err != nil {
		t.Fatalf("AlbumArtwork: %v", err)
	}
	if art == "" {
		t.Fatal("expected non-empty artwork")
	}
	if art[:len("data:image/jpeg;base64,")] != "data:image/jpeg;base64," {
		t.Errorf("unexpected prefix: %q", art[:30])
	}
}

func TestAlbumArtworkMissingAlbum(t *testing.T) {
	db := openTestDB(t)
	art, err := db.AlbumArtwork(999)
	if err != nil {
		t.Fatalf("AlbumArtwork: %v", err)
	}
	if art != "" {
		t.Errorf("expected empty for missing album, got %q", art)
	}
}

func TestGetAlbumTracks(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (1, 1, 'OK Computer', 1997)")
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, track_number, disc_number, duration_ms, file_path)
		VALUES (1, 1, 1, 'Airbag', 1, 1, 282000, '/music/airbag.flac')`)
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, track_number, disc_number, duration_ms, file_path)
		VALUES (2, 1, 1, 'Paranoid Android', 2, 1, 386000, '/music/paranoid.flac')`)

	tracks, err := db.GetAlbumTracks(1)
	if err != nil {
		t.Fatalf("GetAlbumTracks: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
	if tracks[0].Title != "Airbag" {
		t.Errorf("first track = %q, want 'Airbag'", tracks[0].Title)
	}
	if tracks[0].Artist != "Radiohead" {
		t.Errorf("artist = %q, want 'Radiohead'", tracks[0].Artist)
	}
	if tracks[1].TrackNumber != 2 {
		t.Errorf("track number = %d, want 2", tracks[1].TrackNumber)
	}
}

func TestGetAlbumTracksEmpty(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title) VALUES (1, 1, 'Album')")

	tracks, err := db.GetAlbumTracks(1)
	if err != nil {
		t.Fatalf("GetAlbumTracks: %v", err)
	}
	if len(tracks) != 0 {
		t.Errorf("expected empty, got %d tracks", len(tracks))
	}
}

func TestGetAlbumTracksDiscOrdering(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title) VALUES (1, 1, 'Double Album')")
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, track_number, disc_number, file_path)
		VALUES (1, 1, 1, 'Disc 2 Track 1', 1, 2, '/music/d2t1.flac')`)
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, track_number, disc_number, file_path)
		VALUES (2, 1, 1, 'Disc 1 Track 1', 1, 1, '/music/d1t1.flac')`)

	tracks, err := db.GetAlbumTracks(1)
	if err != nil {
		t.Fatalf("GetAlbumTracks: %v", err)
	}
	if tracks[0].DiscNumber != 1 {
		t.Errorf("first track disc = %d, want 1", tracks[0].DiscNumber)
	}
	if tracks[1].DiscNumber != 2 {
		t.Errorf("second track disc = %d, want 2", tracks[1].DiscNumber)
	}
}
