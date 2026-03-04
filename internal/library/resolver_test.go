package library

import "testing"

func TestIsServerPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"server://srv-1/track-42", true},
		{"server://", true},
		{"/home/user/music/song.flac", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsServerPath(tt.path); got != tt.want {
			t.Errorf("IsServerPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestParseServerPath(t *testing.T) {
	serverID, remoteID, err := ParseServerPath("server://srv-1/track-42")
	if err != nil {
		t.Fatalf("ParseServerPath: %v", err)
	}
	if serverID != "srv-1" {
		t.Errorf("serverID = %q, want 'srv-1'", serverID)
	}
	if remoteID != "track-42" {
		t.Errorf("remoteID = %q, want 'track-42'", remoteID)
	}
}

func TestParseServerPathInvalid(t *testing.T) {
	_, _, err := ParseServerPath("server://no-slash")
	if err == nil {
		t.Error("expected error for path without second slash")
	}
}

func TestParseServerPathNestedRemoteID(t *testing.T) {
	serverID, remoteID, err := ParseServerPath("server://srv/a/b/c")
	if err != nil {
		t.Fatalf("ParseServerPath: %v", err)
	}
	if serverID != "srv" {
		t.Errorf("serverID = %q", serverID)
	}
	if remoteID != "a/b/c" {
		t.Errorf("remoteID = %q, want 'a/b/c'", remoteID)
	}
}

func TestResolveLocalPath(t *testing.T) {
	db := openTestDB(t)
	r := NewPathResolver(db)

	path, err := r.Resolve("/home/user/music/song.flac")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if path != "/home/user/music/song.flac" {
		t.Errorf("got %q, want unchanged local path", path)
	}
}
