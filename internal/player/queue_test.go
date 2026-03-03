package player

import "testing"

func TestNewQueue(t *testing.T) {
	q := NewQueue()
	if q.Len() != 0 {
		t.Fatalf("expected empty queue, got %d", q.Len())
	}
	if q.Position() != -1 {
		t.Fatalf("expected position -1, got %d", q.Position())
	}
	if q.Current() != nil {
		t.Fatal("expected nil current track")
	}
}

func TestReplace(t *testing.T) {
	q := NewQueue()
	tracks := []QueueTrack{
		{TrackID: 1, Title: "A", FilePath: "/a.flac"},
		{TrackID: 2, Title: "B", FilePath: "/b.flac"},
		{TrackID: 3, Title: "C", FilePath: "/c.flac"},
	}

	q.Replace(tracks, 1)
	if q.Len() != 3 {
		t.Fatalf("expected 3 tracks, got %d", q.Len())
	}
	if q.Position() != 1 {
		t.Fatalf("expected position 1, got %d", q.Position())
	}
	if q.Current().Title != "B" {
		t.Fatalf("expected current track B, got %s", q.Current().Title)
	}
}

func TestReplaceWithEmpty(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{{TrackID: 1}}, 0)
	q.Replace(nil, 0)
	if q.Len() != 0 {
		t.Fatalf("expected empty queue, got %d", q.Len())
	}
	if q.Position() != -1 {
		t.Fatalf("expected position -1, got %d", q.Position())
	}
}

func TestReplaceClampsBadStartAt(t *testing.T) {
	q := NewQueue()
	tracks := []QueueTrack{{TrackID: 1}, {TrackID: 2}}

	q.Replace(tracks, 99)
	if q.Position() != 0 {
		t.Fatalf("expected position clamped to 0, got %d", q.Position())
	}

	q.Replace(tracks, -1)
	if q.Position() != 0 {
		t.Fatalf("expected position clamped to 0, got %d", q.Position())
	}
}

func TestReplaceClearsShuffle(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{{TrackID: 1}, {TrackID: 2}, {TrackID: 3}}, 0)
	q.SetShuffle(true)
	if !q.Shuffled() {
		t.Fatal("expected shuffle to be on")
	}
	q.Replace([]QueueTrack{{TrackID: 4}}, 0)
	if q.Shuffled() {
		t.Fatal("expected shuffle to be cleared after Replace")
	}
}

func TestAppend(t *testing.T) {
	q := NewQueue()
	q.Append(QueueTrack{TrackID: 1, Title: "A"})
	if q.Len() != 1 {
		t.Fatalf("expected 1 track, got %d", q.Len())
	}
	if q.Position() != 0 {
		t.Fatalf("expected position 0, got %d", q.Position())
	}

	q.Append(QueueTrack{TrackID: 2, Title: "B"})
	if q.Len() != 2 {
		t.Fatalf("expected 2 tracks, got %d", q.Len())
	}
	if q.Position() != 0 {
		t.Fatalf("expected position still 0, got %d", q.Position())
	}
}

func TestInsertAfterCurrent(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
		{TrackID: 3, Title: "C"},
	}, 1) // position = 1 (B)

	q.InsertAfterCurrent(QueueTrack{TrackID: 4, Title: "X"})
	if q.Len() != 4 {
		t.Fatalf("expected 4 tracks, got %d", q.Len())
	}
	tracks := q.Tracks()
	if tracks[2].Title != "X" {
		t.Fatalf("expected X at index 2, got %s", tracks[2].Title)
	}
	if q.Position() != 1 {
		t.Fatalf("expected position still 1, got %d", q.Position())
	}
}

func TestInsertAfterCurrentEmpty(t *testing.T) {
	q := NewQueue()
	q.InsertAfterCurrent(QueueTrack{TrackID: 1, Title: "A"})
	if q.Len() != 1 {
		t.Fatalf("expected 1 track, got %d", q.Len())
	}
	if q.Position() != 0 {
		t.Fatalf("expected position 0, got %d", q.Position())
	}
}

func TestRemove(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
		{TrackID: 3, Title: "C"},
	}, 1)

	// Remove before current - position should shift down.
	wasCurrent := q.Remove(0)
	if wasCurrent {
		t.Fatal("expected wasCurrent to be false")
	}
	if q.Position() != 0 {
		t.Fatalf("expected position 0, got %d", q.Position())
	}
	if q.Current().Title != "B" {
		t.Fatalf("expected current B, got %s", q.Current().Title)
	}
}

func TestRemoveCurrent(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
		{TrackID: 3, Title: "C"},
	}, 1)

	wasCurrent := q.Remove(1)
	if !wasCurrent {
		t.Fatal("expected wasCurrent to be true")
	}
	// Position stays at 1, now pointing to C.
	if q.Current().Title != "C" {
		t.Fatalf("expected current C, got %s", q.Current().Title)
	}
}

