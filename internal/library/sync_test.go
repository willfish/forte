package library

import (
	"context"
	"testing"

	"github.com/willfish/forte/internal/streaming"
)

func TestServerFilePath(t *testing.T) {
	got := serverFilePath("srv-1", "track-42")
	want := "server://srv-1/track-42"
	if got != want {
		t.Errorf("serverFilePath = %q, want %q", got, want)
	}
}

func TestNewProviderSubsonic(t *testing.T) {
	srv := Server{ID: "1", Name: "Test", Type: "subsonic", URL: "http://localhost", Username: "u", Password: "p"}
	p, err := NewServerProvider(srv)
	if err != nil {
		t.Fatalf("NewServerProvider subsonic: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewProviderJellyfin(t *testing.T) {
	srv := Server{ID: "1", Name: "Test", Type: "jellyfin", URL: "http://localhost", Username: "u", Password: "p"}
	p, err := NewServerProvider(srv)
	if err != nil {
		t.Fatalf("NewServerProvider jellyfin: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewProviderUnknown(t *testing.T) {
	srv := Server{ID: "1", Name: "Test", Type: "unknown", URL: "http://localhost"}
	_, err := NewServerProvider(srv)
	if err == nil {
		t.Error("expected error for unknown server type")
	}
}

func TestSyncAlbumFetchesAlbumDetailOnce(t *testing.T) {
	db := openTestDB(t)
	provider := &countingProvider{
		album: streaming.Album{ID: "album-1", Title: "Album", Artist: "Artist"},
		tracks: []streaming.Track{
			{ID: "track-1", Title: "Track 1", Artist: "Artist", TrackNumber: 1},
			{ID: "track-2", Title: "Track 2", Artist: "Artist", TrackNumber: 2},
		},
	}

	seen, err := syncAlbum(context.Background(), db, provider, Server{ID: "srv-1"}, provider.album)
	if err != nil {
		t.Fatalf("syncAlbum: %v", err)
	}
	if provider.getAlbumCalls != 1 {
		t.Fatalf("GetAlbum calls = %d, want 1", provider.getAlbumCalls)
	}
	if len(seen) != 2 {
		t.Fatalf("seen paths = %d, want 2", len(seen))
	}
	if seen[0] != "server://srv-1/track-1" || seen[1] != "server://srv-1/track-2" {
		t.Fatalf("seen paths = %#v", seen)
	}
}

type countingProvider struct {
	album         streaming.Album
	tracks        []streaming.Track
	getAlbumCalls int
}

func (p *countingProvider) Ping() error { return nil }

func (p *countingProvider) GetArtists() ([]streaming.Artist, error) { return nil, nil }

func (p *countingProvider) GetAlbums(string, int, int) ([]streaming.Album, error) {
	return []streaming.Album{p.album}, nil
}

func (p *countingProvider) GetAlbum(string) (streaming.Album, []streaming.Track, error) {
	p.getAlbumCalls++
	return p.album, p.tracks, nil
}

func (p *countingProvider) StreamURL(trackID string) string { return trackID }

func (p *countingProvider) CoverArtURL(string) string { return "" }

func (p *countingProvider) Search(string) (streaming.SearchResults, error) {
	return streaming.SearchResults{}, nil
}

func TestSyncAlbumKeepsTrackIDAndPlaylistMembership(t *testing.T) {
	db := openTestDB(t)
	provider := &countingProvider{
		album: streaming.Album{ID: "album-1", Title: "Album", Artist: "Artist"},
		tracks: []streaming.Track{
			{ID: "track-1", Title: "Track 1", Artist: "Artist", TrackNumber: 1},
		},
	}
	srv := Server{ID: "srv-1"}

	if _, err := syncAlbum(context.Background(), db, provider, srv, provider.album); err != nil {
		t.Fatalf("first syncAlbum: %v", err)
	}

	var trackID int64
	if err := db.QueryRow("SELECT id FROM tracks WHERE file_path = ?", "server://srv-1/track-1").Scan(&trackID); err != nil {
		t.Fatalf("select track: %v", err)
	}
	mustExec(t, db, "INSERT INTO playlists (id, name) VALUES (1, 'Favourites')")
	mustExec(t, db, "INSERT INTO playlist_tracks (playlist_id, track_id, position) VALUES (1, ?, 0)", trackID)
	mustExec(t, db, "INSERT INTO play_history (track_id) VALUES (?)", trackID)

	provider.tracks[0].Title = "Track 1 (remaster)"
	if _, err := syncAlbum(context.Background(), db, provider, srv, provider.album); err != nil {
		t.Fatalf("second syncAlbum: %v", err)
	}

	var gotID int64
	var title string
	if err := db.QueryRow("SELECT id, title FROM tracks WHERE file_path = ?", "server://srv-1/track-1").Scan(&gotID, &title); err != nil {
		t.Fatalf("select after resync: %v", err)
	}
	if gotID != trackID {
		t.Fatalf("track id = %d, want %d", gotID, trackID)
	}
	if title != "Track 1 (remaster)" {
		t.Fatalf("title = %q, want remastered title", title)
	}

	var playlistCount, historyCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM playlist_tracks WHERE track_id = ?", trackID).Scan(&playlistCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT COUNT(*) FROM play_history WHERE track_id = ?", trackID).Scan(&historyCount); err != nil {
		t.Fatal(err)
	}
	if playlistCount != 1 || historyCount != 1 {
		t.Fatalf("playlist=%d history=%d, want 1 and 1", playlistCount, historyCount)
	}
}
