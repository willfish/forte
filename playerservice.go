package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/metadata"
	"github.com/willfish/forte/internal/player"
	"github.com/willfish/forte/internal/scrobbling/lastfm"
	"github.com/willfish/forte/internal/scrobbling/listenbrainz"
	"github.com/willfish/forte/internal/system"
)

// PlayerService exposes audio playback controls to the frontend.
type PlayerService struct {
	engine         *player.Engine
	queue          *player.Queue
	db             *library.DB
	resolver       *library.PathResolver
	mpris          *system.MPRIS
	notifier       *system.Notifier
	toasts         *player.Notifications
	isServerOnline func(string) bool         // set by main.go to check server health
	onTrayUpdate   func(title, artist string) // set by main.go for tooltip updates
	manualSkip     int32                      // atomic: set before explicit Next/Previous to suppress auto-advance
	stopSave       chan struct{}
	tickerDone     chan struct{} // closed when the ticker goroutine exits

	// Scrobble tracking state (protected by scrobbleMu).
	scrobbleMu      sync.Mutex
	scrobbleTrackID int64
	scrobbleElapsed time.Duration
	scrobbled       bool

	// Radio mode state (protected by radioMu).
	radioMu         sync.RWMutex
	radioMode       bool
	radioName       string
	radioStreamURL  string
	radioArtworkURL string
	radioLastTitle  string // last ICY stream title, for change detection
	savedQueue      []player.QueueTrack
	savedPosition   int
}

