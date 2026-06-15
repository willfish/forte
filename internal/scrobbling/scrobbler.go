// Package scrobbling coordinates playback events with configured scrobble services.
package scrobbling

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/willfish/forte/internal/library"
	"github.com/willfish/forte/internal/scrobbling/lastfm"
	"github.com/willfish/forte/internal/scrobbling/listenbrainz"
)

// Track is the playback metadata needed for scrobbling.
type Track struct {
	ID         int64
	Artist     string
	Title      string
	Album      string
	DurationMs int
}

// QueuedTrack is a failed scrobble ready to retry.
type QueuedTrack struct {
	Track
	Timestamp int64
}

// Store is the persistence interface used by Coordinator.
type Store interface {
	LoadScrobbleConfig() (library.ScrobbleConfig, error)
	LoadListenBrainzConfig() (library.ListenBrainzConfig, error)
	RecordPlay(trackID int64, durationPlayedMs int) error
	EnqueueScrobble(service, trackJSON string, timestamp int64) error
	PruneScrobbleQueue() error
	PendingScrobbles(service string, limit int) ([]library.ScrobbleQueueEntry, error)
	RemoveScrobble(id int64) error
	MarkScrobbleAttempt(id int64) error
}

// LastFM submits Last.fm now-playing and scrobble requests.
type LastFM interface {
	NowPlaying(library.ScrobbleConfig, Track) error
	Scrobble(library.ScrobbleConfig, Track, int64) error
	ScrobbleBatch(library.ScrobbleConfig, []QueuedTrack) error
}

// ListenBrainz submits ListenBrainz now-playing and scrobble requests.
type ListenBrainz interface {
	NowPlaying(library.ListenBrainzConfig, Track) error
	Scrobble(library.ListenBrainzConfig, Track, int64) error
	ScrobbleBatch(library.ListenBrainzConfig, []QueuedTrack) error
}

// Coordinator tracks playback progress and submits scrobbles.
type Coordinator struct {
	store        Store
	lastfm       LastFM
	listenbrainz ListenBrainz
	now          func() time.Time
	async        func(func())

	mu        sync.Mutex
	current   Track
	elapsed   time.Duration
	scrobbled bool
}

// Option configures a Coordinator.
type Option func(*Coordinator)

// WithLastFM replaces the Last.fm adapter.
func WithLastFM(client LastFM) Option {
	return func(c *Coordinator) {
		c.lastfm = client
	}
}

// WithListenBrainz replaces the ListenBrainz adapter.
func WithListenBrainz(client ListenBrainz) Option {
	return func(c *Coordinator) {
		c.listenbrainz = client
	}
}

// WithNow replaces the clock used for scrobble timestamps.
func WithNow(now func() time.Time) Option {
	return func(c *Coordinator) {
		c.now = now
	}
}

// WithAsync replaces asynchronous execution. Tests can run functions inline.
func WithAsync(async func(func())) Option {
	return func(c *Coordinator) {
		c.async = async
	}
}

