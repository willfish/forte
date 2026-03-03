package library

import (
	"os"
	"path/filepath"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := OpenDB(path)
	if err != nil {
		t.Fatalf("OpenDB() error: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenDB(t *testing.T) {
	db := openTestDB(t)

	// Verify WAL mode is enabled.
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal", mode)
	}

	// Verify foreign keys are enabled.
	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("PRAGMA foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("foreign_keys = %d, want 1", fk)
	}
}

func TestMigrationIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")

	// Open twice - second open should be a no-op for migrations.
	db1, err := OpenDB(path)
	if err != nil {
		t.Fatalf("first OpenDB: %v", err)
	}
	db1.Close()

	db2, err := OpenDB(path)
	if err != nil {
		t.Fatalf("second OpenDB: %v", err)
	}
	db2.Close()
}

func TestTablesExist(t *testing.T) {
	db := openTestDB(t)

	tables := []string{"artists", "albums", "tracks", "genres", "track_genres", "playlists", "playlist_tracks"}
	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %q not found: %v", table, err)
		}
	}

	// FTS5 virtual table.
	var name string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='fts_tracks'").Scan(&name)
	if err != nil {
		t.Errorf("fts_tracks virtual table not found: %v", err)
	}
}

func TestInsertAndQueryTrack(t *testing.T) {
	db := openTestDB(t)

	// Insert an artist.
	res, err := db.Exec("INSERT INTO artists (name) VALUES (?)", "Test Artist")
	if err != nil {
		t.Fatalf("insert artist: %v", err)
	}
	artistID, _ := res.LastInsertId()

	// Insert an album.
	res, err = db.Exec("INSERT INTO albums (artist_id, title, year) VALUES (?, ?, ?)", artistID, "Test Album", 2024)
	if err != nil {
		t.Fatalf("insert album: %v", err)
	}
	albumID, _ := res.LastInsertId()

	// Insert a track.
	res, err = db.Exec(`INSERT INTO tracks (album_id, artist_id, title, track_number, duration_ms, file_path, format)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, albumID, artistID, "Test Song", 1, 240000, "/music/test.flac", "FLAC")
	if err != nil {
		t.Fatalf("insert track: %v", err)
	}
	trackID, _ := res.LastInsertId()

	// Verify the track is queryable.
	var title string
	if err := db.QueryRow("SELECT title FROM tracks WHERE id = ?", trackID).Scan(&title); err != nil {
		t.Fatalf("query track: %v", err)
	}
	if title != "Test Song" {
		t.Errorf("title = %q, want %q", title, "Test Song")
	}
}

func TestFTS5Search(t *testing.T) {
	db := openTestDB(t)

	// Insert artist + album + track.
	db.Exec("INSERT INTO artists (name) VALUES (?)", "Beethoven")
	db.Exec("INSERT INTO albums (artist_id, title, year) VALUES (1, 'Symphony No. 9', 1824)")
	db.Exec(`INSERT INTO tracks (album_id, artist_id, title, track_number, file_path)
		VALUES (1, 1, 'Ode to Joy', 4, '/music/ode.flac')`)

	// Populate FTS index (content-less table: stores rowid only, columns for matching).
	_, err := db.Exec("INSERT INTO fts_tracks (rowid, title, artist, album, genre) VALUES (1, 'Ode to Joy', 'Beethoven', 'Symphony No. 9', 'Classical')")
	if err != nil {
		t.Fatalf("insert fts: %v", err)
	}

	// Exact match - join FTS rowid back to tracks table.
	var title string
	err = db.QueryRow(`SELECT t.title FROM fts_tracks f
		JOIN tracks t ON t.id = f.rowid
		WHERE fts_tracks MATCH 'Beethoven'`).Scan(&title)
	if err != nil {
		t.Fatalf("FTS exact match: %v", err)
	}
	if title != "Ode to Joy" {
		t.Errorf("title = %q, want %q", title, "Ode to Joy")
	}

	// Prefix match.
	err = db.QueryRow(`SELECT t.title FROM fts_tracks f
		JOIN tracks t ON t.id = f.rowid
		WHERE fts_tracks MATCH 'beet*'`).Scan(&title)
	if err != nil {
		t.Fatalf("FTS prefix match: %v", err)
	}
	if title != "Ode to Joy" {
		t.Errorf("title = %q, want %q", title, "Ode to Joy")
	}

	// No match.
	var rowid int64
	err = db.QueryRow("SELECT rowid FROM fts_tracks WHERE fts_tracks MATCH 'zzzzz'").Scan(&rowid)
	if err == nil {
		t.Error("expected no match for 'zzzzz'")
	}
}

func TestForeignKeyConstraint(t *testing.T) {
	db := openTestDB(t)

	// Inserting a track with a non-existent artist_id should fail.
	_, err := db.Exec(`INSERT INTO tracks (artist_id, title, file_path) VALUES (999, 'Orphan', '/music/orphan.flac')`)
	if err == nil {
		t.Error("expected foreign key error for non-existent artist_id")
	}
}

func TestUniqueFilePath(t *testing.T) {
	db := openTestDB(t)

	db.Exec("INSERT INTO artists (name) VALUES ('A')")
	_, err := db.Exec(`INSERT INTO tracks (artist_id, title, file_path) VALUES (1, 'T1', '/music/a.flac')`)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = db.Exec(`INSERT INTO tracks (artist_id, title, file_path) VALUES (1, 'T2', '/music/a.flac')`)
	if err == nil {
		t.Error("expected unique constraint error for duplicate file_path")
	}
}

func TestCueSheetColumns(t *testing.T) {
	db := openTestDB(t)

	db.Exec("INSERT INTO artists (name) VALUES ('A')")
	db.Exec("INSERT INTO albums (artist_id, title) VALUES (1, 'Album')")

	_, err := db.Exec(`INSERT INTO tracks (album_id, artist_id, title, file_path, cue_file_path, start_ms, end_ms)
		VALUES (1, 1, 'CUE Track', '/music/album.flac', '/music/album.cue', 0, 330000)`)
	if err != nil {
		t.Fatalf("insert CUE track: %v", err)
	}

	var cuePath string
	var startMs, endMs int
	err = db.QueryRow("SELECT cue_file_path, start_ms, end_ms FROM tracks WHERE id = 1").Scan(&cuePath, &startMs, &endMs)
	if err != nil {
		t.Fatalf("query CUE track: %v", err)
	}
	if cuePath != "/music/album.cue" {
		t.Errorf("cue_file_path = %q, want %q", cuePath, "/music/album.cue")
	}
	if startMs != 0 || endMs != 330000 {
		t.Errorf("start/end = %d/%d, want 0/330000", startMs, endMs)
	}
}

func TestServerColumns(t *testing.T) {
	db := openTestDB(t)

	db.Exec("INSERT INTO artists (name) VALUES ('A')")
	_, err := db.Exec(`INSERT INTO tracks (artist_id, title, file_path, server_id, remote_id)
		VALUES (1, 'Remote Track', '', 'subsonic-1', 'tr-42')`)
	if err != nil {
		t.Fatalf("insert server track: %v", err)
	}

	var serverID, remoteID string
	err = db.QueryRow("SELECT server_id, remote_id FROM tracks WHERE id = 1").Scan(&serverID, &remoteID)
	if err != nil {
		t.Fatalf("query server track: %v", err)
	}
	if serverID != "subsonic-1" || remoteID != "tr-42" {
		t.Errorf("server_id/remote_id = %q/%q, want subsonic-1/tr-42", serverID, remoteID)
	}
}

func TestCascadeDelete(t *testing.T) {
	db := openTestDB(t)

	db.Exec("INSERT INTO artists (name) VALUES ('A')")
	db.Exec("INSERT INTO genres (name) VALUES ('Rock')")
	db.Exec(`INSERT INTO tracks (artist_id, title, file_path) VALUES (1, 'T', '/a.flac')`)
	db.Exec("INSERT INTO track_genres (track_id, genre_id) VALUES (1, 1)")

	// Deleting the track should cascade to track_genres.
	db.Exec("DELETE FROM tracks WHERE id = 1")

	var count int
	db.QueryRow("SELECT COUNT(*) FROM track_genres").Scan(&count)
	if count != 0 {
		t.Errorf("track_genres count = %d after cascade delete, want 0", count)
	}
}

func TestOpenDBInvalidPath(t *testing.T) {
	_, err := OpenDB("/nonexistent/dir/test.db")
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestOpenDBCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.db")
	db, err := OpenDB(path)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	db.Close()

	if _, err := os.Stat(path); err != nil {
		t.Errorf("database file not created: %v", err)
	}
}
