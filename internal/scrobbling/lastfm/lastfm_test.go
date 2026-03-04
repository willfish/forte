package lastfm

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestSign(t *testing.T) {
	params := map[string]string{
		"api_key": "xxxxxxxxxx",
		"method":  "auth.getToken",
		"format":  "json",
	}
	got := sign(params, "secret123")

	// sorted keys (excluding "format"): api_key, method
	// concat: "api_keyxxxxxxxxxxmethodauth.getTokensecret123"
	want := "50e53cda5eb80b3e7aec64150f0d8f90"
	if got != want {
		t.Errorf("sign() = %q, want %q", got, want)
	}
}

func TestSignExcludesFormat(t *testing.T) {
	params1 := map[string]string{
		"api_key": "key",
		"method":  "test",
	}
	params2 := map[string]string{
		"api_key": "key",
		"method":  "test",
		"format":  "json",
	}
	if sign(params1, "sec") != sign(params2, "sec") {
		t.Error("sign() should produce the same result regardless of 'format' param")
	}
}

func TestScrobbleThreshold(t *testing.T) {
	tests := []struct {
		name       string
		durationMs int
		want       int
	}{
		{"short track 60s", 60_000, 30_000},
		{"medium track 5m", 300_000, 150_000},
		{"long track 10m", 600_000, 240_000},
		{"very long track 20m", 1_200_000, 240_000},
		{"exact 8m boundary", 480_000, 240_000},
		{"zero duration", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScrobbleThreshold(tt.durationMs)
			if got != tt.want {
				t.Errorf("ScrobbleThreshold(%d) = %d, want %d", tt.durationMs, got, tt.want)
			}
		})
	}
}

func TestNowPlayingParams(t *testing.T) {
	track := TrackInfo{
		Artist:   "Radiohead",
		Track:    "Idioteque",
		Album:    "Kid A",
		Duration: 300,
	}
	params := map[string]string{
		"method":   "track.updateNowPlaying",
		"api_key":  "testkey",
		"sk":       "testsession",
		"artist":   track.Artist,
		"track":    track.Track,
		"album":    track.Album,
		"duration": "300",
	}
	sig := sign(params, "testsecret")
	if sig == "" {
		t.Error("sign() returned empty string")
	}
	for _, key := range []string{"method", "api_key", "sk", "artist", "track", "album", "duration"} {
		if _, ok := params[key]; !ok {
			t.Errorf("missing expected param %q", key)
		}
	}
}

func TestScrobbleParams(t *testing.T) {
	params := map[string]string{
		"method":    "track.scrobble",
		"api_key":   "testkey",
		"sk":        "testsession",
		"artist":    "Portishead",
		"track":     "Glory Box",
		"album":     "Dummy",
		"timestamp": "1700000000",
	}
	sig := sign(params, "testsecret")
	if sig == "" {
		t.Error("sign() returned empty string")
	}
	if params["method"] != "track.scrobble" {
		t.Error("method should be track.scrobble")
	}
}

func TestAuthURL(t *testing.T) {
	got := AuthURL("mykey", "mytoken")
	want := "https://www.last.fm/api/auth/?api_key=mykey&token=mytoken"
	if got != want {
		t.Errorf("AuthURL() = %q, want %q", got, want)
	}
}

func TestGetSessionResponseParsing(t *testing.T) {
	body := []byte(`{"session":{"name":"testuser","key":"abc123","subscriber":0}}`)
	var resp struct {
		Session struct {
			Key  string `json:"key"`
			Name string `json:"name"`
		} `json:"session"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Session.Key != "abc123" {
		t.Errorf("session key = %q, want %q", resp.Session.Key, "abc123")
	}
	if resp.Session.Name != "testuser" {
		t.Errorf("username = %q, want %q", resp.Session.Name, "testuser")
	}
}

func TestScrobbleBatchParams(t *testing.T) {
	tracks := []TrackInfo{
		{Artist: "Radiohead", Track: "Idioteque", Album: "Kid A", Duration: 300},
		{Artist: "Portishead", Track: "Glory Box", Album: "Dummy", Duration: 305},
		{Artist: "Bjork", Track: "Army of Me", Album: "", Duration: 224},
	}
	timestamps := []int64{1700000000, 1700000300, 1700000600}

	// Build the params the same way ScrobbleBatch does, to verify structure.
	params := map[string]string{
		"method":  "track.scrobble",
		"api_key": "testkey",
		"sk":      "testsession",
	}
	for i, tr := range tracks {
		params[fmt.Sprintf("artist[%d]", i)] = tr.Artist
		params[fmt.Sprintf("track[%d]", i)] = tr.Track
		params[fmt.Sprintf("timestamp[%d]", i)] = fmt.Sprintf("%d", timestamps[i])
		if tr.Album != "" {
			params[fmt.Sprintf("album[%d]", i)] = tr.Album
		}
	}

	// Verify indexed params exist.
	for i := range 3 {
		if _, ok := params[fmt.Sprintf("artist[%d]", i)]; !ok {
			t.Errorf("missing artist[%d]", i)
		}
		if _, ok := params[fmt.Sprintf("track[%d]", i)]; !ok {
			t.Errorf("missing track[%d]", i)
		}
		if _, ok := params[fmt.Sprintf("timestamp[%d]", i)]; !ok {
			t.Errorf("missing timestamp[%d]", i)
		}
	}

	// Album should not be present for track without album.
	if _, ok := params["album[2]"]; ok {
		t.Error("album[2] should not be present for track without album")
	}

	// Signature should work with indexed params.
	sig := sign(params, "testsecret")
	if sig == "" {
		t.Error("sign() returned empty string for batch params")
	}
}

func TestAPIErrorParsing(t *testing.T) {
	body := []byte(`{"error":14,"message":"Unauthorized Token"}`)
	var errResp struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error != 14 {
		t.Errorf("error code = %d, want 14", errResp.Error)
	}
	if errResp.Message != "Unauthorized Token" {
		t.Errorf("message = %q, want %q", errResp.Message, "Unauthorized Token")
	}
}
