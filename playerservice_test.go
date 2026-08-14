package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/player"
)

func TestPauseKeepsRadioMode(t *testing.T) {
	p := &PlayerService{}
	p.radioMode = true
	p.radioName = "Test Radio"
	p.radioStreamURL = "https://example.com/stream"

	p.Pause()

	if !p.radioMode {
		t.Fatal("Pause() should keep radio mode active")
	}
	if p.radioName != "Test Radio" || p.radioStreamURL != "https://example.com/stream" {
		t.Fatalf("radio metadata should be preserved: name=%q stream=%q", p.radioName, p.radioStreamURL)
	}
}

func TestStopStopsRadioMode(t *testing.T) {
	p := &PlayerService{}
	p.radioMode = true
	p.radioName = "Test Radio"
	p.radioStreamURL = "https://example.com/stream"

	p.Stop()

	if p.radioMode {
		t.Fatal("Stop() should stop radio mode")
	}
	if p.radioName != "" || p.radioStreamURL != "" {
		t.Fatalf("radio metadata was not cleared: name=%q stream=%q", p.radioName, p.radioStreamURL)
	}
}

func TestStopRadioCancelsReconnectGeneration(t *testing.T) {
	p := &PlayerService{
		radioMode:             true,
		radioStationUUID:      "st-1",
		radioReconnectGen:     4,
		radioReconnectPending: true,
	}

	p.StopRadio()

	if p.radioReconnectStillCurrent("st-1", 4) {
		t.Fatal("old reconnect loop should no longer be current after StopRadio")
	}
	if p.radioReconnectPending {
		t.Fatal("reconnect should not stay pending after StopRadio")
	}
}

func TestRadioReconnectStillCurrent(t *testing.T) {
	p := &PlayerService{radioMode: true, radioStationUUID: "st-1", radioReconnectGen: 3}

	if !p.radioReconnectStillCurrent("st-1", 3) {
		t.Fatal("same station and generation should still be current")
	}
	if p.radioReconnectStillCurrent("st-2", 3) {
		t.Fatal("a different station should not be current")
	}
	if p.radioReconnectStillCurrent("st-1", 4) {
		t.Fatal("a stale reconnect generation should not be current")
	}

	p.radioMode = false
	if p.radioReconnectStillCurrent("st-1", 3) {
		t.Fatal("stopped radio should not stay current")
	}
}

func TestShouldRestartRadio(t *testing.T) {
	if !shouldRestartRadio(true, "http://example.com/stream", player.StateStopped) {
		t.Fatal("stopped radio with a stream URL should restart")
	}
	if shouldRestartRadio(true, "http://example.com/stream", player.StatePlaying) {
		t.Fatal("playing radio should not restart")
	}
	if shouldRestartRadio(true, "http://example.com/stream", player.StatePaused) {
		t.Fatal("paused radio should resume, not restart")
	}
	if shouldRestartRadio(false, "http://example.com/stream", player.StateStopped) {
		t.Fatal("library playback should not restart as radio")
	}
	if shouldRestartRadio(true, "", player.StateStopped) {
		t.Fatal("radio without a stream URL should not restart")
	}
}

func TestPlayerServiceShutdownIsIdempotent(t *testing.T) {
	p := &PlayerService{}

	if err := p.ServiceShutdown(); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := p.ServiceShutdown(); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
}

func TestCleanRadioMediaTitleFiltersStreamFallbacks(t *testing.T) {
	tests := []struct {
		name      string
		title     string
		streamURL string
		want      string
	}{
		{
			name:      "full stream URL",
			title:     "https://example.com/live/radio.m3u8",
			streamURL: "https://example.com/live/radio.m3u8",
			want:      "",
		},
		{
			name:      "stream filename",
			title:     "radio.pls",
			streamURL: "https://example.com/live/radio.pls",
			want:      "",
		},
		{
			name:      "HLS playlist filename",
			title:     "bbc_radio_three-audio=320000.norewind.m3u8",
			streamURL: "https://example.com/live/bbc_radio_three",
			want:      "",
		},
		{
			name:      "real ICY title",
			title:     "Artist - Track",
			streamURL: "https://example.com/live/radio.m3u8",
			want:      "Artist - Track",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanRadioMediaTitle(tt.title, tt.streamURL); got != tt.want {
				t.Fatalf("cleanRadioMediaTitle(%q, %q) = %q, want %q", tt.title, tt.streamURL, got, tt.want)
			}
		})
	}
}

