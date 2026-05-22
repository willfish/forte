package library

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
)

// RadioFavourite represents a saved radio station.
type RadioFavourite struct {
	StationUUID string
	Name        string
	StreamURL   string
	FaviconURL  string
	Tags        string
	AddedAt     string
	Pinned      bool
}

// RadioCustomStation represents a user-defined radio station.
type RadioCustomStation struct {
	StationUUID string
	Name        string
	StreamURL   string
	FaviconURL  string
	Tags        string
	CreatedAt   string
	UpdatedAt   string
}

// RadioHistoryEntry represents a recently played radio station.
type RadioHistoryEntry struct {
	StationUUID  string
	Name         string
	StreamURL    string
	FaviconURL   string
	Tags         string
	TrackTitle   string
	PlayCount    int
	LastError    string
	LastPlayedAt string
}

// AppPreferences stores product-level preferences.
type AppPreferences struct {
	LibraryEnabled   bool
	StartLastStation bool
	AutoReconnect    bool
	ShowTitlebar     bool
}

// AddRadioFavourite saves a radio station to favourites.
func (db *DB) AddRadioFavourite(f RadioFavourite) error {
	_, err := db.Exec(
		`INSERT INTO radio_favourites (station_uuid, name, stream_url, favicon_url, tags, pinned)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(station_uuid) DO UPDATE SET
			name = excluded.name,
			stream_url = excluded.stream_url,
			favicon_url = excluded.favicon_url,
			tags = excluded.tags`,
		f.StationUUID, f.Name, f.StreamURL, f.FaviconURL, f.Tags, boolToInt(f.Pinned),
	)
	if err != nil {
		return fmt.Errorf("add radio favourite: %w", err)
	}
	return nil
}

// SetRadioFavouritePinned pins or unpins a favourite station.
func (db *DB) SetRadioFavouritePinned(stationUUID string, pinned bool) error {
	_, err := db.Exec("UPDATE radio_favourites SET pinned = ? WHERE station_uuid = ?", boolToInt(pinned), stationUUID)
	if err != nil {
		return fmt.Errorf("pin radio favourite: %w", err)
	}
	return nil
}

// RemoveRadioFavourite removes a radio station from favourites.
func (db *DB) RemoveRadioFavourite(stationUUID string) error {
	_, err := db.Exec("DELETE FROM radio_favourites WHERE station_uuid = ?", stationUUID)
	if err != nil {
		return fmt.Errorf("remove radio favourite: %w", err)
	}
	return nil
}

