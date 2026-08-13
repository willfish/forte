package player

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	mpv "github.com/gen2brain/go-mpv"
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

func TestPauseDuringLoadStaysPaused(t *testing.T) {
	e := newTestEngine(t)
	path := writeTestWAV(t)

	loaded := make(chan struct{})
	var once sync.Once
	e.onTrackChange = func() {
		once.Do(func() { close(loaded) })
	}

	if err := e.Play(path); err != nil {
		t.Fatalf("Play() real wav error: %v", err)
	}
	e.Pause()

	select {
	case <-loaded:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for mpv to load generated wav")
	}

	if s := e.State(); s != StatePaused {
		t.Fatalf("expected StatePaused after load completes, got %s", s)
	}
}

func TestStopDuringLoadStaysStopped(t *testing.T) {
	e := newTestEngine(t)
	path := writeTestWAV(t)

	loaded := make(chan struct{})
	e.onTrackChange = func() {
		close(loaded)
	}

	if err := e.Play(path); err != nil {
		t.Fatalf("Play() real wav error: %v", err)
	}
	e.Stop()

	select {
	case <-loaded:
		t.Fatal("onTrackChange should not run after stop during load")
	case <-time.After(500 * time.Millisecond):
	}

	if s := e.State(); s != StateStopped {
		t.Fatalf("expected StateStopped after load race, got %s", s)
	}
}

func TestStopClearsPauseProperty(t *testing.T) {
	e := newTestEngine(t)
	path := writeTestWAV(t)

	if err := e.Play(path); err != nil {
		t.Fatalf("Play() real wav error: %v", err)
	}
	e.Pause()
	e.Stop()

	if s := e.State(); s != StateStopped {
		t.Fatalf("expected StateStopped after Stop(), got %s", s)
	}
	if paused := readPauseProperty(t, e); paused {
		t.Fatal("expected mpv pause property to be false after Stop()")
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	e := newTestEngine(t)

	e.Close()
	e.Close()
}

func TestPlayAfterPausedStopStartsUnpaused(t *testing.T) {
	e := newTestEngine(t)
	path := writeTestWAV(t)

	if err := e.Play(path); err != nil {
		t.Fatalf("Play() real wav error: %v", err)
	}
	e.Pause()
	e.Stop()

	if err := e.Play(path); err != nil {
		t.Fatalf("Play() after paused stop error: %v", err)
	}
	if s := e.State(); s != StatePlaying {
		t.Fatalf("expected StatePlaying after Play(), got %s", s)
	}
	if paused := readPauseProperty(t, e); paused {
		t.Fatal("expected mpv pause property to be false after Play()")
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

func readPauseProperty(t *testing.T, e *Engine) bool {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()

	v, err := e.handle.GetProperty("pause", mpv.FormatFlag)
	if err != nil {
		t.Fatalf("GetProperty(pause): %v", err)
	}
	paused, ok := v.(bool)
	if !ok {
		t.Fatalf("pause property has type %T, want bool", v)
	}
	return paused
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

func TestPlayLoadsRealWAVFile(t *testing.T) {
	e := newTestEngine(t)
	path := writeTestWAV(t)

	loaded := make(chan struct{})
	var once sync.Once
	e.onTrackChange = func() {
		once.Do(func() { close(loaded) })
	}

	if err := e.Play(path); err != nil {
		t.Fatalf("Play() real wav error: %v", err)
	}

	select {
	case <-loaded:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for mpv to load generated wav")
	}

	if s := e.State(); s != StatePlaying {
		t.Fatalf("expected StatePlaying after loading real wav, got %s", s)
	}
}

func TestGaplessOptionSet(t *testing.T) {
	e := newTestEngine(t)

	v := e.handle.GetPropertyString("gapless-audio")
	if v != "yes" {
		t.Fatalf("expected gapless-audio=yes, got %q", v)
	}
}

func TestStreamReconnectOptionsSet(t *testing.T) {
	e := newTestEngine(t)

	v := e.handle.GetPropertyString("stream-lavf-o")
	if v != "reconnect=1,reconnect_streamed=1,reconnect_on_network_error=1,reconnect_delay_max=5" {
		t.Fatalf("expected lavf reconnect options, got %q", v)
	}
}

func TestEOFInvokesStreamErrorWhenReconnectOnEOF(t *testing.T) {
	e := newTestEngine(t)
	e.SetReconnectOnEOF(true)

	called := make(chan struct{})
	var once sync.Once
	e.SetOnStreamError(func() {
		once.Do(func() { close(called) })
	})
	e.SetOnPlaylistEnd(func() {
		t.Error("onPlaylistEnd should not run when radio EOF reconnects")
	})

	if err := e.Play(writeTestWAV(t)); err != nil {
		t.Fatalf("Play() error: %v", err)
	}

	select {
	case <-called:
	case <-time.After(4 * time.Second):
		t.Fatal("expected onStreamError after radio stream EOF")
	}
}

func TestEOFDoesNotInvokeStreamErrorByDefault(t *testing.T) {
	e := newTestEngine(t)

	errored := make(chan struct{})
	e.SetOnStreamError(func() { close(errored) })

	if err := e.Play(writeTestWAV(t)); err != nil {
		t.Fatalf("Play() error: %v", err)
	}

	select {
	case <-errored:
		t.Fatal("onStreamError should not run on EOF without reconnectOnEOF")
	case <-time.After(2 * time.Second):
	}
}

func TestStopDoesNotInvokeStreamErrorWhenReconnectOnEOF(t *testing.T) {
	e := newTestEngine(t)
	e.SetReconnectOnEOF(true)

	errored := make(chan struct{})
	e.SetOnStreamError(func() { close(errored) })

	if err := e.Play(writeTestWAV(t)); err != nil {
		t.Fatalf("Play() error: %v", err)
	}
	e.Stop()

	select {
	case <-errored:
		t.Fatal("onStreamError should not run after Stop()")
	case <-time.After(300 * time.Millisecond):
	}
}

func writeTestWAV(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tone.wav")
	const sampleRate = 44100
	const durationSeconds = 1
	const channels = 1
	const bitsPerSample = 16
	const samples = sampleRate * durationSeconds
	const dataBytes = samples * channels * bitsPerSample / 8

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create wav: %v", err)
	}
	defer func() { _ = f.Close() }()

	writeString := func(s string) {
		if _, err := f.WriteString(s); err != nil {
			t.Fatalf("write wav: %v", err)
		}
	}
	write := func(v any) {
		if err := binary.Write(f, binary.LittleEndian, v); err != nil {
			t.Fatalf("write wav: %v", err)
		}
	}

	writeString("RIFF")
	write(uint32(36 + dataBytes))
	writeString("WAVE")
	writeString("fmt ")
	write(uint32(16))
	write(uint16(1))
	write(uint16(channels))
	write(uint32(sampleRate))
	write(uint32(sampleRate * channels * bitsPerSample / 8))
	write(uint16(channels * bitsPerSample / 8))
	write(uint16(bitsPerSample))
	writeString("data")
	write(uint32(dataBytes))

	for i := 0; i < samples; i++ {
		angle := 2 * math.Pi * 440 * float64(i) / sampleRate
		sample := int16(math.Sin(angle) * math.MaxInt16 * 0.2)
		write(sample)
	}

	return path
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
