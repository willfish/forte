package library

import "fmt"

// RadioFavourite represents a saved radio station.
type RadioFavourite struct {
	StationUUID string
	Name        string
	StreamURL   string
	FaviconURL  string
	Tags        string
	AddedAt     string
}

// AddRadioFavourite saves a radio station to favourites.
func (db *DB) AddRadioFavourite(f RadioFavourite) error {
	_, err := db.Exec(
		`INSERT OR IGNORE INTO radio_favourites (station_uuid, name, stream_url, favicon_url, tags)
		 VALUES (?, ?, ?, ?, ?)`,
		f.StationUUID, f.Name, f.StreamURL, f.FaviconURL, f.Tags,
	)
	if err != nil {
		return fmt.Errorf("add radio favourite: %w", err)
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
		"SELECT station_uuid, name, stream_url, favicon_url, tags, added_at FROM radio_favourites ORDER BY name",
	)
	if err != nil {
		return nil, fmt.Errorf("get radio favourites: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var favs []RadioFavourite
	for rows.Next() {
		var f RadioFavourite
		if err := rows.Scan(&f.StationUUID, &f.Name, &f.StreamURL, &f.FaviconURL, &f.Tags, &f.AddedAt); err != nil {
			return nil, fmt.Errorf("scan radio favourite: %w", err)
		}
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
