package player

import (
	"testing"
)

func TestNewEngine(t *testing.T) {
	e, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error: %v", err)
	}
	defer e.Close()

	v := e.Version()
	if v == "" {
		t.Fatal("expected non-empty mpv version")
	}
	t.Logf("mpv version: %s", v)
}

func TestPlayNonExistentFile(t *testing.T) {
	e, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error: %v", err)
	}
	defer e.Close()

	// loadfile queues the file asynchronously, so the command itself succeeds
	// even for non-existent files. mpv will emit an end-file event with an error.
	if err := e.Play("/nonexistent/file.flac"); err != nil {
		t.Fatalf("Play() error: %v", err)
	}
}
