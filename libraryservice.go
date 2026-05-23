package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/browser"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/artistinfo"
	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/radio"
	"github.com/willfish/forte/internal/scrobbling/lastfm"
	"github.com/willfish/forte/internal/scrobbling/listenbrainz"
	"github.com/willfish/forte/internal/streaming/jellyfin"
	"github.com/willfish/forte/internal/streaming/subsonic"
)

// LibraryService exposes the music library to the frontend.
type LibraryService struct {
	db            *library.DB
	lifecycleMu   sync.Mutex
	health        *library.HealthMonitor
	syncCancel    context.CancelFunc
	syncDone      chan struct{}
	onThemeChange func(theme string)
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

	prefs, err := db.GetAppPreferences()
	if err != nil {
		return fmt.Errorf("load preferences: %w", err)
	}

	s.lifecycleMu.Lock()
	if prefs.LibraryEnabled {
		s.startLibraryRuntimeLocked()
	}
	s.lifecycleMu.Unlock()
	return nil
}

// ServiceShutdown closes the library database when the application exits.
func (s *LibraryService) ServiceShutdown() error {
	s.lifecycleMu.Lock()
	s.stopLibraryRuntimeLocked()
	s.lifecycleMu.Unlock()

	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *LibraryService) startLibraryRuntimeLocked() {
	if s.db == nil {
		return
	}
	if s.health == nil {
		s.health = library.NewHealthMonitor(s.db)
	}
	s.health.Start()
	if s.syncCancel != nil {
		return
	}

	syncCtx, cancelSync := context.WithCancel(context.Background())
	s.syncCancel = cancelSync
	s.syncDone = make(chan struct{})
	go func(done chan<- struct{}) {
		defer close(done)

		if err := library.SyncAllServers(syncCtx, s.db); err != nil && syncCtx.Err() == nil {
			log.Printf("server sync: %v", err)
		}

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := library.SyncAllServers(syncCtx, s.db); err != nil && syncCtx.Err() == nil {
					log.Printf("server sync: %v", err)
				}
			case <-syncCtx.Done():
				return
			}
		}
	}(s.syncDone)
}

func (s *LibraryService) stopLibraryRuntimeLocked() {
	if s.syncCancel != nil {
		s.syncCancel()
	}
	if s.syncDone != nil {
		<-s.syncDone
	}
	s.syncCancel = nil
	s.syncDone = nil
	if s.health != nil {
		s.health.Stop()
	}
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
	ServerID   string `json:"serverId"`
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
			ServerID:   a.ServerID,
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
	ServerID    string `json:"serverId"`
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
			ServerID:    t.ServerID,
		}
	}
	return result, nil
}

