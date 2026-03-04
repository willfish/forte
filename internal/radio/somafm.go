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

type somafmPlaylist struct {
	URL    string `json:"url"`
	Format string `json:"format"`
}

type somafmChannel struct {
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	Genre     string           `json:"genre"`
	Image     string           `json:"xlimage"`
	Playlists []somafmPlaylist `json:"playlists"`
}

type somafmResponse struct {
	Channels []somafmChannel `json:"channels"`
}

// SomaFMClient fetches and caches SomaFM channel data.
// It provides artwork fallback for RadioBrowser stations and
// a curated channel listing for source filtering.
type SomaFMClient struct {
	mu        sync.Mutex
	channels  []somafmChannel
	artIndex  map[string]string // channel id -> xlimage URL
	fetchedAt time.Time
}

// NewSomaFMClient creates a new SomaFM client.
func NewSomaFMClient() *SomaFMClient {
	return &SomaFMClient{}
}

const somafmCacheTTL = 24 * time.Hour

// LookupArtwork returns the artwork URL for a SomaFM station identified
// by its homepage URL. Returns empty string if not applicable.
func (s *SomaFMClient) LookupArtwork(homepage string) string {
	if homepage == "" {
		return ""
	}
	id := extractSomaFMID(homepage)
	if id == "" {
		return ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureFetched(); err != nil {
		return ""
	}

	return s.artIndex[id]
}

// Stations returns all SomaFM channels as Station values.
func (s *SomaFMClient) Stations() ([]Station, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureFetched(); err != nil {
		return nil, err
	}

	stations := make([]Station, 0, len(s.channels))
	for _, ch := range s.channels {
		streamURL := ""
		for _, pl := range ch.Playlists {
			if pl.Format == "mp3" {
				streamURL = pl.URL
				break
			}
		}
		if streamURL == "" {
			continue
		}
		stations = append(stations, Station{
			UUID:      "somafm-" + ch.ID,
			Name:      ch.Title,
			Homepage:  "https://somafm.com/" + ch.ID + "/",
			StreamURL: streamURL,
			Favicon:   ch.Image,
			Tags:      ch.Genre,
		})
	}
	return stations, nil
}

func (s *SomaFMClient) ensureFetched() error {
	if s.channels != nil && time.Since(s.fetchedAt) <= somafmCacheTTL {
		return nil
	}
	return s.fetch()
}

func (s *SomaFMClient) fetch() error {
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

	artIndex := make(map[string]string, len(data.Channels))
	for _, ch := range data.Channels {
		if ch.Image != "" {
			artIndex[ch.ID] = ch.Image
		}
	}

	s.channels = data.Channels
	s.artIndex = artIndex
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
