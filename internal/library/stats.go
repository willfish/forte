package library

import "fmt"

// StatEntry represents a ranked item in top-artists/albums/tracks results.
type StatEntry struct {
	Name       string
	SecondLine string // artist name for albums/tracks, empty for artists
	PlayCount  int
	TotalMs    int64
}

// RecentPlay represents a single recently played track.
type RecentPlay struct {
	TrackID    int64
	Title      string
	Artist     string
	Album      string
	DurationMs int
	PlayedAt   string
}

// RecordPlay inserts a play into the history table.
func (db *DB) RecordPlay(trackID int64, durationPlayedMs int) error {
	_, err := db.Exec(
		"INSERT INTO play_history (track_id, duration_played_ms) VALUES (?, ?)",
		trackID, durationPlayedMs,
	)
	return err
}

// TopArtists returns the most-played artists within the given period.
func (db *DB) TopArtists(period string, limit int) ([]StatEntry, error) {
	where := periodCondition(period)
	q := fmt.Sprintf(`
		SELECT a.name, '', COUNT(*) AS play_count, COALESCE(SUM(ph.duration_played_ms), 0)
		FROM play_history ph
		JOIN tracks t ON t.id = ph.track_id
		JOIN artists a ON a.id = t.artist_id
		%s
		GROUP BY a.id
		ORDER BY play_count DESC
		LIMIT ?`, where)

	return db.queryStats(q, limit)
}

// TopAlbums returns the most-played albums within the given period.
func (db *DB) TopAlbums(period string, limit int) ([]StatEntry, error) {
	where := periodCondition(period)
	q := fmt.Sprintf(`
		SELECT al.title, a.name, COUNT(*) AS play_count, COALESCE(SUM(ph.duration_played_ms), 0)
		FROM play_history ph
		JOIN tracks t ON t.id = ph.track_id
		JOIN albums al ON al.id = t.album_id
		JOIN artists a ON a.id = al.artist_id
		%s
		GROUP BY al.id
		ORDER BY play_count DESC
		LIMIT ?`, where)

	return db.queryStats(q, limit)
}

// TopTracks returns the most-played tracks within the given period.
func (db *DB) TopTracks(period string, limit int) ([]StatEntry, error) {
	where := periodCondition(period)
	q := fmt.Sprintf(`
		SELECT t.title, a.name, COUNT(*) AS play_count, COALESCE(SUM(ph.duration_played_ms), 0)
		FROM play_history ph
		JOIN tracks t ON t.id = ph.track_id
		JOIN artists a ON a.id = t.artist_id
		%s
		GROUP BY t.id
		ORDER BY play_count DESC
		LIMIT ?`, where)

	return db.queryStats(q, limit)
}

// RecentlyPlayed returns the most recent plays.
func (db *DB) RecentlyPlayed(limit int) ([]RecentPlay, error) {
	rows, err := db.Query(`
		SELECT ph.track_id, t.title, a.name, COALESCE(al.title, ''), t.duration_ms, ph.played_at
		FROM play_history ph
		JOIN tracks t ON t.id = ph.track_id
		JOIN artists a ON a.id = t.artist_id
		LEFT JOIN albums al ON al.id = t.album_id
		ORDER BY ph.played_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []RecentPlay
	for rows.Next() {
		var r RecentPlay
		if err := rows.Scan(&r.TrackID, &r.Title, &r.Artist, &r.Album, &r.DurationMs, &r.PlayedAt); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

// queryStats is a shared helper for the top-N queries.
func (db *DB) queryStats(query string, limit int) ([]StatEntry, error) {
	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []StatEntry
	for rows.Next() {
		var e StatEntry
		if err := rows.Scan(&e.Name, &e.SecondLine, &e.PlayCount, &e.TotalMs); err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

// periodCondition returns a SQL WHERE clause fragment for filtering by period.
// Supported values: "7d", "30d", "12m", "all" (or empty).
func periodCondition(period string) string {
	switch period {
	case "7d":
		return "WHERE ph.played_at >= datetime('now', '-7 days')"
	case "30d":
		return "WHERE ph.played_at >= datetime('now', '-30 days')"
	case "12m":
		return "WHERE ph.played_at >= datetime('now', '-12 months')"
	default:
		return ""
	}
}
