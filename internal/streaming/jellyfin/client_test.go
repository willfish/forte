package jellyfin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const testUserID = "user-abc-123"
const testToken = "fake-token-xyz"

// cannedServer returns an httptest.Server with route matching on method + path.
// Auth is pre-registered so all subsequent paths use the known userID.
func cannedServer(t *testing.T, routes map[string]any) *httptest.Server {
	t.Helper()

	// Always register auth endpoint.
	if _, ok := routes["POST /Users/AuthenticateByName"]; !ok {
		routes["POST /Users/AuthenticateByName"] = authResponse{
			AccessToken: testToken,
			User:        authUser{ID: testUserID},
		}
	}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path

		body, ok := routes[key]
		if !ok {
			t.Errorf("unexpected request: %s", key)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func newClient(srv *httptest.Server) *Client {
	return NewWithHTTPClient(srv.URL, "admin", "admin", srv.Client())
}

func TestPing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/System/Ping" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, "", "", srv.Client())
	if err := c.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestPingFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, "", "", srv.Client())
	if err := c.Ping(); err == nil {
		t.Fatal("expected error from Ping")
	}
}

func TestGetArtists(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"GET /Artists/AlbumArtists": itemsResponse{
			Items: []itemJSON{
				{ID: "ar-1", Name: "ABBA", ChildCount: 10},
				{ID: "ar-2", Name: "Beatles", ChildCount: 13},
			},
			TotalRecordCount: 2,
		},
	})
	defer srv.Close()

	c := newClient(srv)
	artists, err := c.GetArtists()
	if err != nil {
		t.Fatalf("GetArtists: %v", err)
	}

	if len(artists) != 2 {
		t.Fatalf("got %d artists, want 2", len(artists))
	}
	if artists[0].Name != "ABBA" {
		t.Errorf("artists[0].Name = %q, want ABBA", artists[0].Name)
	}
	if artists[0].AlbumCount != 10 {
		t.Errorf("artists[0].AlbumCount = %d, want 10", artists[0].AlbumCount)
	}
	if artists[1].AlbumCount != 13 {
		t.Errorf("artists[1].AlbumCount = %d, want 13", artists[1].AlbumCount)
	}
}

func TestGetAlbums(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"GET /Users/" + testUserID + "/Items": itemsResponse{
			Items: []itemJSON{
				{
					ID: "al-1", Name: "Abbey Road",
					AlbumArtist:    "Beatles",
					AlbumArtists:   []nameIDPair{{Name: "Beatles", ID: "ar-2"}},
					ProductionYear: 1969, ChildCount: 17,
				},
				{
					ID: "al-2", Name: "Arrival",
					AlbumArtist:    "ABBA",
					AlbumArtists:   []nameIDPair{{Name: "ABBA", ID: "ar-1"}},
					ProductionYear: 1976, ChildCount: 10,
				},
			},
			TotalRecordCount: 2,
		},
	})
	defer srv.Close()

	c := newClient(srv)
	albums, err := c.GetAlbums("alphabeticalByName", 0, 20)
	if err != nil {
		t.Fatalf("GetAlbums: %v", err)
	}

	if len(albums) != 2 {
		t.Fatalf("got %d albums, want 2", len(albums))
	}
	if albums[0].Title != "Abbey Road" {
		t.Errorf("albums[0].Title = %q, want Abbey Road", albums[0].Title)
	}
	if albums[0].Year != 1969 {
		t.Errorf("albums[0].Year = %d, want 1969", albums[0].Year)
	}
	if albums[0].TrackCount != 17 {
		t.Errorf("albums[0].TrackCount = %d, want 17", albums[0].TrackCount)
	}
	if albums[0].ArtistID != "ar-2" {
		t.Errorf("albums[0].ArtistID = %q, want ar-2", albums[0].ArtistID)
	}
}

