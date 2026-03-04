package library

import "time"

// ScrobbleQueueEntry represents a failed scrobble waiting to be retried.
type ScrobbleQueueEntry struct {
	ID            int64
	Service       string
	TrackJSON     string
	Timestamp     int64
	Attempts      int
	LastAttemptAt string
	CreatedAt     string
}

// EnqueueScrobble adds a failed scrobble to the retry queue.
func (db *DB) EnqueueScrobble(service, trackJSON string, timestamp int64) error {
	_, err := db.Exec(
		`INSERT INTO scrobble_queue (service, track_json, timestamp) VALUES (?, ?, ?)`,
		service, trackJSON, timestamp,
	)
	return err
}

// PendingScrobbles returns entries for a service that are ready for retry,
// respecting exponential backoff (2^attempts minutes, capped at 64 min).
// Results are ordered oldest-first, limited to limit rows.
func (db *DB) PendingScrobbles(service string, limit int) ([]ScrobbleQueueEntry, error) {
	rows, err := db.Query(
		`SELECT id, service, track_json, timestamp, attempts, last_attempt_at, created_at
		 FROM scrobble_queue
		 WHERE service = ?
		 ORDER BY timestamp ASC`,
		service,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	now := time.Now()
	var result []ScrobbleQueueEntry
	for rows.Next() {
		var e ScrobbleQueueEntry
		if err := rows.Scan(&e.ID, &e.Service, &e.TrackJSON, &e.Timestamp,
			&e.Attempts, &e.LastAttemptAt, &e.CreatedAt); err != nil {
			return nil, err
		}

		// Skip entries still within their backoff window.
		if e.Attempts > 0 && e.LastAttemptAt != "" {
			lastAttempt, err := time.Parse("2006-01-02 15:04:05", e.LastAttemptAt)
			if err == nil {
				delay := backoffMinutes(e.Attempts)
				if now.Before(lastAttempt.Add(delay)) {
					continue
				}
			}
		}

		result = append(result, e)
		if len(result) >= limit {
			break
		}
	}
	return result, rows.Err()
}

// backoffMinutes returns 2^attempts minutes, capped at 64 minutes.
func backoffMinutes(attempts int) time.Duration {
	mins := 1
	for range attempts {
		mins *= 2
		if mins >= 64 {
			return 64 * time.Minute
		}
	}
	return time.Duration(mins) * time.Minute
}

// RemoveScrobble deletes an entry from the queue (after successful submission).
func (db *DB) RemoveScrobble(id int64) error {
	_, err := db.Exec(`DELETE FROM scrobble_queue WHERE id = ?`, id)
	return err
}

// MarkScrobbleAttempt increments the attempt counter and records the current time.
func (db *DB) MarkScrobbleAttempt(id int64) error {
	_, err := db.Exec(
		`UPDATE scrobble_queue SET attempts = attempts + 1, last_attempt_at = datetime('now') WHERE id = ?`,
		id,
	)
	return err
}

// PruneScrobbleQueue removes entries older than 30 days.
func (db *DB) PruneScrobbleQueue() error {
	_, err := db.Exec(`DELETE FROM scrobble_queue WHERE created_at < datetime('now', '-30 days')`)
	return err
}

// ScrobbleQueueSize returns the total number of entries in the queue.
func (db *DB) ScrobbleQueueSize() (int, error) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM scrobble_queue`).Scan(&count)
	return count, err
}
