package library

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewWatcher(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	w, err := NewWatcher(scanner)
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}
}

func TestWatcherPauseResume(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	w, err := NewWatcher(scanner)
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	if w.isPaused() {
		t.Error("watcher should not be paused initially")
	}

	w.Pause()
	if !w.isPaused() {
		t.Error("watcher should be paused after Pause()")
	}

	w.Resume()
	if w.isPaused() {
		t.Error("watcher should not be paused after Resume()")
	}
}

func TestWatcherCancellation(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	w, err := NewWatcher(scanner)
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Watch(ctx, []string{dir})
	}()

	// Give the watcher a moment to start, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Errorf("Watch() error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Watch() did not return after context cancellation")
	}
}

func TestWatcherDetectsRemove(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	// Pre-populate the database with a track.
	mustExec(t, db, "INSERT INTO artists (name) VALUES ('A')")
	mustExec(t, db, `INSERT INTO tracks (artist_id, title, file_path) VALUES (1, 'T', '/tmp/test-track.flac')`)

	w, err := NewWatcher(scanner)
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	// Simulate removal by calling handleRemove directly.
	w.handleRemove("/tmp/test-track.flac")

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM tracks WHERE file_path = '/tmp/test-track.flac'").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Errorf("track count = %d after remove, want 0", count)
	}
}

func TestWatcherDetectsNewFile(t *testing.T) {
	db := openTestDB(t)
	scanner := NewScanner(db)

	w, err := NewWatcher(scanner)
	if err != nil {
		t.Fatalf("NewWatcher() error: %v", err)
	}

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = w.Close()
	})

	done := make(chan error, 1)
	go func() {
		done <- w.Watch(ctx, []string{dir})
	}()

	// Wait for watcher to start.
	time.Sleep(100 * time.Millisecond)

	// Create a new audio file (dummy content, tag reading will fail but that's ok).
	if err := os.WriteFile(filepath.Join(dir, "new-track.flac"), []byte("dummy"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait for debounce + processing.
	time.Sleep(300 * time.Millisecond)

	cancel()
	<-done
}
