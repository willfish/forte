package userconfig

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/willfish/forte/internal/library"
)

// BuildFromDB reads exportable user state from the library database.
func BuildFromDB(db *library.DB) (File, error) {
	out := newExportFile()

	prefs, err := db.GetAppPreferences()
	if err != nil {
		return File{}, fmt.Errorf("export app preferences: %w", err)
	}
	out.App = AppSection{
		LibraryEnabled:   prefs.LibraryEnabled,
		StartLastStation: prefs.StartLastStation,
		AutoReconnect:    prefs.AutoReconnect,
		ShowTitlebar:     prefs.ShowTitlebar,
		LogLevel:         prefs.LogLevel,
	}

	favs, err := db.GetRadioFavourites()
	if err != nil {
		return File{}, fmt.Errorf("export radio favourites: %w", err)
	}
	out.Radio.Favourites = make([]FavouriteEntry, len(favs))
	for i, f := range favs {
		out.Radio.Favourites[i] = FavouriteEntry{
			StationUUID: f.StationUUID,
			Name:        f.Name,
			StreamURL:   f.StreamURL,
			FaviconURL:  f.FaviconURL,
			Homepage:    f.Homepage,
			Country:     f.Country,
			Codec:       f.Codec,
			Bitrate:     f.Bitrate,
			Tags:        f.Tags,
			Pinned:      f.Pinned,
		}
	}

	custom, err := db.GetCustomRadioStations()
	if err != nil {
		return File{}, fmt.Errorf("export custom radio stations: %w", err)
	}
	out.Radio.CustomStations = make([]CustomStationEntry, len(custom))
	for i, st := range custom {
		out.Radio.CustomStations[i] = CustomStationEntry{
			StationUUID: st.StationUUID,
			Name:        st.Name,
			StreamURL:   st.StreamURL,
			FaviconURL:  st.FaviconURL,
			Homepage:    st.Homepage,
			Country:     st.Country,
			Codec:       st.Codec,
			Bitrate:     st.Bitrate,
			Tags:        st.Tags,
		}
	}

	return out, nil
}

// WriteFile encodes cfg as TOML and writes it to path, creating parent directories.
func WriteFile(path string, cfg File) error {
	if cfg.SchemaVersion == 0 {
		cfg.SchemaVersion = SchemaVersion
	}
	if cfg.ExportedAt == "" {
		cfg.ExportedAt = newExportFile().ExportedAt
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ExportToDBPath writes the canonical config file for the current database state.
func ExportToDBPath(db *library.DB) (string, error) {
	path, err := DefaultPath()
	if err != nil {
		return "", err
	}
	cfg, err := BuildFromDB(db)
	if err != nil {
		return "", err
	}
	if err := WriteFile(path, cfg); err != nil {
		return "", err
	}
	return path, nil
}
