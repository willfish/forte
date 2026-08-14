package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/metadata"
	"github.com/willfish/forte/internal/player"
	"github.com/willfish/forte/internal/scrobbling"
	"github.com/willfish/forte/internal/system"
)

// PlayerService exposes audio playback controls to the frontend.
type PlayerService struct {
	engine         *player.Engine
	runtime        *player.Runtime
	db             *library.DB
	resolver       *library.PathResolver
	scrobbler      *scrobbling.Coordinator
	mpris          *system.MPRIS
	notifier       *system.Notifier
	toasts         *player.Notifications
	isServerOnline func(string) bool          // set by main.go to check server health
	onTrayUpdate   func(title, artist string) // set by main.go for tooltip updates
	stopSave       chan struct{}
	tickerDone     chan struct{} // closed when the ticker goroutine exits
	shutdownOnce   sync.Once
	shutdownErr    error

	// Radio mode state (protected by radioMu).
	radioMu               sync.RWMutex
	radioMode             bool
	radioStationUUID      string
	radioName             string
	radioStreamURL        string
	radioArtworkURL       string
	radioHomepage         string
	radioTags             string
	radioLastTitle        string // last ICY stream title, for change detection
	radioReconnectPending bool
	radioReconnectGen     uint64 // bumped to cancel an in-flight reconnect loop
	savedQueue            []player.QueueTrack
	savedPosition         int
}

