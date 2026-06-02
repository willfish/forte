// Package userconfig exports and imports sectioned Forte user configuration (TOML).
package userconfig

import "time"

const (
	// SchemaVersion is the current on-disk config schema.
	SchemaVersion = 1
	// FileName is the canonical config file under the Forte config directory.
	FileName = "config.toml"
)

// File is the top-level config document written to ~/.config/forte/config.toml.
type File struct {
	SchemaVersion int          `toml:"schemaVersion"`
	ExportedAt    string       `toml:"exportedAt,omitempty"`
	App           AppSection   `toml:"app"`
	Radio         RadioSection `toml:"radio"`
}

// AppSection holds product-level preferences (SQLite app_preferences).
type AppSection struct {
	LibraryEnabled   bool   `toml:"libraryEnabled"`
	StartLastStation bool   `toml:"startLastStation"`
	AutoReconnect    bool   `toml:"autoReconnect"`
	ShowTitlebar     bool   `toml:"showTitlebar"`
	LogLevel         string `toml:"logLevel"`
}

// RadioSection holds radio favourites and custom stations.
type RadioSection struct {
	Favourites     []FavouriteEntry     `toml:"favourites"`
	CustomStations []CustomStationEntry `toml:"customStations"`
}

// FavouriteEntry is one saved radio favourite including pin state.
type FavouriteEntry struct {
	StationUUID string `toml:"stationUuid"`
	Name        string `toml:"name"`
	StreamURL   string `toml:"streamUrl"`
	FaviconURL  string `toml:"faviconUrl"`
	Homepage    string `toml:"homepage"`
	Country     string `toml:"country"`
	Codec       string `toml:"codec"`
	Bitrate     int    `toml:"bitrate"`
	Tags        string `toml:"tags"`
	Pinned      bool   `toml:"pinned"`
}

// CustomStationEntry is one user-defined radio stream.
type CustomStationEntry struct {
	StationUUID string `toml:"stationUuid"`
	Name        string `toml:"name"`
	StreamURL   string `toml:"streamUrl"`
	FaviconURL  string `toml:"faviconUrl"`
	Homepage    string `toml:"homepage"`
	Country     string `toml:"country"`
	Codec       string `toml:"codec"`
	Bitrate     int    `toml:"bitrate"`
	Tags        string `toml:"tags"`
}

// ImportResult reports which sections were applied during import.
type ImportResult struct {
	Path            string
	SectionsApplied []string
	SectionsSkipped []string
	Warnings        []string
}

func newExportFile() File {
	return File{
		SchemaVersion: SchemaVersion,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
	}
}
