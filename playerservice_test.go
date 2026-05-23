package main

import "testing"

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
