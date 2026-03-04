package radio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type somafmChannel struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Image   string `json:"xlimage"`
}

type somafmResponse struct {
	Channels []somafmChannel `json:"channels"`
}

// SomaFMArtwork fetches and caches SomaFM channel artwork.
// It provides a fallback for stations whose favicon is missing
// in the RadioBrowser API.
type SomaFMArtwork struct {
	mu        sync.Mutex
	channels  map[string]string // channel id -> xlimage URL
	fetchedAt time.Time
}

// NewSomaFMArtwork creates a new SomaFM artwork cache.
func NewSomaFMArtwork() *SomaFMArtwork {
	return &SomaFMArtwork{}
}

const somafmCacheTTL = 24 * time.Hour

// Lookup returns the artwork URL for a station if it's a SomaFM station
// with a missing favicon. Returns empty string if not applicable.
func (s *SomaFMArtwork) Lookup(homepage string) string {
	if homepage == "" {
		return ""
	}
	id := extractSomaFMID(homepage)
	if id == "" {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.channels == nil || time.Since(s.fetchedAt) > somafmCacheTTL {
		if err := s.fetch(); err != nil {
			return ""
		}
	}

	return s.channels[id]
}

func (s *SomaFMArtwork) fetch() error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://somafm.com/channels.json")
	if err != nil {
		return fmt.Errorf("somafm fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("somafm status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("somafm read: %w", err)
	}

	var data somafmResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return fmt.Errorf("somafm parse: %w", err)
	}

	channels := make(map[string]string, len(data.Channels))
	for _, ch := range data.Channels {
		if ch.Image != "" {
			channels[ch.ID] = ch.Image
		}
	}

	s.channels = channels
	s.fetchedAt = time.Now()
	return nil
}

// extractSomaFMID extracts the channel ID from a SomaFM homepage URL.
// e.g. "https://somafm.com/groovesalad/" -> "groovesalad"
//
//	"https://somafm.com/player/#/now-playing/dronezone" -> "dronezone"
func extractSomaFMID(homepage string) string {
	if !strings.Contains(homepage, "somafm.com") {
		return ""
	}

	// Strip protocol and host.
	path := homepage
	if idx := strings.Index(path, "somafm.com/"); idx >= 0 {
		path = path[idx+len("somafm.com/"):]
	} else {
		return ""
	}

	// Remove trailing slash and fragments.
	path = strings.TrimRight(path, "/")
	if idx := strings.Index(path, "#"); idx >= 0 {
		path = path[idx+1:]
	}

	// Handle player URLs like "player/#/now-playing/dronezone".
	if strings.Contains(path, "now-playing/") {
		parts := strings.Split(path, "now-playing/")
		if len(parts) == 2 {
			return strings.TrimRight(parts[1], "/")
		}
	}

	// Handle direct URLs like "groovesalad" or "insound".
	// Skip paths that contain other segments (e.g. "img3/something").
	if !strings.Contains(path, "/") && path != "" && path != "player" {
		return path
	}

	return ""
}