// SearchResult is the JSON-friendly search result type exposed to the frontend.
type SearchResult struct {
	TrackID     int64  `json:"trackId"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	Genre       string `json:"genre"`
	TrackNumber int    `json:"trackNumber"`
	DiscNumber  int    `json:"discNumber"`
	DurationMs  int    `json:"durationMs"`
	FilePath    string `json:"filePath"`
	Source      string `json:"source"`
	ServerID    string `json:"serverId"`
}

// Search searches the library for tracks matching the query.
func (s *LibraryService) Search(query string, limit int) ([]SearchResult, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	rows, err := s.db.Search(query, limit)
	if err != nil {
		return nil, err
	}
	results := make([]SearchResult, len(rows))
	for i, r := range rows {
		results[i] = SearchResult{
			TrackID:     r.TrackID,
			Title:       r.Title,
			Artist:      r.Artist,
			Album:       r.Album,
			Genre:       r.Genre,
			TrackNumber: r.TrackNumber,
			DiscNumber:  r.DiscNumber,
			DurationMs:  r.DurationMs,
			FilePath:    r.FilePath,
			Source:      sourceFromServerID(r.ServerID),
			ServerID:    r.ServerID,
		}
	}
	return results, nil
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
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	Username    string `json:"username"`
	Password    string `json:"password,omitempty"`
	HasPassword bool   `json:"hasPassword"`
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
			ID:          srv.ID,
			Name:        srv.Name,
			Type:        srv.Type,
			URL:         srv.URL,
			Username:    srv.Username,
			HasPassword: srv.Password != "",
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
	password := cfg.Password
	if password == "" {
		existing, err := s.db.GetServer(cfg.ID)
		if err != nil {
			return err
		}
		password = existing.Password
	}
	return s.db.UpdateServer(library.Server{
		ID:       cfg.ID,
		Name:     cfg.Name,
		Type:     cfg.Type,
		URL:      cfg.URL,
		Username: cfg.Username,
		Password: password,
	})
}

// DeleteServer removes a streaming server configuration.
func (s *LibraryService) DeleteServer(id string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.DeleteServer(id)
}

// ServerStatusJSON is the JSON-friendly server status type exposed to the frontend.
type ServerStatusJSON struct {
	ServerID string `json:"serverId"`
	Name     string `json:"name"`
	Online   bool   `json:"online"`
}

// GetServerStatuses returns the online/offline status of all configured servers.
func (s *LibraryService) GetServerStatuses() ([]ServerStatusJSON, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	servers, err := s.db.GetServers()
	if err != nil {
		return nil, err
	}

	// Build a name lookup.
	nameMap := make(map[string]string, len(servers))
	for _, srv := range servers {
		nameMap[srv.ID] = srv.Name
	}

	var result []ServerStatusJSON
	if s.health != nil {
		for _, st := range s.health.Statuses() {
			result = append(result, ServerStatusJSON{
				ServerID: st.ServerID,
				Name:     nameMap[st.ServerID],
				Online:   st.Online,
			})
		}
	}

	// Include servers not yet pinged (assumed online).
	seen := make(map[string]bool, len(result))
	for _, r := range result {
		seen[r.ServerID] = true
	}
	for _, srv := range servers {
		if !seen[srv.ID] {
			result = append(result, ServerStatusJSON{
				ServerID: srv.ID,
				Name:     srv.Name,
				Online:   true,
			})
		}
	}
	return result, nil
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

// ScrobbleConfigJSON is the JSON-friendly scrobble config exposed to the frontend.
// APISecret is intentionally omitted.
type ScrobbleConfigJSON struct {
	APIKey     string `json:"apiKey"`
	SessionKey string `json:"sessionKey"`
	Username   string `json:"username"`
	Enabled    bool   `json:"enabled"`
}

// GetScrobbleConfig returns the current Last.fm scrobble configuration.
func (s *LibraryService) GetScrobbleConfig() (ScrobbleConfigJSON, error) {
	if s.db == nil {
		return ScrobbleConfigJSON{}, fmt.Errorf("library not initialised")
	}
	cfg, err := s.db.LoadScrobbleConfig()
	if err != nil {
		return ScrobbleConfigJSON{}, err
	}
	return ScrobbleConfigJSON{
		APIKey:     cfg.APIKey,
		SessionKey: cfg.SessionKey,
		Username:   cfg.Username,
		Enabled:    cfg.Enabled,
	}, nil
}

// SaveScrobbleAPIKeys saves the Last.fm API key and secret.
func (s *LibraryService) SaveScrobbleAPIKeys(apiKey, apiSecret string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	cfg, err := s.db.LoadScrobbleConfig()
	if err != nil {
		return err
	}
	cfg.APIKey = apiKey
	cfg.APISecret = apiSecret
	return s.db.SaveScrobbleConfig(cfg)
}

// StartLastFmAuth begins the Last.fm auth flow: requests a token and opens
// the browser for user approval. Returns the token for use with CompleteLastFmAuth.
func (s *LibraryService) StartLastFmAuth() (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("library not initialised")
	}
	cfg, err := s.db.LoadScrobbleConfig()
	if err != nil {
		return "", err
	}
	if cfg.APIKey == "" || cfg.APISecret == "" {
		return "", fmt.Errorf("API key and secret must be configured first")
	}
	token, err := lastfm.GetToken(cfg.APIKey, cfg.APISecret)
	if err != nil {
		return "", err
	}
	authURL := lastfm.AuthURL(cfg.APIKey, token)
	if err := browser.OpenURL(authURL); err != nil {
		log.Printf("lastfm: failed to open browser: %v", err)
	}
	return token, nil
}

// CompleteLastFmAuth exchanges the authorized token for a session key.
func (s *LibraryService) CompleteLastFmAuth(token string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	cfg, err := s.db.LoadScrobbleConfig()
	if err != nil {
		return err
	}
	sessionKey, username, err := lastfm.GetSession(cfg.APIKey, cfg.APISecret, token)
	if err != nil {
		return err
	}
	cfg.SessionKey = sessionKey
	cfg.Username = username
	cfg.Enabled = true
	return s.db.SaveScrobbleConfig(cfg)
}

// DisconnectLastFm clears the Last.fm session key and username.
func (s *LibraryService) DisconnectLastFm() error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	cfg, err := s.db.LoadScrobbleConfig()
	if err != nil {
		return err
	}
	cfg.SessionKey = ""
	cfg.Username = ""
	cfg.Enabled = false
	return s.db.SaveScrobbleConfig(cfg)
}

// SetScrobbleEnabled toggles scrobbling on or off.
func (s *LibraryService) SetScrobbleEnabled(enabled bool) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	cfg, err := s.db.LoadScrobbleConfig()
	if err != nil {
		return err
	}
	cfg.Enabled = enabled
	return s.db.SaveScrobbleConfig(cfg)
}

// ListenBrainzConfigJSON is the JSON-friendly ListenBrainz config exposed to the frontend.
// UserToken is intentionally omitted.
type ListenBrainzConfigJSON struct {
	Username string `json:"username"`
	Enabled  bool   `json:"enabled"`
}

// GetListenBrainzConfig returns the current ListenBrainz configuration.
func (s *LibraryService) GetListenBrainzConfig() (ListenBrainzConfigJSON, error) {
	if s.db == nil {
		return ListenBrainzConfigJSON{}, fmt.Errorf("library not initialised")
	}
	cfg, err := s.db.LoadListenBrainzConfig()
	if err != nil {
		return ListenBrainzConfigJSON{}, err
	}
	return ListenBrainzConfigJSON{
		Username: cfg.Username,
		Enabled:  cfg.Enabled,
	}, nil
}

// ConnectListenBrainz validates the user token, retrieves the username, and saves
// the configuration with scrobbling enabled.
func (s *LibraryService) ConnectListenBrainz(userToken string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	username, err := listenbrainz.ValidateToken(userToken)
	if err != nil {
		return err
	}
	return s.db.SaveListenBrainzConfig(library.ListenBrainzConfig{
		UserToken: userToken,
		Username:  username,
		Enabled:   true,
	})
}

// DisconnectListenBrainz clears the ListenBrainz token and username.
func (s *LibraryService) DisconnectListenBrainz() error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.SaveListenBrainzConfig(library.ListenBrainzConfig{})
}

// SetListenBrainzEnabled toggles ListenBrainz scrobbling on or off.
func (s *LibraryService) SetListenBrainzEnabled(enabled bool) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	cfg, err := s.db.LoadListenBrainzConfig()
	if err != nil {
		return err
	}
	cfg.Enabled = enabled
	return s.db.SaveListenBrainzConfig(cfg)
}

// GetScrobbleQueueSize returns the number of scrobbles waiting for retry.
func (s *LibraryService) GetScrobbleQueueSize() (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("library not initialised")
	}
	return s.db.ScrobbleQueueSize()
}

// StatEntryJSON is the JSON-friendly stat entry type exposed to the frontend.
type StatEntryJSON struct {
	Name       string `json:"name"`
	SecondLine string `json:"secondLine"`
	PlayCount  int    `json:"playCount"`
	TotalMs    int64  `json:"totalMs"`
}

// RecentPlayJSON is the JSON-friendly recent play type exposed to the frontend.
type RecentPlayJSON struct {
	TrackID    int64  `json:"trackId"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	DurationMs int    `json:"durationMs"`
	PlayedAt   string `json:"playedAt"`
}

