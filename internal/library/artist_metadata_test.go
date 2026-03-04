package library

import "testing"

func TestGetArtistByName(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (2, 'Bjork')")

	id, err := db.GetArtistByName("Radiohead")
	if err != nil {
		t.Fatalf("GetArtistByName: %v", err)
	}
	if id != 1 {
		t.Errorf("got id %d, want 1", id)
	}

	_, err = db.GetArtistByName("Nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent artist")
	}
}

func TestSaveAndGetArtistMeta(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")

	meta := ArtistMeta{
		Bio:      "English rock band",
		ImageURL: "https://example.com/radiohead.jpg",
		Similar:  []SimilarArtist{{Name: "Muse"}, {Name: "Coldplay"}},
		MbID:     "a74b1b7f-71a5-4011-9441-d0b5e4122711",
		MbArea:   "United Kingdom",
		MbType:   "Group",
		MbBegin:  "1985",
		MbEnd:    "",
		MbTags:   "rock, alternative",
	}

	if err := db.SaveArtistMeta(1, meta); err != nil {
		t.Fatalf("SaveArtistMeta: %v", err)
	}

	got, err := db.GetArtistMeta(1)
	if err != nil {
		t.Fatalf("GetArtistMeta: %v", err)
	}
	if got == nil {
		t.Fatal("GetArtistMeta returned nil")
	}
	if got.Bio != "English rock band" {
		t.Errorf("bio = %q, want 'English rock band'", got.Bio)
	}
	if got.ImageURL != "https://example.com/radiohead.jpg" {
		t.Errorf("image_url = %q", got.ImageURL)
	}
	if len(got.Similar) != 2 {
		t.Fatalf("similar count = %d, want 2", len(got.Similar))
	}
	if got.Similar[0].Name != "Muse" {
		t.Errorf("similar[0] = %q, want 'Muse'", got.Similar[0].Name)
	}
	if got.MbArea != "United Kingdom" {
		t.Errorf("mb_area = %q", got.MbArea)
	}
}

func TestGetArtistMetaMissingEntry(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")

	got, err := db.GetArtistMeta(1)
	if err != nil {
		t.Fatalf("GetArtistMeta: %v", err)
	}
	if got != nil {
		t.Error("expected nil for missing entry")
	}
}

func TestGetArtistMetaExpired(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")

	// Insert with a fetched_at 60 days ago.
	mustExec(t, db, `INSERT INTO artist_metadata (artist_id, bio, fetched_at)
		VALUES (1, 'old bio', datetime('now', '-60 days'))`)

	got, err := db.GetArtistMeta(1)
	if err != nil {
		t.Fatalf("GetArtistMeta: %v", err)
	}
	if got != nil {
		t.Error("expected nil for expired cache entry")
	}
}

func TestSaveArtistMetaUpsert(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")

	meta1 := ArtistMeta{Bio: "first bio"}
	if err := db.SaveArtistMeta(1, meta1); err != nil {
		t.Fatalf("first save: %v", err)
	}

	meta2 := ArtistMeta{Bio: "updated bio"}
	if err := db.SaveArtistMeta(1, meta2); err != nil {
		t.Fatalf("second save: %v", err)
	}

	got, err := db.GetArtistMeta(1)
	if err != nil {
		t.Fatalf("GetArtistMeta: %v", err)
	}
	if got == nil || got.Bio != "updated bio" {
		t.Errorf("expected 'updated bio', got %v", got)
	}
}

func TestGetArtistAlbums(t *testing.T) {
	db := openTestDB(t)
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (1, 1, 'OK Computer', 1997)")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (2, 1, 'Kid A', 2000)")
	mustExec(t, db, "INSERT INTO artists (id, name) VALUES (2, 'Bjork')")
	mustExec(t, db, "INSERT INTO albums (id, artist_id, title, year) VALUES (3, 2, 'Homogenic', 1997)")

	albums, err := db.GetArtistAlbums(1)
	if err != nil {
		t.Fatalf("GetArtistAlbums: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("got %d albums, want 2", len(albums))
	}
	if albums[0].Title != "OK Computer" {
		t.Errorf("first album = %q, want 'OK Computer'", albums[0].Title)
	}
	if albums[1].Title != "Kid A" {
		t.Errorf("second album = %q, want 'Kid A'", albums[1].Title)
	}
}
