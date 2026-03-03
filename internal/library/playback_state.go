package library

// PlaybackState holds persisted playback state for session restore.
type PlaybackState struct {
	QueueJSON       string
	Position        int
	TrackPositionMs int
	Volume          int
	Shuffle         bool
	RepeatMode      string
}

// SavePlaybackState upserts the single playback state row.
func (db *DB) SavePlaybackState(s PlaybackState) error {
	shuffle := 0
	if s.Shuffle {
		shuffle = 1
	}
	_, err := db.Exec(`
		UPDATE playback_state SET
			queue_json = ?,
			position = ?,
			track_position_ms = ?,
			volume = ?,
			shuffle = ?,
			repeat_mode = ?
		WHERE id = 1`,
		s.QueueJSON, s.Position, s.TrackPositionMs, s.Volume, shuffle, s.RepeatMode,
	)
	return err
}

// LoadPlaybackState reads the persisted playback state.
// Returns a zero-value state if no row exists.
func (db *DB) LoadPlaybackState() (PlaybackState, error) {
	var s PlaybackState
	var shuffle int
	err := db.QueryRow(`
		SELECT queue_json, position, track_position_ms, volume, shuffle, repeat_mode
		FROM playback_state WHERE id = 1`,
	).Scan(&s.QueueJSON, &s.Position, &s.TrackPositionMs, &s.Volume, &shuffle, &s.RepeatMode)
	if err != nil {
		return PlaybackState{}, err
	}
	s.Shuffle = shuffle != 0
	return s, nil
}