// GetTopArtists returns the most-played artists for the given period.
func (s *LibraryService) GetTopArtists(period string, limit int) ([]StatEntryJSON, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	entries, err := s.db.TopArtists(period, limit)
	if err != nil {
		return nil, err
	}
	return toStatJSON(entries), nil
}

// GetTopAlbums returns the most-played albums for the given period.
func (s *LibraryService) GetTopAlbums(period string, limit int) ([]StatEntryJSON, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	entries, err := s.db.TopAlbums(period, limit)
	if err != nil {
		return nil, err
	}
	return toStatJSON(entries), nil
}

// GetTopTracks returns the most-played tracks for the given period.
func (s *LibraryService) GetTopTracks(period string, limit int) ([]StatEntryJSON, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	entries, err := s.db.TopTracks(period, limit)
	if err != nil {
		return nil, err
	}
	return toStatJSON(entries), nil
}

// GetRecentlyPlayed returns the most recently played tracks.
func (s *LibraryService) GetRecentlyPlayed(limit int) ([]RecentPlayJSON, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	plays, err := s.db.RecentlyPlayed(limit)
	if err != nil {
		return nil, err
	}
	result := make([]RecentPlayJSON, len(plays))
	for i, p := range plays {
		result[i] = RecentPlayJSON{
			TrackID:    p.TrackID,
			Title:      p.Title,
			Artist:     p.Artist,
			Album:      p.Album,
			DurationMs: p.DurationMs,
			PlayedAt:   p.PlayedAt,
		}
	}
	return result, nil
}

