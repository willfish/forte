package metadata

import (
	"testing"
)

func TestParseFilename(t *testing.T) {
	tests := []struct {
		path      string
		wantTitle string
		wantTrack int
	}{
		{"/music/01 - Track Name.flac", "Track Name", 1},
		{"/music/12 - Song Title.mp3", "Song Title", 12},
		{"/music/01. First Song.flac", "First Song", 1},
		{"/music/03 Third Track.ogg", "Third Track", 3},
		{"/music/Song Without Number.flac", "Song Without Number", 0},
		{"/music/01 - Track - With Dash.flac", "Track - With Dash", 1},
		{"/music/.flac", "", 0},
	}
	for _, tt := range tests {
		title, track := parseFilename(tt.path)
		if title != tt.wantTitle || track != tt.wantTrack {
			t.Errorf("parseFilename(%q) = (%q, %d), want (%q, %d)", tt.path, title, track, tt.wantTitle, tt.wantTrack)
		}
	}
}

func TestParseNum(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"1", 1},
		{"12", 12},
		{"3/12", 3},
		{"01/10", 1},
		{"abc", 0},
	}
	for _, tt := range tests {
		if got := parseNum(tt.input); got != tt.want {
			t.Errorf("parseNum(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestFirst(t *testing.T) {
	if got := first(nil); got != "" {
		t.Errorf("first(nil) = %q, want empty", got)
	}
	if got := first([]string{}); got != "" {
		t.Errorf("first([]) = %q, want empty", got)
	}
	if got := first([]string{"a", "b"}); got != "a" {
		t.Errorf("first([a,b]) = %q, want a", got)
	}
}

func TestParentDirName(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/music/Artist/Album/01.flac", "Album"},
		{"/music/track.mp3", "music"},
		{"track.mp3", "."},
	}
	for _, tt := range tests {
		if got := parentDirName(tt.path); got != tt.want {
			t.Errorf("parentDirName(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestReadTagsNonExistentFile(t *testing.T) {
	_, err := ReadTags("/nonexistent/file.flac")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestReadArtworkNonExistentFile(t *testing.T) {
	_, _, err := ReadArtwork("/nonexistent/file.flac")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}
