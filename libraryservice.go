package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/streaming/jellyfin"
	"github.com/willfish/forte/internal/streaming/subsonic"
)

// LibraryService exposes the music library to the frontend.
type LibraryService struct {
	db       *library.DB
	stopSync chan struct{}
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

	// Start background server sync: immediate + every 15 minutes.
	s.stopSync = make(chan struct{})
	go func() {
		// Initial sync on startup.
		if err := library.SyncAllServers(context.Background(), s.db); err != nil {
			log.Printf("server sync: %v", err)
		}

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := library.SyncAllServers(context.Background(), s.db); err != nil {
					log.Printf("server sync: %v", err)
				}
			case <-s.stopSync:
				return
			}
		}
	}()

	return nil
}

// ServiceShutdown closes the library database when the application exits.
func (s *LibraryService) ServiceShutdown() error {
	if s.stopSync != nil {
		close(s.stopSync)
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// sourceFromServerID returns "local" or "server" based on the server_id value.
func sourceFromServerID(serverID string) string {
	if serverID == "" {
		return "local"
	}
	return "server"
}

// Album is the JSON-friendly album type exposed to the frontend.
type Album struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Year       int    `json:"year"`
	TrackCount int    `json:"trackCount"`
	Source     string `json:"source"`
}

// GetAlbums returns all albums sorted by the given field and direction.
// Source: "" (all, deduped), "local", "server".
func (s *LibraryService) GetAlbums(sort string, order string, source string) ([]Album, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}

	albums, err := s.db.GetAlbums(sort, order, source)
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
			Source:     sourceFromServerID(a.ServerID),
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
	Source      string `json:"source"`
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
			Source:      sourceFromServerID(t.ServerID),
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

// SyncServers triggers an immediate sync of all server catalogs.
func (s *LibraryService) SyncServers() error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return library.SyncAllServers(context.Background(), s.db)
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

// ServerConfig is the JSON-friendly server configuration type exposed to the frontend.
type ServerConfig struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// GetServers returns all configured streaming servers.
func (s *LibraryService) GetServers() ([]ServerConfig, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	servers, err := s.db.GetServers()
	if err != nil {
		return nil, err
	}
	result := make([]ServerConfig, len(servers))
	for i, srv := range servers {
		result[i] = ServerConfig{
			ID:       srv.ID,
			Name:     srv.Name,
			Type:     srv.Type,
			URL:      srv.URL,
			Username: srv.Username,
			Password: srv.Password,
		}
	}
	return result, nil
}

// AddServer adds a new streaming server configuration.
func (s *LibraryService) AddServer(cfg ServerConfig) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	id, err := newUUID()
	if err != nil {
		return fmt.Errorf("generate id: %w", err)
	}
	return s.db.AddServer(library.Server{
		ID:       id,
		Name:     cfg.Name,
		Type:     cfg.Type,
		URL:      cfg.URL,
		Username: cfg.Username,
		Password: cfg.Password,
	})
}

// UpdateServer updates an existing streaming server configuration.
func (s *LibraryService) UpdateServer(cfg ServerConfig) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.UpdateServer(library.Server{
		ID:       cfg.ID,
		Name:     cfg.Name,
		Type:     cfg.Type,
		URL:      cfg.URL,
		Username: cfg.Username,
		Password: cfg.Password,
	})
}

// DeleteServer removes a streaming server configuration.
func (s *LibraryService) DeleteServer(id string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.DeleteServer(id)
}

// TestConnection tests connectivity to a streaming server without persisting it.
func (s *LibraryService) TestConnection(cfg ServerConfig) error {
	switch cfg.Type {
	case "subsonic":
		return subsonic.New(cfg.URL, cfg.Username, cfg.Password).Ping()
	case "jellyfin":
		return jellyfin.New(cfg.URL, cfg.Username, cfg.Password).Ping()
	default:
		return fmt.Errorf("unknown server type: %s", cfg.Type)
	}
}

// newUUID generates a random UUID v4 string.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