func toStatJSON(entries []library.StatEntry) []StatEntryJSON {
	result := make([]StatEntryJSON, len(entries))
	for i, e := range entries {
		result[i] = StatEntryJSON{
			Name:       e.Name,
			SecondLine: e.SecondLine,
			PlayCount:  e.PlayCount,
			TotalMs:    e.TotalMs,
		}
	}
	return result
}

// SimilarArtistJSON is the JSON-friendly similar artist type exposed to the frontend.
type SimilarArtistJSON struct {
	Name      string `json:"name"`
	InLibrary bool   `json:"inLibrary"`
}

// ArtistInfoJSON is the JSON-friendly artist info type exposed to the frontend.
type ArtistInfoJSON struct {
	Name        string              `json:"name"`
	Bio         string              `json:"bio"`
	ImageURL    string              `json:"imageUrl"`
	Area        string              `json:"area"`
	Type        string              `json:"type"`
	ActiveYears string              `json:"activeYears"`
	Similar     []SimilarArtistJSON `json:"similar"`
	Albums      []Album             `json:"albums"`
	Tags        string              `json:"tags"`
}

// GetArtistInfo returns metadata for the named artist, using cache with 30-day TTL.
func (s *LibraryService) GetArtistInfo(artistName string) (ArtistInfoJSON, error) {
	if s.db == nil {
		return ArtistInfoJSON{}, fmt.Errorf("library not initialised")
	}

	artistID, err := s.db.GetArtistByName(artistName)
	if err != nil {
		return ArtistInfoJSON{}, fmt.Errorf("artist not found: %w", err)
	}

	// Check cache.
	cached, err := s.db.GetArtistMeta(artistID)
	if err != nil {
		return ArtistInfoJSON{}, err
	}

	var meta library.ArtistMeta
	if cached != nil {
		meta = *cached
	} else {
		// Fetch from external services.
		apiKey := ""
		cfg, err := s.db.LoadScrobbleConfig()
		if err == nil {
			apiKey = cfg.APIKey
		}

		result, err := artistinfo.Fetch(apiKey, artistName)
		if err != nil {
			return ArtistInfoJSON{}, err
		}

		meta = library.ArtistMeta{
			Bio:      result.Bio,
			ImageURL: result.ImageURL,
			MbID:     result.MbID,
			MbArea:   result.MbArea,
			MbType:   result.MbType,
			MbBegin:  result.MbBegin,
			MbEnd:    result.MbEnd,
			MbTags:   result.MbTags,
		}
		for _, sim := range result.Similar {
			meta.Similar = append(meta.Similar, library.SimilarArtist{Name: sim.Name})
		}

		_ = s.db.SaveArtistMeta(artistID, meta)
	}

	// Build albums list.
	dbAlbums, err := s.db.GetArtistAlbums(artistID)
	if err != nil {
		return ArtistInfoJSON{}, err
	}
	albums := make([]Album, len(dbAlbums))
	for i, a := range dbAlbums {
		albums[i] = Album{
			ID:         a.ID,
			Title:      a.Title,
			Artist:     a.Artist,
			Year:       a.Year,
			TrackCount: a.TrackCount,
			Source:     sourceFromServerID(a.ServerID),
			ServerID:   a.ServerID,
		}
	}

	// Check which similar artists are in the library.
	similar := make([]SimilarArtistJSON, len(meta.Similar))
	for i, sim := range meta.Similar {
		_, lookupErr := s.db.GetArtistByName(sim.Name)
		similar[i] = SimilarArtistJSON{
			Name:      sim.Name,
			InLibrary: lookupErr == nil,
		}
	}

	activeYears := meta.MbBegin
	if meta.MbEnd != "" {
		activeYears += " - " + meta.MbEnd
	} else if meta.MbBegin != "" {
		activeYears += " - present"
	}

	return ArtistInfoJSON{
		Name:        artistName,
		Bio:         meta.Bio,
		ImageURL:    meta.ImageURL,
		Area:        meta.MbArea,
		Type:        meta.MbType,
		ActiveYears: activeYears,
		Similar:     similar,
		Albums:      albums,
		Tags:        meta.MbTags,
	}, nil
}

