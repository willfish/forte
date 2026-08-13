package main

import (
	"testing"

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
