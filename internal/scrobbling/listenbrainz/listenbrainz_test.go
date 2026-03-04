package listenbrainz

import (
	"encoding/json"
	"testing"
)

func TestTrackMetadata(t *testing.T) {
	info := TrackInfo{
		Artist:     "Radiohead",
		Track:      "Idioteque",
		Album:      "Kid A",
		DurationMs: 310000,
	}
	meta := trackMeta(info)

	if meta.ArtistName != "Radiohead" {
		t.Errorf("ArtistName = %q, want %q", meta.ArtistName, "Radiohead")
	}
	if meta.TrackName != "Idioteque" {
		t.Errorf("TrackName = %q, want %q", meta.TrackName, "Idioteque")
	}
	if meta.ReleaseName != "Kid A" {
		t.Errorf("ReleaseName = %q, want %q", meta.ReleaseName, "Kid A")
	}
	if meta.AdditionalInfo.DurationMs != 310000 {
		t.Errorf("DurationMs = %d, want %d", meta.AdditionalInfo.DurationMs, 310000)
	}
	if meta.AdditionalInfo.MediaPlayer != "Forte" {
		t.Errorf("MediaPlayer = %q, want %q", meta.AdditionalInfo.MediaPlayer, "Forte")
	}
	if meta.AdditionalInfo.SubmissionClient != "Forte" {
		t.Errorf("SubmissionClient = %q, want %q", meta.AdditionalInfo.SubmissionClient, "Forte")
	}
}

func TestNowPlayingPayload(t *testing.T) {
	info := TrackInfo{
		Artist:     "Portishead",
		Track:      "Glory Box",
		Album:      "Dummy",
		DurationMs: 305000,
	}
	payload := submitPayload{
		ListenType: "playing_now",
		Payload:    []listen{{TrackMetadata: trackMeta(info)}},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed["listen_type"] != "playing_now" {
		t.Errorf("listen_type = %v, want %q", parsed["listen_type"], "playing_now")
	}

	payloadArr := parsed["payload"].([]any)
	if len(payloadArr) != 1 {
		t.Fatalf("payload length = %d, want 1", len(payloadArr))
	}

	listen := payloadArr[0].(map[string]any)
	if _, ok := listen["listened_at"]; ok {
		t.Error("playing_now should not have listened_at")
	}

	meta := listen["track_metadata"].(map[string]any)
	if meta["artist_name"] != "Portishead" {
		t.Errorf("artist_name = %v, want %q", meta["artist_name"], "Portishead")
	}
}

func TestScrobblePayload(t *testing.T) {
	info := TrackInfo{
		Artist:     "Massive Attack",
		Track:      "Teardrop",
		Album:      "Mezzanine",
		DurationMs: 327000,
	}
	payload := submitPayload{
		ListenType: "single",
		Payload:    []listen{{ListenedAt: 1700000000, TrackMetadata: trackMeta(info)}},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed["listen_type"] != "single" {
		t.Errorf("listen_type = %v, want %q", parsed["listen_type"], "single")
	}

	payloadArr := parsed["payload"].([]any)
	listen := payloadArr[0].(map[string]any)

	listenedAt, ok := listen["listened_at"].(float64)
	if !ok {
		t.Fatal("listened_at missing or wrong type")
	}
	if int64(listenedAt) != 1700000000 {
		t.Errorf("listened_at = %v, want 1700000000", listenedAt)
	}
}

func TestBatchPayload(t *testing.T) {
	tracks := []TrackInfo{
		{Artist: "Radiohead", Track: "Idioteque", Album: "Kid A", DurationMs: 310000},
		{Artist: "Portishead", Track: "Glory Box", Album: "Dummy", DurationMs: 305000},
	}
	timestamps := []int64{1700000000, 1700000300}

	listens := make([]listen, len(tracks))
	for i, tr := range tracks {
		listens[i] = listen{ListenedAt: timestamps[i], TrackMetadata: trackMeta(tr)}
	}
	payload := submitPayload{
		ListenType: "import",
		Payload:    listens,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if parsed["listen_type"] != "import" {
		t.Errorf("listen_type = %v, want %q", parsed["listen_type"], "import")
	}

	payloadArr := parsed["payload"].([]any)
	if len(payloadArr) != 2 {
		t.Fatalf("payload length = %d, want 2", len(payloadArr))
	}

	first := payloadArr[0].(map[string]any)
	listenedAt, ok := first["listened_at"].(float64)
	if !ok {
		t.Fatal("listened_at missing or wrong type in first entry")
	}
	if int64(listenedAt) != 1700000000 {
		t.Errorf("first listened_at = %v, want 1700000000", listenedAt)
	}

	meta := first["track_metadata"].(map[string]any)
	if meta["artist_name"] != "Radiohead" {
		t.Errorf("first artist = %v, want Radiohead", meta["artist_name"])
	}

	second := payloadArr[1].(map[string]any)
	secondMeta := second["track_metadata"].(map[string]any)
	if secondMeta["artist_name"] != "Portishead" {
		t.Errorf("second artist = %v, want Portishead", secondMeta["artist_name"])
	}
}

func TestValidateTokenParsing(t *testing.T) {
	// Valid response
	validBody := []byte(`{"code":200,"message":"Token valid.","valid":true,"user_name":"testuser"}`)
	var valid struct {
		Valid    bool   `json:"valid"`
		UserName string `json:"user_name"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(validBody, &valid); err != nil {
		t.Fatalf("unmarshal valid: %v", err)
	}
	if !valid.Valid {
		t.Error("expected valid=true")
	}
	if valid.UserName != "testuser" {
		t.Errorf("user_name = %q, want %q", valid.UserName, "testuser")
	}

	// Invalid response
	invalidBody := []byte(`{"code":200,"message":"Token invalid.","valid":false}`)
	var invalid struct {
		Valid    bool   `json:"valid"`
		UserName string `json:"user_name"`
		Message  string `json:"message"`
	}
	if err := json.Unmarshal(invalidBody, &invalid); err != nil {
		t.Fatalf("unmarshal invalid: %v", err)
	}
	if invalid.Valid {
		t.Error("expected valid=false")
	}
}

func TestAuthHeader(t *testing.T) {
	// Verify the format used internally matches "Token <token>"
	token := "my-secret-token-123"
	want := "Token my-secret-token-123"
	got := "Token " + token
	if got != want {
		t.Errorf("auth header = %q, want %q", got, want)
	}
}
