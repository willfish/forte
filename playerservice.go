package main

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/metadata"
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

// Enqueue appends a track to the playlist for gapless playback.
func (p *PlayerService) Enqueue(path string) error {
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	return p.engine.Enqueue(path)
}

// PlayAll replaces the playlist and plays the given tracks in order.
func (p *PlayerService) PlayAll(paths []string) error {
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	return p.engine.PlayAll(paths)
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

// MediaTitle returns the title of the currently playing track.
func (p *PlayerService) MediaTitle() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.MediaTitle()
}

// MediaArtist returns the artist of the currently playing track.
func (p *PlayerService) MediaArtist() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.MediaArtist()
}

// MediaAlbum returns the album of the currently playing track.
func (p *PlayerService) MediaAlbum() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.MediaAlbum()
}

// Next skips to the next track in the playlist.
func (p *PlayerService) Next() {
	if p.engine != nil {
		p.engine.Next()
	}
}

// Previous skips to the previous track in the playlist.
func (p *PlayerService) Previous() {
	if p.engine != nil {
		p.engine.Previous()
	}
}

// Artwork returns the album artwork for the currently playing track
// as a base64-encoded data URI, or an empty string if unavailable.
func (p *PlayerService) Artwork() string {
	if p.engine == nil {
		return ""
	}
	path := p.engine.MediaPath()
	if path == "" {
		return ""
	}
	data, mime, err := metadata.ReadArtwork(path)
	if err != nil || len(data) == 0 {
		return ""
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// SetReplayGain sets the ReplayGain mode: "track", "album", or "no" (off).
func (p *PlayerService) SetReplayGain(mode string) error {
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	return p.engine.SetReplayGain(mode)
}

// ReplayGain returns the current ReplayGain mode.
func (p *PlayerService) ReplayGain() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.ReplayGain()
}

// Version returns the mpv library version string.
func (p *PlayerService) Version() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.Version()
}
