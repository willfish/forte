package userconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/logx"
)

// ApplyToDB imports supported sections from cfg into the library database.
//
// Merge rules (schema v1):
//   - app: always overwrites app_preferences.
//   - radio: when opts.MergeRadio is true, inserts only new stationUuid values;
//     when false, upserts by stationUuid (file wins for matches).
func ApplyToDB(db *library.DB, cfg File, opts ApplyOptions) (ImportResult, error) {
	if err := ValidateSchema(cfg); err != nil {
		return ImportResult{}, err
	}

	result := ImportResult{}

	if err := importApp(db, cfg.App); err != nil {
		return result, fmt.Errorf("import app: %w", err)
	}
	result.SectionsApplied = append(result.SectionsApplied, "app")

	if len(cfg.Radio.Favourites) > 0 || len(cfg.Radio.CustomStations) > 0 {
		if err := importRadio(db, cfg.Radio, opts); err != nil {
			return result, fmt.Errorf("import radio: %w", err)
		}
		result.SectionsApplied = append(result.SectionsApplied, "radio")
	}

	return result, nil
}

// ImportFile loads path and applies supported sections using opts.
func ImportFile(db *library.DB, path string, opts ApplyOptions) (ImportResult, error) {
	cfg, err := LoadFile(path)
	if err != nil {
		return ImportResult{}, err
	}
	result, err := ApplyToDB(db, cfg, opts)
	result.Path = path
	return result, err
}

// ImportDefaultIfPresent loads the canonical config file when it exists.
// Schema errors are returned; a missing file is not an error.
func ImportDefaultIfPresent(db *library.DB) (ImportResult, error) {
	path, err := DefaultPath()
	if err != nil {
		return ImportResult{}, err
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return ImportResult{Path: path}, nil
		}
		return ImportResult{}, fmt.Errorf("stat config: %w", err)
	}
	result, err := ImportFile(db, path, MergeOptions)
	if err != nil {
		logx.Logger().Warn("user config import failed", "path", path, "error", err)
		return result, err
	}
	if len(result.SectionsApplied) > 0 {
		logx.Logger().Info("user config imported", "path", path, "sections", strings.Join(result.SectionsApplied, ","))
	}
	return result, nil
}

func importApp(db *library.DB, app AppSection) error {
	logLevel := strings.TrimSpace(app.LogLevel)
	if logLevel == "" {
		logLevel = logx.LevelWarn
	}
	return db.SaveAppPreferences(library.AppPreferences{
		LibraryEnabled:   app.LibraryEnabled,
		StartLastStation: app.StartLastStation,
		AutoReconnect:    app.AutoReconnect,
		ShowTitlebar:     app.ShowTitlebar,
		LogLevel:         logLevel,
	})
}

func importRadio(db *library.DB, radio RadioSection, opts ApplyOptions) error {
	customExisting := map[string]struct{}{}
	if opts.MergeRadio {
		stations, err := db.GetCustomRadioStations()
		if err != nil {
			return err
		}
		for _, st := range stations {
			customExisting[st.StationUUID] = struct{}{}
		}
	}

	for _, fav := range radio.Favourites {
		if strings.TrimSpace(fav.StationUUID) == "" || strings.TrimSpace(fav.StreamURL) == "" {
			continue
		}
		if opts.MergeRadio {
			exists, err := db.IsRadioFavourite(fav.StationUUID)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
		}
		if err := db.AddRadioFavourite(library.RadioFavourite{
			StationUUID: fav.StationUUID,
			Name:        fav.Name,
			StreamURL:   fav.StreamURL,
			FaviconURL:  fav.FaviconURL,
			Homepage:    fav.Homepage,
			Country:     fav.Country,
			Codec:       fav.Codec,
			Bitrate:     fav.Bitrate,
			Tags:        fav.Tags,
			Pinned:      fav.Pinned,
		}); err != nil {
			return err
		}
	}
	for _, st := range radio.CustomStations {
		if strings.TrimSpace(st.StreamURL) == "" {
			continue
		}
		uuid := st.StationUUID
		if uuid == "" {
			uuid = library.CustomRadioStationUUID(st.StreamURL)
		}
		if opts.MergeRadio {
			if _, ok := customExisting[uuid]; ok {
				continue
			}
		}
		if _, err := db.AddCustomRadioStation(library.RadioCustomStation{
			StationUUID: uuid,
			Name:        st.Name,
			StreamURL:   st.StreamURL,
			FaviconURL:  st.FaviconURL,
			Homepage:    st.Homepage,
			Country:     st.Country,
			Codec:       st.Codec,
			Bitrate:     st.Bitrate,
			Tags:        st.Tags,
		}); err != nil {
			return err
		}
	}
	return nil
}