func TestGetAlbum(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"GET /Users/" + testUserID + "/Items/al-1": itemJSON{
			ID: "al-1", Name: "Abbey Road",
			AlbumArtist:    "Beatles",
			AlbumArtists:   []nameIDPair{{Name: "Beatles", ID: "ar-2"}},
			ProductionYear: 1969, ChildCount: 2,
		},
		"GET /Users/" + testUserID + "/Items": itemsResponse{
			Items: []itemJSON{
				{
					ID: "tr-1", Name: "Come Together", Type: "Audio",
					AlbumArtist:       "Beatles",
					AlbumArtists:      []nameIDPair{{Name: "Beatles", ID: "ar-2"}},
					Album:             "Abbey Road",
					AlbumID:           "al-1",
					RunTimeTicks:      2590000000,
					IndexNumber:       1,
					ParentIndexNumber: 1,
					Genres:            []string{"Rock"},
					MediaSources:      []mediaSource{{Container: "flac", Size: 35000000}},
				},
				{
					ID: "tr-2", Name: "Something", Type: "Audio",
					AlbumArtist:       "Beatles",
					AlbumArtists:      []nameIDPair{{Name: "Beatles", ID: "ar-2"}},
					Album:             "Abbey Road",
					AlbumID:           "al-1",
					RunTimeTicks:      1820000000,
					IndexNumber:       2,
					ParentIndexNumber: 1,
					Genres:            []string{"Rock"},
					MediaSources:      []mediaSource{{Container: "flac", Size: 25000000}},
				},
			},
			TotalRecordCount: 2,
		},
	})
	defer srv.Close()

	c := newClient(srv)
	album, tracks, err := c.GetAlbum("al-1")
	if err != nil {
		t.Fatalf("GetAlbum: %v", err)
	}

	if album.Title != "Abbey Road" {
		t.Errorf("album.Title = %q, want Abbey Road", album.Title)
	}
	if album.CoverArtID != "al-1" {
		t.Errorf("album.CoverArtID = %q, want al-1", album.CoverArtID)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}
	if tracks[0].DurationMs != 259000 {
		t.Errorf("tracks[0].DurationMs = %d, want 259000", tracks[0].DurationMs)
	}
	if tracks[1].DurationMs != 182000 {
		t.Errorf("tracks[1].DurationMs = %d, want 182000", tracks[1].DurationMs)
	}
	if tracks[0].TrackNumber != 1 {
		t.Errorf("tracks[0].TrackNumber = %d, want 1", tracks[0].TrackNumber)
	}
	if tracks[0].Genre != "Rock" {
		t.Errorf("tracks[0].Genre = %q, want Rock", tracks[0].Genre)
	}
	if tracks[0].ContentType != "audio/flac" {
		t.Errorf("tracks[0].ContentType = %q, want audio/flac", tracks[0].ContentType)
	}
	if tracks[0].Size != 35000000 {
		t.Errorf("tracks[0].Size = %d, want 35000000", tracks[0].Size)
	}
}

func TestStreamURL(t *testing.T) {
	srv := cannedServer(t, map[string]any{})
	defer srv.Close()

	c := newClient(srv)
	// Force auth so token is populated.
	c.authenticate()

	u := c.StreamURL("tr-42")
	if !strings.HasPrefix(u, srv.URL+"/Audio/tr-42/stream?") {
		t.Errorf("unexpected URL prefix: %s", u)
	}
	if !strings.Contains(u, "static=true") {
		t.Errorf("URL missing static=true: %s", u)
	}
	if !strings.Contains(u, "api_key="+testToken) {
		t.Errorf("URL missing api_key: %s", u)
	}
}

func TestCoverArtURL(t *testing.T) {
	c := New("http://localhost:8096", "", "")
	u := c.CoverArtURL("al-1")

	if u != "http://localhost:8096/Items/al-1/Images/Primary?maxWidth=300" {
		t.Errorf("unexpected URL: %s", u)
	}
}

func TestSearch(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"GET /Users/" + testUserID + "/Items": itemsResponse{
			Items: []itemJSON{
				{ID: "ar-2", Name: "Beatles", Type: "MusicArtist", ChildCount: 13},
				{
					ID: "al-1", Name: "Abbey Road", Type: "MusicAlbum",
					AlbumArtist:    "Beatles",
					AlbumArtists:   []nameIDPair{{Name: "Beatles", ID: "ar-2"}},
					ProductionYear: 1969, ChildCount: 17,
				},
				{
					ID: "tr-1", Name: "Come Together", Type: "Audio",
					AlbumArtist:  "Beatles",
					AlbumArtists: []nameIDPair{{Name: "Beatles", ID: "ar-2"}},
					Album:        "Abbey Road", AlbumID: "al-1",
					RunTimeTicks: 2590000000,
				},
			},
			TotalRecordCount: 3,
		},
	})
	defer srv.Close()

	c := newClient(srv)
	results, err := c.Search("beatles")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Artists) != 1 {
		t.Errorf("got %d artists, want 1", len(results.Artists))
	}
	if len(results.Albums) != 1 {
		t.Errorf("got %d albums, want 1", len(results.Albums))
	}
	if len(results.Tracks) != 1 {
		t.Errorf("got %d tracks, want 1", len(results.Tracks))
	}
}

func TestSearchEmpty(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"GET /Users/" + testUserID + "/Items": itemsResponse{
			Items:            []itemJSON{},
			TotalRecordCount: 0,
		},
	})
	defer srv.Close()

	c := newClient(srv)
	results, err := c.Search("zzzzz")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Artists) != 0 || len(results.Albums) != 0 || len(results.Tracks) != 0 {
		t.Errorf("expected empty results, got artists=%d albums=%d tracks=%d",
			len(results.Artists), len(results.Albums), len(results.Tracks))
	}
}

func TestAuthFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Users/AuthenticateByName" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, "bad", "bad", srv.Client())
	_, err := c.GetArtists()
	if err == nil {
		t.Fatal("expected auth error")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention 401, got: %v", err)
	}
}

func TestAuthCalledOnce(t *testing.T) {
	var authCalls atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/Users/AuthenticateByName" {
			authCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(authResponse{
				AccessToken: testToken,
				User:        authUser{ID: testUserID},
			})
			return
		}

		// Respond to any GET with an empty items response.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(itemsResponse{})
	}))
	defer srv.Close()

	c := NewWithHTTPClient(srv.URL, "admin", "admin", srv.Client())

	// Make multiple requests.
	_, _ = c.GetArtists()
	_, _ = c.GetAlbums("alphabeticalByName", 0, 10)
	_, _ = c.Search("test")

	if n := authCalls.Load(); n != 1 {
		t.Errorf("auth called %d times, want 1", n)
	}
}
