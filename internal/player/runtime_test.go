package player

import (
	"reflect"
	"testing"
)

type fakeRuntimeEngine struct {
	playAllCalls        [][]string
	nextCalls           int
	previousCalls       int
	stopCalls           int
	seekCalls           []float64
	replaceUpcomingCall [][]string
	loopFile            []bool
	state               PlaybackState
	onTrackChange       func()
	onPlaylistEnd       func()
}

func (e *fakeRuntimeEngine) PlayAll(paths []string) error {
	e.playAllCalls = append(e.playAllCalls, append([]string(nil), paths...))
	e.state = StatePlaying
	return nil
}

func (e *fakeRuntimeEngine) Play(string) error {
	e.state = StatePlaying
	return nil
}

func (e *fakeRuntimeEngine) Enqueue(string) error { return nil }

func (e *fakeRuntimeEngine) Stop() {
	e.stopCalls++
	e.state = StateStopped
}

func (e *fakeRuntimeEngine) Pause()                { e.state = StatePaused }
func (e *fakeRuntimeEngine) Resume()               { e.state = StatePlaying }
func (e *fakeRuntimeEngine) Next()                 { e.nextCalls++ }
func (e *fakeRuntimeEngine) Previous()             { e.previousCalls++ }
func (e *fakeRuntimeEngine) Seek(seconds float64)  { e.seekCalls = append(e.seekCalls, seconds) }
func (e *fakeRuntimeEngine) SetVolume(int)         {}
func (e *fakeRuntimeEngine) Volume() int           { return 50 }
func (e *fakeRuntimeEngine) Position() float64     { return 12.5 }
func (e *fakeRuntimeEngine) Duration() float64     { return 180 }
func (e *fakeRuntimeEngine) State() PlaybackState  { return e.state }
func (e *fakeRuntimeEngine) MediaTitle() string    { return "" }
func (e *fakeRuntimeEngine) MediaArtist() string   { return "" }
func (e *fakeRuntimeEngine) MediaAlbum() string    { return "" }
func (e *fakeRuntimeEngine) MediaPath() string     { return "" }
func (e *fakeRuntimeEngine) SetLoopFile(loop bool) { e.loopFile = append(e.loopFile, loop) }
func (e *fakeRuntimeEngine) ReplaceUpcoming(paths []string) {
	e.replaceUpcomingCall = append(e.replaceUpcomingCall, append([]string(nil), paths...))
}
func (e *fakeRuntimeEngine) SetOnTrackChange(fn func()) { e.onTrackChange = fn }
func (e *fakeRuntimeEngine) SetOnPlaylistEnd(fn func()) { e.onPlaylistEnd = fn }

func TestRuntimePlayQueueReplacesQueueAndStartsPlayback(t *testing.T) {
	engine := &fakeRuntimeEngine{}
	runtime := NewRuntime(engine, RuntimeOptions{ResolvePaths: prefixResolved})
	tracks := testRuntimeTracks()

	if err := runtime.PlayQueue(tracks, 1); err != nil {
		t.Fatalf("PlayQueue() error: %v", err)
	}

	if got := runtime.QueuePosition(); got != 1 {
		t.Fatalf("QueuePosition() = %d, want 1", got)
	}
	wantPaths := []string{"resolved:/music/b.flac", "resolved:/music/c.flac"}
	if !reflect.DeepEqual(engine.playAllCalls, [][]string{wantPaths}) {
		t.Fatalf("PlayAll calls = %#v, want %#v", engine.playAllCalls, [][]string{wantPaths})
	}
}

func TestRuntimeExplicitSkipSuppressesNextTrackChange(t *testing.T) {
	engine := &fakeRuntimeEngine{}
	runtime := NewRuntime(engine, RuntimeOptions{ResolvePaths: prefixResolved})
	if err := runtime.PlayQueue(testRuntimeTracks(), 0); err != nil {
		t.Fatalf("PlayQueue() error: %v", err)
	}

	runtime.Next()
	engine.onTrackChange()

	if got := runtime.QueuePosition(); got != 1 {
		t.Fatalf("QueuePosition() after explicit skip callback = %d, want 1", got)
	}
	if engine.nextCalls != 1 {
		t.Fatalf("Next calls = %d, want 1", engine.nextCalls)
	}
}