// GetArtistByName returns the artist ID for the given name.
func (s *LibraryService) GetArtistByName(name string) (int64, error) {
	if s.db == nil {
		return 0, fmt.Errorf("library not initialised")
	}
	return s.db.GetArtistByName(name)
}

// RadioStationJSON is the JSON-friendly radio station type exposed to the frontend.
type RadioStationJSON struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	StreamURL string `json:"streamUrl"`
	Favicon   string `json:"favicon"`
	Country   string `json:"country"`
	Tags      string `json:"tags"`
	Bitrate   int    `json:"bitrate"`
	Codec     string `json:"codec"`
	Votes     int    `json:"votes"`
	Clicks    int    `json:"clicks"`
}

var somafmClient = radio.NewSomaFMClient()

func stationsToJSON(stations []radio.Station) []RadioStationJSON {
	result := make([]RadioStationJSON, len(stations))
	favicons := resolveStationFavicons(stations)
	for i, s := range stations {
		result[i] = RadioStationJSON{
			UUID:      s.UUID,
			Name:      s.Name,
			StreamURL: normalizeRadioStreamURL(s.StreamURL),
			Favicon:   favicons[i],
			Country:   s.Country,
			Tags:      s.Tags,
			Bitrate:   s.Bitrate,
			Codec:     s.Codec,
			Votes:     s.Votes,
			Clicks:    s.Clicks,
		}
	}
	return result
}

func resolveStationFavicons(stations []radio.Station) []string {
	favicons := make([]string, len(stations))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, station := range stations {
		i, station := i, station
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			favicons[i] = resolveRadioArtwork(station.Favicon, station.Homepage)
		}()
	}
	wg.Wait()
	return favicons
}

func normalizeRadioStreamURL(streamURL string) string {
	u, err := url.Parse(streamURL)
	if err != nil {
		return streamURL
	}
	switch strings.ToLower(u.Hostname()) {
	case "timesradio.wireless.radio":
		return "https://times.live.stream.broadcasting.news/stream"
	default:
		return streamURL
	}
}

var radioClient = radio.NewClient()

// SearchRadioStations searches for radio stations by name.
func (s *LibraryService) SearchRadioStations(query string, limit int) ([]RadioStationJSON, error) {
	stations, err := radioClient.Search(query, limit)
	if err != nil {
		return nil, err
	}
	return stationsToJSON(stations), nil
}

// SearchRadioStationsFiltered searches with optional country, codec, and tag filters.
func (s *LibraryService) SearchRadioStationsFiltered(country, codec, tag string, limit int) ([]RadioStationJSON, error) {
	stations, err := radioClient.SearchFiltered(country, codec, tag, limit)
	if err != nil {
		return nil, err
	}
	return stationsToJSON(stations), nil
}

