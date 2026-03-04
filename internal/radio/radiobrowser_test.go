package radio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseStations(t *testing.T) {
	raw := `[
		{
			"stationuuid": "abc-123",
			"name": "Test Radio",
			"url_resolved": "https://stream.example.com/radio",
			"favicon": "https://example.com/icon.png",
			"country": "United Kingdom",
			"tags": "rock,indie",
			"bitrate": 128,
			"codec": "MP3",
			"votes": 42,
			"clickcount": 100
		},
		{
			"stationuuid": "def-456",
			"name": "Jazz FM",
			"url_resolved": "https://stream.example.com/jazz",
			"favicon": "",
			"country": "Germany",
			"tags": "jazz,smooth",
			"bitrate": 256,
			"codec": "AAC",
			"votes": 10,
			"clickcount": 50
		}
	]`

	stations, err := parseStations([]byte(raw))
	if err != nil {
		t.Fatalf("parseStations: %v", err)
	}

	if len(stations) != 2 {
		t.Fatalf("expected 2 stations, got %d", len(stations))
	}

	s := stations[0]
	if s.UUID != "abc-123" {
		t.Errorf("UUID = %q, want %q", s.UUID, "abc-123")
	}
	if s.Name != "Test Radio" {
		t.Errorf("Name = %q, want %q", s.Name, "Test Radio")
	}
	if s.StreamURL != "https://stream.example.com/radio" {
		t.Errorf("StreamURL = %q", s.StreamURL)
	}
	if s.Favicon != "https://example.com/icon.png" {
		t.Errorf("Favicon = %q", s.Favicon)
	}
	if s.Country != "United Kingdom" {
		t.Errorf("Country = %q", s.Country)
	}
	if s.Tags != "rock,indie" {
		t.Errorf("Tags = %q", s.Tags)
	}
	if s.Bitrate != 128 {
		t.Errorf("Bitrate = %d", s.Bitrate)
	}
	if s.Codec != "MP3" {
		t.Errorf("Codec = %q", s.Codec)
	}
	if s.Votes != 42 {
		t.Errorf("Votes = %d", s.Votes)
	}
	if s.Clicks != 100 {
		t.Errorf("Clicks = %d", s.Clicks)
	}
}

func TestParseStationsEmpty(t *testing.T) {
	stations, err := parseStations([]byte("[]"))
	if err != nil {
		t.Fatalf("parseStations: %v", err)
	}
	if len(stations) != 0 {
		t.Fatalf("expected 0 stations, got %d", len(stations))
	}
}

func TestParseStationsInvalidJSON(t *testing.T) {
	_, err := parseStations([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestClientSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/stations/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "jazz" {
			t.Errorf("expected name=jazz, got %s", r.URL.Query().Get("name"))
		}
		if r.Header.Get("User-Agent") != "Forte/1.0" {
			t.Errorf("unexpected User-Agent: %s", r.Header.Get("User-Agent"))
		}

		stations := []Station{
			{UUID: "abc", Name: "Jazz FM", StreamURL: "https://stream.example.com/jazz"},
		}
		_ = json.NewEncoder(w).Encode(stations)
	}))
	defer server.Close()

	c := NewClient()
	c.servers = []string{server.URL}

	stations, err := c.Search("jazz", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(stations) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations))
	}
	if stations[0].Name != "Jazz FM" {
		t.Errorf("Name = %q, want %q", stations[0].Name, "Jazz FM")
	}
}

func TestClientTopVoted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/stations/topvote" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		stations := []Station{
			{UUID: "top1", Name: "Popular Radio", Votes: 999},
		}
		_ = json.NewEncoder(w).Encode(stations)
	}))
	defer server.Close()

	c := NewClient()
	c.servers = []string{server.URL}

	stations, err := c.TopVoted(5)
	if err != nil {
		t.Fatalf("TopVoted: %v", err)
	}
	if len(stations) != 1 {
		t.Fatalf("expected 1 station, got %d", len(stations))
	}
	if stations[0].Votes != 999 {
		t.Errorf("Votes = %d, want 999", stations[0].Votes)
	}
}

func TestClientFallback(t *testing.T) {
	// First server returns error, second succeeds.
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badServer.Close()

	goodServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]Station{{UUID: "ok", Name: "Working"}})
	}))
	defer goodServer.Close()

	c := NewClient()
	c.servers = []string{badServer.URL, goodServer.URL}

	// Run multiple times to exercise the shuffle + fallback.
	for i := 0; i < 5; i++ {
		stations, err := c.Search("test", 10)
		if err != nil {
			t.Fatalf("attempt %d: Search: %v", i, err)
		}
		if len(stations) != 1 {
			t.Fatalf("attempt %d: expected 1 station, got %d", i, len(stations))
		}
	}
}

func TestClientAllMirrorsFail(t *testing.T) {
	badServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer badServer.Close()

	c := NewClient()
	c.servers = []string{badServer.URL}

	_, err := c.Search("test", 10)
	if err == nil {
		t.Fatal("expected error when all mirrors fail")
	}
}
