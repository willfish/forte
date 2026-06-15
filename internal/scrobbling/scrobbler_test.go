package scrobbling

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/willfish/forte/internal/library"
)

type fakeStore struct {
	lastfmCfg       library.ScrobbleConfig
	listenbrainzCfg library.ListenBrainzConfig
	recordedTrackID int64
	recordedMs      int
	enqueuedService string
	enqueuedJSON    string
	enqueuedTS      int64
	pending         map[string][]library.ScrobbleQueueEntry
	removed         []int64
	marked          []int64
	pruned          bool
}

func (s *fakeStore) LoadScrobbleConfig() (library.ScrobbleConfig, error) {
	return s.lastfmCfg, nil
}

func (s *fakeStore) LoadListenBrainzConfig() (library.ListenBrainzConfig, error) {
	return s.listenbrainzCfg, nil
}

func (s *fakeStore) RecordPlay(trackID int64, durationPlayedMs int) error {
	s.recordedTrackID = trackID
	s.recordedMs = durationPlayedMs
	return nil
}

func (s *fakeStore) EnqueueScrobble(service, trackJSON string, timestamp int64) error {
	s.enqueuedService = service
	s.enqueuedJSON = trackJSON
	s.enqueuedTS = timestamp
	return nil
}

func (s *fakeStore) PruneScrobbleQueue() error {
	s.pruned = true
	return nil
}

func (s *fakeStore) PendingScrobbles(service string, limit int) ([]library.ScrobbleQueueEntry, error) {
	entries := s.pending[service]
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return entries, nil
}

func (s *fakeStore) RemoveScrobble(id int64) error {
	s.removed = append(s.removed, id)
	return nil
}

func (s *fakeStore) MarkScrobbleAttempt(id int64) error {
	s.marked = append(s.marked, id)
	return nil
}

type fakeLastFM struct {
	scrobbleErr      error
	scrobbleBatchErr error
	scrobbles        int
	batches          [][]QueuedTrack
}

func (f *fakeLastFM) NowPlaying(library.ScrobbleConfig, Track) error { return nil }

func (f *fakeLastFM) Scrobble(library.ScrobbleConfig, Track, int64) error {
	f.scrobbles++
	return f.scrobbleErr
}

func (f *fakeLastFM) ScrobbleBatch(_ library.ScrobbleConfig, tracks []QueuedTrack) error {
	f.batches = append(f.batches, tracks)
	return f.scrobbleBatchErr
}

type fakeListenBrainz struct{}

func (fakeListenBrainz) NowPlaying(library.ListenBrainzConfig, Track) error { return nil }

func (fakeListenBrainz) Scrobble(library.ListenBrainzConfig, Track, int64) error { return nil }

func (fakeListenBrainz) ScrobbleBatch(library.ListenBrainzConfig, []QueuedTrack) error {
	return nil
}

func TestCoordinatorQueuesFailedLastFMScrobbleAfterThresholdOnce(t *testing.T) {
	store := &fakeStore{
		lastfmCfg: library.ScrobbleConfig{
			APIKey:     "key",
			APISecret:  "secret",
			SessionKey: "session",
			Enabled:    true,
		},
	}
	lastfm := &fakeLastFM{scrobbleErr: errors.New("offline")}
	now := time.Unix(1_700_000_000, 0)
	c := NewCoordinator(store, WithLastFM(lastfm), WithListenBrainz(fakeListenBrainz{}), WithNow(func() time.Time {
		return now
	}), WithAsync(func(fn func()) { fn() }))

	c.TrackStarted(Track{
		ID:         42,
		Artist:     "Nina Simone",
		Title:      "Feeling Good",
		Album:      "I Put a Spell on You",
		DurationMs: 4_000,
	})

	c.Tick("playing")
	c.Tick("playing")
	c.Tick("playing")

	if store.recordedTrackID != 42 {
		t.Fatalf("recorded track ID = %d, want 42", store.recordedTrackID)
	}
	if store.recordedMs != 2_000 {
		t.Fatalf("recorded duration = %d, want 2000", store.recordedMs)
	}
	if lastfm.scrobbles != 1 {
		t.Fatalf("Last.fm scrobbles = %d, want 1", lastfm.scrobbles)
	}
	if store.enqueuedService != "lastfm" {
		t.Fatalf("enqueued service = %q, want lastfm", store.enqueuedService)
	}
	if store.enqueuedTS != now.Unix() {
		t.Fatalf("enqueued timestamp = %d, want %d", store.enqueuedTS, now.Unix())
	}

	var queued struct {
		Artist     string `json:"artist"`
		Track      string `json:"track"`
		Album      string `json:"album"`
		DurationMs int    `json:"duration_ms"`
	}
	if err := json.Unmarshal([]byte(store.enqueuedJSON), &queued); err != nil {
		t.Fatalf("queued JSON: %v", err)
	}
	if queued.Artist != "Nina Simone" || queued.Track != "Feeling Good" || queued.Album != "I Put a Spell on You" || queued.DurationMs != 4_000 {
		t.Fatalf("queued track = %#v", queued)
	}
}

func TestCoordinatorFlushQueueRemovesSuccessfulLastFMEntries(t *testing.T) {
	store := &fakeStore{
		lastfmCfg: library.ScrobbleConfig{
			APIKey:     "key",
			APISecret:  "secret",
			SessionKey: "session",
			Enabled:    true,
		},
		pending: map[string][]library.ScrobbleQueueEntry{
			"lastfm": {
				{
					ID:        10,
					TrackJSON: `{"artist":"Bessie Smith","track":"Backwater Blues","album":"The Collection","duration_ms":180000}`,
					Timestamp: 1_700_000_010,
				},
			},
		},
	}
	lastfm := &fakeLastFM{}
	c := NewCoordinator(store, WithLastFM(lastfm), WithListenBrainz(fakeListenBrainz{}), WithAsync(func(fn func()) { fn() }))

	c.FlushQueue()

	if !store.pruned {
		t.Fatal("expected scrobble queue to be pruned")
	}
	if len(lastfm.batches) != 1 {
		t.Fatalf("Last.fm batches = %d, want 1", len(lastfm.batches))
	}
	got := lastfm.batches[0][0]
	if got.ID != 0 || got.Artist != "Bessie Smith" || got.Title != "Backwater Blues" || got.Album != "The Collection" || got.DurationMs != 180_000 || got.Timestamp != 1_700_000_010 {
		t.Fatalf("batched track = %#v", got)
	}
	if len(store.removed) != 1 || store.removed[0] != 10 {
		t.Fatalf("removed entries = %#v, want [10]", store.removed)
	}
	if len(store.marked) != 0 {
		t.Fatalf("marked entries = %#v, want none", store.marked)
	}
}
