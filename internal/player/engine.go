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
	done   chan struct{} // closed when event loop exits
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
		{"gapless-audio", "yes"},
		{"replaygain", "track"},
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
		done:   make(chan struct{}),
	}

	go e.eventLoop()

	return e, nil
}

// Play loads and plays the audio file at the given path.
// This replaces the current playlist.
func (e *Engine) Play(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.handle.Command([]string{"loadfile", path, "replace"}); err != nil {
		return fmt.Errorf("mpv loadfile: %w", err)
	}

	e.state = StatePlaying
	return nil
}

// Enqueue appends a track to the playlist for gapless playback.
func (e *Engine) Enqueue(path string) error {
	if err := e.handle.Command([]string{"loadfile", path, "append"}); err != nil {
		return fmt.Errorf("mpv enqueue: %w", err)
	}
	return nil
}

// PlayAll replaces the playlist and plays the given tracks in order.
func (e *Engine) PlayAll(paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Load the first track (replaces playlist).
	if err := e.handle.Command([]string{"loadfile", paths[0], "replace"}); err != nil {
		return fmt.Errorf("mpv loadfile: %w", err)
	}
	e.state = StatePlaying

	// Append the rest for gapless transitions.
	for _, p := range paths[1:] {
		if err := e.handle.Command([]string{"loadfile", p, "append"}); err != nil {
			return fmt.Errorf("mpv enqueue: %w", err)
		}
	}

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

// SetReplayGain sets the ReplayGain mode: "track", "album", or "no" (off).
func (e *Engine) SetReplayGain(mode string) error {
	switch mode {
	case "track", "album", "no":
		return e.handle.SetPropertyString("replaygain", mode)
	default:
		return fmt.Errorf("invalid replaygain mode: %q (expected track, album, or no)", mode)
	}
}

// ReplayGain returns the current ReplayGain mode.
func (e *Engine) ReplayGain() string {
	return e.handle.GetPropertyString("replaygain")
}

// Close shuts down the mpv instance.
func (e *Engine) Close() {
	close(e.stop)
	<-e.done // wait for the event loop goroutine to exit
	e.handle.TerminateDestroy()
}

// Version returns the mpv library version string.
func (e *Engine) Version() string {
	return e.handle.GetPropertyString("mpv-version")
}

func (e *Engine) eventLoop() {
	defer close(e.done)
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
			ef := event.EndFile()
			if ef.Reason == mpv.EndFileEOF || ef.Reason == mpv.EndFileStop || ef.Reason == mpv.EndFileError {
				// Check if mpv has more playlist entries queued.
				pos, _ := e.handle.GetProperty("playlist-pos", mpv.FormatInt64)
				if pos == nil || pos.(int64) < 0 {
					e.mu.Lock()
					e.state = StateStopped
					e.mu.Unlock()
					slog.Debug("playlist finished")
				} else {
					slog.Debug("track ended, next track queued")
				}
			}
		case mpv.EventFileLoaded:
			e.mu.Lock()
			e.state = StatePlaying
			e.mu.Unlock()
			slog.Debug("file loaded")
		case mpv.EventShutdown:
			return
		}
	}
}
