package library

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/willfish/forte/internal/metadata"
)

// audioExtensions lists file extensions recognised as audio files.
var audioExtensions = map[string]bool{
	".flac": true, ".mp3": true, ".opus": true, ".ogg": true,
	".m4a": true, ".aac": true, ".wav": true, ".wv": true,
	".mpc": true, ".ape": true,
}

// Progress reports scanner progress.
type Progress struct {
	Scanned int
	Total   int
}

// Scanner populates the library database from music directories.
type Scanner struct {
	db *DB
}

// NewScanner creates a scanner for the given database.
func NewScanner(db *DB) *Scanner {
	return &Scanner{db: db}
}

// Scan walks the given directories and populates the database.
// Progress is reported on the optional progress channel.
// The scan is cancellable via context.
func (s *Scanner) Scan(ctx context.Context, dirs []string, progress chan<- Progress) error {
	// Collect all audio file paths first for progress reporting.
	var paths []string
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			return fmt.Errorf("scan dir: %w", err)
		}
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				slog.Warn("walk error", "path", path, "err", err)
				return nil // skip inaccessible paths
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if d.IsDir() {
				return nil
			}
			if isAudioFile(path) {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("walk %s: %w", dir, err)
		}
	}

	total := len(paths)
	scanned := 0

	// Process in batches.
	const batchSize = 100
	for i := 0; i < len(paths); i += batchSize {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		end := i + batchSize
		if end > len(paths) {
			end = len(paths)
		}
		batch := paths[i:end]

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin tx: %w", err)
		}

		for _, path := range batch {
			if ctx.Err() != nil {
				_ = tx.Rollback()
				return ctx.Err()
			}
			if err := s.processFile(ctx, tx, path); err != nil {
				slog.Warn("skipping file", "path", path, "err", err)
			}
			scanned++
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit batch: %w", err)
		}

		if progress != nil {
			select {
			case progress <- Progress{Scanned: scanned, Total: total}:
			default:
			}
		}
	}

	return nil
}

func (s *Scanner) processFile(ctx context.Context, tx *sql.Tx, path string) error {
	// Check if file already exists and hasn't changed.
	info, err := statFile(path)
	if err != nil {
		return err
	}

	var existingID int64
	var existingModTime string
	var existingSize int64
	err = tx.QueryRowContext(ctx,
		"SELECT id, file_mod_time, file_size FROM tracks WHERE file_path = ?", path,
	).Scan(&existingID, &existingModTime, &existingSize)

	if err == nil && existingModTime == info.modTime && existingSize == info.size {
		return nil // unchanged, skip
	}

	meta, err := metadata.ReadTags(path)
	if err != nil {
		return fmt.Errorf("read tags: %w", err)
	}

	artistID, err := s.upsertArtist(ctx, tx, meta.Artist)
	if err != nil {
		return fmt.Errorf("upsert artist: %w", err)
	}

	var albumID *int64
	if meta.Album != "" {
		id, err := s.upsertAlbum(ctx, tx, artistID, meta.Album, meta.Year)
		if err != nil {
			return fmt.Errorf("upsert album: %w", err)
		}
		albumID = &id
	}

	if meta.Genre != "" {
		if err := s.upsertGenre(ctx, tx, meta.Genre); err != nil {
			return fmt.Errorf("upsert genre: %w", err)
		}
	}

	trackID, err := s.upsertTrack(ctx, tx, albumID, artistID, meta, path, info)
	if err != nil {
		return fmt.Errorf("upsert track: %w", err)
	}

	if meta.Genre != "" {
		if err := s.linkGenre(ctx, tx, trackID, meta.Genre); err != nil {
			return fmt.Errorf("link genre: %w", err)
		}
	}

	// Update FTS index.
	if err := s.upsertFTS(ctx, tx, trackID, meta); err != nil {
		return fmt.Errorf("upsert fts: %w", err)
	}

	return nil
}

func (s *Scanner) upsertArtist(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	return upsertArtist(ctx, tx, name)
}

func (s *Scanner) upsertAlbum(ctx context.Context, tx *sql.Tx, artistID int64, title string, year int) (int64, error) {
	return upsertAlbum(ctx, tx, artistID, title, year, "", "")
}

