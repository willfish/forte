package player

import (
	"testing"
	"time"

	mpv "github.com/gen2brain/go-mpv"
)

func TestStreamEndIsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		reason         mpv.Reason
		reconnectOnEOF bool
		want           bool
	}{
		{name: "error always reconnects", reason: mpv.EndFileError, reconnectOnEOF: false, want: true},
		{name: "eof reconnects when enabled", reason: mpv.EndFileEOF, reconnectOnEOF: true, want: true},
		{name: "eof is finished track by default", reason: mpv.EndFileEOF, reconnectOnEOF: false, want: false},
		{name: "stop is not a stream error", reason: mpv.EndFileStop, reconnectOnEOF: true, want: false},
		{name: "quit is not a stream error", reason: mpv.EndFileQuit, reconnectOnEOF: true, want: false},
		{name: "redirect is not a stream error", reason: mpv.EndFileRedirect, reconnectOnEOF: true, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StreamEndIsError(tt.reason, tt.reconnectOnEOF); got != tt.want {
				t.Fatalf("StreamEndIsError(%v, %v) = %v, want %v", tt.reason, tt.reconnectOnEOF, got, tt.want)
			}
		})
	}
}

func TestRadioReconnectDelay(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		attempt int
		jitter  float64
		want    time.Duration
	}{
		{name: "first attempt full jitter", attempt: 1, jitter: 1, want: time.Second},
		{name: "first attempt zero jitter", attempt: 1, jitter: 0, want: 0},
		{name: "second attempt", attempt: 2, jitter: 1, want: 2 * time.Second},
		{name: "third attempt", attempt: 3, jitter: 1, want: 4 * time.Second},
		{name: "fourth attempt", attempt: 4, jitter: 1, want: 8 * time.Second},
		{name: "fifth attempt", attempt: 5, jitter: 1, want: 16 * time.Second},
		{name: "caps at 30s", attempt: 6, jitter: 1, want: 30 * time.Second},
		{name: "stays capped", attempt: 20, jitter: 1, want: 30 * time.Second},
		{name: "half jitter", attempt: 3, jitter: 0.5, want: 2 * time.Second},
		{name: "zero attempt treated as first", attempt: 0, jitter: 1, want: time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RadioReconnectDelay(tt.attempt, tt.jitter); got != tt.want {
				t.Fatalf("RadioReconnectDelay(%d, %v) = %v, want %v", tt.attempt, tt.jitter, got, tt.want)
			}
		})
	}
}
