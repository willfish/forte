package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIsAudioFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/music/song.flac", true},
		{"/music/song.FLAC", true},
		{"/music/song.mp3", true},
		{"/music/song.opus", true},
		{"/music/song.ogg", true},
		{"/music/song.m4a", true},
		{"/music/song.aac", true},
		{"/music/song.wav", true},
		{"/music/song.wv", true},
		{"/music/song.mpc", true},
		{"/music/song.ape", true},
		{"/music/cover.jpg", false},
		{"/music/notes.txt", false},
		{"/music/data.bin", false},
		{"/music/noext", false},
	}
	for _, tt := range tests {
		if got := isAudioFile(tt.path); got != tt.want {
			t.Errorf("isAudioFile(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestFormatFromExt(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"song.flac", "FLAC"},
		{"song.mp3", "MP3"},
		{"song.opus", "Opus"},
		{"song.ogg", "OGG"},
		{"song.m4a", "M4A"},
		{"song.wav", "WAV"},
		{"song.wv", "WavPack"},
	}
	for _, tt := range tests {
		if got := formatFromExt(tt.path); got != tt.want {
			t.Errorf("formatFromExt(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestScanEmptyDir(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	dir := t.TempDir()
	err := scanner.Scan(context.Background(), []string{dir}, nil)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&count); err != nil {
		t.Fatalf("count tracks: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 tracks, got %d", count)
	}
}

func TestScanSkipsNonAudioFiles(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	dir := t.TempDir()
	// Create non-audio files.
	for _, name := range []string{"cover.jpg", "notes.txt", "README.md"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("test"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	err := scanner.Scan(context.Background(), []string{dir}, nil)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&count); err != nil {
		t.Fatalf("count tracks: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 tracks after scanning non-audio files, got %d", count)
	}
}

func TestScanCancellation(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	dir := t.TempDir()
	// Create some dummy audio files (will fail tag reading but that's ok).
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, "track"+string(rune('0'+i))+".flac")
		if err := os.WriteFile(name, []byte("not a real flac"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := scanner.Scan(ctx, []string{dir}, nil)
	if err == nil {
		t.Error("expected context cancellation error")
	}
}

func TestScanProgress(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	dir := t.TempDir()
	// Create dummy audio files.
	for i := 0; i < 3; i++ {
		name := filepath.Join(dir, "track"+string(rune('0'+i))+".flac")
		if err := os.WriteFile(name, []byte("not a real flac"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	progress := make(chan Progress, 10)
	// Scan will find 3 .flac files but fail to read tags (dummy content).
	// That's fine - progress should still report the total.
	_ = scanner.Scan(context.Background(), []string{dir}, progress)
	close(progress)

	var lastProgress Progress
	for p := range progress {
		lastProgress = p
	}
	if lastProgress.Total != 3 {
		t.Errorf("progress.Total = %d, want 3", lastProgress.Total)
	}
}

func TestScanNonExistentDir(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	err := scanner.Scan(context.Background(), []string{"/nonexistent/path"}, nil)
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}
