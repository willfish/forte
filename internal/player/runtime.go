package player

import (
	"fmt"
	"sync/atomic"
)

// RuntimeEngine is the playback engine surface Runtime needs to coordinate
// queue state with engine commands.
type RuntimeEngine interface {
	Play(path string) error
	Enqueue(path string) error
	PlayAll(paths []string) error
	Pause()
	Resume()
	Stop()
	Seek(seconds float64)
	SetVolume(percent int)
	Volume() int
	Position() float64
	Duration() float64
	State() PlaybackState
	MediaTitle() string
	MediaArtist() string
	MediaAlbum() string
	MediaPath() string
	Next()
	Previous()
	SetLoopFile(loop bool)
	ReplaceUpcoming(paths []string)
	SetOnTrackChange(fn func())
	SetOnPlaylistEnd(fn func())
}

// RuntimeOptions configures adapter hooks around core playback orchestration.
type RuntimeOptions struct {
	ResolvePaths   func([]string) []string
	OnTrackChanged func()
	OnStopped      func()
}

// Runtime coordinates queue state, repeat/shuffle modes, and engine commands.
type Runtime struct {
	engine     RuntimeEngine
	queue      *Queue
	resolve    func([]string) []string
	manualSkip int32

	onTrackChanged func()
	onStopped      func()
}

// NewRuntime wires an engine into the queue runtime.
func NewRuntime(engine RuntimeEngine, opts RuntimeOptions) *Runtime {
	r := &Runtime{
		engine:         engine,
		queue:          NewQueue(),
		resolve:        opts.ResolvePaths,
		onTrackChanged: opts.OnTrackChanged,
		onStopped:      opts.OnStopped,
	}
	if r.resolve == nil {
		r.resolve = func(paths []string) []string { return paths }
	}
	if engine != nil {
		engine.SetOnTrackChange(r.handleTrackChange)
		engine.SetOnPlaylistEnd(r.handlePlaylistEnd)
	}
	return r
}

func (r *Runtime) handleTrackChange() {
	if atomic.CompareAndSwapInt32(&r.manualSkip, 1, 0) {
		return
	}
	r.queue.Next()
	if r.onTrackChanged != nil {
		r.onTrackChanged()
	}
}

func (r *Runtime) handlePlaylistEnd() {
	if r.queue.Repeat() != RepeatAll {
		if r.onStopped != nil {
			r.onStopped()
		}
		return
	}
	r.queue.SetPosition(0)
	paths := r.queue.Paths(0)
	if len(paths) == 0 {
		return
	}
	atomic.StoreInt32(&r.manualSkip, 1)
	_ = r.engine.PlayAll(r.resolve(paths))
}

// PlayQueue replaces the queue and starts playback from startAt.
func (r *Runtime) PlayQueue(tracks []QueueTrack, startAt int) error {
	if r.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	r.queue.Replace(tracks, startAt)
	paths := r.queue.Paths(startAt)
	if len(paths) == 0 {
		return nil
	}
	atomic.StoreInt32(&r.manualSkip, 1)
	return r.engine.PlayAll(r.resolve(paths))
}

func (r *Runtime) QueueAppend(track QueueTrack) {
	r.queue.Append(track)
	r.syncUpcoming()
}

func (r *Runtime) QueueInsertNext(track QueueTrack) {
	r.queue.InsertAfterCurrent(track)
	r.syncUpcoming()
}

func (r *Runtime) QueueRemove(index int) error {
	wasCurrent := r.queue.Remove(index)
	if r.engine == nil {
		return nil
	}
	if !wasCurrent {
		r.syncUpcoming()
		return nil
	}
	cur := r.queue.Current()
	if cur == nil {
		r.engine.Stop()
		return nil
	}
	paths := r.queue.Paths(r.queue.Position())
	if len(paths) == 0 {
		return nil
	}
	atomic.StoreInt32(&r.manualSkip, 1)
	return r.engine.PlayAll(r.resolve(paths))
}

func (r *Runtime) QueueMove(from, to int) {
	r.queue.Move(from, to)
	r.syncUpcoming()
}

func (r *Runtime) syncUpcoming() {
	if r.engine == nil {
		return
	}
	pos := r.queue.Position()
	if pos < 0 {
		return
	}
	r.engine.ReplaceUpcoming(r.resolve(r.queue.Paths(pos + 1)))
}

func (r *Runtime) QueueClear() {
	r.queue.Clear()
	if r.engine != nil {
		r.engine.Stop()
	}
	if r.onStopped != nil {
		r.onStopped()
	}
}

func (r *Runtime) QueueTracks() []QueueTrack {
	return r.queue.Tracks()
}

func (r *Runtime) QueuePosition() int {
	return r.queue.Position()
}

func (r *Runtime) CurrentTrack() *QueueTrack {
	return r.queue.Current()
}

