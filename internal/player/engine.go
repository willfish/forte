// Package player handles audio playback via mpv.
package player

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	mpv "github.com/gen2brain/go-mpv"
)

// ErrMpvNotFound is returned when libmpv cannot be loaded.
var ErrMpvNotFound = errors.New(
	"mpv not found: install mpv (e.g. 'sudo apt install libmpv-dev' or 'nix-shell -p mpv')",
)

// Engine wraps an mpv instance for audio playback.
type Engine struct {
	mu      sync.Mutex
	handle  *mpv.Mpv
	playing bool
	stop    chan struct{}
}

// NewEngine initialises mpv for audio-only playback.
// Returns ErrMpvNotFound if libmpv is not installed.
func NewEngine() (*Engine, error) {
	m := mpv.New()
	if m == nil {
		return nil, ErrMpvNotFound
	}

	if err := m.SetOptionString("audio-display", "no"); err != nil {
		m.TerminateDestroy()
		return nil, fmt.Errorf("mpv set audio-display: %w", err)
	}

	if err := m.SetOptionString("vo", "null"); err != nil {
		m.TerminateDestroy()
		return nil, fmt.Errorf("mpv set vo: %w", err)
	}

	if err := m.SetOptionString("terminal", "no"); err != nil {
		m.TerminateDestroy()
		return nil, fmt.Errorf("mpv set terminal: %w", err)
	}

	if err := m.Initialize(); err != nil {
		m.TerminateDestroy()
		return nil, fmt.Errorf("mpv initialize: %w", err)
	}

	e := &Engine{
		handle: m,
		stop:   make(chan struct{}),
	}

	go e.eventLoop()

	return e, nil
}

// Play loads and plays the audio file at the given path.
func (e *Engine) Play(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.handle.Command([]string{"loadfile", path}); err != nil {
		return fmt.Errorf("mpv loadfile: %w", err)
	}

	e.playing = true
	return nil
}

// Stop stops the current playback.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	_ = e.handle.Command([]string{"stop"})
	e.playing = false
}

// Playing reports whether audio is currently playing.
func (e *Engine) Playing() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.playing
}

// Close shuts down the mpv instance.
func (e *Engine) Close() {
	close(e.stop)
	e.handle.TerminateDestroy()
}

// Version returns the mpv library version string.
func (e *Engine) Version() string {
	return e.handle.GetPropertyString("mpv-version")
}

func (e *Engine) eventLoop() {
	for {
		select {
		case <-e.stop:
			return
		default:
		}

		event := e.handle.WaitEvent(0.5)
		if event == nil {
			continue
		}

		switch event.EventID {
		case mpv.EventEnd:
			e.mu.Lock()
			e.playing = false
			e.mu.Unlock()
			slog.Debug("playback ended")
		case mpv.EventFileLoaded:
			slog.Debug("file loaded")
		case mpv.EventShutdown:
			return
		}
	}
}
