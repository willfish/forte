//go:build !linux

package system

// MPRIS is a no-op media-control integration on platforms without MPRIS2.
type MPRIS struct{}

// NewMPRIS returns a no-op MPRIS integration.
func NewMPRIS(_ PlayerControl) (*MPRIS, error) {
	return &MPRIS{}, nil
}

// Close releases the no-op integration.
func (m *MPRIS) Close() {}

// UpdatePlaybackStatus records no platform state on unsupported platforms.
func (m *MPRIS) UpdatePlaybackStatus(_ string) {}

// UpdateMetadata records no platform state on unsupported platforms.
func (m *MPRIS) UpdateMetadata(_, _, _, _ string, _ int, _ int64) {}

// UpdateVolume records no platform state on unsupported platforms.
func (m *MPRIS) UpdateVolume(_ int) {}

// UpdateShuffle records no platform state on unsupported platforms.
func (m *MPRIS) UpdateShuffle(_ bool) {}

// UpdateLoopStatus records no platform state on unsupported platforms.
func (m *MPRIS) UpdateLoopStatus(_ string) {}

// UpdatePosition records no platform state on unsupported platforms.
func (m *MPRIS) UpdatePosition(_ float64) {}

// ClearMetadata records no platform state on unsupported platforms.
func (m *MPRIS) ClearMetadata() {}