func (r *Runtime) QueueLen() int {
	return r.queue.Len()
}

func (r *Runtime) QueuePaths(from int) []string {
	return r.queue.Paths(from)
}

func (r *Runtime) AdvanceQueue() bool {
	return r.queue.Next()
}

func (r *Runtime) PlayFromQueuePosition() error {
	if r.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	paths := r.queue.Paths(r.queue.Position())
	if len(paths) == 0 {
		return nil
	}
	atomic.StoreInt32(&r.manualSkip, 1)
	return r.engine.PlayAll(r.resolve(paths))
}

func (r *Runtime) ReplaceQueue(tracks []QueueTrack, startAt int) {
	r.queue.Replace(tracks, startAt)
}

func (r *Runtime) SetQueuePosition(pos int) {
	r.queue.SetPosition(pos)
}

func (r *Runtime) SetShuffle(enabled bool) {
	r.queue.SetShuffle(enabled)
	r.syncUpcoming()
}

func (r *Runtime) Shuffled() bool {
	return r.queue.Shuffled()
}

func (r *Runtime) SetRepeat(mode string) {
	var rm RepeatMode
	switch mode {
	case "all":
		rm = RepeatAll
	case "one":
		rm = RepeatOne
	default:
		rm = RepeatOff
	}
	r.queue.SetRepeat(rm)
	if r.engine != nil {
		r.engine.SetLoopFile(rm == RepeatOne)
	}
}

func (r *Runtime) Repeat() string {
	return r.queue.Repeat().String()
}

func (r *Runtime) Play(path string) error {
	if r.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	return r.engine.Play(r.resolve([]string{path})[0])
}

func (r *Runtime) Enqueue(path string) error {
	if r.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	return r.engine.Enqueue(r.resolve([]string{path})[0])
}

func (r *Runtime) PlayAll(paths []string) error {
	if r.engine == nil {
		return fmt.Errorf("player not initialised")
	}
	return r.engine.PlayAll(r.resolve(paths))
}

func (r *Runtime) Pause() {
	if r.engine != nil {
		r.engine.Pause()
	}
}

func (r *Runtime) Resume() {
	if r.engine != nil {
		r.engine.Resume()
	}
}

func (r *Runtime) Stop() {
	if r.engine != nil {
		r.engine.Stop()
	}
	if r.onStopped != nil {
		r.onStopped()
	}
}

func (r *Runtime) Seek(seconds float64) {
	if r.engine != nil {
		r.engine.Seek(seconds)
	}
}

func (r *Runtime) SetVolume(percent int) {
	if r.engine != nil {
		r.engine.SetVolume(percent)
	}
}

func (r *Runtime) Volume() int {
	if r.engine == nil {
		return 0
	}
	return r.engine.Volume()
}

func (r *Runtime) Position() float64 {
	if r.engine == nil {
		return 0
	}
	return r.engine.Position()
}

func (r *Runtime) Duration() float64 {
	if r.engine == nil {
		return 0
	}
	return r.engine.Duration()
}

func (r *Runtime) State() PlaybackState {
	if r.engine == nil {
		return StateStopped
	}
	return r.engine.State()
}

func (r *Runtime) MediaTitle() string {
	if r.engine == nil {
		return ""
	}
	return r.engine.MediaTitle()
}

func (r *Runtime) MediaArtist() string {
	if r.engine == nil {
		return ""
	}
	return r.engine.MediaArtist()
}

func (r *Runtime) MediaAlbum() string {
	if r.engine == nil {
		return ""
	}
	return r.engine.MediaAlbum()
}

func (r *Runtime) MediaPath() string {
	if r.engine == nil {
		return ""
	}
	return r.engine.MediaPath()
}

func (r *Runtime) Next() {
	if r.engine == nil {
		return
	}
	repeat := r.queue.Repeat()
	if repeat == RepeatOne {
		r.engine.Seek(0)
		return
	}
	if !r.queue.Next() {
		return
	}
	atomic.StoreInt32(&r.manualSkip, 1)
	if repeat == RepeatAll && r.queue.Position() == 0 {
		paths := r.queue.Paths(0)
		if len(paths) > 0 {
			_ = r.engine.PlayAll(r.resolve(paths))
		}
		return
	}
	r.engine.Next()
}

func (r *Runtime) Previous() {
	if r.engine == nil {
		return
	}
	repeat := r.queue.Repeat()
	if repeat == RepeatOne {
		r.engine.Seek(0)
		return
	}
	if !r.queue.Previous() {
		return
	}
	atomic.StoreInt32(&r.manualSkip, 1)
	if repeat == RepeatAll && r.queue.Position() == r.queue.Len()-1 {
		paths := r.queue.Paths(r.queue.Position())
		if len(paths) > 0 {
			_ = r.engine.PlayAll(r.resolve(paths))
		}
		return
	}
	r.engine.Previous()
}
