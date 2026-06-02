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

func TestRadioHomepagePersistence(t *testing.T) {
	db := openTestDB(t)

	fav := RadioFavourite{
		StationUUID: "abc-123",
		Name:        "Jazz FM",
		StreamURL:   "https://stream.example.com/jazz",
		FaviconURL:  "https://example.com/icon.png",
		Homepage:    "https://jazzfm.com/",
		Tags:        "jazz",
	}
	if err := db.AddRadioFavourite(fav); err != nil {
		t.Fatalf("AddRadioFavourite: %v", err)
	}

	favs, err := db.GetRadioFavourites()
	if err != nil {
		t.Fatalf("GetRadioFavourites: %v", err)
	}
	if len(favs) != 1 || favs[0].Homepage != fav.Homepage {
		t.Fatalf("favourite homepage = %q, want %q", favs[0].Homepage, fav.Homepage)
	}

	entry := RadioHistoryEntry{
		StationUUID: fav.StationUUID,
		Name:        fav.Name,
		StreamURL:   fav.StreamURL,
		FaviconURL:  fav.FaviconURL,
		Homepage:    fav.Homepage,
		Tags:        fav.Tags,
	}
	if err := db.RecordRadioPlayback(entry); err != nil {
		t.Fatalf("RecordRadioPlayback: %v", err)
	}

	history, err := db.GetRadioHistory(5)
	if err != nil {
		t.Fatalf("GetRadioHistory: %v", err)
	}
	if len(history) != 1 || history[0].Homepage != fav.Homepage {
		t.Fatalf("history homepage = %q, want %q", history[0].Homepage, fav.Homepage)
	}
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

func TestSetRadioFavouritePinnedOrdersPinnedFirst(t *testing.T) {
	db := testDB(t)

	favourites := []RadioFavourite{
		{StationUUID: "z-last", Name: "Zulu FM", StreamURL: "https://stream.example.com/zulu"},
		{StationUUID: "a-pinned", Name: "Alpha FM", StreamURL: "https://stream.example.com/alpha"},
	}
	for _, fav := range favourites {
		if err := db.AddRadioFavourite(fav); err != nil {
			t.Fatalf("AddRadioFavourite: %v", err)
		}
	}
	if err := db.SetRadioFavouritePinned("a-pinned", true); err != nil {
		t.Fatalf("SetRadioFavouritePinned: %v", err)
	}

	got, err := db.GetRadioFavourites()
	if err != nil {
		t.Fatalf("GetRadioFavourites: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 favourites, got %d", len(got))
	}
	if got[0].StationUUID != "a-pinned" || !got[0].Pinned {
		t.Fatalf("expected pinned station first, got %#v", got[0])
	}
	if got[1].Pinned {
		t.Fatalf("expected second station to be unpinned, got %#v", got[1])
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

func TestCustomRadioStationsRoundTrip(t *testing.T) {
	db := testDB(t)

	station := RadioCustomStation{
		StationUUID: "custom-123",
		Name:        "Kitchen Stream",
		StreamURL:   "https://stream.example.com/kitchen",
		FaviconURL:  "https://example.com/kitchen.png",
		Tags:        "home,ambient",
	}
	if _, err := db.AddCustomRadioStation(station); err != nil {
		t.Fatalf("AddCustomRadioStation: %v", err)
	}

	got, err := db.GetCustomRadioStations()
	if err != nil {
		t.Fatalf("GetCustomRadioStations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 custom station, got %d", len(got))
	}
	if got[0].StationUUID != station.StationUUID || got[0].Name != station.Name || got[0].StreamURL != station.StreamURL {
		t.Fatalf("custom station mismatch: %#v", got[0])
	}
	if got[0].CreatedAt == "" {
		t.Error("CreatedAt should not be empty")
	}

	if err := db.DeleteCustomRadioStation(station.StationUUID); err != nil {
		t.Fatalf("DeleteCustomRadioStation: %v", err)
	}
	got, err = db.GetCustomRadioStations()
	if err != nil {
		t.Fatalf("GetCustomRadioStations after delete: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no custom stations after delete, got %d", len(got))
	}
}

func TestRadioHistoryRecordsTitlesAndErrors(t *testing.T) {
	db := testDB(t)

	entry := RadioHistoryEntry{
		StationUUID: "st-1",
		Name:        "Jazz FM",
		StreamURL:   "https://stream.example.com/jazz",
		FaviconURL:  "https://example.com/jazz.png",
		Tags:        "jazz",
	}
	if err := db.RecordRadioPlayback(entry); err != nil {
		t.Fatalf("RecordRadioPlayback: %v", err)
	}
	if err := db.UpdateRadioHistoryTitle("st-1", "Artist - Track"); err != nil {
		t.Fatalf("UpdateRadioHistoryTitle: %v", err)
	}
	if err := db.MarkRadioHistoryError("st-1", "stream lost"); err != nil {
		t.Fatalf("MarkRadioHistoryError: %v", err)
	}

	got, err := db.GetRadioHistory(10)
	if err != nil {
		t.Fatalf("GetRadioHistory: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(got))
	}
	if got[0].TrackTitle != "Artist - Track" {
		t.Fatalf("TrackTitle = %q", got[0].TrackTitle)
	}
	if got[0].LastError != "stream lost" {
		t.Fatalf("LastError = %q", got[0].LastError)
	}
	if got[0].PlayCount != 1 {
		t.Fatalf("PlayCount = %d", got[0].PlayCount)
	}
	if err := db.RecordRadioPlayback(entry); err != nil {
		t.Fatalf("RecordRadioPlayback second: %v", err)
	}
	got, err = db.GetRadioHistory(10)
	if err != nil {
		t.Fatalf("GetRadioHistory after second play: %v", err)
	}
	if got[0].PlayCount != 2 || got[0].LastError != "" {
		t.Fatalf("expected second play to increment and clear error, got %#v", got[0])
	}
	if err := db.ClearRadioHistory(); err != nil {
		t.Fatalf("ClearRadioHistory: %v", err)
	}
	got, err = db.GetRadioHistory(10)
	if err != nil {
		t.Fatalf("GetRadioHistory after clear: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty history after clear, got %d", len(got))
	}
}

func TestAppPreferencesRoundTrip(t *testing.T) {
	db := testDB(t)

	prefs, err := db.GetAppPreferences()
	if err != nil {
		t.Fatalf("GetAppPreferences: %v", err)
	}
	if prefs.LibraryEnabled || !prefs.StartLastStation || !prefs.AutoReconnect || prefs.ShowTitlebar || prefs.LogLevel != "warn" {
		t.Fatalf("unexpected default preferences: %#v", prefs)
	}

	want := AppPreferences{
		LibraryEnabled:   true,
		StartLastStation: false,
		AutoReconnect:    false,
		ShowTitlebar:     true,
		LogLevel:         "debug",
	}
	if err := db.SaveAppPreferences(want); err != nil {
		t.Fatalf("SaveAppPreferences: %v", err)
	}
	got, err := db.GetAppPreferences()
	if err != nil {
		t.Fatalf("GetAppPreferences after save: %v", err)
	}
	if got != want {
		t.Fatalf("preferences mismatch: got %#v want %#v", got, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
