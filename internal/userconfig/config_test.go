package userconfig_test

import (
	"path/filepath"
	"testing"

	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/userconfig"
)

func openTestDB(t *testing.T) *library.DB {
	t.Helper()
	db, err := library.OpenDB(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestExportImportRoundTrip(t *testing.T) {
	db := openTestDB(t)

	if err := db.AddRadioFavourite(library.RadioFavourite{
		StationUUID: "uuid-1",
		Name:        "Jazz FM",
		StreamURL:   "https://stream.example/jazz",
		Homepage:    "https://jazz.example",
		Country:     "UK",
		Codec:       "MP3",
		Bitrate:     128,
		Tags:        "jazz",
		Pinned:      true,
	}); err != nil {
		t.Fatalf("AddRadioFavourite: %v", err)
	}
	if _, err := db.AddCustomRadioStation(library.RadioCustomStation{
		StationUUID: "custom-1",
		Name:        "My Stream",
		StreamURL:   "https://stream.example/custom",
		Tags:        "talk",
	}); err != nil {
		t.Fatalf("AddCustomRadioStation: %v", err)
	}
	if err := db.SaveAppPreferences(library.AppPreferences{
		LibraryEnabled:   true,
		StartLastStation: false,
		AutoReconnect:    true,
		ShowTitlebar:     true,
		LogLevel:         "info",
	}); err != nil {
		t.Fatalf("SaveAppPreferences: %v", err)
	}

	path := filepath.Join(t.TempDir(), "config.toml")
	cfg, err := userconfig.BuildFromDB(db)
	if err != nil {
		t.Fatalf("BuildFromDB: %v", err)
	}
	if err := userconfig.WriteFile(path, cfg); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fresh := openTestDB(t)
	if _, err := userconfig.ImportFile(fresh, path, userconfig.ReplaceOptions); err != nil {
		t.Fatalf("ImportFile: %v", err)
	}

	prefs, err := fresh.GetAppPreferences()
	if err != nil {
		t.Fatalf("GetAppPreferences: %v", err)
	}
	if !prefs.LibraryEnabled || prefs.StartLastStation || !prefs.AutoReconnect || !prefs.ShowTitlebar || prefs.LogLevel != "info" {
		t.Fatalf("app preferences mismatch: %#v", prefs)
	}

	favs, err := fresh.GetRadioFavourites()
	if err != nil {
		t.Fatalf("GetRadioFavourites: %v", err)
	}
	if len(favs) != 1 || favs[0].StationUUID != "uuid-1" || !favs[0].Pinned {
		t.Fatalf("favourites mismatch: %#v", favs)
	}

	custom, err := fresh.GetCustomRadioStations()
	if err != nil {
		t.Fatalf("GetCustomRadioStations: %v", err)
	}
	if len(custom) != 1 || custom[0].Name != "My Stream" {
		t.Fatalf("custom stations mismatch: %#v", custom)
	}
}

func TestMergeRadioPreservesExistingFavourites(t *testing.T) {
	db := openTestDB(t)

	if err := db.AddRadioFavourite(library.RadioFavourite{
		StationUUID: "in-db",
		Name:        "Live station",
		StreamURL:   "https://stream.example/live",
		Pinned:      true,
	}); err != nil {
		t.Fatalf("AddRadioFavourite: %v", err)
	}

	path := filepath.Join(t.TempDir(), "config.toml")
	cfg := userconfig.File{
		SchemaVersion: userconfig.SchemaVersion,
		Radio: userconfig.RadioSection{
			Favourites: []userconfig.FavouriteEntry{{
				StationUUID: "from-file",
				Name:        "File station",
				StreamURL:   "https://stream.example/file",
				Pinned:      false,
			}, {
				StationUUID: "in-db",
				Name:        "Stale name",
				StreamURL:   "https://stream.example/stale",
				Pinned:      false,
			}},
		},
	}
	if err := userconfig.WriteFile(path, cfg); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := userconfig.ImportFile(db, path, userconfig.MergeOptions); err != nil {
		t.Fatalf("ImportFile merge: %v", err)
	}

	favs, err := db.GetRadioFavourites()
	if err != nil {
		t.Fatalf("GetRadioFavourites: %v", err)
	}
	if len(favs) != 2 {
		t.Fatalf("want 2 favourites, got %#v", favs)
	}
	byUUID := map[string]library.RadioFavourite{}
	for _, f := range favs {
		byUUID[f.StationUUID] = f
	}
	if byUUID["in-db"].Name != "Live station" || !byUUID["in-db"].Pinned {
		t.Fatalf("existing favourite was clobbered: %#v", byUUID["in-db"])
	}
	if byUUID["from-file"].Name != "File station" {
		t.Fatalf("file favourite not added: %#v", byUUID["from-file"])
	}
}

func TestValidateSchemaRejectsUnknownVersion(t *testing.T) {
	err := userconfig.ValidateSchema(userconfig.File{SchemaVersion: 99})
	if err == nil {
		t.Fatal("expected schema error")
	}
}
