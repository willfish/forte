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