// ServiceStartup initialises the mpv engine when the application starts.
func (p *PlayerService) ServiceStartup(_ context.Context, _ application.ServiceOptions) error {
	e, err := player.NewEngine()
	if err != nil {
		return fmt.Errorf("player startup: %w", err)
	}
	p.engine = e
	p.runtime = player.NewRuntime(e, player.RuntimeOptions{
		ResolvePaths: p.resolvePaths,
		OnTrackChanged: func() {
			p.pushMPRISMetadata()
			p.startScrobbleTracking()
		},
		OnStopped: p.clearPlaybackMetadata,
	})
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
	p.scrobbler = scrobbling.NewCoordinator(db)

	// When mpv fails to play a stream, stop radio or skip past offline server tracks.
	e.SetOnStreamError(func() {
		p.radioMu.RLock()
		isRadio := p.radioMode
		p.radioMu.RUnlock()
		if isRadio {
			p.handleRadioStreamError()
			return
		}
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
	p.startLastRadioStation()

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

func (p *PlayerService) startLastRadioStation() {
	if p.db == nil {
		return
	}
	prefs, err := p.db.GetAppPreferences()
	if err != nil || !prefs.StartLastStation {
		return
	}
	history, err := p.db.GetRadioHistory(1)
	if err != nil || len(history) == 0 {
		return
	}
	last := history[0]
	go func() {
		if err := p.playRadioStation(last.StationUUID, last.Name, last.StreamURL, last.FaviconURL, last.Homepage, last.Tags, last.Country, last.Codec, last.Bitrate, false, true); err != nil {
			log.Printf("start last radio station: %v", err)
		}
	}()
}

// ServiceShutdown cleans up the mpv engine when the application exits.
func (p *PlayerService) ServiceShutdown() error {
	p.shutdownOnce.Do(func() {
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
			p.shutdownErr = p.db.Close()
		}
	})
	return p.shutdownErr
}

func (p *PlayerService) pushMPRISMetadata() {
	p.radioMu.RLock()
	radioMode := p.radioMode
	p.radioMu.RUnlock()
	if radioMode {
		return
	}
	if p.runtime == nil {
		return
	}
	cur := p.runtime.CurrentTrack()
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

func (p *PlayerService) clearPlaybackMetadata() {
	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("stopped")
		p.mpris.ClearMetadata()
	}
}

func (p *PlayerService) saveState() {
	if p.db == nil || p.runtime == nil {
		return
	}

	p.radioMu.RLock()
	radioMode := p.radioMode
	savedQueue := p.savedQueue
	savedPosition := p.savedPosition
	p.radioMu.RUnlock()

	tracks := p.runtime.QueueTracks()
	position := p.runtime.QueuePosition()
	if radioMode {
		tracks = savedQueue
		position = savedPosition
	}

	queueJSON, err := json.Marshal(tracks)
	if err != nil {
		log.Printf("save state: marshal queue: %v", err)
		return
	}

	var posMs int
	if p.engine != nil && !radioMode {
		posMs = int(p.engine.Position() * 1000)
	}

	vol := 100
	if p.engine != nil {
		vol = p.engine.Volume()
	}

	state := library.PlaybackState{
		QueueJSON:       string(queueJSON),
		Position:        position,
		TrackPositionMs: posMs,
		Volume:          vol,
		Shuffle:         p.runtime.Shuffled(),
		RepeatMode:      p.runtime.Repeat(),
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
	if p.runtime == nil {
		return
	}
	p.runtime.ReplaceQueue(valid, pos)

	repeatMode := "off"
	switch state.RepeatMode {
	case "all":
		repeatMode = "all"
	case "one":
		repeatMode = "one"
	}
	p.runtime.SetRepeat(repeatMode)

	if state.Shuffle {
		p.runtime.SetShuffle(true)
	}

	// Set volume and repeat-one loop on the engine.
	if p.engine != nil {
		p.engine.SetVolume(state.Volume)

		// Load the playlist but start paused.
		paths := p.runtime.QueuePaths(pos)
		if len(paths) > 0 {
			if err := p.runtime.PlayFromQueuePosition(); err == nil {
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
	if p.runtime == nil {
		return fmt.Errorf("player not initialised")
	}
	p.leaveRadioModeWithoutRestore()
	err := p.runtime.PlayQueue(tracks, startAt)
	p.pushMPRISMetadata()
	p.startScrobbleTracking()
	return err
}

// leaveRadioModeWithoutRestore drops radio chrome without putting the
// pre-radio library queue back. Used when the user starts a library queue.
func (p *PlayerService) leaveRadioModeWithoutRestore() {
	p.radioMu.Lock()
	if !p.radioMode {
		p.radioMu.Unlock()
		return
	}
	p.radioMode = false
	p.radioStationUUID = ""
	p.radioName = ""
	p.radioStreamURL = ""
	p.radioArtworkURL = ""
	p.radioHomepage = ""
	p.radioTags = ""
	p.radioLastTitle = ""
	p.radioReconnectPending = false
	p.savedQueue = nil
	p.savedPosition = 0
	p.radioMu.Unlock()
}

// QueueAppend adds a track to the end of the queue.
// If nothing is playing, it does not start playback.
func (p *PlayerService) QueueAppend(track player.QueueTrack) {
	if p.runtime != nil {
		p.runtime.QueueAppend(track)
	}
}

// QueueInsertNext inserts a track immediately after the current track.
func (p *PlayerService) QueueInsertNext(track player.QueueTrack) {
	if p.runtime != nil {
		p.runtime.QueueInsertNext(track)
	}
}

// QueueRemove removes the track at the given index.
// If the removed track was the current track, playback restarts from
// the new current position.
func (p *PlayerService) QueueRemove(index int) error {
	if p.runtime == nil {
		return nil
	}
	return p.runtime.QueueRemove(index)
}

// QueueMove moves a track from one position to another.
func (p *PlayerService) QueueMove(from, to int) {
	if p.runtime != nil {
		p.runtime.QueueMove(from, to)
	}
}

// QueueClear clears the queue and stops playback.
func (p *PlayerService) QueueClear() {
	if p.runtime != nil {
		p.runtime.QueueClear()
	}
	if p.onTrayUpdate != nil {
		p.onTrayUpdate("", "")
	}
}

// GetQueue returns all tracks in the queue.
func (p *PlayerService) GetQueue() []player.QueueTrack {
	if p.runtime == nil {
		return nil
	}
	return p.runtime.QueueTracks()
}

// GetQueuePosition returns the current queue position (-1 if empty).
func (p *PlayerService) GetQueuePosition() int {
	if p.runtime == nil {
		return -1
	}
	return p.runtime.QueuePosition()
}

// SetShuffle enables or disables shuffle mode.
// When toggled, the mpv playlist is rebuilt to match the new order.
func (p *PlayerService) SetShuffle(enabled bool) {
	if p.runtime != nil {
		p.runtime.SetShuffle(enabled)
	}
	if p.mpris != nil {
		p.mpris.UpdateShuffle(enabled)
	}
}

// GetShuffle returns whether shuffle mode is active.
func (p *PlayerService) GetShuffle() bool {
	if p.runtime == nil {
		return false
	}
	return p.runtime.Shuffled()
}

// SetRepeat sets the repeat mode: "off", "all", or "one".
func (p *PlayerService) SetRepeat(mode string) {
	if p.runtime != nil {
		p.runtime.SetRepeat(mode)
	}
	if p.mpris != nil {
		p.mpris.UpdateLoopStatus(mode)
	}
}

// GetRepeat returns the current repeat mode as a string.
func (p *PlayerService) GetRepeat() string {
	if p.runtime == nil {
		return "off"
	}
	return p.runtime.Repeat()
}

// Play starts playback of the audio file at the given path.
func (p *PlayerService) Play(path string) error {
	if p.runtime == nil {
		return fmt.Errorf("player not initialised")
	}
	return p.runtime.Play(path)
}

// Enqueue appends a track to the playlist for gapless playback.
func (p *PlayerService) Enqueue(path string) error {
	if p.runtime == nil {
		return fmt.Errorf("player not initialised")
	}
	return p.runtime.Enqueue(path)
}

// PlayAll replaces the playlist and plays the given tracks in order.
func (p *PlayerService) PlayAll(paths []string) error {
	if p.runtime == nil {
		return fmt.Errorf("player not initialised")
	}
	return p.runtime.PlayAll(paths)
}

// Pause pauses the current playback.
func (p *PlayerService) Pause() {
	if p.runtime != nil {
		p.runtime.Pause()
	}
	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("paused")
	}
}

// Resume resumes paused playback.
func (p *PlayerService) Resume() {
	if p.restartStoppedRadio() {
		return
	}
	if p.runtime != nil {
		p.runtime.Resume()
	}
	if p.mpris != nil {
		p.mpris.UpdatePlaybackStatus("playing")
	}
}

func (p *PlayerService) restartStoppedRadio() bool {
	p.radioMu.RLock()
	radioMode := p.radioMode
	stationUUID := p.radioStationUUID
	name := p.radioName
	streamURL := p.radioStreamURL
	artworkURL := p.radioArtworkURL
	homepage := p.radioHomepage
	tags := p.radioTags
	p.radioMu.RUnlock()

	state := player.StateStopped
	if p.engine != nil {
		state = p.engine.State()
	}
	if !shouldRestartRadio(radioMode, streamURL, state) {
		return false
	}

	go func() {
		if err := p.playRadioStation(stationUUID, name, streamURL, artworkURL, homepage, tags, "", "", 0, false, true); err != nil {
			slog.Warn("radio restart failed", "station", name, "err", err)
		}
	}()
	return true
}

func shouldRestartRadio(radioMode bool, streamURL string, state player.PlaybackState) bool {
	return radioMode && streamURL != "" && state == player.StateStopped
}

// Stop halts the current playback.
func (p *PlayerService) Stop() {
	p.radioMu.RLock()
	isRadio := p.radioMode
	p.radioMu.RUnlock()
	if isRadio {
		p.StopRadio()
		return
	}

	if p.runtime != nil {
		p.runtime.Stop()
	}
	if p.onTrayUpdate != nil {
		p.onTrayUpdate("", "")
	}
}

// Seek seeks to the given position in seconds.
func (p *PlayerService) Seek(seconds float64) {
	if p.runtime != nil {
		p.runtime.Seek(seconds)
	}
}

// SetVolume sets the volume (0-100).
func (p *PlayerService) SetVolume(percent int) {
	if p.runtime != nil {
		p.runtime.SetVolume(percent)
	}
	if p.mpris != nil {
		p.mpris.UpdateVolume(percent)
	}
}

// Volume returns the current volume (0-100).
func (p *PlayerService) Volume() int {
	if p.runtime == nil {
		return 0
	}
	return p.runtime.Volume()
}

// Position returns the current playback position in seconds.
func (p *PlayerService) Position() float64 {
	if p.runtime == nil {
		return 0
	}
	return p.runtime.Position()
}

// Duration returns the duration of the current track in seconds.
func (p *PlayerService) Duration() float64 {
	if p.runtime == nil {
		return 0
	}
	return p.runtime.Duration()
}

// State returns the current playback state as a string.
func (p *PlayerService) State() string {
	if p.runtime == nil {
		return "stopped"
	}
	return p.runtime.State().String()
}

// MediaTitle returns the title of the currently playing track.
// In radio mode, filters out the raw stream URL (shown when no ICY metadata is available).
func (p *PlayerService) MediaTitle() string {
	if p.runtime == nil {
		return ""
	}
	t := p.runtime.MediaTitle()

	p.radioMu.RLock()
	isRadio := p.radioMode
	streamURL := p.radioStreamURL
	p.radioMu.RUnlock()

	if isRadio {
		return cleanRadioMediaTitle(t, streamURL)
	}
	return t
}

func cleanRadioMediaTitle(title, streamURL string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return ""
	}
	lowerTitle := strings.ToLower(t)
	lowerStreamURL := strings.ToLower(strings.TrimSpace(streamURL))
	if lowerStreamURL != "" && lowerTitle == lowerStreamURL {
		return ""
	}
	if strings.HasPrefix(lowerTitle, "http://") || strings.HasPrefix(lowerTitle, "https://") {
		return ""
	}
	for _, ext := range []string{".m3u8", ".m3u", ".pls", ".xspf", ".asx"} {
		if strings.Contains(lowerTitle, ext) {
			return ""
		}
	}
	if streamURL != "" {
		if u, err := url.Parse(streamURL); err == nil {
			base := strings.ToLower(path.Base(u.Path))
			if base != "." && base != "/" && base != "" && lowerTitle == base {
				return ""
			}
		}
	}
	return t
}

// MediaArtist returns the artist of the currently playing track.
func (p *PlayerService) MediaArtist() string {
	if p.runtime == nil {
		return ""
	}
	return p.runtime.MediaArtist()
}

// MediaAlbum returns the album of the currently playing track.
func (p *PlayerService) MediaAlbum() string {
	if p.runtime == nil {
		return ""
	}
	return p.runtime.MediaAlbum()
}

// MediaPath returns the file path of the currently playing track.
func (p *PlayerService) MediaPath() string {
	if p.runtime == nil {
		return ""
	}
	return p.runtime.MediaPath()
}

// logicalMediaPath is the path exposed over IPC. It is the queue's
// library path (local file or server://…), never mpv's resolved stream URL
// which may contain Subsonic/Jellyfin credentials.
func (p *PlayerService) logicalMediaPath() string {
	p.radioMu.RLock()
	radioMode := p.radioMode
	p.radioMu.RUnlock()
	if radioMode {
		return ""
	}
	if p.runtime == nil {
		return ""
	}
	cur := p.runtime.CurrentTrack()
	if cur == nil {
		return ""
	}
	return cur.FilePath
}

// Next skips to the next track in the queue.
func (p *PlayerService) Next() {
	if p.runtime != nil {
		p.runtime.Next()
	}
}

// Previous skips to the previous track in the queue.
func (p *PlayerService) Previous() {
	if p.runtime != nil {
		p.runtime.Previous()
	}
}

// Artwork returns the album artwork for the currently playing track
// as a base64-encoded data URI, or an empty string if unavailable.
func (p *PlayerService) Artwork() string {
	if p.runtime == nil {
		return ""
	}
	// Use the queue's file_path (which may be server://) rather than engine's media path.
	cur := p.runtime.CurrentTrack()
	if cur == nil {
		return ""
	}
	if library.IsServerPath(cur.FilePath) {
		// For server tracks, look up artwork from the album in the DB.
		return p.serverTrackArtwork(cur.TrackID)
	}
	path := p.runtime.MediaPath()
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
	if p.runtime == nil || p.engine == nil {
		return
	}
	cur := p.runtime.CurrentTrack()
	if cur != nil && library.IsServerPath(cur.FilePath) {
		serverID, _, _ := library.ParseServerPath(cur.FilePath)
		if p.isServerOnline != nil && !p.isServerOnline(serverID) {
			p.toasts.Push(fmt.Sprintf("Skipped \"%s\" - server offline", cur.Title), "warn")
		} else {
			p.toasts.Push(fmt.Sprintf("Failed to play \"%s\"", cur.Title), "error")
		}
	}

	// Try advancing through the queue to find a playable track.
	maxAttempts := p.runtime.QueueLen()
	for range maxAttempts {
		if !p.runtime.AdvanceQueue() {
			break
		}
		next := p.runtime.CurrentTrack()
		if next == nil {
			break
		}
		if !library.IsServerPath(next.FilePath) {
			// Local track, play it.
			if len(p.runtime.QueuePaths(p.runtime.QueuePosition())) > 0 {
				_ = p.runtime.PlayFromQueuePosition()
				p.pushMPRISMetadata()
			}
			return
		}
		serverID, _, _ := library.ParseServerPath(next.FilePath)
		if p.isServerOnline == nil || p.isServerOnline(serverID) {
			// Server track on an online server, try it.
			if len(p.runtime.QueuePaths(p.runtime.QueuePosition())) > 0 {
				_ = p.runtime.PlayFromQueuePosition()
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
// sends now-playing notifications if configured.
func (p *PlayerService) startScrobbleTracking() {
	if p.scrobbler == nil {
		return
	}
	if p.runtime == nil {
		return
	}
	cur := p.runtime.CurrentTrack()
	if cur == nil {
		return
	}
	p.scrobbler.TrackStarted(scrobbling.Track{
		ID:         cur.TrackID,
		Artist:     cur.Artist,
		Title:      cur.Title,
		Album:      cur.Album,
		DurationMs: cur.DurationMs,
	})
}

// checkScrobble accumulates play time and submits a scrobble when the
// threshold is reached (50% of duration or 4 minutes, whichever is first).
func (p *PlayerService) checkScrobble() {
	if p.scrobbler != nil {
		p.scrobbler.Tick(p.State())
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

// flushScrobbleQueue retries pending scrobbles for all services.
func (p *PlayerService) flushScrobbleQueue() {
	if p.scrobbler != nil {
		p.scrobbler.FlushQueue()
	}
}

// PlayRadio starts playback of a radio stream. It saves the current library
// queue and enters radio mode where next/prev/shuffle/repeat are disabled.
func (p *PlayerService) PlayRadio(stationName, streamURL, artworkURL string) error {
	return p.playRadioStation(library.CustomRadioStationUUID(streamURL), stationName, streamURL, artworkURL, "", "", "", "", 0, true, true)
}

// PlayRadioStation starts playback of a radio stream with stable station metadata.
func (p *PlayerService) PlayRadioStation(stationUUID, stationName, streamURL, artworkURL, homepage, tags, country, codec string, bitrate int) error {
	return p.playRadioStation(stationUUID, stationName, streamURL, artworkURL, homepage, tags, country, codec, bitrate, true, true)
}

func (p *PlayerService) playRadioStation(stationUUID, stationName, streamURL, artworkURL, homepage, tags, country, codec string, bitrate int, countPlay, resetReconnect bool) error {
	if stationUUID == "" {
		stationUUID = library.CustomRadioStationUUID(streamURL)
	}
	if p.engine == nil {
		return fmt.Errorf("player not initialised")
	}

	p.radioMu.Lock()
	if !p.radioMode {
		if p.runtime != nil {
			p.savedQueue = p.runtime.QueueTracks()
			p.savedPosition = p.runtime.QueuePosition()
		}
	}
	p.radioMode = true
	p.radioStationUUID = stationUUID
	p.radioName = stationName
	p.radioStreamURL = streamURL
	p.radioArtworkURL = artworkURL
	p.radioHomepage = homepage
	p.radioTags = tags
	p.radioLastTitle = ""
	if resetReconnect {
		p.radioReconnectPending = false
		p.radioReconnectGen++
	}
	p.radioMu.Unlock()
	p.engine.SetReconnectOnEOF(true)

	if artworkURL != "" {
		go p.fetchRadioArtwork(artworkURL)
	}

	if p.runtime != nil {
		p.runtime.QueueClear()
	}
	if err := p.engine.Play(streamURL); err != nil {
		return fmt.Errorf("play radio: %w", err)
	}

	if countPlay && p.db != nil {
		_ = p.db.RecordRadioPlayback(library.RadioHistoryEntry{
			StationUUID: stationUUID,
			Name:        stationName,
			StreamURL:   streamURL,
			FaviconURL:  artworkURL,
			Homepage:    homepage,
			Country:     country,
			Codec:       codec,
			Bitrate:     bitrate,
			Tags:        tags,
		})
	}

	if p.mpris != nil {
		p.mpris.UpdateMetadata(stationName, "Radio", "", streamURL, 0, 0)
		p.mpris.UpdatePlaybackStatus("playing")
	}
	if p.onTrayUpdate != nil {
		p.onTrayUpdate(stationName, "Radio")
	}
	if p.notifier != nil && countPlay {
		p.notifier.Notify(stationName, "Radio", nil)
	}

	return nil
}

// fetchRadioArtwork downloads the artwork URL and stores it as a data: URI
// so the frontend never makes external HTTP requests through WebKit.
func (p *PlayerService) fetchRadioArtwork(url string) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url) //nolint:noctx // fire-and-forget background fetch
	if err != nil {
		return // leave original URL as fallback
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return
	}

	const maxSize = 1 << 20 // 1 MB
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil || len(data) == 0 {
		return
	}

	mime := resp.Header.Get("Content-Type")
	if mime == "" {
		mime = "image/png"
	}
	dataURI := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)

	p.radioMu.Lock()
	// Only update if we're still playing the same station.
	if p.radioMode && p.radioArtworkURL == url {
		p.radioArtworkURL = dataURI
	}
	p.radioMu.Unlock()
}

// StopRadio stops the current radio stream and restores the library queue.
func (p *PlayerService) StopRadio() {
	p.radioMu.Lock()
	if !p.radioMode {
		p.radioMu.Unlock()
		return
	}

	p.radioMode = false
	p.radioStationUUID = ""
	p.radioName = ""
	p.radioStreamURL = ""
	p.radioArtworkURL = ""
	p.radioHomepage = ""
	p.radioTags = ""
	p.radioLastTitle = ""
	p.radioReconnectPending = false
	p.radioReconnectGen++
	savedQueue := p.savedQueue
	savedPosition := p.savedPosition
	p.savedQueue = nil
	p.savedPosition = 0
	p.radioMu.Unlock()

	if p.engine != nil {
		p.engine.SetReconnectOnEOF(false)
		p.engine.Stop()
	}

	// Restore the library queue.
	if len(savedQueue) > 0 {
		pos := savedPosition
		if pos < 0 || pos >= len(savedQueue) {
			pos = 0
		}
		if p.runtime != nil {
			p.runtime.ReplaceQueue(savedQueue, pos)
		}
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

	t := cleanRadioMediaTitle(p.engine.MediaTitle(), streamURL)

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
	if p.db != nil {
		p.radioMu.RLock()
		stationUUID := p.radioStationUUID
		p.radioMu.RUnlock()
		if stationUUID != "" {
			_ = p.db.UpdateRadioHistoryTitle(stationUUID, t)
		}
	}
}

func (p *PlayerService) handleRadioStreamError() {
	p.radioMu.Lock()
	if !p.radioMode {
		p.radioMu.Unlock()
		return
	}
	if p.radioReconnectPending {
		p.radioMu.Unlock()
		return
	}
	p.radioReconnectPending = true
	gen := p.radioReconnectGen
	stationUUID := p.radioStationUUID
	name := p.radioName
	streamURL := p.radioStreamURL
	artworkURL := p.radioArtworkURL
	homepage := p.radioHomepage
	tags := p.radioTags
	p.radioMu.Unlock()

	if p.db != nil && stationUUID != "" {
		_ = p.db.MarkRadioHistoryError(stationUUID, "stream lost")
	}

	prefs := library.AppPreferences{AutoReconnect: true}
	if p.db != nil {
		if loaded, err := p.db.GetAppPreferences(); err == nil {
			prefs = loaded
		}
	}
	if !prefs.AutoReconnect {
		slog.Warn("radio stream lost", "station", name)
		p.toasts.Push("Radio stream lost", "warn")
		p.StopRadio()
		return
	}

	slog.Warn("radio stream lost, reconnecting", "station", name)
	p.toasts.Push("Radio stream lost, reconnecting...", "warn")
	go func() {
		for attempt := 1; ; attempt++ {
			delay := player.RadioReconnectDelay(attempt, rand.Float64())
			time.Sleep(delay)
			if !p.radioReconnectStillCurrent(stationUUID, gen) {
				return
			}
			slog.Warn("radio reconnect attempt", "station", name, "attempt", attempt, "delay", delay)
			if err := p.playRadioStation(stationUUID, name, streamURL, artworkURL, homepage, tags, "", "", 0, false, false); err != nil {
				slog.Warn("radio reconnect failed", "station", name, "attempt", attempt, "err", err)
				continue
			}
			p.radioMu.Lock()
			p.radioReconnectPending = false
			p.radioMu.Unlock()
			slog.Warn("radio reconnected", "station", name, "attempt", attempt)
			p.toasts.Push("Radio reconnected", "info")
			return
		}
	}()
}

func (p *PlayerService) radioReconnectStillCurrent(stationUUID string, gen uint64) bool {
	p.radioMu.RLock()
	defer p.radioMu.RUnlock()
	return p.radioMode && p.radioStationUUID == stationUUID && p.radioReconnectGen == gen
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

// PlaybackStatus holds all frequently-polled playback state in a single struct,
// reducing Wails IPC round-trips from ~13/poll to 1/poll.
type PlaybackStatus struct {
	State        string  `json:"state"`
	Position     float64 `json:"position"`
	Duration     float64 `json:"duration"`
	Volume       int     `json:"volume"`
	Title        string  `json:"title"`
	Artist       string  `json:"artist"`
	Album        string  `json:"album"`
	Shuffle      bool    `json:"shuffle"`
	Repeat       string  `json:"repeat"`
	MediaPath    string  `json:"mediaPath"`
	RadioMode    bool    `json:"radioMode"`
	RadioUUID    string  `json:"radioUuid"`
	RadioStation string  `json:"radioStation"`
	RadioArtwork string  `json:"radioArtwork"`
}

// GetPlaybackStatus returns all frequently-polled playback state in a single
// IPC call. This reduces WebKit message churn from ~60 messages/s to ~4/s,
// working around a use-after-free bug in WebKitGTK 2.50.5's FormDataElement
// destructor that triggers under high IPC volume.
func (p *PlayerService) GetPlaybackStatus() PlaybackStatus {
	s := PlaybackStatus{
		State:     p.State(),
		Position:  p.Position(),
		Duration:  p.Duration(),
		Volume:    p.Volume(),
		Title:     p.MediaTitle(),
		Artist:    p.MediaArtist(),
		Album:     p.MediaAlbum(),
		MediaPath: p.logicalMediaPath(),
		Shuffle:   p.GetShuffle(),
		Repeat:    p.GetRepeat(),
	}

	p.radioMu.RLock()
	s.RadioMode = p.radioMode
	s.RadioUUID = p.radioStationUUID
	s.RadioStation = p.radioName
	s.RadioArtwork = p.radioArtworkURL
	p.radioMu.RUnlock()

	return s
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
