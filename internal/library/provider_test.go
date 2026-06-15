package library

import (
	"strings"
	"testing"
)

func TestNewServerProviderCreatesKnownProviders(t *testing.T) {
	tests := []struct {
		name       string
		serverType string
		streamURL  string
	}{
		{
			name:       "subsonic",
			serverType: "subsonic",
			streamURL:  "https://music.example/rest/stream.view?",
		},
		{
			name:       "jellyfin",
			serverType: "jellyfin",
			streamURL:  "https://music.example/Audio/track-1/stream?static=true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewServerProvider(Server{
				Type:     tt.serverType,
				URL:      "https://music.example",
				Username: "user",
				Password: "pass",
			})
			if err != nil {
				t.Fatalf("NewServerProvider: %v", err)
			}
			if got := provider.StreamURL("track-1"); !strings.HasPrefix(got, tt.streamURL) {
				t.Fatalf("StreamURL() = %q, want prefix %q", got, tt.streamURL)
			}
		})
	}
}

func TestNewServerProviderRejectsUnknownProvider(t *testing.T) {
	_, err := NewServerProvider(Server{Type: "plex"})
	if err == nil {
		t.Fatal("NewServerProvider() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "unknown server type: plex") {
		t.Fatalf("NewServerProvider() error = %q", err)
	}
}
