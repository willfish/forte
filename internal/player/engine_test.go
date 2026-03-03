package player

import (
	"testing"
)

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := NewEngine()
	if err != nil {
		t.Fatalf("NewEngine() error: %v", err)
	}
	t.Cleanup(func() { e.Close() })
	return e
}

func TestNewEngine(t *testing.T) {
	e := newTestEngine(t)

	v := e.Version()
	if v == "" {
		t.Fatal("expected non-empty mpv version")
	}
	t.Logf("mpv version: %s", v)
}

func TestInitialState(t *testing.T) {
	e := newTestEngine(t)

	if s := e.State(); s != StateStopped {
		t.Fatalf("expected StateStopped, got %s", s)
	}
}

func TestPlayNonExistentFile(t *testing.T) {
	e := newTestEngine(t)

	// loadfile queues the file asynchronously, so the command itself succeeds
	// even for non-existent files. mpv will emit an end-file event with an error.
	if err := e.Play("/nonexistent/file.flac"); err != nil {
		t.Fatalf("Play() error: %v", err)
	}

	if s := e.State(); s != StatePlaying {
		t.Fatalf("expected StatePlaying after Play(), got %s", s)
	}
}

func TestPauseResume(t *testing.T) {
	e := newTestEngine(t)

	// Pause in stopped state should be a no-op.
	e.Pause()
	if s := e.State(); s != StateStopped {
		t.Fatalf("expected StateStopped after Pause() in stopped state, got %s", s)
	}

	// Play, then pause.
	if err := e.Play("/nonexistent/file.flac"); err != nil {
		t.Fatalf("Play() error: %v", err)
	}
	e.Pause()
	if s := e.State(); s != StatePaused {
		t.Fatalf("expected StatePaused, got %s", s)
	}

	// Resume.
	e.Resume()
	if s := e.State(); s != StatePlaying {
		t.Fatalf("expected StatePlaying after Resume(), got %s", s)
	}

	// Resume when already playing should be a no-op.
	e.Resume()
	if s := e.State(); s != StatePlaying {
		t.Fatalf("expected StatePlaying after redundant Resume(), got %s", s)
	}
}

func TestStopResetsState(t *testing.T) {
	e := newTestEngine(t)

	if err := e.Play("/nonexistent/file.flac"); err != nil {
		t.Fatalf("Play() error: %v", err)
	}
	e.Stop()
	if s := e.State(); s != StateStopped {
		t.Fatalf("expected StateStopped after Stop(), got %s", s)
	}
}

func TestVolume(t *testing.T) {
	e := newTestEngine(t)

	e.SetVolume(50)
	if v := e.Volume(); v != 50 {
		t.Fatalf("expected volume 50, got %d", v)
	}

	// Clamp to bounds.
	e.SetVolume(-10)
	if v := e.Volume(); v != 0 {
		t.Fatalf("expected volume 0 after setting -10, got %d", v)
	}

	e.SetVolume(200)
	if v := e.Volume(); v != 100 {
		t.Fatalf("expected volume 100 after setting 200, got %d", v)
	}
}

func TestSeekWhileStopped(t *testing.T) {
	e := newTestEngine(t)

	// Seek in stopped state should be a no-op (no panic).
	e.Seek(30.0)
}

func TestEnqueue(t *testing.T) {
	e := newTestEngine(t)

	// Enqueue without playing first should succeed (appends to empty playlist).
	if err := e.Enqueue("/nonexistent/a.flac"); err != nil {
		t.Fatalf("Enqueue() error: %v", err)
	}
}

func TestPlayAll(t *testing.T) {
	e := newTestEngine(t)

	paths := []string{"/nonexistent/a.flac", "/nonexistent/b.flac", "/nonexistent/c.flac"}
	if err := e.PlayAll(paths); err != nil {
		t.Fatalf("PlayAll() error: %v", err)
	}

	if s := e.State(); s != StatePlaying {
		t.Fatalf("expected StatePlaying after PlayAll(), got %s", s)
	}
}

func TestPlayAllEmpty(t *testing.T) {
	e := newTestEngine(t)

	if err := e.PlayAll(nil); err != nil {
		t.Fatalf("PlayAll(nil) error: %v", err)
	}

	if s := e.State(); s != StateStopped {
		t.Fatalf("expected StateStopped after PlayAll(nil), got %s", s)
	}
}

func TestGaplessOptionSet(t *testing.T) {
	e := newTestEngine(t)

	v := e.handle.GetPropertyString("gapless-audio")
	if v != "yes" {
		t.Fatalf("expected gapless-audio=yes, got %q", v)
	}
}

func TestReplayGainDefault(t *testing.T) {
	e := newTestEngine(t)

	if v := e.ReplayGain(); v != "track" {
		t.Fatalf("expected default replaygain=track, got %q", v)
	}
}

func TestSetReplayGain(t *testing.T) {
	e := newTestEngine(t)

	for _, mode := range []string{"album", "no", "track"} {
		if err := e.SetReplayGain(mode); err != nil {
			t.Fatalf("SetReplayGain(%q) error: %v", mode, err)
		}
		if v := e.ReplayGain(); v != mode {
			t.Fatalf("expected replaygain=%q, got %q", mode, v)
		}
	}
}

func TestSetReplayGainInvalid(t *testing.T) {
	e := newTestEngine(t)

	if err := e.SetReplayGain("bogus"); err == nil {
		t.Fatal("expected error for invalid replaygain mode")
	}
}

func TestPlaybackStateString(t *testing.T) {
	tests := []struct {
		state PlaybackState
		want  string
	}{
		{StateStopped, "stopped"},
		{StatePlaying, "playing"},
		{StatePaused, "paused"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("PlaybackState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
