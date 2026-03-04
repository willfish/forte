package library

import "testing"

func TestSaveAndLoadPlaybackState(t *testing.T) {
	db := openTestDB(t)

	state := PlaybackState{
		QueueJSON:       `[{"path":"/a.flac"}]`,
		Position:        3,
		TrackPositionMs: 45000,
		Volume:          80,
		Shuffle:         true,
		RepeatMode:      "all",
	}

	if err := db.SavePlaybackState(state); err != nil {
		t.Fatalf("SavePlaybackState: %v", err)
	}

	got, err := db.LoadPlaybackState()
	if err != nil {
		t.Fatalf("LoadPlaybackState: %v", err)
	}

	if got.QueueJSON != state.QueueJSON {
		t.Errorf("QueueJSON = %q", got.QueueJSON)
	}
	if got.Position != 3 {
		t.Errorf("Position = %d, want 3", got.Position)
	}
	if got.TrackPositionMs != 45000 {
		t.Errorf("TrackPositionMs = %d", got.TrackPositionMs)
	}
	if got.Volume != 80 {
		t.Errorf("Volume = %d", got.Volume)
	}
	if !got.Shuffle {
		t.Error("Shuffle = false, want true")
	}
	if got.RepeatMode != "all" {
		t.Errorf("RepeatMode = %q", got.RepeatMode)
	}
}

func TestLoadPlaybackStateDefault(t *testing.T) {
	db := openTestDB(t)

	got, err := db.LoadPlaybackState()
	if err != nil {
		t.Fatalf("LoadPlaybackState: %v", err)
	}
	if got.Position != -1 {
		t.Errorf("default Position = %d, want -1", got.Position)
	}
	if got.Volume != 100 {
		t.Errorf("default Volume = %d, want 100", got.Volume)
	}
	if got.Shuffle {
		t.Error("default Shuffle = true, want false")
	}
	if got.RepeatMode != "off" {
		t.Errorf("default RepeatMode = %q, want 'off'", got.RepeatMode)
	}
}

func TestSavePlaybackStateOverwrite(t *testing.T) {
	db := openTestDB(t)

	_ = db.SavePlaybackState(PlaybackState{Volume: 50})
	_ = db.SavePlaybackState(PlaybackState{Volume: 75})

	got, _ := db.LoadPlaybackState()
	if got.Volume != 75 {
		t.Errorf("Volume = %d, want 75", got.Volume)
	}
}

func TestPlaybackStateShuffleFalse(t *testing.T) {
	db := openTestDB(t)

	_ = db.SavePlaybackState(PlaybackState{Shuffle: true})
	_ = db.SavePlaybackState(PlaybackState{Shuffle: false})

	got, _ := db.LoadPlaybackState()
	if got.Shuffle {
		t.Error("Shuffle = true, want false")
	}
}
