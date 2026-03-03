package library

import (
	"testing"
)

func seedSearchData(t *testing.T, db *DB) {
	t.Helper()
	mustExec(t, db, "INSERT INTO artists (name) VALUES ('Beethoven')")
	mustExec(t, db, "INSERT INTO artists (name) VALUES ('Mozart')")
	mustExec(t, db, "INSERT INTO albums (artist_id, title, year) VALUES (1, 'Symphony No. 9', 1824)")
	mustExec(t, db, "INSERT INTO albums (artist_id, title, year) VALUES (2, 'Requiem', 1791)")
	mustExec(t, db, "INSERT INTO genres (name) VALUES ('Classical')")
	mustExec(t, db, "INSERT INTO genres (name) VALUES ('Romantic')")

	mustExec(t, db, `INSERT INTO tracks (album_id, artist_id, title, track_number, disc_number, duration_ms, file_path)
		VALUES (1, 1, 'Ode to Joy', 4, 1, 240000, '/music/ode.flac')`)
	mustExec(t, db, `INSERT INTO tracks (album_id, artist_id, title, track_number, disc_number, duration_ms, file_path)
		VALUES (1, 1, 'Allegro ma non troppo', 1, 1, 960000, '/music/allegro.flac')`)
	mustExec(t, db, `INSERT INTO tracks (album_id, artist_id, title, track_number, disc_number, duration_ms, file_path)
		VALUES (2, 2, 'Lacrimosa', 7, 1, 210000, '/music/lacrimosa.flac')`)

	mustExec(t, db, "INSERT INTO track_genres (track_id, genre_id) VALUES (1, 1)")
	mustExec(t, db, "INSERT INTO track_genres (track_id, genre_id) VALUES (1, 2)")
	mustExec(t, db, "INSERT INTO track_genres (track_id, genre_id) VALUES (2, 1)")
	mustExec(t, db, "INSERT INTO track_genres (track_id, genre_id) VALUES (3, 1)")

	// Populate FTS index.
	mustExec(t, db, "INSERT INTO fts_tracks (rowid, title, artist, album, genre) VALUES (1, 'Ode to Joy', 'Beethoven', 'Symphony No. 9', 'Classical Romantic')")
	mustExec(t, db, "INSERT INTO fts_tracks (rowid, title, artist, album, genre) VALUES (2, 'Allegro ma non troppo', 'Beethoven', 'Symphony No. 9', 'Classical')")
	mustExec(t, db, "INSERT INTO fts_tracks (rowid, title, artist, album, genre) VALUES (3, 'Lacrimosa', 'Mozart', 'Requiem', 'Classical')")
}

func TestSearchExactMatch(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	results, err := db.Search("Beethoven", 100)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for 'Beethoven', got %d", len(results))
	}
}

func TestSearchPrefixMatch(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	results, err := db.Search("beet", 100)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for prefix 'beet', got %d", len(results))
	}
}

func TestSearchByTitle(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	results, err := db.Search("Lacrimosa", 100)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'Lacrimosa', got %d", len(results))
	}
	if results[0].Artist != "Mozart" {
		t.Errorf("artist = %q, want Mozart", results[0].Artist)
	}
}

func TestSearchByAlbum(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	results, err := db.Search("Requiem", 100)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result for 'Requiem', got %d", len(results))
	}
}

func TestSearchMultiWord(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	results, err := db.Search("Beethoven Symphony", 100)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results for multi-word query, got %d", len(results))
	}
}

func TestSearchNoMatch(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	results, err := db.Search("zzzzzzz", 100)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	results, err := db.Search("", 100)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) < 3 {
		t.Errorf("expected at least 3 results for empty query, got %d", len(results))
	}
}

func TestSearchLimit(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	results, err := db.Search("", 1)
	if err != nil {
		t.Fatalf("Search() error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with limit=1, got %d", len(results))
	}
}

func TestSearchSpecialCharacters(t *testing.T) {
	db := openTestDB(t)
	seedSearchData(t, db)

	// These should not cause SQL errors.
	for _, q := range []string{"foo\"bar", "test'quote", "a AND b", "a OR b", "*", "NOT x"} {
		_, err := db.Search(q, 100)
		if err != nil {
			t.Errorf("Search(%q) error: %v", q, err)
		}
	}
}

func TestSanitizeFTS(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "\"hello\"*"},
		{"hello world", "\"hello\"* \"world\"*"},
		{"", ""},
		{"foo\"bar", "\"foobar\"*"},
	}
	for _, tt := range tests {
		if got := sanitizeFTS(tt.input); got != tt.want {
			t.Errorf("sanitizeFTS(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
