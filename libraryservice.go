package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/library"
)

// LibraryService exposes the music library to the frontend.
type LibraryService struct {
	db *library.DB
}

// ServiceStartup opens the library database when the application starts.
func (s *LibraryService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("config dir: %w", err)
	}

	dbDir := filepath.Join(dataDir, "forte")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	db, err := library.OpenDB(filepath.Join(dbDir, "library.db"))
	if err != nil {
		return fmt.Errorf("library startup: %w", err)
	}
	s.db = db
	return nil
}

// ServiceShutdown closes the library database when the application exits.
func (s *LibraryService) ServiceShutdown() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Album is the JSON-friendly album type exposed to the frontend.
type Album struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Year       int    `json:"year"`
	TrackCount int    `json:"trackCount"`
}

// GetAlbums returns all albums sorted by the given field and direction.
func (s *LibraryService) GetAlbums(sort string, order string) ([]Album, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}

	albums, err := s.db.GetAlbums(sort, order)
	if err != nil {
		return nil, err
	}

	result := make([]Album, len(albums))
	for i, a := range albums {
		result[i] = Album{
			ID:         a.ID,
			Title:      a.Title,
			Artist:     a.Artist,
			Year:       a.Year,
			TrackCount: a.TrackCount,
		}
	}
	return result, nil
}

// AlbumArtwork returns the artwork for an album as a base64 data URI.
func (s *LibraryService) AlbumArtwork(albumID int64) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("library not initialised")
	}
	return s.db.AlbumArtwork(albumID)
}

// AlbumTrack is the JSON-friendly track type exposed to the frontend.
type AlbumTrack struct {
	TrackID     int64  `json:"trackId"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	TrackNumber int    `json:"trackNumber"`
	DiscNumber  int    `json:"discNumber"`
	DurationMs  int    `json:"durationMs"`
	FilePath    string `json:"filePath"`
}

// GetAlbumTracks returns the tracks for a given album.
func (s *LibraryService) GetAlbumTracks(albumID int64) ([]AlbumTrack, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}

	tracks, err := s.db.GetAlbumTracks(albumID)
	if err != nil {
		return nil, err
	}

	result := make([]AlbumTrack, len(tracks))
	for i, t := range tracks {
		result[i] = AlbumTrack{
			TrackID:     t.TrackID,
			Title:       t.Title,
			Artist:      t.Artist,
			TrackNumber: t.TrackNumber,
			DiscNumber:  t.DiscNumber,
			DurationMs:  t.DurationMs,
			FilePath:    t.FilePath,
		}
	}
	return result, nil
}

// Search searches the library for tracks matching the query.
func (s *LibraryService) Search(query string, limit int) ([]library.SearchResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	return s.db.Search(query, limit)
}

// Playlist is the JSON-friendly playlist type exposed to the frontend.
type Playlist struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GetPlaylists returns all playlists.
func (s *LibraryService) GetPlaylists() ([]Playlist, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	playlists, err := s.db.GetPlaylists()
	if err != nil {
		return nil, err
	}
	result := make([]Playlist, len(playlists))
	for i, p := range playlists {
		result[i] = Playlist{ID: p.ID, Name: p.Name}
	}
	return result, nil
}

// CreatePlaylist creates a new playlist and returns its ID.
func (s *LibraryService) CreatePlaylist(name string) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("library not initialised")
	}
	return s.db.CreatePlaylist(name)
}

// RenamePlaylist renames a playlist.
func (s *LibraryService) RenamePlaylist(id int64, name string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.RenamePlaylist(id, name)
}

// DeletePlaylist deletes a playlist.
func (s *LibraryService) DeletePlaylist(id int64) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.DeletePlaylist(id)
}

// PlaylistTrack is the JSON-friendly playlist track type exposed to the frontend.
type PlaylistTrack struct {
	TrackID    int64  `json:"trackId"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	DurationMs int    `json:"durationMs"`
	FilePath   string `json:"filePath"`
	Position   int    `json:"position"`
}

// GetPlaylistTracks returns the tracks in a playlist.
func (s *LibraryService) GetPlaylistTracks(playlistID int64) ([]PlaylistTrack, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	tracks, err := s.db.GetPlaylistTracks(playlistID)
	if err != nil {
		return nil, err
	}
	result := make([]PlaylistTrack, len(tracks))
	for i, t := range tracks {
		result[i] = PlaylistTrack{
			TrackID:    t.TrackID,
			Title:      t.Title,
			Artist:     t.Artist,
			Album:      t.Album,
			DurationMs: t.DurationMs,
			FilePath:   t.FilePath,
			Position:   t.Position,
		}
	}
	return result, nil
}

// AddTrackToPlaylist adds a track to a playlist.
func (s *LibraryService) AddTrackToPlaylist(playlistID, trackID int64) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.AddTrackToPlaylist(playlistID, trackID)
}

// RemoveTrackFromPlaylist removes a track from a playlist.
func (s *LibraryService) RemoveTrackFromPlaylist(playlistID, trackID int64) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.RemoveTrackFromPlaylist(playlistID, trackID)
}

// MoveTrackInPlaylist moves a track from one position to another.
func (s *LibraryService) MoveTrackInPlaylist(playlistID int64, fromPos, toPos int) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.MoveTrackInPlaylist(playlistID, fromPos, toPos)
}
