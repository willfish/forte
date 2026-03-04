package radio

import (
	"testing"
	"time"
)

func TestExtractSomaFMID(t *testing.T) {
	tests := []struct {
		name     string
		homepage string
		want     string
	}{
		{"direct URL", "https://somafm.com/groovesalad/", "groovesalad"},
		{"no trailing slash", "https://somafm.com/dronezone", "dronezone"},
		{"http scheme", "http://somafm.com/insound/", "insound"},
		{"player URL", "https://somafm.com/player/#/now-playing/metal", "metal"},
		{"not somafm", "https://example.com/station/", ""},
		{"empty", "", ""},
		{"root URL", "https://somafm.com/", ""},
		{"player without station", "https://somafm.com/player", ""},
		{"image path", "https://somafm.com/img3/something", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSomaFMID(tt.homepage)
			if got != tt.want {
				t.Errorf("extractSomaFMID(%q) = %q, want %q", tt.homepage, got, tt.want)
			}
		})
	}
}

func TestSomaFMClientLookupArtwork(t *testing.T) {
	s := NewSomaFMClient()

	// Pre-populate the cache to avoid hitting the network.
	s.mu.Lock()
	s.channels = []somafmChannel{
		{ID: "groovesalad", Title: "Groove Salad", Image: "https://somafm.com/logos/512/groovesalad512.png"},
		{ID: "dronezone", Title: "Drone Zone", Image: "https://somafm.com/logos/512/dronezone512.png"},
		{ID: "insound", Title: "The In-Sound", Image: "https://somafm.com/logos/512/insound512.jpg"},
	}
	s.artIndex = map[string]string{
		"groovesalad": "https://somafm.com/logos/512/groovesalad512.png",
		"dronezone":   "https://somafm.com/logos/512/dronezone512.png",
		"insound":     "https://somafm.com/logos/512/insound512.jpg",
	}
	s.fetchedAt = time.Now()
	s.mu.Unlock()

	tests := []struct {
		name     string
		homepage string
		want     string
	}{
		{"known channel", "https://somafm.com/groovesalad/", "https://somafm.com/logos/512/groovesalad512.png"},
		{"player URL", "https://somafm.com/player/#/now-playing/dronezone", "https://somafm.com/logos/512/dronezone512.png"},
		{"channel without favicon", "https://somafm.com/insound/", "https://somafm.com/logos/512/insound512.jpg"},
		{"non-SomaFM", "https://example.com/radio/", ""},
		{"empty homepage", "", ""},
		{"unknown channel", "https://somafm.com/doesnotexist/", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.LookupArtwork(tt.homepage)
			if got != tt.want {
				t.Errorf("LookupArtwork(%q) = %q, want %q", tt.homepage, got, tt.want)
			}
		})
	}
}

func TestSomaFMClientStations(t *testing.T) {
	s := NewSomaFMClient()

	s.mu.Lock()
	s.channels = []somafmChannel{
		{
			ID: "groovesalad", Title: "Groove Salad", Genre: "ambient",
			Image: "https://somafm.com/logos/512/groovesalad512.png",
			Playlists: []somafmPlaylist{
				{URL: "https://api.somafm.com/groovesalad.pls", Format: "mp3"},
				{URL: "https://api.somafm.com/groovesalad130.pls", Format: "aac"},
			},
		},
		{
			ID: "dronezone", Title: "Drone Zone", Genre: "ambient|space",
			Image: "https://somafm.com/logos/512/dronezone512.png",
			Playlists: []somafmPlaylist{
				{URL: "https://api.somafm.com/dronezone130.pls", Format: "aac"},
				{URL: "https://api.somafm.com/dronezone.pls", Format: "mp3"},
			},
		},
		{
			ID: "nostream", Title: "No Stream", Genre: "test",
			Image: "https://somafm.com/logos/512/nostream512.png",
			Playlists: []somafmPlaylist{
				{URL: "https://api.somafm.com/nostream130.pls", Format: "aac"},
			},
		},
	}
	s.artIndex = map[string]string{
		"groovesalad": "https://somafm.com/logos/512/groovesalad512.png",
		"dronezone":   "https://somafm.com/logos/512/dronezone512.png",
	}
	s.fetchedAt = time.Now()
	s.mu.Unlock()

	stations, err := s.Stations()
	if err != nil {
		t.Fatalf("Stations() error: %v", err)
	}

	// Should skip "nostream" (no mp3 playlist).
	if len(stations) != 2 {
		t.Fatalf("got %d stations, want 2", len(stations))
	}

	if stations[0].UUID != "somafm-groovesalad" {
		t.Errorf("stations[0].UUID = %q, want %q", stations[0].UUID, "somafm-groovesalad")
	}
	if stations[0].StreamURL != "https://api.somafm.com/groovesalad.pls" {
		t.Errorf("stations[0].StreamURL = %q, want mp3 URL", stations[0].StreamURL)
	}
	if stations[0].Tags != "ambient" {
		t.Errorf("stations[0].Tags = %q, want %q", stations[0].Tags, "ambient")
	}
	if stations[1].StreamURL != "https://api.somafm.com/dronezone.pls" {
		t.Errorf("stations[1].StreamURL = %q, want mp3 URL", stations[1].StreamURL)
	}
}
