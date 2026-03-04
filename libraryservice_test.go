package main

import (
	"path/filepath"
	"testing"

	"github.com/willfish/forte/internal/library"
)

// openTestService creates a LibraryService backed by a temp SQLite database.
func openTestService(t *testing.T) *LibraryService {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := library.OpenDB(path)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &LibraryService{db: db}
}

// seedTrack inserts an artist, album, and track into the test database.
// Returns the track ID (always 1 for the first call).
func seedTrack(t *testing.T, s *LibraryService) {
	t.Helper()
	mustExecService(t, s, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExecService(t, s, "INSERT INTO albums (id, artist_id, title, year) VALUES (1, 1, 'OK Computer', 1997)")
	mustExecService(t, s, `INSERT INTO tracks (id, album_id, artist_id, title, track_number, disc_number, duration_ms, file_path)
		VALUES (1, 1, 1, 'Airbag', 1, 1, 282000, '/music/airbag.flac')`)
	mustExecService(t, s, `INSERT INTO fts_tracks (rowid, title, artist, album, genre)
		VALUES (1, 'Airbag', 'Radiohead', 'OK Computer', 'Rock')`)
}

func mustExecService(t *testing.T, s *LibraryService, query string, args ...any) {
	t.Helper()
	if _, err := s.db.Exec(query, args...); err != nil {
		t.Fatalf("exec %q: %v", query, err)
	}
}

// --- nil db error tests ---

func TestNilDB(t *testing.T) {
	s := &LibraryService{}

	if _, err := s.GetAlbums("title", "asc", ""); err == nil {
		t.Error("GetAlbums: expected error")
	}
	if _, err := s.AlbumArtwork(1); err == nil {
		t.Error("AlbumArtwork: expected error")
	}
	if _, err := s.GetAlbumTracks(1); err == nil {
		t.Error("GetAlbumTracks: expected error")
	}
	if _, err := s.Search("q", 10); err == nil {
		t.Error("Search: expected error")
	}
	if err := s.SyncServers(); err == nil {
		t.Error("SyncServers: expected error")
	}
	if _, err := s.GetPlaylists(); err == nil {
		t.Error("GetPlaylists: expected error")
	}
	if _, err := s.CreatePlaylist("x"); err == nil {
		t.Error("CreatePlaylist: expected error")
	}
	if err := s.RenamePlaylist(1, "x"); err == nil {
		t.Error("RenamePlaylist: expected error")
	}
	if err := s.DeletePlaylist(1); err == nil {
		t.Error("DeletePlaylist: expected error")
	}
	if _, err := s.GetPlaylistTracks(1); err == nil {
		t.Error("GetPlaylistTracks: expected error")
	}
	if err := s.AddTrackToPlaylist(1, 1); err == nil {
		t.Error("AddTrackToPlaylist: expected error")
	}
	if err := s.RemoveTrackFromPlaylist(1, 1); err == nil {
		t.Error("RemoveTrackFromPlaylist: expected error")
	}
	if err := s.MoveTrackInPlaylist(1, 0, 1); err == nil {
		t.Error("MoveTrackInPlaylist: expected error")
	}
	if _, err := s.GetServers(); err == nil {
		t.Error("GetServers: expected error")
	}
	if err := s.AddServer(ServerConfig{}); err == nil {
		t.Error("AddServer: expected error")
	}
	if err := s.UpdateServer(ServerConfig{}); err == nil {
		t.Error("UpdateServer: expected error")
	}
	if err := s.DeleteServer("x"); err == nil {
		t.Error("DeleteServer: expected error")
	}
	if _, err := s.GetServerStatuses(); err == nil {
		t.Error("GetServerStatuses: expected error")
	}
	if _, err := s.GetScrobbleConfig(); err == nil {
		t.Error("GetScrobbleConfig: expected error")
	}
	if err := s.SaveScrobbleAPIKeys("k", "s"); err == nil {
		t.Error("SaveScrobbleAPIKeys: expected error")
	}
	if err := s.SetScrobbleEnabled(true); err == nil {
		t.Error("SetScrobbleEnabled: expected error")
	}
	if err := s.DisconnectLastFm(); err == nil {
		t.Error("DisconnectLastFm: expected error")
	}
	if _, err := s.GetListenBrainzConfig(); err == nil {
		t.Error("GetListenBrainzConfig: expected error")
	}
	if err := s.DisconnectListenBrainz(); err == nil {
		t.Error("DisconnectListenBrainz: expected error")
	}
	if err := s.SetListenBrainzEnabled(true); err == nil {
		t.Error("SetListenBrainzEnabled: expected error")
	}
	if _, err := s.GetScrobbleQueueSize(); err == nil {
		t.Error("GetScrobbleQueueSize: expected error")
	}
	if _, err := s.GetTopArtists("all", 10); err == nil {
		t.Error("GetTopArtists: expected error")
	}
	if _, err := s.GetTopAlbums("all", 10); err == nil {
		t.Error("GetTopAlbums: expected error")
	}
	if _, err := s.GetTopTracks("all", 10); err == nil {
		t.Error("GetTopTracks: expected error")
	}
	if _, err := s.GetRecentlyPlayed(10); err == nil {
		t.Error("GetRecentlyPlayed: expected error")
	}
	if _, err := s.GetArtistByName("x"); err == nil {
		t.Error("GetArtistByName: expected error")
	}
	if _, err := s.GetArtistInfo("x"); err == nil {
		t.Error("GetArtistInfo: expected error")
	}
	if _, err := s.StartLastFmAuth(); err == nil {
		t.Error("StartLastFmAuth: expected error")
	}
	if err := s.CompleteLastFmAuth("tok"); err == nil {
		t.Error("CompleteLastFmAuth: expected error")
	}
	if err := s.ConnectListenBrainz("tok"); err == nil {
		t.Error("ConnectListenBrainz: expected error")
	}
}

// --- Albums ---

func TestGetAlbumsServiceMapping(t *testing.T) {
	s := openTestService(t)
	mustExecService(t, s, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")
	mustExecService(t, s, "INSERT INTO albums (id, artist_id, title, year, server_id) VALUES (1, 1, 'OK Computer', 1997, '')")
	mustExecService(t, s, "INSERT INTO albums (id, artist_id, title, year, server_id) VALUES (2, 1, 'Kid A', 2000, 'srv-1')")

	albums, err := s.GetAlbums("title", "asc", "")
	if err != nil {
		t.Fatalf("GetAlbums: %v", err)
	}
	if len(albums) != 2 {
		t.Fatalf("got %d albums, want 2", len(albums))
	}
	// Kid A comes first alphabetically.
	if albums[0].Title != "Kid A" {
		t.Errorf("first album = %q", albums[0].Title)
	}
	if albums[0].Source != "server" {
		t.Errorf("Kid A source = %q, want 'server'", albums[0].Source)
	}
	if albums[1].Source != "local" {
		t.Errorf("OK Computer source = %q, want 'local'", albums[1].Source)
	}
}

func TestGetAlbumsEmpty(t *testing.T) {
	s := openTestService(t)
	albums, err := s.GetAlbums("title", "asc", "")
	if err != nil {
		t.Fatalf("GetAlbums: %v", err)
	}
	if len(albums) != 0 {
		t.Errorf("expected empty, got %d", len(albums))
	}
}

func TestAlbumArtworkService(t *testing.T) {
	s := openTestService(t)
	mustExecService(t, s, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExecService(t, s, "INSERT INTO albums (id, artist_id, title) VALUES (1, 1, 'Album')")

	art, err := s.AlbumArtwork(1)
	if err != nil {
		t.Fatalf("AlbumArtwork: %v", err)
	}
	if art != "" {
		t.Errorf("expected empty, got %q", art)
	}
}

func TestGetAlbumTracksServiceMapping(t *testing.T) {
	s := openTestService(t)
	seedTrack(t, s)

	tracks, err := s.GetAlbumTracks(1)
	if err != nil {
		t.Fatalf("GetAlbumTracks: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks, want 1", len(tracks))
	}
	if tracks[0].Title != "Airbag" {
		t.Errorf("title = %q", tracks[0].Title)
	}
	if tracks[0].Artist != "Radiohead" {
		t.Errorf("artist = %q", tracks[0].Artist)
	}
	if tracks[0].Source != "local" {
		t.Errorf("source = %q, want 'local'", tracks[0].Source)
	}
	if tracks[0].DurationMs != 282000 {
		t.Errorf("durationMs = %d", tracks[0].DurationMs)
	}
}

// --- Search ---

func TestSearchService(t *testing.T) {
	s := openTestService(t)
	seedTrack(t, s)

	results, err := s.Search("Airbag", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Title != "Airbag" {
		t.Errorf("title = %q", results[0].Title)
	}
	if results[0].Album != "OK Computer" {
		t.Errorf("album = %q", results[0].Album)
	}
	if results[0].Source != "local" {
		t.Errorf("source = %q, want 'local'", results[0].Source)
	}
}

func TestSearchEmpty(t *testing.T) {
	s := openTestService(t)
	seedTrack(t, s)

	// Empty query returns all tracks.
	results, err := s.Search("", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestSearchNoMatch(t *testing.T) {
	s := openTestService(t)
	seedTrack(t, s)

	results, err := s.Search("nonexistent", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// --- Playlists ---

func TestPlaylistServiceCRUD(t *testing.T) {
	s := openTestService(t)

	id, err := s.CreatePlaylist("My Playlist")
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	playlists, err := s.GetPlaylists()
	if err != nil {
		t.Fatalf("GetPlaylists: %v", err)
	}
	if len(playlists) != 1 {
		t.Fatalf("got %d playlists, want 1", len(playlists))
	}
	if playlists[0].Name != "My Playlist" {
		t.Errorf("name = %q", playlists[0].Name)
	}

	if err := s.RenamePlaylist(id, "Renamed"); err != nil {
		t.Fatalf("RenamePlaylist: %v", err)
	}
	playlists, _ = s.GetPlaylists()
	if playlists[0].Name != "Renamed" {
		t.Errorf("after rename = %q", playlists[0].Name)
	}

	if err := s.DeletePlaylist(id); err != nil {
		t.Fatalf("DeletePlaylist: %v", err)
	}
	playlists, _ = s.GetPlaylists()
	if len(playlists) != 0 {
		t.Errorf("expected empty after delete, got %d", len(playlists))
	}
}

func TestPlaylistTracksService(t *testing.T) {
	s := openTestService(t)
	seedTrack(t, s)
	mustExecService(t, s, `INSERT INTO tracks (id, album_id, artist_id, title, duration_ms, file_path)
		VALUES (2, 1, 1, 'Paranoid Android', 386000, '/music/paranoid.flac')`)

	plID, _ := s.CreatePlaylist("Test")
	if err := s.AddTrackToPlaylist(plID, 1); err != nil {
		t.Fatalf("AddTrackToPlaylist: %v", err)
	}
	if err := s.AddTrackToPlaylist(plID, 2); err != nil {
		t.Fatalf("AddTrackToPlaylist: %v", err)
	}

	tracks, err := s.GetPlaylistTracks(plID)
	if err != nil {
		t.Fatalf("GetPlaylistTracks: %v", err)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
	if tracks[0].Title != "Airbag" {
		t.Errorf("first track = %q", tracks[0].Title)
	}
	if tracks[0].Position != 0 {
		t.Errorf("position = %d, want 0", tracks[0].Position)
	}

	if err := s.RemoveTrackFromPlaylist(plID, 1); err != nil {
		t.Fatalf("RemoveTrackFromPlaylist: %v", err)
	}
	tracks, _ = s.GetPlaylistTracks(plID)
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks after remove, want 1", len(tracks))
	}
}

func TestMoveTrackInPlaylistService(t *testing.T) {
	s := openTestService(t)
	mustExecService(t, s, "INSERT INTO artists (id, name) VALUES (1, 'A')")
	mustExecService(t, s, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (1, 1, 'T1', '/1.flac')`)
	mustExecService(t, s, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (2, 1, 'T2', '/2.flac')`)
	mustExecService(t, s, `INSERT INTO tracks (id, artist_id, title, file_path) VALUES (3, 1, 'T3', '/3.flac')`)

	plID, _ := s.CreatePlaylist("PL")
	_ = s.AddTrackToPlaylist(plID, 1)
	_ = s.AddTrackToPlaylist(plID, 2)
	_ = s.AddTrackToPlaylist(plID, 3)

	if err := s.MoveTrackInPlaylist(plID, 0, 2); err != nil {
		t.Fatalf("MoveTrackInPlaylist: %v", err)
	}

	tracks, _ := s.GetPlaylistTracks(plID)
	if len(tracks) != 3 {
		t.Fatalf("got %d tracks", len(tracks))
	}
	// Expected: T2, T3, T1
	if tracks[0].Title != "T2" || tracks[1].Title != "T3" || tracks[2].Title != "T1" {
		t.Errorf("order = [%q, %q, %q], want [T2, T3, T1]",
			tracks[0].Title, tracks[1].Title, tracks[2].Title)
	}
}

// --- Servers ---

func TestServerServiceCRUD(t *testing.T) {
	s := openTestService(t)

	err := s.AddServer(ServerConfig{
		Name:     "My Navidrome",
		Type:     "subsonic",
		URL:      "http://localhost:4533",
		Username: "admin",
		Password: "pass",
	})
	if err != nil {
		t.Fatalf("AddServer: %v", err)
	}

	servers, err := s.GetServers()
	if err != nil {
		t.Fatalf("GetServers: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("got %d servers, want 1", len(servers))
	}
	if servers[0].Name != "My Navidrome" {
		t.Errorf("name = %q", servers[0].Name)
	}
	if servers[0].ID == "" {
		t.Error("expected non-empty UUID")
	}

	// Update.
	servers[0].Name = "Renamed"
	if err := s.UpdateServer(servers[0]); err != nil {
		t.Fatalf("UpdateServer: %v", err)
	}
	servers, _ = s.GetServers()
	if servers[0].Name != "Renamed" {
		t.Errorf("after update = %q", servers[0].Name)
	}

	// Delete.
	if err := s.DeleteServer(servers[0].ID); err != nil {
		t.Fatalf("DeleteServer: %v", err)
	}
	servers, _ = s.GetServers()
	if len(servers) != 0 {
		t.Errorf("expected empty after delete, got %d", len(servers))
	}
}

func TestGetServerStatusesNoHealth(t *testing.T) {
	s := openTestService(t)
	// health is nil - servers should still appear as online.
	mustExecService(t, s, "INSERT INTO servers (id, name, type, url, username, password) VALUES ('s1', 'Srv', 'subsonic', 'http://localhost', 'u', 'p')")

	statuses, err := s.GetServerStatuses()
	if err != nil {
		t.Fatalf("GetServerStatuses: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("got %d statuses, want 1", len(statuses))
	}
	if statuses[0].ServerID != "s1" {
		t.Errorf("serverID = %q", statuses[0].ServerID)
	}
	if statuses[0].Name != "Srv" {
		t.Errorf("name = %q", statuses[0].Name)
	}
	if !statuses[0].Online {
		t.Error("expected online = true for unpinged server")
	}
}

func TestTestConnectionUnknownType(t *testing.T) {
	s := openTestService(t)
	err := s.TestConnection(ServerConfig{Type: "unknown"})
	if err == nil {
		t.Error("expected error for unknown server type")
	}
}

// --- Scrobble config ---

func TestScrobbleConfigService(t *testing.T) {
	s := openTestService(t)

	cfg, err := s.GetScrobbleConfig()
	if err != nil {
		t.Fatalf("GetScrobbleConfig: %v", err)
	}
	if cfg.APIKey != "" {
		t.Errorf("default APIKey = %q", cfg.APIKey)
	}

	if err := s.SaveScrobbleAPIKeys("mykey", "mysecret"); err != nil {
		t.Fatalf("SaveScrobbleAPIKeys: %v", err)
	}
	cfg, _ = s.GetScrobbleConfig()
	if cfg.APIKey != "mykey" {
		t.Errorf("APIKey = %q, want 'mykey'", cfg.APIKey)
	}

	if err := s.SetScrobbleEnabled(true); err != nil {
		t.Fatalf("SetScrobbleEnabled: %v", err)
	}
	cfg, _ = s.GetScrobbleConfig()
	if !cfg.Enabled {
		t.Error("expected enabled = true")
	}

	if err := s.DisconnectLastFm(); err != nil {
		t.Fatalf("DisconnectLastFm: %v", err)
	}
	cfg, _ = s.GetScrobbleConfig()
	if cfg.Enabled {
		t.Error("expected enabled = false after disconnect")
	}
	if cfg.SessionKey != "" {
		t.Errorf("sessionKey = %q after disconnect", cfg.SessionKey)
	}
	if cfg.Username != "" {
		t.Errorf("username = %q after disconnect", cfg.Username)
	}
}

// --- ListenBrainz config ---

func TestListenBrainzConfigService(t *testing.T) {
	s := openTestService(t)

	cfg, err := s.GetListenBrainzConfig()
	if err != nil {
		t.Fatalf("GetListenBrainzConfig: %v", err)
	}
	if cfg.Username != "" {
		t.Errorf("default username = %q", cfg.Username)
	}

	if err := s.SetListenBrainzEnabled(true); err != nil {
		t.Fatalf("SetListenBrainzEnabled: %v", err)
	}
	cfg, _ = s.GetListenBrainzConfig()
	if !cfg.Enabled {
		t.Error("expected enabled = true")
	}

	if err := s.DisconnectListenBrainz(); err != nil {
		t.Fatalf("DisconnectListenBrainz: %v", err)
	}
	cfg, _ = s.GetListenBrainzConfig()
	if cfg.Enabled {
		t.Error("expected enabled = false after disconnect")
	}
}

// --- Scrobble queue ---

func TestScrobbleQueueSizeService(t *testing.T) {
	s := openTestService(t)

	size, err := s.GetScrobbleQueueSize()
	if err != nil {
		t.Fatalf("GetScrobbleQueueSize: %v", err)
	}
	if size != 0 {
		t.Errorf("expected 0, got %d", size)
	}
}

// --- Stats ---

func TestStatsServiceEmpty(t *testing.T) {
	s := openTestService(t)

	artists, err := s.GetTopArtists("all", 10)
	if err != nil {
		t.Fatalf("GetTopArtists: %v", err)
	}
	if len(artists) != 0 {
		t.Errorf("expected empty, got %d", len(artists))
	}

	albums, err := s.GetTopAlbums("all", 10)
	if err != nil {
		t.Fatalf("GetTopAlbums: %v", err)
	}
	if len(albums) != 0 {
		t.Errorf("expected empty, got %d", len(albums))
	}

	tracks, err := s.GetTopTracks("all", 10)
	if err != nil {
		t.Fatalf("GetTopTracks: %v", err)
	}
	if len(tracks) != 0 {
		t.Errorf("expected empty, got %d", len(tracks))
	}

	recent, err := s.GetRecentlyPlayed(10)
	if err != nil {
		t.Fatalf("GetRecentlyPlayed: %v", err)
	}
	if len(recent) != 0 {
		t.Errorf("expected empty, got %d", len(recent))
	}
}

func TestStatsServiceWithPlays(t *testing.T) {
	s := openTestService(t)
	seedTrack(t, s)
	mustExecService(t, s, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (1, 200000)")
	mustExecService(t, s, "INSERT INTO play_history (track_id, duration_played_ms) VALUES (1, 282000)")

	artists, err := s.GetTopArtists("all", 10)
	if err != nil {
		t.Fatalf("GetTopArtists: %v", err)
	}
	if len(artists) != 1 {
		t.Fatalf("got %d artists, want 1", len(artists))
	}
	if artists[0].Name != "Radiohead" {
		t.Errorf("artist = %q", artists[0].Name)
	}
	if artists[0].PlayCount != 2 {
		t.Errorf("playCount = %d, want 2", artists[0].PlayCount)
	}

	tracks, err := s.GetTopTracks("all", 10)
	if err != nil {
		t.Fatalf("GetTopTracks: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("got %d tracks, want 1", len(tracks))
	}
	if tracks[0].Name != "Airbag" {
		t.Errorf("track = %q", tracks[0].Name)
	}
	if tracks[0].TotalMs != 482000 {
		t.Errorf("totalMs = %d, want 482000", tracks[0].TotalMs)
	}

	albums, err := s.GetTopAlbums("all", 10)
	if err != nil {
		t.Fatalf("GetTopAlbums: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("got %d albums, want 1", len(albums))
	}
	if albums[0].Name != "OK Computer" {
		t.Errorf("album = %q", albums[0].Name)
	}

	recent, err := s.GetRecentlyPlayed(10)
	if err != nil {
		t.Fatalf("GetRecentlyPlayed: %v", err)
	}
	if len(recent) != 2 {
		t.Fatalf("got %d recent plays, want 2", len(recent))
	}
	if recent[0].Title != "Airbag" {
		t.Errorf("recent title = %q", recent[0].Title)
	}
	if recent[0].Artist != "Radiohead" {
		t.Errorf("recent artist = %q", recent[0].Artist)
	}
}

// --- Artist info ---

func TestGetArtistByNameService(t *testing.T) {
	s := openTestService(t)
	mustExecService(t, s, "INSERT INTO artists (id, name) VALUES (1, 'Radiohead')")

	id, err := s.GetArtistByName("Radiohead")
	if err != nil {
		t.Fatalf("GetArtistByName: %v", err)
	}
	if id != 1 {
		t.Errorf("id = %d, want 1", id)
	}
}

func TestGetArtistByNameNotFound(t *testing.T) {
	s := openTestService(t)
	_, err := s.GetArtistByName("Nonexistent")
	if err == nil {
		t.Error("expected error for missing artist")
	}
}

// --- Helpers ---

func TestSourceFromServerID(t *testing.T) {
	if got := sourceFromServerID(""); got != "local" {
		t.Errorf("empty = %q, want 'local'", got)
	}
	if got := sourceFromServerID("srv-1"); got != "server" {
		t.Errorf("srv-1 = %q, want 'server'", got)
	}
}

func TestNewUUID(t *testing.T) {
	id1, err := newUUID()
	if err != nil {
		t.Fatalf("newUUID: %v", err)
	}
	if len(id1) != 36 {
		t.Errorf("uuid length = %d, want 36", len(id1))
	}

	id2, _ := newUUID()
	if id1 == id2 {
		t.Error("two UUIDs should not be equal")
	}
}

// --- ServiceShutdown ---

func TestServiceShutdown(t *testing.T) {
	s := openTestService(t)
	// ServiceShutdown should not panic even without health or stopSync.
	if err := s.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown: %v", err)
	}
}

func TestServiceShutdownWithStopSync(t *testing.T) {
	s := openTestService(t)
	s.stopSync = make(chan struct{})
	// Should close the channel without blocking.
	if err := s.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown: %v", err)
	}
}

func TestServiceShutdownNilDB(t *testing.T) {
	s := &LibraryService{}
	if err := s.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown nil db: %v", err)
	}
}
