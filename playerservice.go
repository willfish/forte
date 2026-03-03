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

// Stop halts the current playback.
func (p *PlayerService) Stop() {
	if p.engine != nil {
		p.engine.Stop()
	}
}

// Playing reports whether audio is currently playing.
func (p *PlayerService) Playing() bool {
	if p.engine == nil {
		return false
	}
	return p.engine.Playing()
}

// Version returns the mpv library version string.
func (p *PlayerService) Version() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.Version()
}
