package library

import (
	"fmt"
	"strings"
)

// SearchResult represents a single search result.
type SearchResult struct {
	TrackID     int64
	Title       string
	Artist      string
	Album       string
	Genre       string
	TrackNumber int
	DiscNumber  int
	DurationMs  int
	FilePath    string
	ServerID    string
}

// Search queries the FTS5 index and returns matching tracks.
// An empty query returns all tracks (up to limit).
func (db *DB) Search(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 100
	}

	if strings.TrimSpace(query) == "" {
		return db.allTracks(limit)
	}

	return db.ftsSearch(query, limit)
}

func (db *DB) ftsSearch(query string, limit int) ([]SearchResult, error) {
	// Append * for prefix matching ("beet" -> "beet*").
	ftsQuery := sanitizeFTS(query)

	rows, err := db.Query(`
		SELECT t.id, t.title, a.name, COALESCE(al.title, ''),
		       COALESCE(GROUP_CONCAT(DISTINCT g.name), ''),
		       t.track_number, t.disc_number, t.duration_ms, t.file_path, t.server_id
		FROM fts_tracks f
		JOIN tracks t ON t.id = f.rowid
		JOIN artists a ON a.id = t.artist_id
		LEFT JOIN albums al ON al.id = t.album_id
		LEFT JOIN track_genres tg ON tg.track_id = t.id
		LEFT JOIN genres g ON g.id = tg.genre_id
		WHERE fts_tracks MATCH ?
		GROUP BY t.id
		ORDER BY rank
		LIMIT ?
	`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanResults(rows)
}

func (db *DB) allTracks(limit int) ([]SearchResult, error) {
	rows, err := db.Query(`
		SELECT t.id, t.title, a.name, COALESCE(al.title, ''),
		       COALESCE(GROUP_CONCAT(DISTINCT g.name), ''),
		       t.track_number, t.disc_number, t.duration_ms, t.file_path, t.server_id
		FROM tracks t
		JOIN artists a ON a.id = t.artist_id
		LEFT JOIN albums al ON al.id = t.album_id
		LEFT JOIN track_genres tg ON tg.track_id = t.id
		LEFT JOIN genres g ON g.id = tg.genre_id
		GROUP BY t.id
		ORDER BY a.name, al.title, t.disc_number, t.track_number
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("all tracks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return scanResults(rows)
}

func scanResults(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		if err := rows.Scan(
			&r.TrackID, &r.Title, &r.Artist, &r.Album, &r.Genre,
			&r.TrackNumber, &r.DiscNumber, &r.DurationMs, &r.FilePath, &r.ServerID,
		); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// sanitizeFTS escapes special FTS5 characters and appends * for prefix matching.
func sanitizeFTS(query string) string {
	// Split into terms and make each a prefix query.
	terms := strings.Fields(query)
	for i, term := range terms {
		// Escape double quotes in terms.
		term = strings.ReplaceAll(term, "\"", "")
		if term != "" {
			terms[i] = "\"" + term + "\"" + "*"
		}
	}
	return strings.Join(terms, " ")
}
