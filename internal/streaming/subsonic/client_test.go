package subsonic

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// cannedServer returns an httptest.Server that responds with canned JSON
// keyed by the Subsonic method name extracted from the request path.
func cannedServer(t *testing.T, responses map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Path is /rest/{method}.view
		path := strings.TrimPrefix(r.URL.Path, "/rest/")
		method := strings.TrimSuffix(path, ".view")

		body, ok := responses[method]
		if !ok {
			t.Errorf("unexpected method: %s", method)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func wrap(body any) map[string]any {
	return map[string]any{
		"subsonic-response": body,
	}
}

func okBody(extra map[string]any) map[string]any {
	body := map[string]any{
		"status":  "ok",
		"version": "1.16.1",
	}
	for k, v := range extra {
		body[k] = v
	}
	return body
}

func TestPing(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"ping": wrap(okBody(nil)),
	})
	defer srv.Close()

	c := New(srv.URL, "admin", "admin")
	if err := c.Ping(); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestPingAuthFailure(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"ping": wrap(map[string]any{
			"status":  "failed",
			"version": "1.16.1",
			"error":   map[string]any{"code": 40, "message": "Wrong username or password"},
		}),
	})
	defer srv.Close()

	c := New(srv.URL, "bad", "bad")
	err := c.Ping()
	if err == nil {
		t.Fatal("expected auth error")
	}

	apiErr, ok := err.(*apiError)
	if !ok {
		t.Fatalf("expected *apiError, got %T", err)
	}
	if apiErr.Code != 40 {
		t.Errorf("error code = %d, want 40", apiErr.Code)
	}
}

func TestGetArtists(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"getArtists": wrap(okBody(map[string]any{
			"artists": map[string]any{
				"index": []map[string]any{
					{
						"name": "A",
						"artist": []map[string]any{
							{"id": "ar-1", "name": "ABBA", "albumCount": 10},
						},
					},
					{
						"name": "B",
						"artist": []map[string]any{
							{"id": "ar-2", "name": "Beatles", "albumCount": 13},
							{"id": "ar-3", "name": "Bjork", "albumCount": 9},
						},
					},
				},
			},
		})),
	})
	defer srv.Close()

	c := New(srv.URL, "admin", "admin")
	artists, err := c.GetArtists()
	if err != nil {
		t.Fatalf("GetArtists: %v", err)
	}

	if len(artists) != 3 {
		t.Fatalf("got %d artists, want 3", len(artists))
	}
	if artists[0].Name != "ABBA" {
		t.Errorf("artists[0].Name = %q, want ABBA", artists[0].Name)
	}
	if artists[1].AlbumCount != 13 {
		t.Errorf("artists[1].AlbumCount = %d, want 13", artists[1].AlbumCount)
	}
}

func TestGetAlbums(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"getAlbumList2": wrap(okBody(map[string]any{
			"albumList2": map[string]any{
				"album": []map[string]any{
					{"id": "al-1", "title": "Abbey Road", "artist": "Beatles", "artistId": "ar-2", "year": 1969, "songCount": 17, "coverArt": "al-1"},
					{"id": "al-2", "title": "Arrival", "artist": "ABBA", "artistId": "ar-1", "year": 1976, "songCount": 10, "coverArt": "al-2"},
				},
			},
		})),
	})
	defer srv.Close()

	c := New(srv.URL, "admin", "admin")
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
}

