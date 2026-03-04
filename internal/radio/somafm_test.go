package radio

import (
	"encoding/json"
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

// TestSomaFMJSONParsing verifies that the actual SomaFM API response
// format deserializes correctly and produces stations with artwork.
func TestSomaFMJSONParsing(t *testing.T) {
	// Realistic JSON matching the live SomaFM channels.json format.
	raw := `{
		"channels": [
			{
				"id": "groovesalad",
				"title": "Groove Salad",
				"description": "A nicely chilled plate of ambient beats.",
				"dj": "DJ Nickel",
				"genre": "ambient|electronica",
				"image": "https://api.somafm.com/img/groovesalad120.png",
				"largeimage": "https://api.somafm.com/logos/256/groovesalad256.png",
				"xlimage": "https://api.somafm.com/logos/512/groovesalad512.png",
				"listeners": "1200",
				"lastPlaying": "Zero 7 - In The Waiting Line",
				"playlists": [
					{"url": "https://api.somafm.com/groovesalad.pls", "format": "mp3", "quality": "highest"},
					{"url": "https://api.somafm.com/groovesalad130.pls", "format": "aac", "quality": "highest"},
					{"url": "https://api.somafm.com/groovesalad64.pls", "format": "aacp", "quality": "high"}
				]
			},
			{
				"id": "dronezone",
				"title": "Drone Zone",
				"description": "Served best chilled.",
				"genre": "ambient|space",
				"image": "https://api.somafm.com/img/dronezone120.png",
				"largeimage": "https://api.somafm.com/logos/256/dronezone256.png",
				"xlimage": "https://api.somafm.com/logos/512/dronezone512.png",
				"listeners": "800",
				"playlists": [
					{"url": "https://api.somafm.com/dronezone.pls", "format": "mp3", "quality": "highest"},
					{"url": "https://api.somafm.com/dronezone130.pls", "format": "aac", "quality": "highest"}
				]
			}
		]
	}`

	var data somafmResponse
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	if len(data.Channels) != 2 {
		t.Fatalf("got %d channels, want 2", len(data.Channels))
	}

	// Verify deserialization picked up the right image field.
	for _, ch := range data.Channels {
		if ch.Image == "" {
			t.Errorf("channel %q: Image is empty (xlimage not parsed)", ch.ID)
		}
		if ch.ID == "groovesalad" && ch.Image != "https://api.somafm.com/logos/512/groovesalad512.png" {
			t.Errorf("channel %q: Image = %q, want xlimage URL", ch.ID, ch.Image)
		}
	}

	// Populate the client with parsed data and verify Stations() output.
	s := NewSomaFMClient()
	s.mu.Lock()
	s.channels = data.Channels
	s.artIndex = make(map[string]string)
	for _, ch := range data.Channels {
		if ch.Image != "" {
			s.artIndex[ch.ID] = ch.Image
		}
	}
	s.fetchedAt = time.Now()
	s.mu.Unlock()

	stations, err := s.Stations()
	if err != nil {
		t.Fatalf("Stations(): %v", err)
	}

	for _, st := range stations {
		if st.Favicon == "" {
			t.Errorf("station %q: Favicon is empty", st.Name)
		}
		if st.StreamURL == "" {
			t.Errorf("station %q: StreamURL is empty", st.Name)
		}
		if st.Homepage == "" {
			t.Errorf("station %q: Homepage is empty", st.Name)
		}
	}

	// Verify artwork lookup works with parsed data.
	art := s.LookupArtwork("https://somafm.com/groovesalad/")
	if art != "https://api.somafm.com/logos/512/groovesalad512.png" {
		t.Errorf("LookupArtwork = %q, want xlimage URL", art)
	}
}
