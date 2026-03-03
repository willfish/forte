package main

import (
	"context"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/player"
)

// PlayerService exposes audio playback controls to the frontend.
type PlayerService struct {
	engine *player.Engine
}

// ServiceStartup initialises the mpv engine when the application starts.
func (p *PlayerService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	e, err := player.NewEngine()
	if err != nil {
		return fmt.Errorf("player startup: %w", err)
	}
	p.engine = e
	return nil
}

// ServiceShutdown cleans up the mpv engine when the application exits.
func (p *PlayerService) ServiceShutdown() error {
	if p.engine != nil {
		p.engine.Close()
	}
	return nil
}

// Play starts playback of the audio file at the given path.
func (p *PlayerService) Play(path string) error {
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	return p.engine.Play(path)
}

// Pause pauses the current playback.
func (p *PlayerService) Pause() {
	if p.engine != nil {
		p.engine.Pause()
	}
}

// Resume resumes paused playback.
func (p *PlayerService) Resume() {
	if p.engine != nil {
		p.engine.Resume()
	}
}

// Stop halts the current playback.
func (p *PlayerService) Stop() {
	if p.engine != nil {
		p.engine.Stop()
	}
}

// Seek seeks to the given position in seconds.
func (p *PlayerService) Seek(seconds float64) {
	if p.engine != nil {
		p.engine.Seek(seconds)
	}
}

// SetVolume sets the volume (0-100).
func (p *PlayerService) SetVolume(percent int) {
	if p.engine != nil {
		p.engine.SetVolume(percent)
	}
}

// Volume returns the current volume (0-100).
func (p *PlayerService) Volume() int {
	if p.engine == nil {
		return 0
	}
	return p.engine.Volume()
}

// Position returns the current playback position in seconds.
func (p *PlayerService) Position() float64 {
	if p.engine == nil {
		return 0
	}
	return p.engine.Position()
}

// Duration returns the duration of the current track in seconds.
func (p *PlayerService) Duration() float64 {
	if p.engine == nil {
		return 0
	}
	return p.engine.Duration()
}

// State returns the current playback state as a string.
func (p *PlayerService) State() string {
	if p.engine == nil {
		return "stopped"
	}
	return p.engine.State().String()
}

// Version returns the mpv library version string.
func (p *PlayerService) Version() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.Version()
}
