package library

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/willfish/forte/internal/streaming"
	"github.com/willfish/forte/internal/streaming/jellyfin"
	"github.com/willfish/forte/internal/streaming/subsonic"
)

// artworkClient is used for fetching album artwork with a generous timeout.
var artworkClient = &http.Client{Timeout: 30 * time.Second}

// SyncAllServers syncs all configured servers into the local database.
func SyncAllServers(ctx context.Context, db *DB) error {
	servers, err := db.GetServers()
	if err != nil {
		return fmt.Errorf("sync: get servers: %w", err)
	}
	for _, srv := range servers {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := SyncServer(ctx, db, srv); err != nil {
			slog.Warn("sync server failed", "server", srv.Name, "err", err)
		}
	}
	return nil
}

// SyncServer syncs a single server's catalog into the local database.
func SyncServer(ctx context.Context, db *DB, srv Server) error {
	provider, err := newProvider(srv)
	if err != nil {
		return err
	}

	// Collect all album IDs seen during this sync for reconciliation.
	seenAlbumRemoteIDs := make(map[string]bool)
	seenTrackFilePaths := make(map[string]bool)

	// Paginate through all albums.
	const pageSize = 500
	for offset := 0; ; offset += pageSize {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		albums, err := provider.GetAlbums("alphabeticalByName", offset, pageSize)
		if err != nil {
			return fmt.Errorf("sync: get albums at offset %d: %w", offset, err)
		}
		if len(albums) == 0 {
			break
		}

		for _, album := range albums {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			seenAlbumRemoteIDs[album.ID] = true

			if err := syncAlbum(ctx, db, provider, srv, album); err != nil {
				slog.Warn("sync album failed", "album", album.Title, "err", err)
				continue
			}

			// Track all file_paths for this album's tracks.
			albumDetail, tracks, err := provider.GetAlbum(album.ID)
			_ = albumDetail
			if err == nil {
				for _, t := range tracks {
					seenTrackFilePaths[serverFilePath(srv.ID, t.ID)] = true
				}
			}
		}

		if len(albums) < pageSize {
			break
		}
	}

	// Reconcile: remove tracks and albums for this server that were not seen.
	if err := reconcile(ctx, db, srv.ID, seenAlbumRemoteIDs, seenTrackFilePaths); err != nil {
		return fmt.Errorf("sync: reconcile: %w", err)
	}

	return nil
}