// GetRadioFavourites returns all saved radio stations ordered by name.
func (db *DB) GetRadioFavourites() ([]RadioFavourite, error) {
	rows, err := db.Query(
		`SELECT station_uuid, name, stream_url, favicon_url, tags, added_at, pinned
		 FROM radio_favourites
		 ORDER BY pinned DESC, name COLLATE NOCASE`,
	)
	if err != nil {
		return nil, fmt.Errorf("get radio favourites: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var favs []RadioFavourite
	for rows.Next() {
		var f RadioFavourite
		var pinned int
		if err := rows.Scan(&f.StationUUID, &f.Name, &f.StreamURL, &f.FaviconURL, &f.Tags, &f.AddedAt, &pinned); err != nil {
			return nil, fmt.Errorf("scan radio favourite: %w", err)
		}
		f.Pinned = pinned != 0
		favs = append(favs, f)
	}
	return favs, rows.Err()
}

// IsRadioFavourite checks if a station is in favourites.
func (db *DB) IsRadioFavourite(stationUUID string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT 1 FROM radio_favourites WHERE station_uuid = ?", stationUUID).Scan(&exists)
	if err != nil {
		return false, nil // Not found is not an error.
	}
	return true, nil
}

// AddCustomRadioStation saves a user-defined station.
func (db *DB) AddCustomRadioStation(st RadioCustomStation) (RadioCustomStation, error) {
	if st.StationUUID == "" {
		st.StationUUID = CustomRadioStationUUID(st.StreamURL)
	}
	_, err := db.Exec(
		`INSERT INTO radio_custom_stations (station_uuid, name, stream_url, favicon_url, tags)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(station_uuid) DO UPDATE SET
			name = excluded.name,
			stream_url = excluded.stream_url,
			favicon_url = excluded.favicon_url,
			tags = excluded.tags,
			updated_at = datetime('now')`,
		st.StationUUID, st.Name, st.StreamURL, st.FaviconURL, st.Tags,
	)
	if err != nil {
		return st, fmt.Errorf("add custom radio station: %w", err)
	}
	return st, nil
}

// DeleteCustomRadioStation removes a user-defined station.
func (db *DB) DeleteCustomRadioStation(stationUUID string) error {
	_, err := db.Exec("DELETE FROM radio_custom_stations WHERE station_uuid = ?", stationUUID)
	if err != nil {
		return fmt.Errorf("delete custom radio station: %w", err)
	}
	return nil
}

// GetCustomRadioStations returns saved user-defined stations.
func (db *DB) GetCustomRadioStations() ([]RadioCustomStation, error) {
	rows, err := db.Query(
		`SELECT station_uuid, name, stream_url, favicon_url, tags, created_at, updated_at
		 FROM radio_custom_stations
		 ORDER BY name COLLATE NOCASE`,
	)
	if err != nil {
		return nil, fmt.Errorf("get custom radio stations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var stations []RadioCustomStation
	for rows.Next() {
		var st RadioCustomStation
		if err := rows.Scan(&st.StationUUID, &st.Name, &st.StreamURL, &st.FaviconURL, &st.Tags, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan custom radio station: %w", err)
		}
		stations = append(stations, st)
	}
	return stations, rows.Err()
}

// RecordRadioPlayback upserts a station into radio playback history.
func (db *DB) RecordRadioPlayback(st RadioHistoryEntry) error {
	_, err := db.Exec(
		`INSERT INTO radio_history (station_uuid, name, stream_url, favicon_url, tags, track_title, play_count, last_error)
		 VALUES (?, ?, ?, ?, ?, ?, 1, ?)
		 ON CONFLICT(station_uuid) DO UPDATE SET
			name = excluded.name,
			stream_url = excluded.stream_url,
			favicon_url = excluded.favicon_url,
			tags = excluded.tags,
			track_title = excluded.track_title,
			play_count = radio_history.play_count + 1,
			last_error = excluded.last_error,
			last_played_at = datetime('now')`,
		st.StationUUID, st.Name, st.StreamURL, st.FaviconURL, st.Tags, st.TrackTitle, st.LastError,
	)
	if err != nil {
		return fmt.Errorf("record radio playback: %w", err)
	}
	return nil
}

// UpdateRadioHistoryTitle stores the latest ICY title for a history entry.
func (db *DB) UpdateRadioHistoryTitle(stationUUID, title string) error {
	_, err := db.Exec("UPDATE radio_history SET track_title = ? WHERE station_uuid = ?", title, stationUUID)
	if err != nil {
		return fmt.Errorf("update radio title: %w", err)
	}
	return nil
}

// MarkRadioHistoryError stores the latest playback error for a history entry.
func (db *DB) MarkRadioHistoryError(stationUUID, msg string) error {
	_, err := db.Exec("UPDATE radio_history SET last_error = ? WHERE station_uuid = ?", msg, stationUUID)
	if err != nil {
		return fmt.Errorf("mark radio error: %w", err)
	}
	return nil
}

// GetRadioHistory returns recently played radio stations.
func (db *DB) GetRadioHistory(limit int) ([]RadioHistoryEntry, error) {
	if limit <= 0 {
		limit = 25
	}
	rows, err := db.Query(
		`SELECT station_uuid, name, stream_url, favicon_url, tags, track_title, play_count, last_error, last_played_at
		 FROM radio_history
		 ORDER BY last_played_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get radio history: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []RadioHistoryEntry
	for rows.Next() {
		var e RadioHistoryEntry
		if err := rows.Scan(&e.StationUUID, &e.Name, &e.StreamURL, &e.FaviconURL, &e.Tags, &e.TrackTitle, &e.PlayCount, &e.LastError, &e.LastPlayedAt); err != nil {
			return nil, fmt.Errorf("scan radio history: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ClearRadioHistory deletes all radio playback history.
func (db *DB) ClearRadioHistory() error {
	_, err := db.Exec("DELETE FROM radio_history")
	if err != nil {
		return fmt.Errorf("clear radio history: %w", err)
	}
	return nil
}

// GetAppPreferences returns product-level preferences.
func (db *DB) GetAppPreferences() (AppPreferences, error) {
	var libraryEnabled, startLastStation, autoReconnect, showTitlebar int
	err := db.QueryRow(
		`SELECT library_enabled, start_last_station, auto_reconnect, show_titlebar
		 FROM app_preferences WHERE id = 1`,
	).Scan(&libraryEnabled, &startLastStation, &autoReconnect, &showTitlebar)
	if err != nil {
		return AppPreferences{}, fmt.Errorf("get app preferences: %w", err)
	}
	return AppPreferences{
		LibraryEnabled:   libraryEnabled != 0,
		StartLastStation: startLastStation != 0,
		AutoReconnect:    autoReconnect != 0,
		ShowTitlebar:     showTitlebar != 0,
	}, nil
}

// SaveAppPreferences stores product-level preferences.
func (db *DB) SaveAppPreferences(p AppPreferences) error {
	_, err := db.Exec(
		`INSERT INTO app_preferences (id, library_enabled, start_last_station, auto_reconnect, show_titlebar)
		 VALUES (1, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
			library_enabled = excluded.library_enabled,
			start_last_station = excluded.start_last_station,
			auto_reconnect = excluded.auto_reconnect,
			show_titlebar = excluded.show_titlebar`,
		boolToInt(p.LibraryEnabled), boolToInt(p.StartLastStation), boolToInt(p.AutoReconnect), boolToInt(p.ShowTitlebar),
	)
	if err != nil {
		return fmt.Errorf("save app preferences: %w", err)
	}
	return nil
}

// CustomRadioStationUUID returns a stable UUID for a custom stream URL.
func CustomRadioStationUUID(streamURL string) string {
	sum := sha1.Sum([]byte(streamURL))
	return "custom-" + hex.EncodeToString(sum[:8])
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