type stubEngine struct{}

func (stubEngine) Play(string) error           { return nil }
func (stubEngine) Enqueue(string) error        { return nil }
func (stubEngine) PlayAll([]string) error      { return nil }
func (stubEngine) Pause()                      {}
func (stubEngine) Resume()                     {}
func (stubEngine) Stop()                       {}
func (stubEngine) Seek(float64)                {}
func (stubEngine) SetVolume(int)               {}
func (stubEngine) Volume() int                 { return 80 }
func (stubEngine) Position() float64           { return 0 }
func (stubEngine) Duration() float64           { return 0 }
func (stubEngine) State() player.PlaybackState { return player.StateStopped }
func (stubEngine) MediaTitle() string          { return "" }
func (stubEngine) MediaArtist() string         { return "" }
func (stubEngine) MediaAlbum() string          { return "" }
func (stubEngine) MediaPath() string           { return "https://example.invalid/stream?api_key=TEST" }
func (stubEngine) Next()                       {}
func (stubEngine) Previous()                   {}
func (stubEngine) SetLoopFile(bool)            {}
func (stubEngine) ReplaceUpcoming([]string)    {}
func (stubEngine) SetOnTrackChange(func())     {}
func (stubEngine) SetOnPlaylistEnd(func())     {}

func testPlayerDB(t *testing.T) *library.DB {
	t.Helper()
	db, err := library.OpenDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSaveStateKeepsLibraryQueueDuringRadio(t *testing.T) {
	db := testPlayerDB(t)
	tracks := []player.QueueTrack{
		{TrackID: 1, Title: "Airbag", FilePath: "/music/airbag.flac"},
		{TrackID: 2, Title: "Karma Police", FilePath: "/music/karma.flac"},
	}
	rt := player.NewRuntime(stubEngine{}, player.RuntimeOptions{})
	if err := rt.PlayQueue(tracks, 0); err != nil {
		t.Fatal(err)
	}

	p := &PlayerService{db: db, runtime: rt}
	p.radioMode = true
	p.savedQueue = tracks
	p.savedPosition = 1
	rt.QueueClear()

	p.saveState()

	got, err := db.LoadPlaybackState()
	if err != nil {
		t.Fatal(err)
	}
	var saved []player.QueueTrack
	if err := json.Unmarshal([]byte(got.QueueJSON), &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved) != 2 || saved[0].Title != "Airbag" {
		t.Fatalf("saved queue = %#v, want library tracks", saved)
	}
	if got.Position != 1 {
		t.Fatalf("saved position = %d, want 1", got.Position)
	}
}

func TestPlayQueueLeavesRadioWithoutRestoringOldQueue(t *testing.T) {
	rt := player.NewRuntime(stubEngine{}, player.RuntimeOptions{})
	p := &PlayerService{runtime: rt}
	p.radioMode = true
	p.savedQueue = []player.QueueTrack{{Title: "Old", FilePath: "/old.flac"}}
	p.radioName = "Jazz FM"

	album := []player.QueueTrack{{TrackID: 3, Title: "Airbag", FilePath: "/music/airbag.flac"}}
	if err := p.PlayQueue(album, 0); err != nil {
		t.Fatal(err)
	}
	if p.IsRadioMode() {
		t.Fatal("PlayQueue should leave radio mode")
	}
	got := p.GetQueue()
	if len(got) != 1 || got[0].Title != "Airbag" {
		t.Fatalf("queue = %#v, want album tracks", got)
	}
}

func TestGetPlaybackStatusOmitsStreamCredentials(t *testing.T) {
	rt := player.NewRuntime(stubEngine{}, player.RuntimeOptions{})
	tracks := []player.QueueTrack{{
		TrackID:  1,
		Title:    "Airbag",
		FilePath: "server://srv/remote-1",
	}}
	if err := rt.PlayQueue(tracks, 0); err != nil {
		t.Fatal(err)
	}
	p := &PlayerService{runtime: rt}
	status := p.GetPlaybackStatus()
	if status.MediaPath != "server://srv/remote-1" {
		t.Fatalf("MediaPath = %q, want logical server path", status.MediaPath)
	}
	if strings.Contains(status.MediaPath, "api_key") || strings.Contains(status.MediaPath, "t=") {
		t.Fatalf("MediaPath leaked credentials: %q", status.MediaPath)
	}
}
