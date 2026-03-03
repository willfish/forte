// Package player handles audio playback via mpv.
package player

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	mpv "github.com/gen2brain/go-mpv"
)

// PlaybackState represents the current state of the player.
type PlaybackState int

const (
	StateStopped PlaybackState = iota
	StatePlaying
	StatePaused
)

func (s PlaybackState) String() string {
	switch s {
	case StatePlaying:
		return "playing"
	case StatePaused:
		return "paused"
	default:
		return "stopped"
	}
}

// ErrMpvNotFound is returned when libmpv cannot be loaded.
var ErrMpvNotFound = errors.New(
	"mpv not found: install mpv (e.g. 'sudo apt install libmpv-dev' or 'nix-shell -p mpv')",
)

// Engine wraps an mpv instance for audio playback.
type Engine struct {
	mu     sync.Mutex
	handle *mpv.Mpv
	state  PlaybackState
	stop   chan struct{}
}

// NewEngine initialises mpv for audio-only playback.
// Returns ErrMpvNotFound if libmpv is not installed.
func NewEngine() (*Engine, error) {
	m := mpv.New()
	if m == nil {
		return nil, ErrMpvNotFound
	}

	for _, opt := range [][2]string{
		{"audio-display", "no"},
		{"vo", "null"},
		{"terminal", "no"},
	} {
		if err := m.SetOptionString(opt[0], opt[1]); err != nil {
			m.TerminateDestroy()
			return nil, fmt.Errorf("mpv set %s: %w", opt[0], err)
		}
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

	e.state = StatePlaying
	return nil
}

// Pause pauses the current playback.
func (e *Engine) Pause() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state != StatePlaying {
		return
	}
	_ = e.handle.SetProperty("pause", mpv.FormatFlag, true)
	e.state = StatePaused
}

// Resume resumes paused playback.
func (e *Engine) Resume() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state != StatePaused {
		return
	}
	_ = e.handle.SetProperty("pause", mpv.FormatFlag, false)
	e.state = StatePlaying
}

// Stop stops the current playback.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	_ = e.handle.Command([]string{"stop"})
	e.state = StateStopped
}

// Seek seeks to the given position in seconds.
func (e *Engine) Seek(seconds float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.state == StateStopped {
		return
	}
	_ = e.handle.CommandString(fmt.Sprintf("seek %f absolute", seconds))
}

// SetVolume sets the volume (0-100).
func (e *Engine) SetVolume(percent int) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	_ = e.handle.SetProperty("volume", mpv.FormatDouble, float64(percent))
}

// Volume returns the current volume (0-100).
func (e *Engine) Volume() int {
	v, err := e.handle.GetProperty("volume", mpv.FormatDouble)
	if err != nil {
		return 0
	}
	return int(v.(float64))
}

// Position returns the current playback position in seconds.
func (e *Engine) Position() float64 {
	v, err := e.handle.GetProperty("time-pos", mpv.FormatDouble)
	if err != nil {
		return 0
	}
	return v.(float64)
}

// Duration returns the duration of the current track in seconds.
func (e *Engine) Duration() float64 {
	v, err := e.handle.GetProperty("duration", mpv.FormatDouble)
	if err != nil {
		return 0
	}
	return v.(float64)
}

// State returns the current playback state.
func (e *Engine) State() PlaybackState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
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
			e.state = StateStopped
			e.mu.Unlock()
			slog.Debug("playback ended")
		case mpv.EventFileLoaded:
			slog.Debug("file loaded")
		case mpv.EventShutdown:
			return
		}
	}
}