// NewCoordinator creates a scrobbling coordinator backed by store.
func NewCoordinator(store Store, opts ...Option) *Coordinator {
	c := &Coordinator{
		store:        store,
		lastfm:       lastFMAdapter{},
		listenbrainz: listenBrainzAdapter{},
		now:          time.Now,
		async: func(fn func()) {
			go fn()
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// TrackStarted resets progress and sends now-playing notifications.
func (c *Coordinator) TrackStarted(track Track) {
	c.mu.Lock()
	c.current = track
	c.elapsed = 0
	c.scrobbled = false
	c.mu.Unlock()

	if c.store == nil {
		return
	}

	if cfg, err := c.store.LoadScrobbleConfig(); err == nil && cfg.Enabled && cfg.SessionKey != "" {
		c.async(func() {
			if err := c.lastfm.NowPlaying(cfg, track); err != nil {
				log.Printf("lastfm now-playing: %v", err)
			}
		})
	}
	if cfg, err := c.store.LoadListenBrainzConfig(); err == nil && cfg.Enabled && cfg.UserToken != "" {
		c.async(func() {
			if err := c.listenbrainz.NowPlaying(cfg, track); err != nil {
				log.Printf("listenbrainz now-playing: %v", err)
			}
		})
	}
}

// Tick advances scrobble progress by one second when playback is active.
func (c *Coordinator) Tick(playbackState string) {
	c.mu.Lock()
	if playbackState != "playing" || c.scrobbled || c.current.ID == 0 {
		c.mu.Unlock()
		return
	}
	c.elapsed += time.Second
	track := c.current
	threshold := time.Duration(lastfm.ScrobbleThreshold(track.DurationMs)) * time.Millisecond
	if threshold <= 0 || c.elapsed < threshold {
		c.mu.Unlock()
		return
	}

	c.scrobbled = true
	elapsedMs := int(c.elapsed.Milliseconds())
	c.mu.Unlock()
	if c.store == nil {
		return
	}
	_ = c.store.RecordPlay(track.ID, elapsedMs)

	ts := c.now().Unix()
	if cfg, err := c.store.LoadScrobbleConfig(); err == nil && cfg.Enabled && cfg.SessionKey != "" {
		c.async(func() {
			if err := c.lastfm.Scrobble(cfg, track, ts); err != nil {
				log.Printf("lastfm scrobble: %v (queued for retry)", err)
				c.enqueueFailed("lastfm", track, ts)
			}
		})
	}
	if cfg, err := c.store.LoadListenBrainzConfig(); err == nil && cfg.Enabled && cfg.UserToken != "" {
		c.async(func() {
			if err := c.listenbrainz.Scrobble(cfg, track, ts); err != nil {
				log.Printf("listenbrainz scrobble: %v (queued for retry)", err)
				c.enqueueFailed("listenbrainz", track, ts)
			}
		})
	}
}

// FlushQueue retries pending scrobbles for all configured services.
func (c *Coordinator) FlushQueue() {
	if c.store == nil {
		return
	}
	if err := c.store.PruneScrobbleQueue(); err != nil {
		log.Printf("scrobble queue: prune: %v", err)
	}
	c.flushLastFMQueue()
	c.flushListenBrainzQueue()
}

func (c *Coordinator) flushLastFMQueue() {
	cfg, err := c.store.LoadScrobbleConfig()
	if err != nil || !cfg.Enabled || cfg.SessionKey == "" {
		return
	}
	entries, err := c.store.PendingScrobbles("lastfm", 50)
	if err != nil || len(entries) == 0 {
		return
	}

	tracks, ok := c.queuedTracks(entries, "lastfm")
	if !ok {
		return
	}
	if err := c.lastfm.ScrobbleBatch(cfg, tracks); err != nil {
		log.Printf("scrobble queue: lastfm batch: %v", err)
		c.markAttempts(entries)
		return
	}
	c.removeEntries(entries)
	log.Printf("scrobble queue: flushed %d lastfm scrobbles", len(entries))
}

func (c *Coordinator) flushListenBrainzQueue() {
	cfg, err := c.store.LoadListenBrainzConfig()
	if err != nil || !cfg.Enabled || cfg.UserToken == "" {
		return
	}
	entries, err := c.store.PendingScrobbles("listenbrainz", 100)
	if err != nil || len(entries) == 0 {
		return
	}

	tracks, ok := c.queuedTracks(entries, "listenbrainz")
	if !ok {
		return
	}
	if err := c.listenbrainz.ScrobbleBatch(cfg, tracks); err != nil {
		log.Printf("scrobble queue: listenbrainz batch: %v", err)
		c.markAttempts(entries)
		return
	}
	c.removeEntries(entries)
	log.Printf("scrobble queue: flushed %d listenbrainz scrobbles", len(entries))
}

func (c *Coordinator) queuedTracks(entries []library.ScrobbleQueueEntry, service string) ([]QueuedTrack, bool) {
	tracks := make([]QueuedTrack, len(entries))
	for i, entry := range entries {
		var track scrobbleTrackJSON
		if err := json.Unmarshal([]byte(entry.TrackJSON), &track); err != nil {
			log.Printf("scrobble queue: unmarshal %s entry %d: %v", service, entry.ID, err)
			_ = c.store.RemoveScrobble(entry.ID)
			return nil, false
		}
		tracks[i] = QueuedTrack{
			Track: Track{
				Artist:     track.Artist,
				Title:      track.Track,
				Album:      track.Album,
				DurationMs: track.DurationMs,
			},
			Timestamp: entry.Timestamp,
		}
	}
	return tracks, true
}

func (c *Coordinator) markAttempts(entries []library.ScrobbleQueueEntry) {
	for _, entry := range entries {
		_ = c.store.MarkScrobbleAttempt(entry.ID)
	}
}

func (c *Coordinator) removeEntries(entries []library.ScrobbleQueueEntry) {
	for _, entry := range entries {
		_ = c.store.RemoveScrobble(entry.ID)
	}
}

func (c *Coordinator) enqueueFailed(service string, track Track, ts int64) {
	data, err := json.Marshal(scrobbleTrackJSON{
		Artist:     track.Artist,
		Track:      track.Title,
		Album:      track.Album,
		DurationMs: track.DurationMs,
	})
	if err != nil {
		log.Printf("scrobble queue: marshal: %v", err)
		return
	}
	if err := c.store.EnqueueScrobble(service, string(data), ts); err != nil {
		log.Printf("scrobble queue: enqueue: %v", err)
	}
}

type scrobbleTrackJSON struct {
	Artist     string `json:"artist"`
	Track      string `json:"track"`
	Album      string `json:"album"`
	DurationMs int    `json:"duration_ms"`
}

type lastFMAdapter struct{}

func (lastFMAdapter) NowPlaying(cfg library.ScrobbleConfig, track Track) error {
	return lastfm.NowPlaying(cfg.APIKey, cfg.APISecret, cfg.SessionKey, lastfm.TrackInfo{
		Artist:   track.Artist,
		Track:    track.Title,
		Album:    track.Album,
		Duration: track.DurationMs / 1000,
	})
}

func (lastFMAdapter) Scrobble(cfg library.ScrobbleConfig, track Track, ts int64) error {
	return lastfm.Scrobble(cfg.APIKey, cfg.APISecret, cfg.SessionKey, lastfm.TrackInfo{
		Artist:   track.Artist,
		Track:    track.Title,
		Album:    track.Album,
		Duration: track.DurationMs / 1000,
	}, ts)
}

func (lastFMAdapter) ScrobbleBatch(cfg library.ScrobbleConfig, tracks []QueuedTrack) error {
	infos := make([]lastfm.TrackInfo, len(tracks))
	timestamps := make([]int64, len(tracks))
	for i, track := range tracks {
		infos[i] = lastfm.TrackInfo{
			Artist:   track.Artist,
			Track:    track.Title,
			Album:    track.Album,
			Duration: track.DurationMs / 1000,
		}
		timestamps[i] = track.Timestamp
	}
	return lastfm.ScrobbleBatch(cfg.APIKey, cfg.APISecret, cfg.SessionKey, infos, timestamps)
}

type listenBrainzAdapter struct{}

func (listenBrainzAdapter) NowPlaying(cfg library.ListenBrainzConfig, track Track) error {
	return listenbrainz.NowPlaying(cfg.UserToken, listenbrainz.TrackInfo{
		Artist:     track.Artist,
		Track:      track.Title,
		Album:      track.Album,
		DurationMs: track.DurationMs,
	})
}

func (listenBrainzAdapter) Scrobble(cfg library.ListenBrainzConfig, track Track, ts int64) error {
	return listenbrainz.Scrobble(cfg.UserToken, listenbrainz.TrackInfo{
		Artist:     track.Artist,
		Track:      track.Title,
		Album:      track.Album,
		DurationMs: track.DurationMs,
	}, ts)
}

func (listenBrainzAdapter) ScrobbleBatch(cfg library.ListenBrainzConfig, tracks []QueuedTrack) error {
	infos := make([]listenbrainz.TrackInfo, len(tracks))
	timestamps := make([]int64, len(tracks))
	for i, track := range tracks {
		infos[i] = listenbrainz.TrackInfo{
			Artist:     track.Artist,
			Track:      track.Title,
			Album:      track.Album,
			DurationMs: track.DurationMs,
		}
		timestamps[i] = track.Timestamp
	}
	return listenbrainz.ScrobbleBatch(cfg.UserToken, infos, timestamps)
}
