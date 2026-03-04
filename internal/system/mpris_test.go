package system

import (
	"testing"

	"github.com/godbus/dbus/v5"
)

func TestMapPlaybackStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"playing", "Playing"},
		{"paused", "Paused"},
		{"stopped", "Stopped"},
		{"", "Stopped"},
		{"unknown", "Stopped"},
	}
	for _, tt := range tests {
		if got := mapPlaybackStatus(tt.input); got != tt.want {
			t.Errorf("mapPlaybackStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapRepeatToLoopStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"one", "Track"},
		{"all", "Playlist"},
		{"off", "None"},
		{"", "None"},
		{"unknown", "None"},
	}
	for _, tt := range tests {
		if got := mapRepeatToLoopStatus(tt.input); got != tt.want {
			t.Errorf("mapRepeatToLoopStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapLoopStatusToRepeat(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Track", "one"},
		{"Playlist", "all"},
		{"None", "off"},
		{"", "off"},
		{"unknown", "off"},
	}
	for _, tt := range tests {
		if got := mapLoopStatusToRepeat(tt.input); got != tt.want {
			t.Errorf("mapLoopStatusToRepeat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestLoopStatusRoundTrip(t *testing.T) {
	for _, mode := range []string{"one", "all", "off"} {
		loopStatus := mapRepeatToLoopStatus(mode)
		got := mapLoopStatusToRepeat(loopStatus)
		if got != mode {
			t.Errorf("round-trip failed: %q -> %q -> %q", mode, loopStatus, got)
		}
	}
}

func TestClampSeek(t *testing.T) {
	tests := []struct {
		name     string
		current  float64
		offsetUs int64
		duration float64
		want     float64
	}{
		{"forward within bounds", 10.0, 5_000_000, 300.0, 15.0},
		{"backward within bounds", 10.0, -5_000_000, 300.0, 5.0},
		{"clamp to zero", 2.0, -5_000_000, 300.0, 0.0},
		{"clamp to duration", 298.0, 5_000_000, 300.0, 300.0},
		{"zero duration (no upper clamp)", 10.0, 5_000_000, 0.0, 15.0},
		{"exact offset to zero", 5.0, -5_000_000, 300.0, 0.0},
		{"zero offset", 10.0, 0, 300.0, 10.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampSeek(tt.current, tt.offsetUs, tt.duration)
			if got != tt.want {
				t.Errorf("clampSeek(%v, %v, %v) = %v, want %v",
					tt.current, tt.offsetUs, tt.duration, got, tt.want)
			}
		})
	}
}

func TestBuildMetadata(t *testing.T) {
	md := buildMetadata("Song", "Artist", "Album", "/music/song.flac", 240000, 42, "file:///tmp/art.jpg")

	// Track ID: dbus.ObjectPath with the track ID embedded.
	trackID := md["mpris:trackid"].Value().(dbus.ObjectPath)
	if trackID != "/org/forte/track/42" {
		t.Errorf("trackid = %q, want /org/forte/track/42", trackID)
	}

	// Duration: milliseconds * 1000 = microseconds.
	length := md["mpris:length"].Value().(int64)
	if length != 240_000_000 {
		t.Errorf("length = %d, want 240000000", length)
	}

	// Title.
	title := md["xesam:title"].Value().(string)
	if title != "Song" {
		t.Errorf("title = %q, want Song", title)
	}

	// Artist must be []string per MPRIS2 spec.
	artists := md["xesam:artist"].Value().([]string)
	if len(artists) != 1 || artists[0] != "Artist" {
		t.Errorf("artist = %v, want [Artist]", artists)
	}

	// Album.
	album := md["xesam:album"].Value().(string)
	if album != "Album" {
		t.Errorf("album = %q, want Album", album)
	}

	// URL.
	url := md["xesam:url"].Value().(string)
	if url != "file:///music/song.flac" {
		t.Errorf("url = %q, want file:///music/song.flac", url)
	}

	// Art URL.
	artURL := md["mpris:artUrl"].Value().(string)
	if artURL != "file:///tmp/art.jpg" {
		t.Errorf("artUrl = %q, want file:///tmp/art.jpg", artURL)
	}
}

func TestBuildMetadataOmitsEmpty(t *testing.T) {
	md := buildMetadata("", "", "", "", 0, 1, "")

	// Always-present fields.
	if _, ok := md["mpris:trackid"]; !ok {
		t.Error("expected mpris:trackid to always be present")
	}
	if _, ok := md["mpris:length"]; !ok {
		t.Error("expected mpris:length to always be present")
	}

	// Optional fields should be absent.
	for _, key := range []string{"xesam:title", "xesam:artist", "xesam:album", "xesam:url", "mpris:artUrl"} {
		if _, ok := md[key]; ok {
			t.Errorf("expected %q to be absent for empty input", key)
		}
	}
}

func TestEmptyMetadata(t *testing.T) {
	md := emptyMetadata()
	trackID := md["mpris:trackid"].Value().(dbus.ObjectPath)
	if trackID != "/org/mpris/MediaPlayer2/TrackList/NoTrack" {
		t.Errorf("empty trackid = %q, want NoTrack path", trackID)
	}
	if len(md) != 1 {
		t.Errorf("empty metadata has %d keys, want 1", len(md))
	}
}

// mockPlayer implements PlayerControl for testing D-Bus handler callbacks.
type mockPlayer struct {
	pauseCalled    bool
	resumeCalled   bool
	stopCalled     bool
	nextCalled     bool
	previousCalled bool
	seekPos        float64
	volumeSet      int
	shuffleSet     *bool
	repeatSet      string
	state          string
	position       float64
	duration       float64
}

func (m *mockPlayer) Pause()                    { m.pauseCalled = true }
func (m *mockPlayer) Resume()                   { m.resumeCalled = true }
func (m *mockPlayer) Stop()                     { m.stopCalled = true }
func (m *mockPlayer) Next()                     { m.nextCalled = true }
func (m *mockPlayer) Previous()                 { m.previousCalled = true }
func (m *mockPlayer) Seek(s float64)            { m.seekPos = s }
func (m *mockPlayer) SetVolume(p int)           { m.volumeSet = p }
func (m *mockPlayer) Volume() int               { return 50 }
func (m *mockPlayer) Position() float64         { return m.position }
func (m *mockPlayer) Duration() float64         { return m.duration }
func (m *mockPlayer) State() string             { return m.state }
func (m *mockPlayer) MediaPath() string         { return "" }
func (m *mockPlayer) SetShuffle(e bool)         { m.shuffleSet = &e }
func (m *mockPlayer) GetShuffle() bool          { return false }
func (m *mockPlayer) SetRepeat(mode string)     { m.repeatSet = mode }
func (m *mockPlayer) GetRepeat() string         { return "off" }

func TestMprisPlayerPlay(t *testing.T) {
	mock := &mockPlayer{}
	p := &mprisPlayer{m: &MPRIS{player: mock}}
	p.Play()
	if !mock.resumeCalled {
		t.Error("Play() should call Resume()")
	}
}

func TestMprisPlayerPause(t *testing.T) {
	mock := &mockPlayer{}
	p := &mprisPlayer{m: &MPRIS{player: mock}}
	p.Pause()
	if !mock.pauseCalled {
		t.Error("Pause() should call Pause()")
	}
}

func TestMprisPlayerPlayPause(t *testing.T) {
	mock := &mockPlayer{state: "playing"}
	p := &mprisPlayer{m: &MPRIS{player: mock}}
	p.PlayPause()
	if !mock.pauseCalled {
		t.Error("PlayPause() when playing should call Pause()")
	}

	mock = &mockPlayer{state: "paused"}
	p = &mprisPlayer{m: &MPRIS{player: mock}}
	p.PlayPause()
	if !mock.resumeCalled {
		t.Error("PlayPause() when paused should call Resume()")
	}

	mock = &mockPlayer{state: "stopped"}
	p = &mprisPlayer{m: &MPRIS{player: mock}}
	p.PlayPause()
	if !mock.resumeCalled {
		t.Error("PlayPause() when stopped should call Resume()")
	}
}

func TestMprisPlayerStop(t *testing.T) {
	mock := &mockPlayer{}
	p := &mprisPlayer{m: &MPRIS{player: mock}}
	p.Stop()
	if !mock.stopCalled {
		t.Error("Stop() should call Stop()")
	}
}

func TestMprisPlayerNext(t *testing.T) {
	mock := &mockPlayer{}
	p := &mprisPlayer{m: &MPRIS{player: mock}}
	p.Next()
	if !mock.nextCalled {
		t.Error("Next() should call Next()")
	}
}

func TestMprisPlayerPrevious(t *testing.T) {
	mock := &mockPlayer{}
	p := &mprisPlayer{m: &MPRIS{player: mock}}
	p.Previous()
	if !mock.previousCalled {
		t.Error("Previous() should call Previous()")
	}
}

func TestSetPositionOutOfRange(t *testing.T) {
	mock := &mockPlayer{duration: 300.0}
	p := &mprisPlayer{m: &MPRIS{player: mock}}

	// Negative position should be rejected (no Seek called).
	p.SetPosition("/org/forte/track/1", -1_000_000)
	if mock.seekPos != 0 {
		t.Error("SetPosition with negative position should not seek")
	}

	// Beyond duration should be rejected.
	mock.seekPos = 0
	p.SetPosition("/org/forte/track/1", 500_000_000_000)
	if mock.seekPos != 0 {
		t.Error("SetPosition beyond duration should not seek")
	}
}

func TestUpdatePositionNilProps(t *testing.T) {
	// UpdatePosition with nil props should not panic.
	m := &MPRIS{}
	m.UpdatePosition(10.5) // no panic = pass
}

func TestClearMetadataNilProps(t *testing.T) {
	// ClearMetadata with nil props should not panic.
	m := &MPRIS{}
	m.ClearMetadata() // no panic = pass
}