func TestRuntimeTrackChangeAutoAdvancesQueue(t *testing.T) {
	engine := &fakeRuntimeEngine{}
	runtime := NewRuntime(engine, RuntimeOptions{ResolvePaths: prefixResolved})
	if err := runtime.PlayQueue(testRuntimeTracks(), 0); err != nil {
		t.Fatalf("PlayQueue() error: %v", err)
	}

	engine.onTrackChange()
	engine.onTrackChange()

	if got := runtime.QueuePosition(); got != 1 {
		t.Fatalf("QueuePosition() after auto track change = %d, want 1", got)
	}
}

func TestRuntimePlaylistEndRepeatAllWrapsAndReloads(t *testing.T) {
	engine := &fakeRuntimeEngine{}
	runtime := NewRuntime(engine, RuntimeOptions{ResolvePaths: prefixResolved})
	if err := runtime.PlayQueue(testRuntimeTracks(), 2); err != nil {
		t.Fatalf("PlayQueue() error: %v", err)
	}
	runtime.SetRepeat("all")

	engine.onPlaylistEnd()

	if got := runtime.QueuePosition(); got != 0 {
		t.Fatalf("QueuePosition() = %d, want 0", got)
	}
	wantReload := []string{"resolved:/music/a.flac", "resolved:/music/b.flac", "resolved:/music/c.flac"}
	if !reflect.DeepEqual(engine.playAllCalls[len(engine.playAllCalls)-1], wantReload) {
		t.Fatalf("last PlayAll = %#v, want %#v", engine.playAllCalls[len(engine.playAllCalls)-1], wantReload)
	}
}

func TestRuntimeSetShuffleRebuildsUpcomingPlaylist(t *testing.T) {
	engine := &fakeRuntimeEngine{}
	runtime := NewRuntime(engine, RuntimeOptions{ResolvePaths: prefixResolved})
	if err := runtime.PlayQueue(testRuntimeTracks(), 0); err != nil {
		t.Fatalf("PlayQueue() error: %v", err)
	}

	runtime.SetShuffle(true)

	if len(engine.replaceUpcomingCall) != 1 {
		t.Fatalf("ReplaceUpcoming calls = %d, want 1", len(engine.replaceUpcomingCall))
	}
	if got := engine.replaceUpcomingCall[0]; len(got) != 2 {
		t.Fatalf("ReplaceUpcoming paths = %#v, want two upcoming paths", got)
	}
}

func TestRuntimePlaylistEndWithoutRepeatStopsMetadata(t *testing.T) {
	engine := &fakeRuntimeEngine{}
	var stopped int
	runtime := NewRuntime(engine, RuntimeOptions{
		ResolvePaths: prefixResolved,
		OnStopped:    func() { stopped++ },
	})
	if err := runtime.PlayQueue(testRuntimeTracks(), 2); err != nil {
		t.Fatalf("PlayQueue() error: %v", err)
	}

	engine.onPlaylistEnd()

	if got := runtime.QueuePosition(); got != 2 {
		t.Fatalf("QueuePosition() = %d, want 2", got)
	}
	if stopped != 1 {
		t.Fatalf("OnStopped calls = %d, want 1", stopped)
	}
}

func testRuntimeTracks() []QueueTrack {
	return []QueueTrack{
		{TrackID: 1, Title: "A", FilePath: "/music/a.flac"},
		{TrackID: 2, Title: "B", FilePath: "/music/b.flac"},
		{TrackID: 3, Title: "C", FilePath: "/music/c.flac"},
	}
}

func prefixResolved(paths []string) []string {
	resolved := make([]string, len(paths))
	for i, path := range paths {
		resolved[i] = "resolved:" + path
	}
	return resolved
}