// GetRadioStationsByTag returns radio stations matching a tag.
func (s *LibraryService) GetRadioStationsByTag(tag string, limit int) ([]RadioStationJSON, error) {
	stations, err := radioClient.ByTag(tag, limit)
	if err != nil {
		return nil, err
	}
	return stationsToJSON(stations), nil
}

// GetRadioStationsByCountry returns radio stations matching a country.
func (s *LibraryService) GetRadioStationsByCountry(country string, limit int) ([]RadioStationJSON, error) {
	stations, err := radioClient.ByCountry(country, limit)
	if err != nil {
		return nil, err
	}
	return stationsToJSON(stations), nil
}

// GetTopVotedRadioStations returns the top voted radio stations.
func (s *LibraryService) GetTopVotedRadioStations(limit int) ([]RadioStationJSON, error) {
	stations, err := radioClient.TopVoted(limit)
	if err != nil {
		return nil, err
	}
	return stationsToJSON(stations), nil
}

// GetTopClickedRadioStations returns the most clicked radio stations.
func (s *LibraryService) GetTopClickedRadioStations(limit int) ([]RadioStationJSON, error) {
	stations, err := radioClient.TopClicked(limit)
	if err != nil {
		return nil, err
	}
	return stationsToJSON(stations), nil
}

// GetSomaFMStations returns all SomaFM channels.
func (s *LibraryService) GetSomaFMStations() ([]RadioStationJSON, error) {
	stations, err := somafmClient.Stations()
	if err != nil {
		return nil, err
	}
	return stationsToJSON(stations), nil
}

// RadioFavouriteJSON is the JSON-friendly radio favourite type exposed to the frontend.
type RadioFavouriteJSON struct {
	StationUUID string `json:"stationUuid"`
	Name        string `json:"name"`
	StreamURL   string `json:"streamUrl"`
	FaviconURL  string `json:"faviconUrl"`
	Tags        string `json:"tags"`
	AddedAt     string `json:"addedAt"`
	Pinned      bool   `json:"pinned"`
}

// RadioHistoryJSON is the JSON-friendly radio history type exposed to the frontend.
type RadioHistoryJSON struct {
	StationUUID  string `json:"stationUuid"`
	Name         string `json:"name"`
	StreamURL    string `json:"streamUrl"`
	FaviconURL   string `json:"faviconUrl"`
	Tags         string `json:"tags"`
	TrackTitle   string `json:"trackTitle"`
	PlayCount    int    `json:"playCount"`
	LastError    string `json:"lastError"`
	LastPlayedAt string `json:"lastPlayedAt"`
}