// ServiceStartup initialises the mpv engine when the application starts.
func (p *PlayerService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	e, err := player.NewEngine()
	if err != nil {
		return fmt.Errorf("player startup: %w", err)
	}
	p.engine = e
	p.queue = player.NewQueue()
	p.toasts = player.NewNotifications()

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
	p.resolver = library.NewPathResolver(db)

	// When mpv auto-advances to the next track (gapless), advance the queue.
	e.SetOnTrackChange(func() {
		if atomic.CompareAndSwapInt32(&p.manualSkip, 1, 0) {
			return // explicit Next/Previous already updated the queue
		}
		p.queue.Next()
		p.pushMPRISMetadata()
		p.startScrobbleTracking()
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
			_ = p.engine.PlayAll(p.resolvePaths(paths))
		}
	})

	// When mpv fails to play a stream, skip past offline server tracks.
	e.SetOnStreamError(func() {
		p.skipToNextPlayable()
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

	// Periodic save (10s), MPRIS position update (1s), and scrobble queue flush (5m).
	p.stopSave = make(chan struct{})
	p.tickerDone = make(chan struct{})
	go func() {
		defer close(p.tickerDone)
		posTicker := time.NewTicker(1 * time.Second)
		saveTicker := time.NewTicker(10 * time.Second)
		flushTicker := time.NewTicker(5 * time.Minute)
		defer posTicker.Stop()
		defer saveTicker.Stop()
		defer flushTicker.Stop()
		for {
			select {
			case <-posTicker.C:
				if p.mpris != nil && p.engine != nil {
					p.mpris.UpdatePosition(p.engine.Position())
				}
				p.checkScrobble()
				p.checkRadioTitle()
			case <-saveTicker.C:
				p.saveState()
			case <-flushTicker.C:
				p.flushScrobbleQueue()
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
	if p.tickerDone != nil {
		<-p.tickerDone // wait for ticker goroutine to fully exit
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
		if cur.FilePath != "" && !library.IsServerPath(cur.FilePath) {
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

	// Filter out tracks whose files no longer exist (server tracks are always valid).
	valid := make([]player.QueueTrack, 0, len(tracks))
	for _, t := range tracks {
		if library.IsServerPath(t.FilePath) {
			valid = append(valid, t)
		} else if _, err := os.Stat(t.FilePath); err == nil {
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
			if library.IsServerPath(t.FilePath) {
				continue
			}
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
			if err := p.engine.PlayAll(p.resolvePaths(paths)); err == nil {
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
	resolved := p.resolvePaths(paths)
	atomic.StoreInt32(&p.manualSkip, 1) // suppress callback for initial load
	err := p.engine.PlayAll(resolved)
	p.pushMPRISMetadata()
	p.startScrobbleTracking()
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
			return p.engine.PlayAll(p.resolvePaths(paths))
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
	p.engine.ReplaceUpcoming(p.resolvePaths(upcoming))
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
	resolved := p.resolvePaths([]string{path})
	return p.engine.Play(resolved[0])
}

// Enqueue appends a track to the playlist for gapless playback.
func (p *PlayerService) Enqueue(path string) error {
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	resolved := p.resolvePaths([]string{path})
	return p.engine.Enqueue(resolved[0])
}

// PlayAll replaces the playlist and plays the given tracks in order.
func (p *PlayerService) PlayAll(paths []string) error {
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	return p.engine.PlayAll(p.resolvePaths(paths))
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
// In radio mode, filters out the raw stream URL (shown when no ICY metadata is available).
func (p *PlayerService) MediaTitle() string {
	if p.engine == nil {
		return ""
	}
	t := p.engine.MediaTitle()

	p.radioMu.RLock()
	isRadio := p.radioMode
	streamURL := p.radioStreamURL
	p.radioMu.RUnlock()

	if isRadio && t == streamURL {
		return ""
	}
	return t
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
				_ = p.engine.PlayAll(p.resolvePaths(paths))
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
				_ = p.engine.PlayAll(p.resolvePaths(paths))
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
	// Use the queue's file_path (which may be server://) rather than engine's media path.
	cur := p.queue.Current()
	if cur == nil {
		return ""
	}
	if library.IsServerPath(cur.FilePath) {
		// For server tracks, look up artwork from the album in the DB.
		return p.serverTrackArtwork(cur.TrackID)
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

// serverTrackArtwork returns album artwork for a server track by looking up
// its album_id and fetching the stored artwork blob.
func (p *PlayerService) serverTrackArtwork(trackID int64) string {
	if p.db == nil {
		return ""
	}
	var albumID int64
	err := p.db.QueryRow("SELECT COALESCE(album_id, 0) FROM tracks WHERE id = ?", trackID).Scan(&albumID)
	if err != nil || albumID == 0 {
		return ""
	}
	art, _ := p.db.AlbumArtwork(albumID)
	return art
}

// GetToasts returns and clears all pending toast notifications.
func (p *PlayerService) GetToasts() []player.Toast {
	if p.toasts == nil {
		return nil
	}
	return p.toasts.Drain()
}

// skipToNextPlayable advances the queue past any tracks on offline servers,
// playing the first reachable track. Pushes toast notifications for skipped tracks.
func (p *PlayerService) skipToNextPlayable() {
	cur := p.queue.Current()
	if cur != nil && library.IsServerPath(cur.FilePath) {
		serverID, _, _ := library.ParseServerPath(cur.FilePath)
		if p.isServerOnline != nil && !p.isServerOnline(serverID) {
			p.toasts.Push(fmt.Sprintf("Skipped \"%s\" - server offline", cur.Title), "warn")
		} else {
			p.toasts.Push(fmt.Sprintf("Failed to play \"%s\"", cur.Title), "error")
		}
	}

	// Try advancing through the queue to find a playable track.
	maxAttempts := p.queue.Len()
	for range maxAttempts {
		if !p.queue.Next() {
			break
		}
		next := p.queue.Current()
		if next == nil {
			break
		}
		if !library.IsServerPath(next.FilePath) {
			// Local track, play it.
			atomic.StoreInt32(&p.manualSkip, 1)
			paths := p.queue.Paths(p.queue.Position())
			if len(paths) > 0 {
				_ = p.engine.PlayAll(p.resolvePaths(paths))
				p.pushMPRISMetadata()
			}
			return
		}
		serverID, _, _ := library.ParseServerPath(next.FilePath)
		if p.isServerOnline == nil || p.isServerOnline(serverID) {
			// Server track on an online server, try it.
			atomic.StoreInt32(&p.manualSkip, 1)
			paths := p.queue.Paths(p.queue.Position())
			if len(paths) > 0 {
				_ = p.engine.PlayAll(p.resolvePaths(paths))
				p.pushMPRISMetadata()
			}
			return
		}
		p.toasts.Push(fmt.Sprintf("Skipped \"%s\" - server offline", next.Title), "warn")
	}

	// All remaining tracks are on offline servers.
	p.engine.Stop()
	p.toasts.Push("Playback stopped - remaining tracks are on offline servers", "warn")
	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("stopped")
		p.mpris.ClearMetadata()
	}
}

// startScrobbleTracking resets scrobble state for the current track and
// sends a "now playing" notification to Last.fm if configured.
func (p *PlayerService) startScrobbleTracking() {
	cur := p.queue.Current()
	if cur == nil {
		return
	}
	p.scrobbleMu.Lock()
	p.scrobbleTrackID = cur.TrackID
	p.scrobbleElapsed = 0
	p.scrobbled = false
	p.scrobbleMu.Unlock()

	if p.db == nil {
		return
	}

	// Last.fm now-playing.
	cfg, err := p.db.LoadScrobbleConfig()
	if err == nil && cfg.Enabled && cfg.SessionKey != "" {
		track := lastfm.TrackInfo{
			Artist:   cur.Artist,
			Track:    cur.Title,
			Album:    cur.Album,
			Duration: cur.DurationMs / 1000,
		}
		go func() {
			if err := lastfm.NowPlaying(cfg.APIKey, cfg.APISecret, cfg.SessionKey, track); err != nil {
				log.Printf("lastfm now-playing: %v", err)
			}
		}()
	}

	// ListenBrainz now-playing.
	lbCfg, err := p.db.LoadListenBrainzConfig()
	if err == nil && lbCfg.Enabled && lbCfg.UserToken != "" {
		lbTrack := listenbrainz.TrackInfo{
			Artist:     cur.Artist,
			Track:      cur.Title,
			Album:      cur.Album,
			DurationMs: cur.DurationMs,
		}
		go func() {
			if err := listenbrainz.NowPlaying(lbCfg.UserToken, lbTrack); err != nil {
				log.Printf("listenbrainz now-playing: %v", err)
			}
		}()
	}
}

// checkScrobble accumulates play time and submits a scrobble when the
// threshold is reached (50% of duration or 4 minutes, whichever is first).
func (p *PlayerService) checkScrobble() {
	if p.engine == nil {
		return
	}
	if p.engine.State().String() != "playing" {
		return
	}

	cur := p.queue.Current()

	p.scrobbleMu.Lock()
	if p.scrobbled {
		p.scrobbleMu.Unlock()
		return
	}
	p.scrobbleElapsed += time.Second
	if cur == nil || cur.TrackID != p.scrobbleTrackID {
		p.scrobbleMu.Unlock()
		return
	}
	threshold := time.Duration(lastfm.ScrobbleThreshold(cur.DurationMs)) * time.Millisecond
	if threshold <= 0 || p.scrobbleElapsed < threshold {
		p.scrobbleMu.Unlock()
		return
	}
	p.scrobbled = true
	elapsedMs := int(p.scrobbleElapsed.Milliseconds())
	p.scrobbleMu.Unlock()

	if p.db == nil {
		return
	}

	// Record play in local listening history.
	_ = p.db.RecordPlay(cur.TrackID, elapsedMs)

	ts := time.Now().Unix()

	// Last.fm scrobble.
	cfg, err := p.db.LoadScrobbleConfig()
	if err == nil && cfg.Enabled && cfg.SessionKey != "" {
		track := lastfm.TrackInfo{
			Artist:   cur.Artist,
			Track:    cur.Title,
			Album:    cur.Album,
			Duration: cur.DurationMs / 1000,
		}
		go func() {
			if err := lastfm.Scrobble(cfg.APIKey, cfg.APISecret, cfg.SessionKey, track, ts); err != nil {
				log.Printf("lastfm scrobble: %v (queued for retry)", err)
				p.enqueueFailedScrobble("lastfm", track.Artist, track.Track, track.Album, cur.DurationMs, ts)
			}
		}()
	}

	// ListenBrainz scrobble.
	lbCfg, err := p.db.LoadListenBrainzConfig()
	if err == nil && lbCfg.Enabled && lbCfg.UserToken != "" {
		lbTrack := listenbrainz.TrackInfo{
			Artist:     cur.Artist,
			Track:      cur.Title,
			Album:      cur.Album,
			DurationMs: cur.DurationMs,
		}
		go func() {
			if err := listenbrainz.Scrobble(lbCfg.UserToken, lbTrack, ts); err != nil {
				log.Printf("listenbrainz scrobble: %v (queued for retry)", err)
				p.enqueueFailedScrobble("listenbrainz", lbTrack.Artist, lbTrack.Track, lbTrack.Album, cur.DurationMs, ts)
			}
		}()
	}
}

// resolvePaths translates any server:// paths to streaming URLs.
func (p *PlayerService) resolvePaths(paths []string) []string {
	if p.resolver == nil {
		return paths
	}
	resolved := make([]string, len(paths))
	for i, path := range paths {
		r, err := p.resolver.Resolve(path)
		if err != nil {
			log.Printf("resolve path: %v", err)
			resolved[i] = path // pass through on error
		} else {
			resolved[i] = r
		}
	}
	return resolved
}

// scrobbleTrackJSON is the JSON format stored in the queue for retry.
type scrobbleTrackJSON struct {
	Artist     string `json:"artist"`
	Track      string `json:"track"`
	Album      string `json:"album"`
	DurationMs int    `json:"duration_ms"`
}

// enqueueFailedScrobble saves a failed scrobble to the retry queue.
func (p *PlayerService) enqueueFailedScrobble(service, artist, track, album string, durationMs int, ts int64) {
	if p.db == nil {
		return
	}
	data, err := json.Marshal(scrobbleTrackJSON{
		Artist:     artist,
		Track:      track,
		Album:      album,
		DurationMs: durationMs,
	})
	if err != nil {
		log.Printf("scrobble queue: marshal: %v", err)
		return
	}
	if err := p.db.EnqueueScrobble(service, string(data), ts); err != nil {
		log.Printf("scrobble queue: enqueue: %v", err)
	}
}

// flushScrobbleQueue retries pending scrobbles for all services.
func (p *PlayerService) flushScrobbleQueue() {
	if p.db == nil {
		return
	}

	if err := p.db.PruneScrobbleQueue(); err != nil {
		log.Printf("scrobble queue: prune: %v", err)
	}

	p.flushLastFmQueue()
	p.flushListenBrainzQueue()
}

// flushLastFmQueue retries pending Last.fm scrobbles in batches of up to 50.
func (p *PlayerService) flushLastFmQueue() {
	cfg, err := p.db.LoadScrobbleConfig()
	if err != nil || !cfg.Enabled || cfg.SessionKey == "" {
		return
	}

	entries, err := p.db.PendingScrobbles("lastfm", 50)
	if err != nil || len(entries) == 0 {
		return
	}

	tracks := make([]lastfm.TrackInfo, len(entries))
	timestamps := make([]int64, len(entries))
	for i, e := range entries {
		var t scrobbleTrackJSON
		if err := json.Unmarshal([]byte(e.TrackJSON), &t); err != nil {
			log.Printf("scrobble queue: unmarshal lastfm entry %d: %v", e.ID, err)
			_ = p.db.RemoveScrobble(e.ID) // corrupted entry
			return
		}
		tracks[i] = lastfm.TrackInfo{
			Artist:   t.Artist,
			Track:    t.Track,
			Album:    t.Album,
			Duration: t.DurationMs / 1000,
		}
		timestamps[i] = e.Timestamp
	}

	if err := lastfm.ScrobbleBatch(cfg.APIKey, cfg.APISecret, cfg.SessionKey, tracks, timestamps); err != nil {
		log.Printf("scrobble queue: lastfm batch: %v", err)
		for _, e := range entries {
			_ = p.db.MarkScrobbleAttempt(e.ID)
		}
		return
	}

	for _, e := range entries {
		_ = p.db.RemoveScrobble(e.ID)
	}
	log.Printf("scrobble queue: flushed %d lastfm scrobbles", len(entries))
}

// flushListenBrainzQueue retries pending ListenBrainz scrobbles in batches of up to 100.
func (p *PlayerService) flushListenBrainzQueue() {
	lbCfg, err := p.db.LoadListenBrainzConfig()
	if err != nil || !lbCfg.Enabled || lbCfg.UserToken == "" {
		return
	}

	entries, err := p.db.PendingScrobbles("listenbrainz", 100)
	if err != nil || len(entries) == 0 {
		return
	}

	tracks := make([]listenbrainz.TrackInfo, len(entries))
	timestamps := make([]int64, len(entries))
	for i, e := range entries {
		var t scrobbleTrackJSON
		if err := json.Unmarshal([]byte(e.TrackJSON), &t); err != nil {
			log.Printf("scrobble queue: unmarshal listenbrainz entry %d: %v", e.ID, err)
			_ = p.db.RemoveScrobble(e.ID) // corrupted entry
			return
		}
		tracks[i] = listenbrainz.TrackInfo{
			Artist:     t.Artist,
			Track:      t.Track,
			Album:      t.Album,
			DurationMs: t.DurationMs,
		}
		timestamps[i] = e.Timestamp
	}

	if err := listenbrainz.ScrobbleBatch(lbCfg.UserToken, tracks, timestamps); err != nil {
		log.Printf("scrobble queue: listenbrainz batch: %v", err)
		for _, e := range entries {
			_ = p.db.MarkScrobbleAttempt(e.ID)
		}
		return
	}

	for _, e := range entries {
		_ = p.db.RemoveScrobble(e.ID)
	}
	log.Printf("scrobble queue: flushed %d listenbrainz scrobbles", len(entries))
}

// PlayRadio starts playback of a radio stream. It saves the current library
// queue and enters radio mode where next/prev/shuffle/repeat are disabled.
func (p *PlayerService) PlayRadio(stationName, streamURL, artworkURL string) error {
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}

	// Save the library queue if not already in radio mode.
	p.radioMu.Lock()
	if !p.radioMode {
		p.savedQueue = p.queue.Tracks()
		p.savedPosition = p.queue.Position()
	}
	p.radioMode = true
	p.radioName = stationName
	p.radioStreamURL = streamURL
	p.radioArtworkURL = artworkURL
	p.radioLastTitle = ""
	p.radioMu.Unlock()

	// Clear the queue and play the stream directly.
	p.queue.Clear()
	if err := p.engine.Play(streamURL); err != nil {
		return fmt.Errorf("play radio: %w", err)
	}

	if p.mpris != nil {
		p.mpris.UpdateMetadata(stationName, "Radio", "", streamURL, 0, 0)
		p.mpris.UpdatePlaybackStatus("playing")
	}
	if p.onTrayUpdate != nil {
		p.onTrayUpdate(stationName, "Radio")
	}
	if p.notifier != nil {
		p.notifier.Notify(stationName, "Radio", nil)
	}

	return nil
}

// StopRadio stops the current radio stream and restores the library queue.
func (p *PlayerService) StopRadio() {
	p.radioMu.Lock()
	if !p.radioMode {
		p.radioMu.Unlock()
		return
	}

	p.radioMode = false
	p.radioName = ""
	p.radioStreamURL = ""
	p.radioArtworkURL = ""
	p.radioLastTitle = ""
	savedQueue := p.savedQueue
	savedPosition := p.savedPosition
	p.savedQueue = nil
	p.savedPosition = 0
	p.radioMu.Unlock()

	if p.engine != nil {
		p.engine.Stop()
	}

	// Restore the library queue.
	if len(savedQueue) > 0 {
		pos := savedPosition
		if pos < 0 || pos >= len(savedQueue) {
			pos = 0
		}
		p.queue.Replace(savedQueue, pos)
	}

	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("stopped")
		p.mpris.ClearMetadata()
	}
	if p.onTrayUpdate != nil {
		p.onTrayUpdate("", "")
	}
}

// checkRadioTitle detects ICY stream title changes during radio playback
// and updates MPRIS metadata, tray tooltip, and desktop notifications.
func (p *PlayerService) checkRadioTitle() {
	p.radioMu.RLock()
	if !p.radioMode || p.engine == nil {
		p.radioMu.RUnlock()
		return
	}
	streamURL := p.radioStreamURL
	name := p.radioName
	lastTitle := p.radioLastTitle
	p.radioMu.RUnlock()

	t := p.engine.MediaTitle()
	// Filter out raw stream URL (shown when no ICY metadata is available).
	if t == streamURL {
		t = ""
	}

	if t == lastTitle {
		return
	}

	p.radioMu.Lock()
	p.radioLastTitle = t
	p.radioMu.Unlock()

	if p.mpris != nil {
		artist := "Radio"
		if t != "" {
			artist = t
		}
		p.mpris.UpdateMetadata(name, artist, "", streamURL, 0, 0)
	}

	if p.onTrayUpdate != nil {
		if t != "" {
			p.onTrayUpdate(name, t)
		} else {
			p.onTrayUpdate(name, "Radio")
		}
	}

	if p.notifier != nil && t != "" {
		p.notifier.Notify(name, t, nil)
	}
}

// IsRadioMode returns whether the player is currently in radio mode.
func (p *PlayerService) IsRadioMode() bool {
	p.radioMu.RLock()
	defer p.radioMu.RUnlock()

	return p.radioMode
}

// RadioStationName returns the name of the currently playing radio station.
func (p *PlayerService) RadioStationName() string {
	p.radioMu.RLock()
	defer p.radioMu.RUnlock()

	return p.radioName
}

// RadioArtworkURL returns the artwork URL of the currently playing radio station.
func (p *PlayerService) RadioArtworkURL() string {
	p.radioMu.RLock()
	defer p.radioMu.RUnlock()

	return p.radioArtworkURL
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
