// Package listenbrainz provides stateless functions for the ListenBrainz API.
package listenbrainz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "https://api.listenbrainz.org"

// TrackInfo holds metadata for a now-playing or scrobble call.
type TrackInfo struct {
	Artist     string
	Track      string
	Album      string
	DurationMs int
}

// ValidateToken checks a user token against the ListenBrainz API
// and returns the associated username.
func ValidateToken(token string) (string, error) {
	req, err := http.NewRequest("GET", baseURL+"/1/validate-token", nil)
	if err != nil {
		return "", fmt.Errorf("listenbrainz: build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("listenbrainz: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("listenbrainz: read response: %w", err)
	}

	var result struct {
		Valid    bool   `json:"valid"`
		UserName string `json:"user_name"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("listenbrainz: parse response: %w", err)
	}
	if !result.Valid {
		return "", fmt.Errorf("listenbrainz: invalid token: %s", result.Message)
	}
	return result.UserName, nil
}

// NowPlaying sends a "playing now" notification to ListenBrainz.
func NowPlaying(token string, t TrackInfo) error {
	payload := submitPayload{
		ListenType: "playing_now",
		Payload:    []listen{{TrackMetadata: trackMeta(t)}},
	}
	return submit(token, payload)
}

// Scrobble submits a completed track listen to ListenBrainz.
func Scrobble(token string, t TrackInfo, listenedAt int64) error {
	payload := submitPayload{
		ListenType: "single",
		Payload:    []listen{{ListenedAt: listenedAt, TrackMetadata: trackMeta(t)}},
	}
	return submit(token, payload)
}

// ScrobbleBatch submits multiple completed track listens in a single API call.
// ListenBrainz uses listen_type "import" for batch submissions.
func ScrobbleBatch(token string, tracks []TrackInfo, timestamps []int64) error {
	if len(tracks) == 0 {
		return nil
	}
	if len(tracks) != len(timestamps) {
		return fmt.Errorf("listenbrainz: tracks and timestamps length mismatch")
	}
	listens := make([]listen, len(tracks))
	for i, t := range tracks {
		listens[i] = listen{ListenedAt: timestamps[i], TrackMetadata: trackMeta(t)}
	}
	payload := submitPayload{
		ListenType: "import",
		Payload:    listens,
	}
	return submit(token, payload)
}

type submitPayload struct {
	ListenType string   `json:"listen_type"`
	Payload    []listen `json:"payload"`
}

type listen struct {
	ListenedAt    int64         `json:"listened_at,omitempty"`
	TrackMetadata trackMetadata `json:"track_metadata"`
}

type trackMetadata struct {
	ArtistName     string         `json:"artist_name"`
	TrackName      string         `json:"track_name"`
	ReleaseName    string         `json:"release_name,omitempty"`
	AdditionalInfo additionalInfo `json:"additional_info"`
}

type additionalInfo struct {
	DurationMs       int    `json:"duration_ms,omitempty"`
	MediaPlayer      string `json:"media_player"`
	SubmissionClient string `json:"submission_client"`
}

func trackMeta(t TrackInfo) trackMetadata {
	return trackMetadata{
		ArtistName:  t.Artist,
		TrackName:   t.Track,
		ReleaseName: t.Album,
		AdditionalInfo: additionalInfo{
			DurationMs:       t.DurationMs,
			MediaPlayer:      "Forte",
			SubmissionClient: "Forte",
		},
	}
}

func submit(token string, payload submitPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("listenbrainz: marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/1/submit-listens", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("listenbrainz: build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("listenbrainz: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("listenbrainz: API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
