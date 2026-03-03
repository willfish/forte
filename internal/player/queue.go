package player

import (
	"math/rand/v2"
	"sync"
)

// RepeatMode controls queue repeat behaviour.
type RepeatMode int

const (
	RepeatOff RepeatMode = iota
	RepeatAll
	RepeatOne
)

func (m RepeatMode) String() string {
	switch m {
	case RepeatAll:
		return "all"
	case RepeatOne:
		return "one"
	default:
		return "off"
	}
}

// QueueTrack holds metadata for a track in the play queue.
type QueueTrack struct {
	TrackID    int64  `json:"trackId"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	DurationMs int    `json:"durationMs"`
	FilePath   string `json:"filePath"`
}

// Queue manages an ordered list of tracks for playback.
type Queue struct {
	mu             sync.RWMutex
	tracks         []QueueTrack // current play order (may be shuffled)
	originalTracks []QueueTrack // saved original order when shuffled; nil otherwise
	position       int          // -1 when empty
	shuffled       bool
	repeat         RepeatMode
}

// NewQueue creates an empty queue.
func NewQueue() *Queue {
	return &Queue{position: -1}
}

// Replace sets the queue to the given tracks and starts at startAt.
// Disables shuffle (new queue = fresh order).
func (q *Queue) Replace(tracks []QueueTrack, startAt int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tracks = make([]QueueTrack, len(tracks))
	copy(q.tracks, tracks)
	q.originalTracks = nil
	q.shuffled = false

	if len(q.tracks) == 0 {
		q.position = -1
	} else if startAt >= 0 && startAt < len(q.tracks) {
		q.position = startAt
	} else {
		q.position = 0
	}
}

// Append adds a track to the end of the queue.
func (q *Queue) Append(track QueueTrack) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tracks = append(q.tracks, track)
	if q.originalTracks != nil {
		q.originalTracks = append(q.originalTracks, track)
	}
	if q.position == -1 {
		q.position = 0
	}
}

// InsertAfterCurrent inserts a track immediately after the current position.
// If the queue is empty, it becomes the only track.
func (q *Queue) InsertAfterCurrent(track QueueTrack) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 || q.position < 0 {
		q.tracks = []QueueTrack{track}
		if q.originalTracks != nil {
			q.originalTracks = append(q.originalTracks, track)
		}
		q.position = 0
		return
	}

	insertAt := q.position + 1
	q.tracks = append(q.tracks, QueueTrack{})
	copy(q.tracks[insertAt+1:], q.tracks[insertAt:])
	q.tracks[insertAt] = track

	if q.originalTracks != nil {
		q.originalTracks = append(q.originalTracks, track)
	}
}

// Remove removes the track at the given index.
// Returns true if the removed track was the current track.
func (q *Queue) Remove(index int) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if index < 0 || index >= len(q.tracks) {
		return false
	}

	if q.originalTracks != nil {
		// Remove the same track from originalTracks by TrackID.
		tid := q.tracks[index].TrackID
		for i, t := range q.originalTracks {
			if t.TrackID == tid {
				q.originalTracks = append(q.originalTracks[:i], q.originalTracks[i+1:]...)
				break
			}
		}
	}

	wasCurrent := index == q.position
	q.tracks = append(q.tracks[:index], q.tracks[index+1:]...)

	if len(q.tracks) == 0 {
		q.position = -1
	} else if index < q.position {
		q.position--
	} else if wasCurrent && q.position >= len(q.tracks) {
		q.position = len(q.tracks) - 1
	}

	return wasCurrent
}

// Move moves a track from one index to another.
func (q *Queue) Move(from, to int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if from < 0 || from >= len(q.tracks) || to < 0 || to >= len(q.tracks) || from == to {
		return
	}

	track := q.tracks[from]

	// Adjust current position.
	switch {
	case q.position == from:
		q.position = to
	case from < q.position && to >= q.position:
		q.position--
	case from > q.position && to <= q.position:
		q.position++
	}

	q.tracks = append(q.tracks[:from], q.tracks[from+1:]...)
	q.tracks = append(q.tracks[:to], append([]QueueTrack{track}, q.tracks[to:]...)...)
}

// Next advances to the next track. Returns false if at the end
// (respecting repeat mode).
func (q *Queue) Next() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 {
		return false
	}

	if q.repeat == RepeatOne {
		return true // position stays, same track replays
	}

	if q.position+1 >= len(q.tracks) {
		if q.repeat == RepeatAll {
			q.position = 0
			return true
		}
		return false
	}
	q.position++
	return true
}

// Previous goes back to the previous track. Returns false if at the start
// (respecting repeat mode).
func (q *Queue) Previous() bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.tracks) == 0 {
		return false
	}

	if q.repeat == RepeatOne {
		return true
	}

	if q.position <= 0 {
		if q.repeat == RepeatAll {
			q.position = len(q.tracks) - 1
			return true
		}
		return false
	}
	q.position--
	return true
}

// SetShuffle enables or disables shuffle mode.
// When enabling, the current track stays at the current position and
// remaining tracks are shuffled using Fisher-Yates.
// When disabling, the original order is restored and position moves to
// the current track's location in the original order.
func (q *Queue) SetShuffle(enabled bool) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if enabled == q.shuffled || len(q.tracks) == 0 {
		return
	}

	if enabled {
		q.originalTracks = make([]QueueTrack, len(q.tracks))
		copy(q.originalTracks, q.tracks)

		// Shuffle everything after the current position.
		remaining := q.tracks[q.position+1:]
		for i := len(remaining) - 1; i > 0; i-- {
			j := rand.IntN(i + 1)
			remaining[i], remaining[j] = remaining[j], remaining[i]
		}
		q.shuffled = true
	} else {
		currentTrack := q.tracks[q.position]
		q.tracks = q.originalTracks
		q.originalTracks = nil

		// Find current track in restored order.
		for i, t := range q.tracks {
			if t.TrackID == currentTrack.TrackID {
				q.position = i
				break
			}
		}
		q.shuffled = false
	}
}

// Shuffled returns whether shuffle mode is active.
func (q *Queue) Shuffled() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.shuffled
}

// SetRepeat sets the repeat mode.
func (q *Queue) SetRepeat(mode RepeatMode) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.repeat = mode
}

// Repeat returns the current repeat mode.
func (q *Queue) Repeat() RepeatMode {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.repeat
}

// Current returns the currently selected track, or nil if empty.
func (q *Queue) Current() *QueueTrack {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.position < 0 || q.position >= len(q.tracks) {
		return nil
	}
	t := q.tracks[q.position]
	return &t
}

// Tracks returns a copy of all tracks in the queue.
func (q *Queue) Tracks() []QueueTrack {
	q.mu.RLock()
	defer q.mu.RUnlock()

	result := make([]QueueTrack, len(q.tracks))
	copy(result, q.tracks)
	return result
}

// Position returns the current position index (-1 if empty).
func (q *Queue) Position() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.position
}

// SetPosition sets the current position.
func (q *Queue) SetPosition(pos int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if pos >= 0 && pos < len(q.tracks) {
		q.position = pos
	}
}

// Len returns the number of tracks in the queue.
func (q *Queue) Len() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.tracks)
}

// Clear empties the queue.
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tracks = nil
	q.originalTracks = nil
	q.shuffled = false
	q.position = -1
}

// Paths returns the file paths of all tracks starting from the given index.
func (q *Queue) Paths(from int) []string {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if from < 0 || from >= len(q.tracks) {
		return nil
	}
	paths := make([]string, len(q.tracks)-from)
	for i, t := range q.tracks[from:] {
		paths[i] = t.FilePath
	}
	return paths
}