func TestRemoveCurrentAtEnd(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
	}, 1)

	wasCurrent := q.Remove(1)
	if !wasCurrent {
		t.Fatal("expected wasCurrent to be true")
	}
	// Position clamped to last track.
	if q.Position() != 0 {
		t.Fatalf("expected position 0, got %d", q.Position())
	}
}

func TestRemoveLastTrack(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{{TrackID: 1}}, 0)
	q.Remove(0)
	if q.Position() != -1 {
		t.Fatalf("expected position -1, got %d", q.Position())
	}
	if q.Len() != 0 {
		t.Fatalf("expected empty queue, got %d", q.Len())
	}
}

func TestRemoveInvalidIndex(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{{TrackID: 1}}, 0)
	wasCurrent := q.Remove(-1)
	if wasCurrent {
		t.Fatal("expected false for invalid index")
	}
	wasCurrent = q.Remove(5)
	if wasCurrent {
		t.Fatal("expected false for out of range index")
	}
}

func TestMove(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
		{TrackID: 3, Title: "C"},
	}, 0) // position = 0 (A)

	// Move A from index 0 to index 2.
	q.Move(0, 2)
	tracks := q.Tracks()
	if tracks[0].Title != "B" || tracks[1].Title != "C" || tracks[2].Title != "A" {
		t.Fatalf("unexpected order: %v %v %v", tracks[0].Title, tracks[1].Title, tracks[2].Title)
	}
	// Position should follow the current track.
	if q.Position() != 2 {
		t.Fatalf("expected position 2 (followed A), got %d", q.Position())
	}
}

func TestMoveAcrossCurrent(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
		{TrackID: 3, Title: "C"},
	}, 1) // position = 1 (B)

	// Move C (index 2) to index 0 - moves past current from right to left.
	q.Move(2, 0)
	if q.Position() != 2 {
		t.Fatalf("expected position 2 (shifted right), got %d", q.Position())
	}
}

func TestNext(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
	}, 0)

	if !q.Next() {
		t.Fatal("expected Next to return true")
	}
	if q.Position() != 1 {
		t.Fatalf("expected position 1, got %d", q.Position())
	}
	if q.Next() {
		t.Fatal("expected Next to return false at end")
	}
}

func TestPrevious(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
	}, 1)

	if !q.Previous() {
		t.Fatal("expected Previous to return true")
	}
	if q.Position() != 0 {
		t.Fatalf("expected position 0, got %d", q.Position())
	}
	if q.Previous() {
		t.Fatal("expected Previous to return false at start")
	}
}

func TestSetPosition(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1},
		{TrackID: 2},
		{TrackID: 3},
	}, 0)

	q.SetPosition(2)
	if q.Position() != 2 {
		t.Fatalf("expected position 2, got %d", q.Position())
	}

	// Invalid positions should be ignored.
	q.SetPosition(-1)
	if q.Position() != 2 {
		t.Fatalf("expected position still 2, got %d", q.Position())
	}
	q.SetPosition(99)
	if q.Position() != 2 {
		t.Fatalf("expected position still 2, got %d", q.Position())
	}
}

func TestClear(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{{TrackID: 1}, {TrackID: 2}}, 0)
	q.Clear()
	if q.Len() != 0 {
		t.Fatalf("expected empty queue, got %d", q.Len())
	}
	if q.Position() != -1 {
		t.Fatalf("expected position -1, got %d", q.Position())
	}
}

func TestPaths(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, FilePath: "/a.flac"},
		{TrackID: 2, FilePath: "/b.flac"},
		{TrackID: 3, FilePath: "/c.flac"},
	}, 0)

	paths := q.Paths(1)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if paths[0] != "/b.flac" || paths[1] != "/c.flac" {
		t.Fatalf("unexpected paths: %v", paths)
	}

	// Invalid from index.
	if q.Paths(-1) != nil {
		t.Fatal("expected nil for negative index")
	}
	if q.Paths(99) != nil {
		t.Fatal("expected nil for out of range index")
	}
}

func TestTracksCopiesSlice(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{{TrackID: 1, Title: "A"}}, 0)
	tracks := q.Tracks()
	tracks[0].Title = "modified"
	if q.Current().Title != "A" {
		t.Fatal("Tracks() should return a copy, not a reference")
	}
}

// --- Repeat mode tests ---

func TestRepeatOneNext(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
	}, 0)
	q.SetRepeat(RepeatOne)

	if !q.Next() {
		t.Fatal("expected Next to return true with repeat-one")
	}
	if q.Position() != 0 {
		t.Fatalf("expected position to stay 0 with repeat-one, got %d", q.Position())
	}
}

