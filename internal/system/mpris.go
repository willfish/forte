// Package system provides Linux desktop integration (MPRIS2, notifications, tray).
package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/godbus/dbus/v5"
	"github.com/godbus/dbus/v5/introspect"
	"github.com/godbus/dbus/v5/prop"
)

const (
	busName    = "org.mpris.MediaPlayer2.forte"
	objectPath = "/org/mpris/MediaPlayer2"

	ifaceRoot   = "org.mpris.MediaPlayer2"
	ifacePlayer = "org.mpris.MediaPlayer2.Player"
)

// readArtworkFn reads album artwork from a media file.
// Set via SetReadArtworkFn to avoid a direct dependency on the metadata package.
var readArtworkFn = func(path string) ([]byte, string, error) { return nil, "", nil }

// SetReadArtworkFn sets the function used to read artwork from media files.
func SetReadArtworkFn(fn func(string) ([]byte, string, error)) {
	readArtworkFn = fn
}

// PlayerControl is the interface that the MPRIS provider uses to control the player.
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
	State() string // "playing", "paused", "stopped"
	MediaPath() string
	SetShuffle(enabled bool)
	GetShuffle() bool
	SetRepeat(mode string) // "off", "all", "one"
	GetRepeat() string
}

// MPRIS provides MPRIS2 D-Bus integration for Forte.
type MPRIS struct {
	mu      sync.Mutex
	conn    *dbus.Conn
	props   *prop.Properties
	player  PlayerControl
	artDir  string
	artPath string // current temp art file path
}