// upsertArtist finds or creates an artist by name.
func upsertArtist(ctx context.Context, tx *sql.Tx, name string) (int64, error) {
	if name == "" {
		name = "Unknown Artist"
	}
	var id int64
	err := tx.QueryRowContext(ctx, "SELECT id FROM artists WHERE name = ?", name).Scan(&id)
	if err == nil {
		return id, nil
	}
	res, err := tx.ExecContext(ctx, "INSERT INTO artists (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// upsertAlbum finds or creates an album. When serverID is non-empty, matches on
// (server_id, remote_id) instead of (artist_id, title) to avoid cross-server collisions.
func upsertAlbum(ctx context.Context, tx *sql.Tx, artistID int64, title string, year int, serverID, remoteID string) (int64, error) {
	var id int64
	if serverID != "" {
		err := tx.QueryRowContext(ctx,
			"SELECT id FROM albums WHERE server_id = ? AND remote_id = ?", serverID, remoteID,
		).Scan(&id)
		if err == nil {
			// Update metadata in case it changed on the server.
			_, _ = tx.ExecContext(ctx,
				"UPDATE albums SET artist_id = ?, title = ?, year = ?, updated_at = datetime('now') WHERE id = ?",
				artistID, title, year, id,
			)
			return id, nil
		}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO albums (artist_id, title, year, server_id, remote_id) VALUES (?, ?, ?, ?, ?)",
			artistID, title, year, serverID, remoteID,
		)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}

	err := tx.QueryRowContext(ctx,
		"SELECT id FROM albums WHERE artist_id = ? AND title = ? AND server_id = ''", artistID, title,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	res, err := tx.ExecContext(ctx,
		"INSERT INTO albums (artist_id, title, year) VALUES (?, ?, ?)", artistID, title, year,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Scanner) upsertGenre(ctx context.Context, tx *sql.Tx, name string) error {
	_, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO genres (name) VALUES (?)", name)
	return err
}

func (s *Scanner) upsertTrack(ctx context.Context, tx *sql.Tx, albumID *int64, artistID int64, meta metadata.TrackMeta, path string, info fileInfo) (int64, error) {
	// Delete existing if present (for re-scan of changed files).
	_, _ = tx.ExecContext(ctx, "DELETE FROM fts_tracks WHERE rowid IN (SELECT id FROM tracks WHERE file_path = ?)", path)
	_, _ = tx.ExecContext(ctx, "DELETE FROM track_genres WHERE track_id IN (SELECT id FROM tracks WHERE file_path = ?)", path)
	_, _ = tx.ExecContext(ctx, "DELETE FROM tracks WHERE file_path = ?", path)

	res, err := tx.ExecContext(ctx, `INSERT INTO tracks
		(album_id, artist_id, title, track_number, disc_number, duration_ms,
		 file_path, file_size, file_mod_time, format, bitrate)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		albumID, artistID, meta.Title, meta.TrackNumber, meta.DiscNumber,
		meta.Duration.Milliseconds(), path, info.size, info.modTime,
		formatFromExt(path), meta.Bitrate,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Scanner) linkGenre(ctx context.Context, tx *sql.Tx, trackID int64, genre string) error {
	_, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO track_genres (track_id, genre_id)
		SELECT ?, id FROM genres WHERE name = ?`, trackID, genre)
	return err
}

func (s *Scanner) upsertFTS(ctx context.Context, tx *sql.Tx, trackID int64, meta metadata.TrackMeta) error {
	_, err := tx.ExecContext(ctx,
		"INSERT INTO fts_tracks (rowid, title, artist, album, genre) VALUES (?, ?, ?, ?, ?)",
		trackID, meta.Title, meta.Artist, meta.Album, meta.Genre,
	)
	return err
}

type fileInfo struct {
	size    int64
	modTime string
}

func statFile(path string) (fileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return fileInfo{}, err
	}
	return fileInfo{
		size:    fi.Size(),
		modTime: fi.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

func isAudioFile(path string) bool {
	return audioExtensions[strings.ToLower(filepath.Ext(path))]
}

func formatFromExt(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".flac":
		return "FLAC"
	case ".mp3":
		return "MP3"
	case ".opus":
		return "Opus"
	case ".ogg":
		return "OGG"
	case ".m4a":
		return "M4A"
	case ".aac":
		return "AAC"
	case ".wav":
		return "WAV"
	case ".wv":
		return "WavPack"
	case ".mpc":
		return "Musepack"
	case ".ape":
		return "APE"
	default:
		return strings.TrimPrefix(ext, ".")
	}
}
