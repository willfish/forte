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
	p, err := newProvider(srv)
	if err != nil {
		t.Fatalf("newProvider subsonic: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewProviderJellyfin(t *testing.T) {
	srv := Server{ID: "1", Name: "Test", Type: "jellyfin", URL: "http://localhost", Username: "u", Password: "p"}
	p, err := newProvider(srv)
	if err != nil {
		t.Fatalf("newProvider jellyfin: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewProviderUnknown(t *testing.T) {
	srv := Server{ID: "1", Name: "Test", Type: "unknown", URL: "http://localhost"}
	_, err := newProvider(srv)
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
