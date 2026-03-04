package artistinfo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ArtistInfo holds metadata fetched from Last.fm.
type ArtistInfo struct {
	Bio      string
	ImageURL string
	Similar  []SimilarArtist
}

// SimilarArtist is a name returned from the similar artists list.
type SimilarArtist struct {
	Name string
}

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// FetchLastFm fetches artist info from the Last.fm API.
func FetchLastFm(apiKey, artistName string) (*ArtistInfo, error) {
	u := "https://ws.audioscrobbler.com/2.0/?" + url.Values{
		"method":  {"artist.getinfo"},
		"artist":  {artistName},
		"api_key": {apiKey},
		"format":  {"json"},
	}.Encode()

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("lastfm request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lastfm read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lastfm status %d: %s", resp.StatusCode, body)
	}

	return parseLastFmResponse(body)
}

// lastFmResponse maps the relevant parts of the Last.fm artist.getinfo JSON.
type lastFmResponse struct {
	Artist struct {
		Bio struct {
			Summary string `json:"summary"`
		} `json:"bio"`
		Image []struct {
			Text string `json:"#text"`
			Size string `json:"size"`
		} `json:"image"`
		Similar struct {
			Artist []struct {
				Name string `json:"name"`
			} `json:"artist"`
		} `json:"similar"`
	} `json:"artist"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

func parseLastFmResponse(body []byte) (*ArtistInfo, error) {
	var resp lastFmResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("lastfm parse: %w", err)
	}
	if resp.Error != 0 {
		return nil, fmt.Errorf("lastfm api error %d: %s", resp.Error, resp.Message)
	}

	info := &ArtistInfo{}

	// Strip HTML from bio summary.
	bio := htmlTagRe.ReplaceAllString(resp.Artist.Bio.Summary, "")
	info.Bio = strings.TrimSpace(bio)

	// Pick the largest available image.
	for _, img := range resp.Artist.Image {
		if img.Text != "" {
			info.ImageURL = img.Text
		}
	}

	for _, s := range resp.Artist.Similar.Artist {
		if s.Name != "" {
			info.Similar = append(info.Similar, SimilarArtist{Name: s.Name})
		}
	}

	return info, nil
}
