package system

import "testing"

func TestNotifierEnabledDefault(t *testing.T) {
	// Manually construct to avoid needing D-Bus.
	n := &Notifier{enabled: true}
	if !n.Enabled() {
		t.Error("expected enabled by default")
	}
}

func TestNotifierSetEnabled(t *testing.T) {
	n := &Notifier{enabled: true}

	n.SetEnabled(false)
	if n.Enabled() {
		t.Error("expected disabled after SetEnabled(false)")
	}

	n.SetEnabled(true)
	if !n.Enabled() {
		t.Error("expected enabled after SetEnabled(true)")
	}
}

func TestNotifyDisabledNoOp(t *testing.T) {
	// With a nil conn, Notify would panic if it tried to use D-Bus.
	// When disabled, it should return early without touching conn.
	n := &Notifier{enabled: false, conn: nil}
	n.Notify("Title", "Body", nil) // no panic = pass
}

func TestNotifyDisabledWithArtwork(t *testing.T) {
	n := &Notifier{enabled: false, conn: nil}
	n.Notify("Title", "Body", []byte{0xFF, 0xD8}) // no panic = pass
}

func TestCloseNilConn(t *testing.T) {
	n := &Notifier{conn: nil, iconDir: ""}
	n.Close() // no panic = pass
}
