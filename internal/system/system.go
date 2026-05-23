// Package system provides desktop integration for media controls and notifications.
package system

// PlayerControl is the interface that desktop media integrations use to control
// the player.
type PlayerControl interface {
	Pause()
	Resume()
	Stop()
	Next()
	Previous()
	Seek(seconds float64)
	SetVolume(percent int)
	Volume() int
	Position() float64
	Duration() float64
	State() string
	MediaPath() string
	SetShuffle(enabled bool)
	GetShuffle() bool
	SetRepeat(mode string)
	GetRepeat() string
}

// readArtworkFn reads album artwork from a media file.
// Set via SetReadArtworkFn to avoid a direct dependency on the metadata package.
var readArtworkFn = func(path string) ([]byte, string, error) { return nil, "", nil }

// SetReadArtworkFn sets the function used to read artwork from media files.
func SetReadArtworkFn(fn func(string) ([]byte, string, error)) {
	readArtworkFn = fn
}
