package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/metadata"
	"github.com/willfish/forte/internal/player"
	"github.com/willfish/forte/internal/system"
)

// PlayerService exposes audio playback controls to the frontend.
type PlayerService struct {
	engine       *player.Engine
	queue        *player.Queue
	db           *library.DB
	mpris        *system.MPRIS
	notifier     *system.Notifier
	onTrayUpdate func(title, artist string) // set by main.go for tooltip updates
	manualSkip   int32                      // atomic: set before explicit Next/Previous to suppress auto-advance
	stopSave     chan struct{}
}

// ServiceStartup initialises the mpv engine when the application starts.
func (p *PlayerService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	e, err := player.NewEngine()
	if err != nil {
		return fmt.Errorf("player startup: %w", err)
	}
	p.engine = e
	p.queue = player.NewQueue()

	// Open the database for persisting playback state.
	dataDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("config dir: %w", err)
	}
	dbDir := filepath.Join(dataDir, "forte")
	if err := os.MkdirAll(dbDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	db, err := library.OpenDB(filepath.Join(dbDir, "library.db"))
	if err != nil {
		return fmt.Errorf("player db: %w", err)
	}
	p.db = db

	// When mpv auto-advances to the next track (gapless), advance the queue.
	e.SetOnTrackChange(func() {
		if atomic.CompareAndSwapInt32(&p.manualSkip, 1, 0) {
			return // explicit Next/Previous already updated the queue
		}
		p.queue.Next()
		p.pushMPRISMetadata()
	})

	// When the mpv playlist ends, loop back if repeat-all is on.
	e.SetOnPlaylistEnd(func() {
		if p.queue.Repeat() != player.RepeatAll {
			if p.mpris != nil {
				p.mpris.UpdatePlaybackStatus("stopped")
				p.mpris.ClearMetadata()
			}
			return
		}
		p.queue.SetPosition(0)
		paths := p.queue.Paths(0)
		if len(paths) > 0 {
			atomic.StoreInt32(&p.manualSkip, 1)
			_ = p.engine.PlayAll(paths)
		}
	})

	// Start MPRIS2 D-Bus provider.
	system.SetReadArtworkFn(metadata.ReadArtwork)
	mpris, err := system.NewMPRIS(p)
	if err != nil {
		log.Printf("mpris: %v (media keys will not work)", err)
	} else {
		p.mpris = mpris
	}

	// Start desktop notifications.
	notifier, err := system.NewNotifier()
	if err != nil {
		log.Printf("notifications: %v (desktop notifications will not work)", err)
	} else {
		p.notifier = notifier
	}

	// Restore saved playback state.
	p.restoreState()

	// Periodic save (10s) and MPRIS position update (1s).
	p.stopSave = make(chan struct{})
	go func() {
		posTicker := time.NewTicker(1 * time.Second)
		saveTicker := time.NewTicker(10 * time.Second)
		defer posTicker.Stop()
		defer saveTicker.Stop()
		for {
			select {
			case <-posTicker.C:
				if p.mpris != nil && p.engine != nil {
					p.mpris.UpdatePosition(p.engine.Position())
				}
			case <-saveTicker.C:
				p.saveState()
			case <-p.stopSave:
				return
			}
		}
	}()

	return nil
}

