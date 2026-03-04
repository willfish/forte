package artistinfo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// MBInfo holds metadata fetched from MusicBrainz.
type MBInfo struct {
	ID             string
	Disambiguation string
	Type           string
	Area           string
	Begin          string
	End            string
	Tags           []string
}

// FetchMusicBrainz searches MusicBrainz for the named artist and returns metadata.
func FetchMusicBrainz(artistName string) (*MBInfo, error) {
	u := "https://musicbrainz.org/ws/2/artist/?" + url.Values{
		"query": {"artist:" + artistName},
		"fmt":   {"json"},
		"limit": {"1"},
	}.Encode()

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz request: %w", err)
	}
	req.Header.Set("User-Agent", "Forte/1.0 (github.com/willfish/forte)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("musicbrainz read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("musicbrainz status %d: %s", resp.StatusCode, body)
	}

	return parseMusicBrainzResponse(body)
}

// mbResponse maps the relevant parts of the MusicBrainz artist search JSON.
type mbResponse struct {
	Artists []struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Disambiguation string `json:"disambiguation"`
		Type           string `json:"type"`
		Area           struct {
			Name string `json:"name"`
		} `json:"area"`
		LifeSpan struct {
			Begin string `json:"begin"`
			End   string `json:"end"`
		} `json:"life-span"`
		Tags []struct {
			Name string `json:"name"`
		} `json:"tags"`
	} `json:"artists"`
}

func parseMusicBrainzResponse(body []byte) (*MBInfo, error) {
	var resp mbResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("musicbrainz parse: %w", err)
	}
	if len(resp.Artists) == 0 {
		return nil, nil
	}

	a := resp.Artists[0]
	info := &MBInfo{
		ID:             a.ID,
		Disambiguation: a.Disambiguation,
		Type:           a.Type,
		Area:           a.Area.Name,
		Begin:          a.LifeSpan.Begin,
		End:            a.LifeSpan.End,
	}
	for _, tag := range a.Tags {
		if tag.Name != "" {
			info.Tags = append(info.Tags, tag.Name)
		}
	}
	return info, nil
}
