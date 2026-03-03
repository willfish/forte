package library

import (
	"encoding/base64"
	"fmt"
)

// Album represents an album in the library.
type Album struct {
	ID         int64
	Title      string
	Artist     string
	Year       int
	TrackCount int
}

// GetAlbums returns albums sorted by the given field and direction.
// Valid sort fields: "title", "artist", "year", "created_at".
// Valid order: "asc", "desc".
func (db *DB) GetAlbums(sort, order string) ([]Album, error) {
	orderClause := albumOrderClause(sort, order)

	rows, err := db.Query(`
		SELECT al.id, al.title, a.name, al.year, al.track_count
		FROM albums al
		JOIN artists a ON a.id = al.artist_id
		ORDER BY `+orderClause+`
		LIMIT 10000
	`)
	if err != nil {
		return nil, fmt.Errorf("get albums: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var albums []Album
	for rows.Next() {
		var al Album
		if err := rows.Scan(&al.ID, &al.Title, &al.Artist, &al.Year, &al.TrackCount); err != nil {
			return nil, fmt.Errorf("scan album: %w", err)
		}
		albums = append(albums, al)
	}
	return albums, rows.Err()
}

// AlbumArtwork returns the artwork for an album as a base64 data URI.
// Returns empty string if no artwork is stored.
func (db *DB) AlbumArtwork(albumID int64) (string, error) {
	var blob []byte
	err := db.QueryRow("SELECT artwork_blob FROM albums WHERE id = ?", albumID).Scan(&blob)
	if err != nil || len(blob) == 0 {
		return "", nil
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(blob), nil
}

// AlbumTracks returns the tracks for an album, ordered by disc and track number.
type AlbumTrack struct {
	TrackID     int64
	Title       string
	Artist      string
	TrackNumber int
	DiscNumber  int
	DurationMs  int
	FilePath    string
}

// GetAlbumTracks returns the tracks for a given album.
func (db *DB) GetAlbumTracks(albumID int64) ([]AlbumTrack, error) {
	rows, err := db.Query(`
		SELECT t.id, t.title, a.name, t.track_number, t.disc_number, t.duration_ms, t.file_path
		FROM tracks t
		JOIN artists a ON a.id = t.artist_id
		WHERE t.album_id = ?
		ORDER BY t.disc_number, t.track_number
	`, albumID)
	if err != nil {
		return nil, fmt.Errorf("get album tracks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tracks []AlbumTrack
	for rows.Next() {
		var t AlbumTrack
		if err := rows.Scan(&t.TrackID, &t.Title, &t.Artist, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.FilePath); err != nil {
			return nil, fmt.Errorf("scan album track: %w", err)
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func albumOrderClause(sort, order string) string {
	dir := "ASC"
	if order == "desc" {
		dir = "DESC"
	}

	switch sort {
	case "artist":
		return "a.name " + dir + ", al.year " + dir + ", al.title " + dir
	case "year":
		return "al.year " + dir + ", a.name ASC, al.title ASC"
	case "created_at":
		return "al.created_at " + dir
	default:
		return "al.title " + dir + ", a.name ASC"
	}
}
