package library

import "fmt"

// Playlist represents a named playlist.
type Playlist struct {
	ID        int64
	Name      string
	CreatedAt string
	UpdatedAt string
}

// PlaylistTrack represents a track within a playlist, including its position.
type PlaylistTrack struct {
	TrackID    int64
	Title      string
	Artist     string
	Album      string
	DurationMs int
	FilePath   string
	Position   int
}

// CreatePlaylist creates a new playlist with the given name.
func (db *DB) CreatePlaylist(name string) (int64, error) {
	result, err := db.Exec(
		"INSERT INTO playlists (name) VALUES (?)", name,
	)
	if err != nil {
		return 0, fmt.Errorf("create playlist: %w", err)
	}
	return result.LastInsertId()
}

// GetPlaylists returns all playlists ordered by name.
func (db *DB) GetPlaylists() ([]Playlist, error) {
	rows, err := db.Query(`
		SELECT id, name, created_at, updated_at
		FROM playlists
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("get playlists: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var playlists []Playlist
	for rows.Next() {
		var p Playlist
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan playlist: %w", err)
		}
		playlists = append(playlists, p)
	}
	return playlists, rows.Err()
}

// RenamePlaylist renames a playlist.
func (db *DB) RenamePlaylist(id int64, name string) error {
	_, err := db.Exec(
		"UPDATE playlists SET name = ?, updated_at = datetime('now') WHERE id = ?",
		name, id,
	)
	if err != nil {
		return fmt.Errorf("rename playlist: %w", err)
	}
	return nil
}

// DeletePlaylist deletes a playlist and its track associations.
func (db *DB) DeletePlaylist(id int64) error {
	_, err := db.Exec("DELETE FROM playlists WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete playlist: %w", err)
	}
	return nil
}

// GetPlaylistTracks returns the tracks in a playlist, ordered by position.
func (db *DB) GetPlaylistTracks(playlistID int64) ([]PlaylistTrack, error) {
	rows, err := db.Query(`
		SELECT t.id, t.title, a.name, COALESCE(al.title, ''), t.duration_ms, t.file_path, pt.position
		FROM playlist_tracks pt
		JOIN tracks t ON t.id = pt.track_id
		JOIN artists a ON a.id = t.artist_id
		LEFT JOIN albums al ON al.id = t.album_id
		WHERE pt.playlist_id = ?
		ORDER BY pt.position
	`, playlistID)
	if err != nil {
		return nil, fmt.Errorf("get playlist tracks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tracks []PlaylistTrack
	for rows.Next() {
		var t PlaylistTrack
		if err := rows.Scan(&t.TrackID, &t.Title, &t.Artist, &t.Album, &t.DurationMs, &t.FilePath, &t.Position); err != nil {
			return nil, fmt.Errorf("scan playlist track: %w", err)
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

// AddTrackToPlaylist adds a track to a playlist at the end.
func (db *DB) AddTrackToPlaylist(playlistID, trackID int64) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO playlist_tracks (playlist_id, track_id, position)
		VALUES (?, ?, (SELECT COALESCE(MAX(position), -1) + 1 FROM playlist_tracks WHERE playlist_id = ?))
	`, playlistID, trackID, playlistID)
	if err != nil {
		return fmt.Errorf("add track to playlist: %w", err)
	}
	_, err = db.Exec(
		"UPDATE playlists SET updated_at = datetime('now') WHERE id = ?", playlistID,
	)
	return err
}

// RemoveTrackFromPlaylist removes a track from a playlist and reorders positions.
func (db *DB) RemoveTrackFromPlaylist(playlistID, trackID int64) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	if _, err := tx.Exec(
		"DELETE FROM playlist_tracks WHERE playlist_id = ? AND track_id = ?",
		playlistID, trackID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("remove track: %w", err)
	}

	// Reorder remaining positions to be contiguous.
	if _, err := tx.Exec(`
		UPDATE playlist_tracks SET position = (
			SELECT COUNT(*) FROM playlist_tracks pt2
			WHERE pt2.playlist_id = playlist_tracks.playlist_id
			AND pt2.position < playlist_tracks.position
		)
		WHERE playlist_id = ?
	`, playlistID); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("reorder positions: %w", err)
	}

	if _, err := tx.Exec(
		"UPDATE playlists SET updated_at = datetime('now') WHERE id = ?", playlistID,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

// MoveTrackInPlaylist moves a track from one position to another within a playlist.
func (db *DB) MoveTrackInPlaylist(playlistID int64, fromPos, toPos int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}

	// Temporarily set moving track to a sentinel position.
	if _, err := tx.Exec(
		"UPDATE playlist_tracks SET position = -1 WHERE playlist_id = ? AND position = ?",
		playlistID, fromPos,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("mark moving track: %w", err)
	}

	if fromPos < toPos {
		// Moving down: shift tracks between from+1 and to upward.
		if _, err := tx.Exec(`
			UPDATE playlist_tracks SET position = position - 1
			WHERE playlist_id = ? AND position > ? AND position <= ?
		`, playlistID, fromPos, toPos); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("shift tracks: %w", err)
		}
	} else {
		// Moving up: shift tracks between to and from-1 downward.
		if _, err := tx.Exec(`
			UPDATE playlist_tracks SET position = position + 1
			WHERE playlist_id = ? AND position >= ? AND position < ?
		`, playlistID, toPos, fromPos); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("shift tracks: %w", err)
		}
	}

	// Place moving track at the target position.
	if _, err := tx.Exec(
		"UPDATE playlist_tracks SET position = ? WHERE playlist_id = ? AND position = -1",
		toPos, playlistID,
	); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("place track: %w", err)
	}

	if _, err := tx.Exec(
		"UPDATE playlists SET updated_at = datetime('now') WHERE id = ?", playlistID,
	); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