// ServiceShutdown cleans up the mpv engine when the application exits.
func (p *PlayerService) ServiceShutdown() error {
	if p.stopSave != nil {
		close(p.stopSave)
	}
	p.saveState()
	if p.notifier != nil {
		p.notifier.Close()
	}
	if p.mpris != nil {
		p.mpris.Close()
	}
	if p.engine != nil {
		p.engine.Close()
	}
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

func (p *PlayerService) pushMPRISMetadata() {
	cur := p.queue.Current()
	if p.mpris != nil {
		if cur == nil {
			p.mpris.ClearMetadata()
		} else {
			p.mpris.UpdateMetadata(cur.Title, cur.Artist, cur.Album, cur.FilePath, cur.DurationMs, cur.TrackID)
			p.mpris.UpdatePlaybackStatus(p.State())
		}
	}

	// Update tray tooltip.
	if p.onTrayUpdate != nil {
		if cur != nil {
			p.onTrayUpdate(cur.Title, cur.Artist)
		} else {
			p.onTrayUpdate("", "")
		}
	}

	// Send desktop notification for the new track.
	if p.notifier != nil && cur != nil {
		var artwork []byte
		if cur.FilePath != "" {
			artwork, _, _ = metadata.ReadArtwork(cur.FilePath)
		}
		body := cur.Artist
		if cur.Album != "" {
			body += " - " + cur.Album
		}
		p.notifier.Notify(cur.Title, body, artwork)
	}
}

func (p *PlayerService) saveState() {
	if p.db == nil || p.queue == nil {
		return
	}
	tracks := p.queue.Tracks()
	queueJSON, err := json.Marshal(tracks)
	if err != nil {
		log.Printf("save state: marshal queue: %v", err)
		return
	}

	var posMs int
	if p.engine != nil {
		posMs = int(p.engine.Position() * 1000)
	}

	vol := 100
	if p.engine != nil {
		vol = p.engine.Volume()
	}

	state := library.PlaybackState{
		QueueJSON:       string(queueJSON),
		Position:        p.queue.Position(),
		TrackPositionMs: posMs,
		Volume:          vol,
		Shuffle:         p.queue.Shuffled(),
		RepeatMode:      p.queue.Repeat().String(),
	}
	if err := p.db.SavePlaybackState(state); err != nil {
		log.Printf("save state: %v", err)
	}
}

func (p *PlayerService) restoreState() {
	if p.db == nil {
		return
	}
	state, err := p.db.LoadPlaybackState()
	if err != nil {
		return // no saved state or error - start fresh
	}

	var tracks []player.QueueTrack
	if err := json.Unmarshal([]byte(state.QueueJSON), &tracks); err != nil {
		return
	}

	// Filter out tracks whose files no longer exist.
	valid := make([]player.QueueTrack, 0, len(tracks))
	for _, t := range tracks {
		if _, err := os.Stat(t.FilePath); err == nil {
			valid = append(valid, t)
		}
	}
	if len(valid) == 0 {
		// Nothing to restore, just set volume.
		if p.engine != nil {
			p.engine.SetVolume(state.Volume)
		}
		return
	}

	// Adjust position if tracks were removed before it.
	pos := state.Position
	removed := 0
	for i, t := range tracks {
		if i < state.Position {
			if _, err := os.Stat(t.FilePath); err != nil {
				removed++
			}
		}
	}
	pos -= removed
	if pos < 0 || pos >= len(valid) {
		pos = 0
	}

	// Restore queue and modes.
	p.queue.Replace(valid, pos)

	var rm player.RepeatMode
	switch state.RepeatMode {
	case "all":
		rm = player.RepeatAll
	case "one":
		rm = player.RepeatOne
	default:
		rm = player.RepeatOff
	}
	p.queue.SetRepeat(rm)

	if state.Shuffle {
		p.queue.SetShuffle(true)
	}

	// Set volume and repeat-one loop on the engine.
	if p.engine != nil {
		p.engine.SetVolume(state.Volume)
		p.engine.SetLoopFile(rm == player.RepeatOne)

		// Load the playlist but start paused.
		paths := p.queue.Paths(pos)
		if len(paths) > 0 {
			atomic.StoreInt32(&p.manualSkip, 1)
			if err := p.engine.PlayAll(paths); err == nil {
				// Pause immediately and seek to saved position.
				p.engine.Pause()
				if state.TrackPositionMs > 0 {
					p.engine.Seek(float64(state.TrackPositionMs) / 1000.0)
				}
			}
		}
	}
}

// PlayQueue replaces the queue with the given tracks and starts playback
// from startAt. This is the primary entry point for playing from the UI.
func (p *PlayerService) PlayQueue(tracks []player.QueueTrack, startAt int) error {
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	p.queue.Replace(tracks, startAt)
	paths := p.queue.Paths(startAt)
	if len(paths) == 0 {
		return nil
	}
	atomic.StoreInt32(&p.manualSkip, 1) // suppress callback for initial load
	err := p.engine.PlayAll(paths)
	p.pushMPRISMetadata()
	return err
}

// QueueAppend adds a track to the end of the queue.
// If nothing is playing, it does not start playback.
func (p *PlayerService) QueueAppend(track player.QueueTrack) {
	p.queue.Append(track)
}

// QueueInsertNext inserts a track immediately after the current track.
func (p *PlayerService) QueueInsertNext(track player.QueueTrack) {
	p.queue.InsertAfterCurrent(track)
}

// QueueRemove removes the track at the given index.
// If the removed track was the current track, playback restarts from
// the new current position.
func (p *PlayerService) QueueRemove(index int) error {
	wasCurrent := p.queue.Remove(index)
	if wasCurrent && p.engine != nil {
		cur := p.queue.Current()
		if cur == nil {
			p.engine.Stop()
			return nil
		}
		paths := p.queue.Paths(p.queue.Position())
		if len(paths) > 0 {
			atomic.StoreInt32(&p.manualSkip, 1)
			return p.engine.PlayAll(paths)
		}
	}
	return nil
}

// QueueMove moves a track from one position to another.
func (p *PlayerService) QueueMove(from, to int) {
	p.queue.Move(from, to)
}

// QueueClear clears the queue and stops playback.
func (p *PlayerService) QueueClear() {
	p.queue.Clear()
	if p.engine != nil {
		p.engine.Stop()
	}
	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("stopped")
		p.mpris.ClearMetadata()
	}
	if p.onTrayUpdate != nil {
		p.onTrayUpdate("", "")
	}
}