func TestRepeatOnePrevious(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
	}, 1)
	q.SetRepeat(RepeatOne)

	if !q.Previous() {
		t.Fatal("expected Previous to return true with repeat-one")
	}
	if q.Position() != 1 {
		t.Fatalf("expected position to stay 1, got %d", q.Position())
	}
}

func TestRepeatAllWrapsForward(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
	}, 1)
	q.SetRepeat(RepeatAll)

	if !q.Next() {
		t.Fatal("expected Next to return true with repeat-all")
	}
	if q.Position() != 0 {
		t.Fatalf("expected position to wrap to 0, got %d", q.Position())
	}
}

func TestRepeatAllWrapsBackward(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
	}, 0)
	q.SetRepeat(RepeatAll)

	if !q.Previous() {
		t.Fatal("expected Previous to return true with repeat-all")
	}
	if q.Position() != 1 {
		t.Fatalf("expected position to wrap to 1, got %d", q.Position())
	}
}

func TestRepeatModeString(t *testing.T) {
	tests := []struct {
		mode RepeatMode
		want string
	}{
		{RepeatOff, "off"},
		{RepeatAll, "all"},
		{RepeatOne, "one"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("RepeatMode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}

func TestRepeatGetterSetter(t *testing.T) {
	q := NewQueue()
	if q.Repeat() != RepeatOff {
		t.Fatal("expected default repeat off")
	}
	q.SetRepeat(RepeatAll)
	if q.Repeat() != RepeatAll {
		t.Fatal("expected repeat all")
	}
}

// --- Shuffle tests ---

func TestShuffleProducesPermutation(t *testing.T) {
	q := NewQueue()
	tracks := make([]QueueTrack, 20)
	for i := range tracks {
		tracks[i] = QueueTrack{TrackID: int64(i + 1), Title: string(rune('A' + i))}
	}
	q.Replace(tracks, 0)
	q.SetShuffle(true)

	if !q.Shuffled() {
		t.Fatal("expected shuffled to be true")
	}

	result := q.Tracks()
	if len(result) != 20 {
		t.Fatalf("expected 20 tracks, got %d", len(result))
	}

	// Current track should still be at position 0.
	if q.Current().TrackID != 1 {
		t.Fatalf("expected current track ID 1, got %d", q.Current().TrackID)
	}

	// All tracks should be present (permutation).
	seen := make(map[int64]bool)
	for _, tr := range result {
		seen[tr.TrackID] = true
	}
	if len(seen) != 20 {
		t.Fatalf("expected all 20 tracks present, got %d unique", len(seen))
	}
}

func TestShuffleOffRestoresOrder(t *testing.T) {
	q := NewQueue()
	tracks := []QueueTrack{
		{TrackID: 1, Title: "A"},
		{TrackID: 2, Title: "B"},
		{TrackID: 3, Title: "C"},
		{TrackID: 4, Title: "D"},
		{TrackID: 5, Title: "E"},
	}
	q.Replace(tracks, 2) // position = 2 (C)

	q.SetShuffle(true)
	// Current track should still be C.
	if q.Current().TrackID != 3 {
		t.Fatalf("expected current track C (ID 3), got %d", q.Current().TrackID)
	}

	q.SetShuffle(false)
	if q.Shuffled() {
		t.Fatal("expected shuffled to be false")
	}

	// Order should be restored.
	result := q.Tracks()
	for i, tr := range result {
		if tr.TrackID != int64(i+1) {
			t.Fatalf("expected track ID %d at position %d, got %d", i+1, i, tr.TrackID)
		}
	}

	// Position should point to C (index 2) in original order.
	if q.Position() != 2 {
		t.Fatalf("expected position 2, got %d", q.Position())
	}
}

func TestShuffleToggleIdempotent(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{{TrackID: 1}, {TrackID: 2}}, 0)

	q.SetShuffle(true)
	q.SetShuffle(true) // should be a no-op
	if !q.Shuffled() {
		t.Fatal("expected still shuffled")
	}

	q.SetShuffle(false)
	q.SetShuffle(false) // should be a no-op
	if q.Shuffled() {
		t.Fatal("expected not shuffled")
	}
}

func TestShuffleEmptyQueue(t *testing.T) {
	q := NewQueue()
	q.SetShuffle(true) // should not panic
	if q.Shuffled() {
		t.Fatal("expected shuffle to stay off for empty queue")
	}
}

func TestClearResetsShuffle(t *testing.T) {
	q := NewQueue()
	q.Replace([]QueueTrack{{TrackID: 1}, {TrackID: 2}}, 0)
	q.SetShuffle(true)
	q.Clear()
	if q.Shuffled() {
		t.Fatal("expected shuffle to be cleared")
	}
}