func syncAlbum(ctx context.Context, db *DB, provider streaming.Provider, srv Server, album streaming.Album) error {
	_, tracks, err := provider.GetAlbum(album.ID)
	if err != nil {
		return fmt.Errorf("get album detail: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	artistID, err := upsertArtist(ctx, tx, album.Artist)
	if err != nil {
		return fmt.Errorf("upsert artist: %w", err)
	}

	albumID, err := upsertAlbum(ctx, tx, artistID, album.Title, album.Year, srv.ID, album.ID)
	if err != nil {
		return fmt.Errorf("upsert album: %w", err)
	}

	// Update track count.
	_, _ = tx.ExecContext(ctx,
		"UPDATE albums SET track_count = ? WHERE id = ?", len(tracks), albumID,
	)

	for _, t := range tracks {
		filePath := serverFilePath(srv.ID, t.ID)

		// Upsert genre if present.
		if t.Genre != "" {
			_, _ = tx.ExecContext(ctx, "INSERT OR IGNORE INTO genres (name) VALUES (?)", t.Genre)
		}

		// Delete existing track data for re-sync.
		_, _ = tx.ExecContext(ctx, "DELETE FROM fts_tracks WHERE rowid IN (SELECT id FROM tracks WHERE file_path = ?)", filePath)
		_, _ = tx.ExecContext(ctx, "DELETE FROM track_genres WHERE track_id IN (SELECT id FROM tracks WHERE file_path = ?)", filePath)
		_, _ = tx.ExecContext(ctx, "DELETE FROM tracks WHERE file_path = ?", filePath)

		res, err := tx.ExecContext(ctx, `INSERT INTO tracks
			(album_id, artist_id, title, track_number, disc_number, duration_ms,
			 file_path, file_size, format, server_id, remote_id)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			albumID, artistID, t.Title, t.TrackNumber, t.DiscNumber,
			t.DurationMs, filePath, t.Size, t.ContentType, srv.ID, t.ID,
		)
		if err != nil {
			slog.Warn("sync: insert track", "track", t.Title, "err", err)
			continue
		}

		trackID, _ := res.LastInsertId()

		if t.Genre != "" {
			_, _ = tx.ExecContext(ctx, `INSERT OR IGNORE INTO track_genres (track_id, genre_id)
				SELECT ?, id FROM genres WHERE name = ?`, trackID, t.Genre)
		}

		// FTS index.
		_, _ = tx.ExecContext(ctx,
			"INSERT INTO fts_tracks (rowid, title, artist, album, genre) VALUES (?, ?, ?, ?, ?)",
			trackID, t.Title, t.Artist, album.Title, t.Genre,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// Fetch and store artwork outside the transaction.
	if album.CoverArtID != "" {
		artURL := provider.CoverArtURL(album.CoverArtID)
		if artURL != "" {
			if data, err := fetchArtwork(artURL); err == nil && len(data) > 0 {
				_, _ = db.ExecContext(ctx,
					"UPDATE albums SET artwork_blob = ? WHERE id = ?", data, albumID,
				)
			}
		}
	}

	return nil
}

func reconcile(ctx context.Context, db *DB, serverID string, seenAlbums map[string]bool, seenTracks map[string]bool) error {
	// Remove tracks not seen in this sync.
	rows, err := db.QueryContext(ctx,
		"SELECT id, file_path, remote_id FROM tracks WHERE server_id = ?", serverID,
	)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	var staleTrackIDs []int64
	for rows.Next() {
		var id int64
		var fp, rid string
		if err := rows.Scan(&id, &fp, &rid); err != nil {
			continue
		}
		if !seenTracks[fp] {
			staleTrackIDs = append(staleTrackIDs, id)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, id := range staleTrackIDs {
		_, _ = db.ExecContext(ctx, "DELETE FROM fts_tracks WHERE rowid = ?", id)
		_, _ = db.ExecContext(ctx, "DELETE FROM track_genres WHERE track_id = ?", id)
		_, _ = db.ExecContext(ctx, "DELETE FROM tracks WHERE id = ?", id)
	}

	// Remove albums not seen in this sync.
	albumRows, err := db.QueryContext(ctx,
		"SELECT id, remote_id FROM albums WHERE server_id = ?", serverID,
	)
	if err != nil {
		return err
	}
	defer func() { _ = albumRows.Close() }()

	var staleAlbumIDs []int64
	for albumRows.Next() {
		var id int64
		var rid string
		if err := albumRows.Scan(&id, &rid); err != nil {
			continue
		}
		if !seenAlbums[rid] {
			staleAlbumIDs = append(staleAlbumIDs, id)
		}
	}
	if err := albumRows.Err(); err != nil {
		return err
	}

	for _, id := range staleAlbumIDs {
		_, _ = db.ExecContext(ctx, "DELETE FROM albums WHERE id = ?", id)
	}

	return nil
}

// serverFilePath returns the synthetic file path for a server track.
func serverFilePath(serverID, remoteID string) string {
	return "server://" + serverID + "/" + remoteID
}

func newProvider(srv Server) (streaming.Provider, error) {
	switch srv.Type {
	case "subsonic":
		return subsonic.New(srv.URL, srv.Username, srv.Password), nil
	case "jellyfin":
		return jellyfin.New(srv.URL, srv.Username, srv.Password), nil
	default:
		return nil, fmt.Errorf("unknown server type: %s", srv.Type)
	}
}

func fetchArtwork(url string) ([]byte, error) {
	resp, err := artworkClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("artwork: status %d", resp.StatusCode)
	}

	// Cap at 5MB to avoid memory issues.
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}
	return data, nil
}