// GetQueue returns all tracks in the queue.
func (p *PlayerService) GetQueue() []player.QueueTrack {
	return p.queue.Tracks()
}

// GetQueuePosition returns the current queue position (-1 if empty).
func (p *PlayerService) GetQueuePosition() int {
	return p.queue.Position()
}

// SetShuffle enables or disables shuffle mode.
// When toggled, the mpv playlist is rebuilt to match the new order.
func (p *PlayerService) SetShuffle(enabled bool) {
	p.queue.SetShuffle(enabled)
	if p.engine == nil {
		return
	}
	// Rebuild the mpv playlist from the track after current.
	pos := p.queue.Position()
	if pos < 0 {
		return
	}
	upcoming := p.queue.Paths(pos + 1)
	p.engine.ReplaceUpcoming(upcoming)
	if p.mpris != nil {
		p.mpris.UpdateShuffle(enabled)
	}
}

// GetShuffle returns whether shuffle mode is active.
func (p *PlayerService) GetShuffle() bool {
	return p.queue.Shuffled()
}

// SetRepeat sets the repeat mode: "off", "all", or "one".
func (p *PlayerService) SetRepeat(mode string) {
	var rm player.RepeatMode
	switch mode {
	case "all":
		rm = player.RepeatAll
	case "one":
		rm = player.RepeatOne
	default:
		rm = player.RepeatOff
	}
	p.queue.SetRepeat(rm)

	if p.engine != nil {
		p.engine.SetLoopFile(rm == player.RepeatOne)
	}
	if p.mpris != nil {
		p.mpris.UpdateLoopStatus(mode)
	}
}

// GetRepeat returns the current repeat mode as a string.
func (p *PlayerService) GetRepeat() string {
	return p.queue.Repeat().String()
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
	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("paused")
	}
}

// Resume resumes paused playback.
func (p *PlayerService) Resume() {
	if p.engine != nil {
		p.engine.Resume()
	}
	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("playing")
	}
}

// Stop halts the current playback.
func (p *PlayerService) Stop() {
	if p.engine != nil {
		p.engine.Stop()
	}
	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("stopped")
		p.mpris.ClearMetadata()
	}
	if p.onTrayUpdate != nil {
		p.onTrayUpdate("", "")
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
	if p.mpris != nil {
		p.mpris.UpdateVolume(percent)
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

// MediaPath returns the file path of the currently playing track.
func (p *PlayerService) MediaPath() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.MediaPath()
}

// Next skips to the next track in the queue.
func (p *PlayerService) Next() {
	if p.engine == nil {
		return
	}
	repeat := p.queue.Repeat()
	if repeat == player.RepeatOne {
		// Repeat-one: seek to start instead of advancing.
		p.engine.Seek(0)
		return
	}
	if p.queue.Next() {
		atomic.StoreInt32(&p.manualSkip, 1)
		if repeat == player.RepeatAll && p.queue.Position() == 0 {
			// Wrapped around: reload playlist from the start.
			paths := p.queue.Paths(0)
			if len(paths) > 0 {
				_ = p.engine.PlayAll(paths)
			}
		} else {
			p.engine.Next()
		}
	}
}

// Previous skips to the previous track in the queue.
func (p *PlayerService) Previous() {
	if p.engine == nil {
		return
	}
	repeat := p.queue.Repeat()
	if repeat == player.RepeatOne {
		p.engine.Seek(0)
		return
	}
	if p.queue.Previous() {
		atomic.StoreInt32(&p.manualSkip, 1)
		if repeat == player.RepeatAll && p.queue.Position() == p.queue.Len()-1 {
			// Wrapped backward: reload playlist from the end.
			paths := p.queue.Paths(p.queue.Position())
			if len(paths) > 0 {
				_ = p.engine.PlayAll(paths)
			}
		} else {
			p.engine.Previous()
		}
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

// SetNotifications enables or disables desktop notifications.
func (p *PlayerService) SetNotifications(enabled bool) {
	if p.notifier != nil {
		p.notifier.SetEnabled(enabled)
	}
}

// GetNotifications returns whether desktop notifications are enabled.
func (p *PlayerService) GetNotifications() bool {
	if p.notifier == nil {
		return false
	}
	return p.notifier.Enabled()
}

// Version returns the mpv library version string.
func (p *PlayerService) Version() string {
	if p.engine == nil {
		return ""
	}
	return p.engine.Version()
}