// RadioCustomStationJSON is the JSON-friendly custom station type exposed to the frontend.
type RadioCustomStationJSON struct {
	StationUUID string `json:"stationUuid"`
	Name        string `json:"name"`
	StreamURL   string `json:"streamUrl"`
	FaviconURL  string `json:"faviconUrl"`
	Tags        string `json:"tags"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// AppPreferencesJSON is the JSON-friendly app preference type exposed to the frontend.
type AppPreferencesJSON struct {
	LibraryEnabled   bool `json:"libraryEnabled"`
	StartLastStation bool `json:"startLastStation"`
	AutoReconnect    bool `json:"autoReconnect"`
	ShowTitlebar     bool `json:"showTitlebar"`
}

// GetRadioFavourites returns all saved radio stations.
func (s *LibraryService) GetRadioFavourites() ([]RadioFavouriteJSON, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	favs, err := s.db.GetRadioFavourites()
	if err != nil {
		return nil, err
	}
	result := make([]RadioFavouriteJSON, len(favs))
	for i, f := range favs {
		result[i] = RadioFavouriteJSON{
			StationUUID: f.StationUUID,
			Name:        f.Name,
			StreamURL:   f.StreamURL,
			FaviconURL:  f.FaviconURL,
			Tags:        f.Tags,
			AddedAt:     f.AddedAt,
			Pinned:      f.Pinned,
		}
	}
	return result, nil
}

// AddRadioFavourite saves a radio station to favourites.
func (s *LibraryService) AddRadioFavourite(stationUUID, name, streamURL, faviconURL, tags string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.AddRadioFavourite(library.RadioFavourite{
		StationUUID: stationUUID,
		Name:        name,
		StreamURL:   streamURL,
		FaviconURL:  faviconURL,
		Tags:        tags,
	})
}

// RemoveRadioFavourite removes a radio station from favourites.
func (s *LibraryService) RemoveRadioFavourite(stationUUID string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.RemoveRadioFavourite(stationUUID)
}

// SetRadioFavouritePinned pins or unpins a favourite station.
func (s *LibraryService) SetRadioFavouritePinned(stationUUID string, pinned bool) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.SetRadioFavouritePinned(stationUUID, pinned)
}

// IsRadioFavourite checks if a station is in favourites.
func (s *LibraryService) IsRadioFavourite(stationUUID string) (bool, error) {
	if s.db == nil {
		return false, fmt.Errorf("library not initialised")
	}
	return s.db.IsRadioFavourite(stationUUID)
}

// AddCustomRadioStation saves a user-defined radio station.
func (s *LibraryService) AddCustomRadioStation(name, streamURL, faviconURL, tags string) (RadioCustomStationJSON, error) {
	if s.db == nil {
		return RadioCustomStationJSON{}, fmt.Errorf("library not initialised")
	}
	if strings.TrimSpace(name) == "" {
		return RadioCustomStationJSON{}, fmt.Errorf("station name is required")
	}
	if err := validateStreamURL(streamURL); err != nil {
		return RadioCustomStationJSON{}, err
	}
	station, err := s.db.AddCustomRadioStation(library.RadioCustomStation{
		Name:       strings.TrimSpace(name),
		StreamURL:  strings.TrimSpace(streamURL),
		FaviconURL: strings.TrimSpace(faviconURL),
		Tags:       strings.TrimSpace(tags),
	})
	if err != nil {
		return RadioCustomStationJSON{}, err
	}
	return customStationToJSON(station), nil
}

// DeleteCustomRadioStation removes a user-defined radio station.
func (s *LibraryService) DeleteCustomRadioStation(stationUUID string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.DeleteCustomRadioStation(stationUUID)
}

// GetCustomRadioStations returns user-defined radio stations.
func (s *LibraryService) GetCustomRadioStations() ([]RadioCustomStationJSON, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	stations, err := s.db.GetCustomRadioStations()
	if err != nil {
		return nil, err
	}
	result := make([]RadioCustomStationJSON, len(stations))
	for i, station := range stations {
		result[i] = customStationToJSON(station)
	}
	return result, nil
}

// GetRadioHistory returns recently played radio stations.
func (s *LibraryService) GetRadioHistory(limit int) ([]RadioHistoryJSON, error) {
	if s.db == nil {
		return nil, fmt.Errorf("library not initialised")
	}
	entries, err := s.db.GetRadioHistory(limit)
	if err != nil {
		return nil, err
	}
	result := make([]RadioHistoryJSON, len(entries))
	for i, entry := range entries {
		result[i] = RadioHistoryJSON{
			StationUUID:  entry.StationUUID,
			Name:         entry.Name,
			StreamURL:    entry.StreamURL,
			FaviconURL:   entry.FaviconURL,
			Tags:         entry.Tags,
			TrackTitle:   entry.TrackTitle,
			PlayCount:    entry.PlayCount,
			LastError:    entry.LastError,
			LastPlayedAt: entry.LastPlayedAt,
		}
	}
	return result, nil
}

// ClearRadioHistory removes all radio playback history.
func (s *LibraryService) ClearRadioHistory() error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	return s.db.ClearRadioHistory()
}

// GetAppPreferences returns product-level preferences.
func (s *LibraryService) GetAppPreferences() (AppPreferencesJSON, error) {
	if s.db == nil {
		return AppPreferencesJSON{}, fmt.Errorf("library not initialised")
	}
	prefs, err := s.db.GetAppPreferences()
	if err != nil {
		return AppPreferencesJSON{}, err
	}
	return AppPreferencesJSON{
		LibraryEnabled:   prefs.LibraryEnabled,
		StartLastStation: prefs.StartLastStation,
		AutoReconnect:    prefs.AutoReconnect,
		ShowTitlebar:     prefs.ShowTitlebar,
	}, nil
}

// SaveAppPreferences stores product-level preferences.
func (s *LibraryService) SaveAppPreferences(prefs AppPreferencesJSON) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}

	current, err := s.db.GetAppPreferences()
	if err != nil {
		return err
	}

	next := library.AppPreferences{
		LibraryEnabled:   prefs.LibraryEnabled,
		StartLastStation: prefs.StartLastStation,
		AutoReconnect:    prefs.AutoReconnect,
		ShowTitlebar:     prefs.ShowTitlebar,
	}
	if err := s.db.SaveAppPreferences(next); err != nil {
		return err
	}

	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if !current.LibraryEnabled && next.LibraryEnabled {
		s.startLibraryRuntimeLocked()
	} else if current.LibraryEnabled && !next.LibraryEnabled {
		s.stopLibraryRuntimeLocked()
	} else if next.LibraryEnabled {
		s.startLibraryRuntimeLocked()
	}
	return nil
}

// SetThemePreference applies the current frontend theme to native desktop chrome.
func (s *LibraryService) SetThemePreference(theme string) error {
	switch theme {
	case "green-dark", "green-light", "blue-dark", "blue-light", "financial-times-dark", "financial-times-light":
	default:
		return fmt.Errorf("unknown theme preference %q", theme)
	}

	if s.onThemeChange != nil {
		s.onThemeChange(theme)
	}
	return nil
}

func customStationToJSON(station library.RadioCustomStation) RadioCustomStationJSON {
	return RadioCustomStationJSON{
		StationUUID: station.StationUUID,
		Name:        station.Name,
		StreamURL:   station.StreamURL,
		FaviconURL:  station.FaviconURL,
		Tags:        station.Tags,
		CreatedAt:   station.CreatedAt,
		UpdatedAt:   station.UpdatedAt,
	}
}

func validateStreamURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid stream URL")
	}
	switch u.Scheme {
	case "http", "https":
		return nil
	default:
		return fmt.Errorf("stream URL must use http or https")
	}
}

// imageProxyCache caches proxied image data URIs keyed by URL.
// inflight tracks URLs currently being fetched; waiters block on the channel.
var imageProxyCache struct {
	sync.Mutex
	m        map[string]string
	inflight map[string]chan struct{}
}

var imageProxyClient = &http.Client{Timeout: 5 * time.Second}

// ProxyImageURL fetches a remote image and returns it as a base64 data URI.
// Results are cached in memory. Concurrent requests for the same URL are
// coalesced into a single HTTP fetch. Returns empty string on failure.
func (s *LibraryService) ProxyImageURL(url string) string {
	if url == "" || (!strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")) {
		return ""
	}

	imageProxyCache.Lock()
	if imageProxyCache.m == nil {
		imageProxyCache.m = make(map[string]string)
		imageProxyCache.inflight = make(map[string]chan struct{})
	}

	// Return cached result immediately.
	if cached, ok := imageProxyCache.m[url]; ok {
		imageProxyCache.Unlock()
		return cached
	}

	// If another goroutine is already fetching this URL, wait for it.
	if ch, ok := imageProxyCache.inflight[url]; ok {
		imageProxyCache.Unlock()
		<-ch
		imageProxyCache.Lock()
		cached := imageProxyCache.m[url]
		imageProxyCache.Unlock()
		return cached
	}

	// Mark this URL as in-flight.
	ch := make(chan struct{})
	imageProxyCache.inflight[url] = ch
	imageProxyCache.Unlock()

	// Fetch the image. Always clean up the in-flight entry and signal waiters.
	dataURI := fetchImage(url)

	imageProxyCache.Lock()
	if dataURI != "" {
		imageProxyCache.m[url] = dataURI
	}
	delete(imageProxyCache.inflight, url)
	imageProxyCache.Unlock()
	close(ch)

	return dataURI
}

func fetchImage(url string) string {
	resp, err := imageProxyClient.Get(url)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024)) // 2MB max
	if err != nil || len(data) == 0 {
		return ""
	}

	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = http.DetectContentType(data)
	}

	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
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
