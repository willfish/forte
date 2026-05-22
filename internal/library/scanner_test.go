package library

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"
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

func TestScanRealAudioFilesAcrossSupportedFormats(t *testing.T) {
	ffmpeg := requireFFmpeg(t)
	db := openTestDB(t)
	scanner := NewScanner(db)

	dir := t.TempDir()
	formats := []struct {
		filename string
		format   string
		codec    string
	}{
		{"01 Forte FLAC.flac", "FLAC", "flac"},
		{"02 Forte MP3.mp3", "MP3", "libmp3lame"},
		{"03 Forte OGG.ogg", "OGG", "libvorbis"},
		{"04 Forte M4A.m4a", "M4A", "aac"},
		{"05 Forte WAV.wav", "WAV", "pcm_s16le"},
	}

	for i, tt := range formats {
		title := fmt.Sprintf("Real %s Track", tt.format)
		path := filepath.Join(dir, tt.filename)
		generateTaggedAudio(t, ffmpeg, path, tt.codec, title, "Forte Test Artist", "Forte Test Album", i+1)
	}

	if err := scanner.Scan(context.Background(), []string{dir}, nil); err != nil {
		t.Fatalf("Scan() real audio: %v", err)
	}

	rows, err := db.Query(`
		SELECT t.title, ar.name, al.title, t.track_number, t.format, t.duration_ms, t.file_path
		FROM tracks t
		JOIN artists ar ON ar.id = t.artist_id
		LEFT JOIN albums al ON al.id = t.album_id
		ORDER BY t.track_number`)
	if err != nil {
		t.Fatalf("query tracks: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var gotFormats []string
	var count int
	for rows.Next() {
		var title, artist, album, format, path string
		var trackNumber, durationMs int
		if err := rows.Scan(&title, &artist, &album, &trackNumber, &format, &durationMs, &path); err != nil {
			t.Fatalf("scan row: %v", err)
		}
		count++
		gotFormats = append(gotFormats, format)
		if artist != "Forte Test Artist" {
			t.Fatalf("artist = %q", artist)
		}
		if album != "Forte Test Album" {
			t.Fatalf("album = %q", album)
		}
		if title == "" {
			t.Fatal("title should be populated from tags")
		}
		if durationMs <= 0 {
			t.Fatalf("duration_ms = %d for %s", durationMs, path)
		}
		if trackNumber != count {
			t.Fatalf("track_number = %d, want %d", trackNumber, count)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if count != len(formats) {
		t.Fatalf("track count = %d, want %d", count, len(formats))
	}

	sort.Strings(gotFormats)
	wantFormats := []string{"FLAC", "M4A", "MP3", "OGG", "WAV"}
	for i := range wantFormats {
		if gotFormats[i] != wantFormats[i] {
			t.Fatalf("formats = %v, want %v", gotFormats, wantFormats)
		}
	}

	results, err := db.Search("Forte Test Artist", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != len(formats) {
		t.Fatalf("search results = %d, want %d", len(results), len(formats))
	}
}

func TestScanChangedRealAudioFileUpdatesExistingTrack(t *testing.T) {
	ffmpeg := requireFFmpeg(t)
	db := openTestDB(t)
	scanner := NewScanner(db)

	dir := t.TempDir()
	path := filepath.Join(dir, "01 Mutable.flac")
	generateTaggedAudio(t, ffmpeg, path, "flac", "Original Title", "Forte Test Artist", "Mutable Album", 1)

	if err := scanner.Scan(context.Background(), []string{dir}, nil); err != nil {
		t.Fatalf("initial Scan(): %v", err)
	}

	generateTaggedAudio(t, ffmpeg, path, "flac", "Updated Title", "Forte Test Artist", "Mutable Album", 1)
	future := time.Now().Add(2 * time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	if err := scanner.Scan(context.Background(), []string{dir}, nil); err != nil {
		t.Fatalf("rescan: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM tracks WHERE file_path = ?", path).Scan(&count); err != nil {
		t.Fatalf("count tracks: %v", err)
	}
	if count != 1 {
		t.Fatalf("track count for path = %d, want 1", count)
	}

	var title string
	if err := db.QueryRow("SELECT title FROM tracks WHERE file_path = ?", path).Scan(&title); err != nil {
		t.Fatalf("select title: %v", err)
	}
	if title != "Updated Title" {
		t.Fatalf("title = %q, want Updated Title", title)
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

func requireFFmpeg(t *testing.T) string {
	t.Helper()
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg is required for real audio scanner integration tests")
	}
	return ffmpeg
}

func generateTaggedAudio(t *testing.T, ffmpeg, path, codec, title, artist, album string, track int) {
	t.Helper()
	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-y",
		"-f", "lavfi",
		"-i", "sine=frequency=440:duration=0.25:sample_rate=44100",
		"-metadata", "title=" + title,
		"-metadata", "artist=" + artist,
		"-metadata", "album=" + album,
		"-metadata", fmt.Sprintf("track=%d", track),
		"-c:a", codec,
	}
	if codec == "aac" {
		args = append(args, "-b:a", "96k")
	}
	args = append(args, path)
	if out, err := exec.Command(ffmpeg, args...).CombinedOutput(); err != nil {
		t.Fatalf("generate audio %s: %v\n%s", path, err, out)
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
