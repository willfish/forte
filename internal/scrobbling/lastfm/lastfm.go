// Package lastfm provides stateless functions for the Last.fm Scrobbling API v2.
package lastfm

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const apiURL = "https://ws.audioscrobbler.com/2.0/"

// TrackInfo holds metadata for a now-playing or scrobble call.
type TrackInfo struct {
	Artist   string
	Track    string
	Album    string
	Duration int // seconds
}

// GetToken requests an unauthorized request token from Last.fm.
func GetToken(apiKey, apiSecret string) (string, error) {
	params := map[string]string{
		"method":  "auth.getToken",
		"api_key": apiKey,
	}
	body, err := apiCall(params, apiSecret)
	if err != nil {
		return "", err
	}
	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("lastfm: parse token response: %w", err)
	}
	if resp.Token == "" {
		return "", fmt.Errorf("lastfm: empty token in response")
	}
	return resp.Token, nil
}

// AuthURL returns the Last.fm URL the user should visit to authorize the token.
func AuthURL(apiKey, token string) string {
	return fmt.Sprintf("https://www.last.fm/api/auth/?api_key=%s&token=%s", url.QueryEscape(apiKey), url.QueryEscape(token))
}

// GetSession exchanges an authorized token for a session key.
func GetSession(apiKey, apiSecret, token string) (sessionKey, username string, err error) {
	params := map[string]string{
		"method":  "auth.getSession",
		"api_key": apiKey,
		"token":   token,
	}
	body, err := apiCall(params, apiSecret)
	if err != nil {
		return "", "", err
	}
	var resp struct {
		Session struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"session"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", "", fmt.Errorf("lastfm: parse session response: %w", err)
	}
	if resp.Session.Key == "" {
		return "", "", fmt.Errorf("lastfm: empty session key in response")
	}
	return resp.Session.Key, resp.Session.Name, nil
}

// NowPlaying sends a "now playing" notification to Last.fm.
func NowPlaying(apiKey, apiSecret, sessionKey string, t TrackInfo) error {
	params := map[string]string{
		"method":  "track.updateNowPlaying",
		"api_key": apiKey,
		"sk":      sessionKey,
		"artist":  t.Artist,
		"track":   t.Track,
	}
	if t.Album != "" {
		params["album"] = t.Album
	}
	if t.Duration > 0 {
		params["duration"] = fmt.Sprintf("%d", t.Duration)
	}
	_, err := apiCall(params, apiSecret)
	return err
}

// Scrobble submits a completed track listen to Last.fm.
func Scrobble(apiKey, apiSecret, sessionKey string, t TrackInfo, timestamp int64) error {
	params := map[string]string{
		"method":    "track.scrobble",
		"api_key":   apiKey,
		"sk":        sessionKey,
		"artist":    t.Artist,
		"track":     t.Track,
		"timestamp": fmt.Sprintf("%d", timestamp),
	}
	if t.Album != "" {
		params["album"] = t.Album
	}
	_, err := apiCall(params, apiSecret)
	return err
}

// ScrobbleThreshold returns the duration in milliseconds after which a track
// should be scrobbled: 50% of duration or 4 minutes, whichever is smaller.
func ScrobbleThreshold(durationMs int) int {
	half := durationMs / 2
	fourMin := 240_000
	if half < fourMin {
		return half
	}
	return fourMin
}

// sign computes the Last.fm API signature: sort params by key, concat
// key+value pairs, append secret, MD5 the result.
func sign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if k == "format" {
			continue // format is excluded from signature
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf strings.Builder
	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteString(params[k])
	}
	buf.WriteString(secret)

	return fmt.Sprintf("%x", md5.Sum([]byte(buf.String())))
}

// apiCall signs and POSTs to the Last.fm API, returning the JSON response body.
func apiCall(params map[string]string, secret string) ([]byte, error) {
	params["format"] = "json"
	params["api_sig"] = sign(params, secret)

	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.PostForm(apiURL, form)
	if err != nil {
		return nil, fmt.Errorf("lastfm: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("lastfm: read response: %w", err)
	}

	// Check for API-level errors.
	var errResp struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &errResp) == nil && errResp.Error != 0 {
		return nil, fmt.Errorf("lastfm: API error %d: %s", errResp.Error, errResp.Message)
	}

	return body, nil
}
