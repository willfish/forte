//go:build !linux

package system

import "testing"

type unsupportedPlayer struct{}

func (unsupportedPlayer) Pause()            {}
func (unsupportedPlayer) Resume()           {}
func (unsupportedPlayer) Stop()             {}
func (unsupportedPlayer) Next()             {}
func (unsupportedPlayer) Previous()         {}
func (unsupportedPlayer) Seek(float64)      {}
func (unsupportedPlayer) SetVolume(int)     {}
func (unsupportedPlayer) Volume() int       { return 50 }
func (unsupportedPlayer) Position() float64 { return 0 }
func (unsupportedPlayer) Duration() float64 { return 0 }
func (unsupportedPlayer) State() string     { return "stopped" }
func (unsupportedPlayer) MediaPath() string { return "" }
func (unsupportedPlayer) SetShuffle(bool)   {}
func (unsupportedPlayer) GetShuffle() bool  { return false }
func (unsupportedPlayer) SetRepeat(string)  {}
func (unsupportedPlayer) GetRepeat() string { return "off" }

func TestUnsupportedMPRISNoOps(t *testing.T) {
	m, err := NewMPRIS(unsupportedPlayer{})
	if err != nil {
		t.Fatalf("NewMPRIS() error = %v", err)
	}

	m.UpdatePlaybackStatus("playing")
	m.UpdateMetadata("Title", "Artist", "Album", "/tmp/track.flac", 123, 1)
	m.UpdateVolume(75)
	m.UpdateShuffle(true)
	m.UpdateLoopStatus("all")
	m.UpdatePosition(10)
	m.ClearMetadata()
	m.Close()
}

func TestUnsupportedNotifierStateAndNoOps(t *testing.T) {
	n, err := NewNotifier()
	if err != nil {
		t.Fatalf("NewNotifier() error = %v", err)
	}
	if !n.Enabled() {
		t.Fatal("notifier should default to enabled")
	}

	n.Notify("Title", "Body", []byte{0xff, 0xd8})
	n.SetEnabled(false)
	if n.Enabled() {
		t.Fatal("notifier should be disabled")
	}
	n.Notify("Title", "Body", nil)
	n.Close()
}
