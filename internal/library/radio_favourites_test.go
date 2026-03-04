package library

import (
	"os"
	"path/filepath"
	"testing"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestRadioFavouritesRoundTrip(t *testing.T) {
	db := testDB(t)

	fav := RadioFavourite{
		StationUUID: "abc-123",
		Name:        "Jazz FM",
		StreamURL:   "https://stream.example.com/jazz",
		FaviconURL:  "https://example.com/icon.png",
		Tags:        "jazz,smooth",
	}

	if err := db.AddRadioFavourite(fav); err != nil {
		t.Fatalf("AddRadioFavourite: %v", err)
	}

	favs, err := db.GetRadioFavourites()
	if err != nil {
		t.Fatalf("GetRadioFavourites: %v", err)
	}
	if len(favs) != 1 {
		t.Fatalf("expected 1 favourite, got %d", len(favs))
	}

	got := favs[0]
	if got.StationUUID != "abc-123" {
		t.Errorf("StationUUID = %q", got.StationUUID)
	}
	if got.Name != "Jazz FM" {
		t.Errorf("Name = %q", got.Name)
	}
	if got.StreamURL != "https://stream.example.com/jazz" {
		t.Errorf("StreamURL = %q", got.StreamURL)
	}
	if got.FaviconURL != "https://example.com/icon.png" {
		t.Errorf("FaviconURL = %q", got.FaviconURL)
	}
	if got.Tags != "jazz,smooth" {
		t.Errorf("Tags = %q", got.Tags)
	}
	if got.AddedAt == "" {
		t.Error("AddedAt should not be empty")
	}
}

func TestRadioFavouriteDuplicate(t *testing.T) {
	db := testDB(t)

	fav := RadioFavourite{
		StationUUID: "abc-123",
		Name:        "Jazz FM",
		StreamURL:   "https://stream.example.com/jazz",
	}

	if err := db.AddRadioFavourite(fav); err != nil {
		t.Fatalf("AddRadioFavourite: %v", err)
	}
	// Adding the same station again should not error (INSERT OR IGNORE).
	if err := db.AddRadioFavourite(fav); err != nil {
		t.Fatalf("AddRadioFavourite duplicate: %v", err)
	}

	favs, err := db.GetRadioFavourites()
	if err != nil {
		t.Fatalf("GetRadioFavourites: %v", err)
	}
	if len(favs) != 1 {
		t.Fatalf("expected 1 favourite after duplicate, got %d", len(favs))
	}
}

func TestRemoveRadioFavourite(t *testing.T) {
	db := testDB(t)

	fav := RadioFavourite{
		StationUUID: "abc-123",
		Name:        "Jazz FM",
		StreamURL:   "https://stream.example.com/jazz",
	}
	if err := db.AddRadioFavourite(fav); err != nil {
		t.Fatalf("AddRadioFavourite: %v", err)
	}

	if err := db.RemoveRadioFavourite("abc-123"); err != nil {
		t.Fatalf("RemoveRadioFavourite: %v", err)
	}

	favs, err := db.GetRadioFavourites()
	if err != nil {
		t.Fatalf("GetRadioFavourites: %v", err)
	}
	if len(favs) != 0 {
		t.Fatalf("expected 0 favourites after removal, got %d", len(favs))
	}
}

func TestIsRadioFavourite(t *testing.T) {
	db := testDB(t)

	ok, err := db.IsRadioFavourite("nonexistent")
	if err != nil {
		t.Fatalf("IsRadioFavourite: %v", err)
	}
	if ok {
		t.Error("expected false for nonexistent station")
	}

	fav := RadioFavourite{
		StationUUID: "abc-123",
		Name:        "Jazz FM",
		StreamURL:   "https://stream.example.com/jazz",
	}
	if err := db.AddRadioFavourite(fav); err != nil {
		t.Fatalf("AddRadioFavourite: %v", err)
	}

	ok, err = db.IsRadioFavourite("abc-123")
	if err != nil {
		t.Fatalf("IsRadioFavourite: %v", err)
	}
	if !ok {
		t.Error("expected true for saved station")
	}
}

func TestGetRadioFavouritesEmpty(t *testing.T) {
	db := testDB(t)

	favs, err := db.GetRadioFavourites()
	if err != nil {
		t.Fatalf("GetRadioFavourites: %v", err)
	}
	if favs != nil {
		t.Errorf("expected nil for empty favourites, got %v", favs)
	}
}

func TestRemoveNonexistentFavourite(t *testing.T) {
	db := testDB(t)

	// Should not error when removing a station that doesn't exist.
	if err := db.RemoveRadioFavourite("nonexistent"); err != nil {
		t.Fatalf("RemoveRadioFavourite: %v", err)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