func TestGetAlbum(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"getAlbum": wrap(okBody(map[string]any{
			"album": map[string]any{
				"id": "al-1", "title": "Abbey Road", "artist": "Beatles", "artistId": "ar-2",
				"year": 1969, "songCount": 2, "coverArt": "al-1",
				"song": []map[string]any{
					{
						"id": "tr-1", "title": "Come Together", "artist": "Beatles", "artistId": "ar-2",
						"album": "Abbey Road", "albumId": "al-1", "coverArt": "al-1",
						"duration": 259, "track": 1, "discNumber": 1,
						"genre": "Rock", "contentType": "audio/flac", "size": 35000000,
					},
					{
						"id": "tr-2", "title": "Something", "artist": "Beatles", "artistId": "ar-2",
						"album": "Abbey Road", "albumId": "al-1", "coverArt": "al-1",
						"duration": 182, "track": 2, "discNumber": 1,
						"genre": "Rock", "contentType": "audio/flac", "size": 25000000,
					},
				},
			},
		})),
	})
	defer srv.Close()

	c := New(srv.URL, "admin", "admin")
	album, tracks, err := c.GetAlbum("al-1")
	if err != nil {
		t.Fatalf("GetAlbum: %v", err)
	}

	if album.Title != "Abbey Road" {
		t.Errorf("album.Title = %q, want Abbey Road", album.Title)
	}
	if len(tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(tracks))
	}

	// Duration should be converted from seconds to milliseconds.
	if tracks[0].DurationMs != 259000 {
		t.Errorf("tracks[0].DurationMs = %d, want 259000", tracks[0].DurationMs)
	}
	if tracks[1].DurationMs != 182000 {
		t.Errorf("tracks[1].DurationMs = %d, want 182000", tracks[1].DurationMs)
	}
	if tracks[0].TrackNumber != 1 {
		t.Errorf("tracks[0].TrackNumber = %d, want 1", tracks[0].TrackNumber)
	}
}

func TestStreamURL(t *testing.T) {
	c := New("http://localhost:4533", "admin", "admin")
	u := c.StreamURL("tr-42")

	if u == "" {
		t.Fatal("StreamURL returned empty string")
	}
	if !strings.HasPrefix(u, "http://localhost:4533/rest/stream.view?") {
		t.Errorf("unexpected URL prefix: %s", u)
	}
	if !strings.Contains(u, "id=tr-42") {
		t.Errorf("URL missing track id: %s", u)
	}
	if !strings.Contains(u, "u=admin") {
		t.Errorf("URL missing username: %s", u)
	}
}

func TestCoverArtURL(t *testing.T) {
	c := New("http://localhost:4533", "admin", "admin")
	u := c.CoverArtURL("al-1")

	if u == "" {
		t.Fatal("CoverArtURL returned empty string")
	}
	if !strings.HasPrefix(u, "http://localhost:4533/rest/getCoverArt.view?") {
		t.Errorf("unexpected URL prefix: %s", u)
	}
	if !strings.Contains(u, "id=al-1") {
		t.Errorf("URL missing cover art id: %s", u)
	}
}

func TestSearch(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"search3": wrap(okBody(map[string]any{
			"searchResult3": map[string]any{
				"artist": []map[string]any{
					{"id": "ar-2", "name": "Beatles", "albumCount": 13},
				},
				"album": []map[string]any{
					{"id": "al-1", "title": "Abbey Road", "artist": "Beatles", "artistId": "ar-2", "year": 1969, "songCount": 17, "coverArt": "al-1"},
				},
				"song": []map[string]any{
					{"id": "tr-1", "title": "Come Together", "artist": "Beatles", "artistId": "ar-2", "album": "Abbey Road", "albumId": "al-1", "duration": 259},
				},
			},
		})),
	})
	defer srv.Close()

	c := New(srv.URL, "admin", "admin")
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
		"search3": wrap(okBody(map[string]any{
			"searchResult3": map[string]any{},
		})),
	})
	defer srv.Close()

	c := New(srv.URL, "admin", "admin")
	results, err := c.Search("zzzzz")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	if len(results.Artists) != 0 || len(results.Albums) != 0 || len(results.Tracks) != 0 {
		t.Errorf("expected empty results, got artists=%d albums=%d tracks=%d",
			len(results.Artists), len(results.Albums), len(results.Tracks))
	}
}

func TestAPIError(t *testing.T) {
	srv := cannedServer(t, map[string]any{
		"getAlbum": wrap(map[string]any{
			"status":  "failed",
			"version": "1.16.1",
			"error":   map[string]any{"code": 70, "message": "Album not found"},
		}),
	})
	defer srv.Close()

	c := New(srv.URL, "admin", "admin")
	_, _, err := c.GetAlbum("nonexistent")
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*apiError)
	if !ok {
		t.Fatalf("expected *apiError, got %T", err)
	}
	if apiErr.Code != 70 {
		t.Errorf("error code = %d, want 70", apiErr.Code)
	}
	if !strings.Contains(apiErr.Error(), "Album not found") {
		t.Errorf("error message = %q, want to contain 'Album not found'", apiErr.Error())
	}
}
