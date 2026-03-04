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

func TestSomaFMArtworkLookup(t *testing.T) {
	s := NewSomaFMArtwork()

	// Pre-populate the cache to avoid hitting the network.
	s.mu.Lock()
	s.channels = map[string]string{
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
			got := s.Lookup(tt.homepage)
			if got != tt.want {
				t.Errorf("Lookup(%q) = %q, want %q", tt.homepage, got, tt.want)
			}
		})
	}
}
