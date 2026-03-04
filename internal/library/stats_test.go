package library

import "testing"

// seedStatsData inserts two artists, two albums, and three tracks for testing.
// Returns the track IDs in order.
func seedStatsData(t *testing.T, db *DB) (int64, int64, int64) {
	t.Helper()
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Artist A')")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (2, 'Artist B')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (1, 1, 'Album A', 2020)")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (2, 2, 'Album B', 2021)")
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, file_path, duration_ms)
		VALUES (1, 1, 1, 'Track A1', '/a1.flac', 200000)`)
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, file_path, duration_ms)
		VALUES (2, 1, 1, 'Track A2', '/a2.flac', 180000)`)
	mustExec(t, db, `INSERT INTO tracks (id, album_id, artist_id, title, file_path, duration_ms)
		VALUES (3, 2, 2, 'Track B1', '/b1.flac', 250000)`)
	return 1, 2, 3
}

func TestRecordPlayAndTopArtists(t *testing.T) {
	db := openTestDB(t)
	t1, _, t3 := seedStatsData(t, db)

	// Play Artist A tracks 3 times, Artist B once.
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 100000)", t3)

	artists, err := db.TopArtists("all", 10)
	if err != nil {
		t.Fatalf("TopArtists: %v", err)
	}
	if len(artists) != 2 {
		t.Fatalf("got %d artists, want 2", len(artists))
	}
	if artists[0].Name != "Artist A" {
		t.Errorf("top artist = %q, want Artist A", artists[0].Name)
	}
	if artists[0].PlayCount != 2 {
		t.Errorf("play count = %d, want 2", artists[0].PlayCount)
	}
	if artists[0].TotalMs != 240000 {
		t.Errorf("total ms = %d, want 240000", artists[0].TotalMs)
	}
}

func TestTopAlbums(t *testing.T) {
	db := openTestDB(t)
	t1, t2, t3 := seedStatsData(t, db)

	// Album A: 3 plays (t1 twice, t2 once). Album B: 1 play.
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 90000)", t2)
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 100000)", t3)

	albums, err := db.TopAlbums("all", 10)
	if err != nil {
		t.Fatalf("TopAlbums: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("got %d albums, want 2", len(albums))
	}
	if albums[0].Name != "Album A" {
		t.Errorf("top album = %q, want Album A", albums[0].Name)
	}
	if albums[0].SecondLine != "Artist A" {
		t.Errorf("second line = %q, want Artist A", albums[0].SecondLine)
	}
	if albums[0].PlayCount != 3 {
		t.Errorf("play count = %d, want 3", albums[0].PlayCount)
	}
}

func TestTopTracks(t *testing.T) {
	db := openTestDB(t)
	t1, _, t3 := seedStatsData(t, db)

	// Track A1: 3 plays, Track B1: 1 play.
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, 100000)", t3)

	tracks, err := db.TopTracks("all", 10)
	if err != nil {
		t.Fatalf("TopTracks: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
	if tracks[0].Name != "Track A1" {
		t.Errorf("top track = %q, want Track A1", tracks[0].Name)
	}
	if tracks[0].SecondLine != "Artist A" {
		t.Errorf("second line = %q, want Artist A", tracks[0].SecondLine)
	}
	if tracks[0].PlayCount != 3 {
		t.Errorf("play count = %d, want 3", tracks[0].PlayCount)
	}
}

func TestRecentlyPlayed(t *testing.T) {
	db := openTestDB(t)
	t1, t2, t3 := seedStatsData(t, db)

	// Insert plays with explicit timestamps for ordering.
	mustExec(t, db, "INSERT INTO play_history (track_id, played_at, duration_played_ms) VALUES (?, '2025-01-01 10:00:00', 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, played_at, duration_played_ms) VALUES (?, '2025-01-01 11:00:00', 90000)", t2)
	mustExec(t, db, "INSERT INTO play_history (track_id, played_at, duration_played_ms) VALUES (?, '2025-01-01 12:00:00', 100000)", t3)

	recent, err := db.RecentlyPlayed(10)
	if err != nil {
		t.Fatalf("RecentlyPlayed: %v", err)
	}
	if len(recent) != 3 {
		t.Fatalf("got %d recent, want 3", len(recent))
	}
	// Most recent first.
	if recent[0].Title != "Track B1" {
		t.Errorf("most recent = %q, want Track B1", recent[0].Title)
	}
	if recent[0].Artist != "Artist B" {
		t.Errorf("artist = %q, want Artist B", recent[0].Artist)
	}
	if recent[0].Album != "Album B" {
		t.Errorf("album = %q, want Album B", recent[0].Album)
	}
}

func TestPeriodFiltering(t *testing.T) {
	db := openTestDB(t)
	t1, _, t3 := seedStatsData(t, db)

	// One play from 60 days ago, one from today.
	mustExec(t, db, "INSERT INTO play_history (track_id, played_at, duration_played_ms) VALUES (?, datetime('now', '-60 days'), 120000)", t1)
	mustExec(t, db, "INSERT INTO play_history (track_id, played_at, duration_played_ms) VALUES (?, datetime('now'), 100000)", t3)

	// 30-day window should only include the recent play.
	artists, err := db.TopArtists("30d", 10)
	if err != nil {
		t.Fatalf("TopArtists 30d: %v", err)
	}
	if len(artists) != 1 {
		t.Fatalf("got %d artists for 30d, want 1", len(artists))
	}
	if artists[0].Name != "Artist B" {
		t.Errorf("30d top artist = %q, want Artist B", artists[0].Name)
	}

	// All time should include both.
	artists, err = db.TopArtists("all", 10)
	if err != nil {
		t.Fatalf("TopArtists all: %v", err)
	}
	if len(artists) != 2 {
		t.Fatalf("got %d artists for all, want 2", len(artists))
	}
}

func TestEmptyHistory(t *testing.T) {
	db := openTestDB(t)

	artists, err := db.TopArtists("all", 10)
	if err != nil {
		t.Fatalf("TopArtists: %v", err)
	}
	if len(artists) != 0 {
		t.Errorf("expected empty artists, got %d", len(artists))
	}

	albums, err := db.TopAlbums("all", 10)
	if err != nil {
		t.Fatalf("TopAlbums: %v", err)
	}
	if len(albums) != 0 {
		t.Errorf("expected empty albums, got %d", len(albums))
	}

	tracks, err := db.TopTracks("all", 10)
	if err != nil {
		t.Fatalf("TopTracks: %v", err)
	}
	if len(tracks) != 0 {
		t.Errorf("expected empty tracks, got %d", len(tracks))
	}

	recent, err := db.RecentlyPlayed(10)
	if err != nil {
		t.Fatalf("RecentlyPlayed: %v", err)
	}
	if len(recent) != 0 {
		t.Errorf("expected empty recent, got %d", len(recent))
	}
}

func TestRecordPlayViaMethod(t *testing.T) {
	db := openTestDB(t)
	t1, _, _ := seedStatsData(t, db)

	if err := db.RecordPlay(t1, 150000); err != nil {
		t.Fatalf("RecordPlay: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM play_history").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("play count = %d, want 1", count)
	}
}