// NewMPRIS creates and registers the MPRIS2 service on the session bus.
func NewMPRIS(player PlayerControl) (*MPRIS, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("mpris: connect session bus: %w", err)
	}

	artDir, err := os.MkdirTemp("", "forte-art-*")
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("mpris: create art dir: %w", err)
	}

	m := &MPRIS{
		conn:   conn,
		player: player,
		artDir: artDir,
	}

	// Export properties.
	propsSpec := prop.Map{
		ifaceRoot: {
			"CanQuit":             {Value: true, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"CanRaise":            {Value: false, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"HasTrackList":        {Value: false, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"Identity":            {Value: "Forte", Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"DesktopEntry":        {Value: "forte", Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"SupportedUriSchemes": {Value: []string{"file"}, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"SupportedMimeTypes":  {Value: []string{"audio/mpeg", "audio/flac", "audio/ogg", "audio/wav", "audio/x-wav"}, Writable: false, Emit: prop.EmitTrue, Callback: nil},
		},
		ifacePlayer: {
			"PlaybackStatus": {Value: "Stopped", Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"LoopStatus":     {Value: "None", Writable: true, Emit: prop.EmitTrue, Callback: m.onLoopStatusChanged},
			"Rate":           {Value: 1.0, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"Shuffle":        {Value: false, Writable: true, Emit: prop.EmitTrue, Callback: m.onShuffleChanged},
			"Volume":         {Value: 1.0, Writable: true, Emit: prop.EmitTrue, Callback: m.onVolumeChanged},
			"Metadata":       {Value: emptyMetadata(), Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"Position":       {Value: int64(0), Writable: false, Emit: prop.EmitFalse, Callback: nil},
			"MinimumRate":    {Value: 1.0, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"MaximumRate":    {Value: 1.0, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"CanGoNext":      {Value: true, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"CanGoPrevious":  {Value: true, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"CanPlay":        {Value: true, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"CanPause":       {Value: true, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"CanSeek":        {Value: true, Writable: false, Emit: prop.EmitTrue, Callback: nil},
			"CanControl":     {Value: true, Writable: false, Emit: prop.EmitConst, Callback: nil},
		},
	}

	cleanup := func() {
		_ = os.RemoveAll(artDir)
		_ = conn.Close()
	}

	m.props, err = prop.Export(conn, dbus.ObjectPath(objectPath), propsSpec)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("mpris: export properties: %w", err)
	}

	// Export method handlers.
	_ = conn.Export(&mprisRoot{m: m}, dbus.ObjectPath(objectPath), ifaceRoot)

	// Use ExportMethodTable for the Player interface to avoid a go vet
	// false positive on the Seek method name (conflicts with io.Seeker).
	pl := &mprisPlayer{m: m}
	_ = conn.ExportMethodTable(map[string]interface{}{
		"Play":        pl.Play,
		"Pause":       pl.Pause,
		"PlayPause":   pl.PlayPause,
		"Stop":        pl.Stop,
		"Next":        pl.Next,
		"Previous":    pl.Previous,
		"Seek":        pl.SeekOffset,
		"SetPosition": pl.SetPosition,
		"OpenUri":     pl.OpenUri,
	}, dbus.ObjectPath(objectPath), ifacePlayer)

	// Export introspection.
	_ = conn.Export(
		introspect.NewIntrospectable(mprisIntrospectNode()),
		dbus.ObjectPath(objectPath),
		"org.freedesktop.DBus.Introspectable",
	)

	// Claim the bus name.
	reply, err := conn.RequestName(busName, dbus.NameFlagReplaceExisting)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("mpris: request name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		cleanup()
		return nil, fmt.Errorf("mpris: name %s already taken", busName)
	}

	return m, nil
}

// Close releases the D-Bus connection and cleans up temp files.
func (m *MPRIS) Close() {
	if m.conn != nil {
		_, _ = m.conn.ReleaseName(busName)
		_ = m.conn.Close()
	}
	if m.artDir != "" {
		_ = os.RemoveAll(m.artDir)
	}
}

// --- D-Bus method handler structs ---

// mprisRoot handles org.mpris.MediaPlayer2 methods.
type mprisRoot struct{ m *MPRIS }

func (r *mprisRoot) Raise() *dbus.Error { return nil }

func (r *mprisRoot) Quit() *dbus.Error {
	os.Exit(0)
	return nil
}

// mprisPlayer handles org.mpris.MediaPlayer2.Player methods.
type mprisPlayer struct{ m *MPRIS }

func (p *mprisPlayer) Play() *dbus.Error {
	p.m.player.Resume()
	return nil
}

func (p *mprisPlayer) Pause() *dbus.Error {
	p.m.player.Pause()
	return nil
}

func (p *mprisPlayer) PlayPause() *dbus.Error {
	if p.m.player.State() == "playing" {
		p.m.player.Pause()
	} else {
		p.m.player.Resume()
	}
	return nil
}

func (p *mprisPlayer) Stop() *dbus.Error {
	p.m.player.Stop()
	return nil
}

func (p *mprisPlayer) Next() *dbus.Error {
	p.m.player.Next()
	return nil
}

func (p *mprisPlayer) Previous() *dbus.Error {
	p.m.player.Previous()
	return nil
}

// clampSeek calculates a new position from a microsecond offset, clamped to [0, duration].
func clampSeek(currentPos float64, offsetUs int64, duration float64) float64 {
	newPos := currentPos + float64(offsetUs)/1e6
	if newPos < 0 {
		newPos = 0
	}
	if duration > 0 && newPos > duration {
		newPos = duration
	}
	return newPos
}

func (p *mprisPlayer) SeekOffset(offsetUs int64) *dbus.Error {
	newPos := clampSeek(p.m.player.Position(), offsetUs, p.m.player.Duration())
	p.m.player.Seek(newPos)

	_ = p.m.conn.Emit(
		dbus.ObjectPath(objectPath),
		ifacePlayer+".Seeked",
		int64(newPos*1e6),
	)
	return nil
}

func (p *mprisPlayer) SetPosition(trackID dbus.ObjectPath, positionUs int64) *dbus.Error {
	newPos := float64(positionUs) / 1e6
	dur := p.m.player.Duration()
	if newPos < 0 || (dur > 0 && newPos > dur) {
		return nil
	}
	p.m.player.Seek(newPos)

	_ = p.m.conn.Emit(
		dbus.ObjectPath(objectPath),
		ifacePlayer+".Seeked",
		positionUs,
	)
	return nil
}

func (p *mprisPlayer) OpenUri(uri string) *dbus.Error {
	return nil
}

// --- Property change callbacks (from external D-Bus Set calls) ---

func (m *MPRIS) onVolumeChanged(c *prop.Change) *dbus.Error {
	vol := c.Value.(float64)
	m.player.SetVolume(int(vol * 100))
	return nil
}

func (m *MPRIS) onShuffleChanged(c *prop.Change) *dbus.Error {
	m.player.SetShuffle(c.Value.(bool))
	return nil
}

func (m *MPRIS) onLoopStatusChanged(c *prop.Change) *dbus.Error {
	m.player.SetRepeat(mapLoopStatusToRepeat(c.Value.(string)))
	return nil
}

// --- State update methods (called by PlayerService to push changes) ---

// mapPlaybackStatus converts an internal state to the MPRIS2 PlaybackStatus.
func mapPlaybackStatus(state string) string {
	switch state {
	case "playing":
		return "Playing"
	case "paused":
		return "Paused"
	default:
		return "Stopped"
	}
}

// UpdatePlaybackStatus pushes a playback state change to D-Bus.
func (m *MPRIS) UpdatePlaybackStatus(state string) {
	m.setProp(ifacePlayer, "PlaybackStatus", dbus.MakeVariant(mapPlaybackStatus(state)))
}

// buildMetadata constructs an MPRIS2 metadata map from track info.
// artURL is optional; pass "" to omit artwork.
func buildMetadata(title, artist, album, filePath string, durationMs int, trackID int64, artURL string) map[string]dbus.Variant {
	md := map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/forte/track/" + strconv.FormatInt(trackID, 10))),
		"mpris:length":  dbus.MakeVariant(int64(durationMs) * 1000),
	}

	if title != "" {
		md["xesam:title"] = dbus.MakeVariant(title)
	}
	if artist != "" {
		md["xesam:artist"] = dbus.MakeVariant([]string{artist})
	}
	if album != "" {
		md["xesam:album"] = dbus.MakeVariant(album)
	}
	if filePath != "" {
		md["xesam:url"] = dbus.MakeVariant("file://" + filePath)
	}
	if artURL != "" {
		md["mpris:artUrl"] = dbus.MakeVariant(artURL)
	}

	return md
}

// UpdateMetadata pushes track metadata to D-Bus.
func (m *MPRIS) UpdateMetadata(title, artist, album, filePath string, durationMs int, trackID int64) {
	artURL := m.exportArtwork()
	md := buildMetadata(title, artist, album, filePath, durationMs, trackID, artURL)
	m.setProp(ifacePlayer, "Metadata", dbus.MakeVariant(md))
}

// UpdateVolume pushes volume to D-Bus (0-100 -> 0.0-1.0).
func (m *MPRIS) UpdateVolume(percent int) {
	m.setProp(ifacePlayer, "Volume", dbus.MakeVariant(float64(percent)/100.0))
}

// UpdateShuffle pushes shuffle state to D-Bus.
func (m *MPRIS) UpdateShuffle(enabled bool) {
	m.setProp(ifacePlayer, "Shuffle", dbus.MakeVariant(enabled))
}

// mapRepeatToLoopStatus converts an internal repeat mode to MPRIS2 LoopStatus.
func mapRepeatToLoopStatus(mode string) string {
	switch mode {
	case "one":
		return "Track"
	case "all":
		return "Playlist"
	default:
		return "None"
	}
}

// mapLoopStatusToRepeat converts an MPRIS2 LoopStatus to an internal repeat mode.
func mapLoopStatusToRepeat(status string) string {
	switch status {
	case "Track":
		return "one"
	case "Playlist":
		return "all"
	default:
		return "off"
	}
}

// UpdateLoopStatus pushes repeat mode to D-Bus.
func (m *MPRIS) UpdateLoopStatus(mode string) {
	m.setProp(ifacePlayer, "LoopStatus", dbus.MakeVariant(mapRepeatToLoopStatus(mode)))
}

// UpdatePosition updates the Position property (no signal emitted per spec).
func (m *MPRIS) UpdatePosition(seconds float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.props == nil {
		return
	}
	m.props.SetMust(ifacePlayer, "Position", int64(seconds*1e6))
}

// ClearMetadata resets metadata to empty (when stopped).
func (m *MPRIS) ClearMetadata() {
	m.setProp(ifacePlayer, "Metadata", dbus.MakeVariant(emptyMetadata()))
}

func (m *MPRIS) setProp(iface, name string, value dbus.Variant) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.props == nil {
		return
	}
	// Use SetMust for internal updates - Set() enforces the Writable flag
	// which rejects writes to read-only properties like PlaybackStatus and
	// Metadata. Those are correctly read-only for external D-Bus clients,
	// but the application itself needs to update them.
	m.props.SetMust(iface, name, value.Value())
}

func (m *MPRIS) exportArtwork() string {
	mediaPath := m.player.MediaPath()
	if mediaPath == "" {
		return ""
	}

	data, _, err := readArtworkFn(mediaPath)
	if err != nil || len(data) == 0 {
		return ""
	}

	path := filepath.Join(m.artDir, "cover.jpg")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return ""
	}
	m.artPath = path
	return "file://" + path
}

func emptyMetadata() map[string]dbus.Variant {
	return map[string]dbus.Variant{
		"mpris:trackid": dbus.MakeVariant(dbus.ObjectPath("/org/mpris/MediaPlayer2/TrackList/NoTrack")),
	}
}

func mprisIntrospectNode() *introspect.Node {
	return &introspect.Node{
		Name: objectPath,
		Interfaces: []introspect.Interface{
			introspect.IntrospectData,
			{
				Name: ifaceRoot,
				Methods: []introspect.Method{
					{Name: "Raise"},
					{Name: "Quit"},
				},
				Properties: []introspect.Property{
					{Name: "CanQuit", Type: "b", Access: "read"},
					{Name: "CanRaise", Type: "b", Access: "read"},
					{Name: "HasTrackList", Type: "b", Access: "read"},
					{Name: "Identity", Type: "s", Access: "read"},
					{Name: "DesktopEntry", Type: "s", Access: "read"},
					{Name: "SupportedUriSchemes", Type: "as", Access: "read"},
					{Name: "SupportedMimeTypes", Type: "as", Access: "read"},
				},
			},
			{
				Name: ifacePlayer,
				Methods: []introspect.Method{
					{Name: "Next"},
					{Name: "Previous"},
					{Name: "Pause"},
					{Name: "Play"},
					{Name: "PlayPause"},
					{Name: "Stop"},
					{Name: "Seek", Args: []introspect.Arg{{Name: "Offset", Type: "x", Direction: "in"}}},
					{Name: "SetPosition", Args: []introspect.Arg{
						{Name: "TrackId", Type: "o", Direction: "in"},
						{Name: "Position", Type: "x", Direction: "in"},
					}},
					{Name: "OpenUri", Args: []introspect.Arg{{Name: "Uri", Type: "s", Direction: "in"}}},
				},
				Properties: []introspect.Property{
					{Name: "PlaybackStatus", Type: "s", Access: "read"},
					{Name: "LoopStatus", Type: "s", Access: "readwrite"},
					{Name: "Rate", Type: "d", Access: "readwrite"},
					{Name: "Shuffle", Type: "b", Access: "readwrite"},
					{Name: "Metadata", Type: "a{sv}", Access: "read"},
					{Name: "Volume", Type: "d", Access: "readwrite"},
					{Name: "Position", Type: "x", Access: "read"},
					{Name: "MinimumRate", Type: "d", Access: "read"},
					{Name: "MaximumRate", Type: "d", Access: "read"},
					{Name: "CanGoNext", Type: "b", Access: "read"},
					{Name: "CanGoPrevious", Type: "b", Access: "read"},
					{Name: "CanPlay", Type: "b", Access: "read"},
					{Name: "CanPause", Type: "b", Access: "read"},
					{Name: "CanSeek", Type: "b", Access: "read"},
					{Name: "CanControl", Type: "b", Access: "read"},
				},
				Signals: []introspect.Signal{
					{Name: "Seeked", Args: []introspect.Arg{{Name: "Position", Type: "x"}}},
				},
			},
		},
	}
}
