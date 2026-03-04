package library

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// SimilarArtist represents an artist similar to the queried one.
type SimilarArtist struct {
	Name string `json:"name"`
}

// ArtistMeta holds cached metadata for an artist.
type ArtistMeta struct {
	ArtistID    int64           `json:"artistId"`
	Bio         string          `json:"bio"`
	ImageURL    string          `json:"imageUrl"`
	Similar     []SimilarArtist `json:"similar"`
	MbID        string          `json:"mbId"`
	MbArea      string          `json:"mbArea"`
	MbType      string          `json:"mbType"`
	MbBegin     string          `json:"mbBegin"`
	MbEnd       string          `json:"mbEnd"`
	MbTags      string          `json:"mbTags"`
	FetchedAt   time.Time       `json:"fetchedAt"`
}

const cacheTTL = 30 * 24 * time.Hour

// GetArtistByName returns the artist ID for the given name, or sql.ErrNoRows if not found.
func (db *DB) GetArtistByName(name string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM artists WHERE name = ? LIMIT 1", name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get artist by name: %w", err)
	}
	return id, nil
}

// GetArtistMeta returns cached metadata for the artist, or nil if missing or expired.
func (db *DB) GetArtistMeta(artistID int64) (*ArtistMeta, error) {
	var meta ArtistMeta
	var similarJSON, fetchedAtStr string

	err := db.QueryRow(`
		SELECT artist_id, bio, image_url, similar_json, mb_id, mb_area, mb_type, mb_begin, mb_end, mb_tags, fetched_at
		FROM artist_metadata WHERE artist_id = ?`, artistID,
	).Scan(
		&meta.ArtistID, &meta.Bio, &meta.ImageURL, &similarJSON,
		&meta.MbID, &meta.MbArea, &meta.MbType, &meta.MbBegin, &meta.MbEnd, &meta.MbTags,
		&fetchedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get artist meta: %w", err)
	}

	fetched, err := time.Parse("2006-01-02 15:04:05", fetchedAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse fetched_at: %w", err)
	}
	if time.Since(fetched) > cacheTTL {
		return nil, nil
	}
	meta.FetchedAt = fetched

	if err := json.Unmarshal([]byte(similarJSON), &meta.Similar); err != nil {
		meta.Similar = nil
	}

	return &meta, nil
}

// SaveArtistMeta upserts artist metadata into the cache.
func (db *DB) SaveArtistMeta(artistID int64, meta ArtistMeta) error {
	similarJSON, err := json.Marshal(meta.Similar)
	if err != nil {
		return fmt.Errorf("marshal similar: %w", err)
	}

	_, err = db.Exec(`
		INSERT INTO artist_metadata (artist_id, bio, image_url, similar_json, mb_id, mb_area, mb_type, mb_begin, mb_end, mb_tags, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
		ON CONFLICT(artist_id) DO UPDATE SET
			bio = excluded.bio,
			image_url = excluded.image_url,
			similar_json = excluded.similar_json,
			mb_id = excluded.mb_id,
			mb_area = excluded.mb_area,
			mb_type = excluded.mb_type,
			mb_begin = excluded.mb_begin,
			mb_end = excluded.mb_end,
			mb_tags = excluded.mb_tags,
			fetched_at = excluded.fetched_at`,
		artistID, meta.Bio, meta.ImageURL, string(similarJSON),
		meta.MbID, meta.MbArea, meta.MbType, meta.MbBegin, meta.MbEnd, meta.MbTags,
	)
	if err != nil {
		return fmt.Errorf("save artist meta: %w", err)
	}
	return nil
}

// GetArtistAlbums returns all albums for a given artist, sorted by year.
func (db *DB) GetArtistAlbums(artistID int64) ([]Album, error) {
	rows, err := db.Query(`
		SELECT a.id, a.title, ar.name, a.year, a.track_count, a.server_id
		FROM albums a
		JOIN artists ar ON ar.id = a.artist_id
		WHERE a.artist_id = ?
		ORDER BY a.year ASC, a.title ASC`, artistID)
	if err != nil {
		return nil, fmt.Errorf("get artist albums: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var albums []Album
	for rows.Next() {
		var a Album
		if err := rows.Scan(&a.ID, &a.Title, &a.Artist, &a.Year, &a.TrackCount, &a.ServerID); err != nil {
			return nil, fmt.Errorf("scan artist album: %w", err)
		}
		albums = append(albums, a)
	}
	return albums, rows.Err()
}
