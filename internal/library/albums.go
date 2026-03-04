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
	ServerID   string
}

// GetAlbums returns albums sorted by the given field and direction.
// Valid sort fields: "title", "artist", "year", "created_at".
// Valid order: "asc", "desc".
// Source filter: "" (all, deduped), "local", "server".
func (db *DB) GetAlbums(sort, order, source string) ([]Album, error) {
	var query string
	switch source {
	case "local":
		query = `
			SELECT al.id, al.title, a.name, al.year, al.track_count, al.server_id
			FROM albums al
			JOIN artists a ON a.id = al.artist_id
			WHERE al.server_id = ''
			ORDER BY ` + directOrderClause(sort, order) + `
			LIMIT 10000`
	case "server":
		query = `
			SELECT al.id, al.title, a.name, al.year, al.track_count, al.server_id
			FROM albums al
			JOIN artists a ON a.id = al.artist_id
			WHERE al.server_id != ''
			ORDER BY ` + directOrderClause(sort, order) + `
			LIMIT 10000`
	default:
		// All albums, deduped: prefer local over server when same title+artist.
		query = `
			SELECT id, title, artist, year, track_count, server_id FROM (
				SELECT al.id, al.title, a.name AS artist, al.year, al.track_count, al.server_id,
					ROW_NUMBER() OVER (
						PARTITION BY LOWER(al.title), LOWER(a.name)
						ORDER BY CASE WHEN al.server_id = '' THEN 0 ELSE 1 END
					) AS rn
				FROM albums al
				JOIN artists a ON a.id = al.artist_id
			) WHERE rn = 1
			ORDER BY ` + dedupOrderClause(sort, order) + `
			LIMIT 10000`
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("get albums: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var albums []Album
	for rows.Next() {
		var al Album
		if err := rows.Scan(&al.ID, &al.Title, &al.Artist, &al.Year, &al.TrackCount, &al.ServerID); err != nil {
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

// AlbumTrack represents a track belonging to an album.
type AlbumTrack struct {
	TrackID     int64
	Title       string
	Artist      string
	TrackNumber int
	DiscNumber  int
	DurationMs  int
	FilePath    string
	ServerID    string
}

// GetAlbumTracks returns the tracks for a given album.
func (db *DB) GetAlbumTracks(albumID int64) ([]AlbumTrack, error) {
	rows, err := db.Query(`
		SELECT t.id, t.title, a.name, t.track_number, t.disc_number, t.duration_ms, t.file_path, t.server_id
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
		if err := rows.Scan(&t.TrackID, &t.Title, &t.Artist, &t.TrackNumber, &t.DiscNumber, &t.DurationMs, &t.FilePath, &t.ServerID); err != nil {
			return nil, fmt.Errorf("scan album track: %w", err)
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

// directOrderClause returns ORDER BY for queries using al.* and a.name aliases.
func directOrderClause(sort, order string) string {
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

// dedupOrderClause returns ORDER BY for the dedup subquery output columns.
func dedupOrderClause(sort, order string) string {
	dir := "ASC"
	if order == "desc" {
		dir = "DESC"
	}
	switch sort {
	case "artist":
		return "artist " + dir + ", year " + dir + ", title " + dir
	case "year":
		return "year " + dir + ", artist ASC, title ASC"
	case "created_at":
		return "id " + dir
	default:
		return "title " + dir + ", artist ASC"
	}
}
