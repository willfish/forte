package library

import (
	"testing"
	"time"
)

func TestEnqueueAndRetrieve(t *testing.T) {
	db := openTestDB(t)

	if err := db.EnqueueScrobble("lastfm", `{"artist":"A","track":"T"}`, 1700000000); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if err := db.EnqueueScrobble("lastfm", `{"artist":"B","track":"U"}`, 1700000001); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	entries, err := db.PendingScrobbles("lastfm", 10)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Timestamp != 1700000000 {
		t.Errorf("first entry timestamp = %d, want 1700000000", entries[0].Timestamp)
	}
	if entries[1].Timestamp != 1700000001 {
		t.Errorf("second entry timestamp = %d, want 1700000001", entries[1].Timestamp)
	}
}

func TestServiceIsolation(t *testing.T) {
	db := openTestDB(t)

	_ = db.EnqueueScrobble("lastfm", `{"artist":"A"}`, 1700000000)
	_ = db.EnqueueScrobble("listenbrainz", `{"artist":"B"}`, 1700000001)

	lfm, _ := db.PendingScrobbles("lastfm", 10)
	lb, _ := db.PendingScrobbles("listenbrainz", 10)

	if len(lfm) != 1 {
		t.Errorf("lastfm entries = %d, want 1", len(lfm))
	}
	if len(lb) != 1 {
		t.Errorf("listenbrainz entries = %d, want 1", len(lb))
	}
}

func TestRemoveScrobble(t *testing.T) {
	db := openTestDB(t)

	_ = db.EnqueueScrobble("lastfm", `{}`, 1700000000)
	entries, _ := db.PendingScrobbles("lastfm", 10)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	if err := db.RemoveScrobble(entries[0].ID); err != nil {
		t.Fatalf("remove: %v", err)
	}

	entries, _ = db.PendingScrobbles("lastfm", 10)
	if len(entries) != 0 {
		t.Errorf("got %d entries after remove, want 0", len(entries))
	}
}

func TestQueueSize(t *testing.T) {
	db := openTestDB(t)

	size, err := db.ScrobbleQueueSize()
	if err != nil {
		t.Fatalf("queue size: %v", err)
	}
	if size != 0 {
		t.Errorf("empty queue size = %d, want 0", size)
	}

	_ = db.EnqueueScrobble("lastfm", `{}`, 1700000000)
	_ = db.EnqueueScrobble("listenbrainz", `{}`, 1700000001)

	size, err = db.ScrobbleQueueSize()
	if err != nil {
		t.Fatalf("queue size: %v", err)
	}
	if size != 2 {
		t.Errorf("queue size = %d, want 2", size)
	}
}

func TestBackoffFiltering(t *testing.T) {
	db := openTestDB(t)

	_ = db.EnqueueScrobble("lastfm", `{}`, 1700000000)

	// Mark one attempt - entry should now have a recent last_attempt_at
	// and backoff of 2 minutes.
	entries, _ := db.PendingScrobbles("lastfm", 10)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	_ = db.MarkScrobbleAttempt(entries[0].ID)

	// Immediately after marking, the entry should be filtered out by backoff.
	entries, _ = db.PendingScrobbles("lastfm", 10)
	if len(entries) != 0 {
		t.Errorf("got %d entries within backoff window, want 0", len(entries))
	}
}

func TestMarkScrobbleAttempt(t *testing.T) {
	db := openTestDB(t)

	_ = db.EnqueueScrobble("lastfm", `{}`, 1700000000)
	entries, _ := db.PendingScrobbles("lastfm", 10)
	id := entries[0].ID

	_ = db.MarkScrobbleAttempt(id)
	_ = db.MarkScrobbleAttempt(id)

	var attempts int
	_ = db.QueryRow("SELECT attempts FROM scrobble_queue WHERE id = ?", id).Scan(&attempts)
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestPruneScrobbleQueue(t *testing.T) {
	db := openTestDB(t)

	// Insert an entry and backdate its created_at to 31 days ago.
	_ = db.EnqueueScrobble("lastfm", `{}`, 1700000000)
	old := time.Now().AddDate(0, 0, -31).UTC().Format("2006-01-02 15:04:05")
	_, _ = db.Exec("UPDATE scrobble_queue SET created_at = ?", old)

	_ = db.EnqueueScrobble("lastfm", `{}`, 1700000001) // recent entry

	if err := db.PruneScrobbleQueue(); err != nil {
		t.Fatalf("prune: %v", err)
	}

	size, _ := db.ScrobbleQueueSize()
	if size != 1 {
		t.Errorf("queue size after prune = %d, want 1", size)
	}
}

func TestBackoffMinutes(t *testing.T) {
	tests := []struct {
		attempts int
		want     time.Duration
	}{
		{0, 1 * time.Minute},
		{1, 2 * time.Minute},
		{2, 4 * time.Minute},
		{3, 8 * time.Minute},
		{4, 16 * time.Minute},
		{5, 32 * time.Minute},
		{6, 64 * time.Minute},
		{7, 64 * time.Minute},  // capped
		{20, 64 * time.Minute}, // capped
	}
	for _, tt := range tests {
		got := backoffMinutes(tt.attempts)
		if got != tt.want {
			t.Errorf("backoffMinutes(%d) = %v, want %v", tt.attempts, got, tt.want)
		}
	}
}

func TestPendingScrobblesLimit(t *testing.T) {
	db := openTestDB(t)

	for i := range 5 {
		_ = db.EnqueueScrobble("lastfm", `{}`, int64(1700000000+i))
	}

	entries, err := db.PendingScrobbles("lastfm", 3)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if len(entries) != 3 {
		t.Errorf("got %d entries with limit 3, want 3", len(entries))
	}
}

