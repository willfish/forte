package radio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func loadSharedStationsFixture(t *testing.T) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "testdata", "radio", "stations.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stations fixture: %v", err)
	}
	return data
}

type stationsFixtureFile struct {
	SomaFM struct {
		UUID      string `json:"uuid"`
		Name      string `json:"name"`
		StreamURL string `json:"streamUrl"`
		Homepage  string `json:"homepage"`
		Favicon   string `json:"favicon"`
		Tags      string `json:"tags"`
	} `json:"somafm"`
}

func TestSomaFMStationsFromSharedFixture(t *testing.T) {
	var fx stationsFixtureFile
	if err := json.Unmarshal(loadSharedStationsFixture(t), &fx); err != nil {
		t.Fatalf("unmarshal stations fixture: %v", err)
	}

	client := NewSomaFMClient()
	client.mu.Lock()
	client.channels = []somafmChannel{
		{
			ID:    "missioncontrol",
			Title: fx.SomaFM.Name,
			Genre: fx.SomaFM.Tags,
			Image: fx.SomaFM.Favicon,
			Playlists: []somafmPlaylist{
				{URL: fx.SomaFM.StreamURL, Format: "mp3"},
			},
		},
	}
	client.fetchedAt = time.Now()
	client.mu.Unlock()

	stations, err := client.Stations()
	if err != nil {
		t.Fatalf("Stations: %v", err)
	}
	if len(stations) != 1 {
		t.Fatalf("got %d stations, want 1", len(stations))
	}
	got := stations[0]
	if got.UUID != fx.SomaFM.UUID {
		t.Errorf("UUID = %q, want %q", got.UUID, fx.SomaFM.UUID)
	}
	if got.Homepage != fx.SomaFM.Homepage {
		t.Errorf("Homepage = %q, want %q", got.Homepage, fx.SomaFM.Homepage)
	}
	if got.StreamURL != fx.SomaFM.StreamURL {
		t.Errorf("StreamURL = %q, want %q", got.StreamURL, fx.SomaFM.StreamURL)
	}
}
