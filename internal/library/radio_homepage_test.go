package library

import "testing"

func TestDeriveHomepageFromStreamURL(t *testing.T) {
	tests := []struct {
		stream string
		want   string
	}{
		{"https://stream.example.com/jazz-fm", "https://stream.example.com/"},
		{"http://radio.host/path/live.mp3", "https://radio.host/"},
		{"", ""},
		{"not-a-url", ""},
		{"http://127.0.0.1/stream", ""},
		{"http://localhost/stream", ""},
	}

	for _, tt := range tests {
		got := DeriveHomepageFromStreamURL(tt.stream)
		if got != tt.want {
			t.Errorf("DeriveHomepageFromStreamURL(%q) = %q, want %q", tt.stream, got, tt.want)
		}
	}
}
