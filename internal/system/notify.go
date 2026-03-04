package system

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/godbus/dbus/v5"
)

const (
	notifyDest   = "org.freedesktop.Notifications"
	notifyPath   = "/org/freedesktop/Notifications"
	notifyIface  = "org.freedesktop.Notifications"
	notifyMethod = notifyIface + ".Notify"
)

// Notifier sends desktop notifications via D-Bus.
type Notifier struct {
	mu         sync.Mutex
	conn       *dbus.Conn
	replacesID uint32
	enabled    bool
	iconDir    string
}

// NewNotifier connects to the session bus for sending notifications.
func NewNotifier() (*Notifier, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("notify: connect session bus: %w", err)
	}

	iconDir, err := os.MkdirTemp("", "forte-notify-*")
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("notify: create icon dir: %w", err)
	}

	return &Notifier{
		conn:    conn,
		enabled: true,
		iconDir: iconDir,
	}, nil
}

// Notify sends a desktop notification. If artwork is non-nil, it is
// used as the notification icon. Each call replaces the previous notification.
func (n *Notifier) Notify(title, body string, artwork []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.enabled {
		return
	}

	icon := "forte"
	if len(artwork) > 0 {
		path := filepath.Join(n.iconDir, "notify-icon.jpg")
		if err := os.WriteFile(path, artwork, 0o644); err == nil {
			icon = path
		}
	}

	obj := n.conn.Object(notifyDest, notifyPath)
	call := obj.Call(
		notifyMethod, 0,
		"Forte",          // app_name
		n.replacesID,     // replaces_id
		icon,             // app_icon
		title,            // summary
		body,             // body
		[]string{},       // actions
		map[string]dbus.Variant{}, // hints
		int32(5000),      // expire_timeout ms
	)
	if call.Err != nil {
		return
	}

	// Store the returned ID so the next notification replaces this one.
	if err := call.Store(&n.replacesID); err == nil {
		return
	}
}

// SetEnabled enables or disables notifications.
func (n *Notifier) SetEnabled(enabled bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.enabled = enabled
}

// Enabled returns whether notifications are enabled.
func (n *Notifier) Enabled() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.enabled
}

// Close releases the D-Bus connection and cleans up temp files.
func (n *Notifier) Close() {
	if n.conn != nil {
		_ = n.conn.Close()
	}
	if n.iconDir != "" {
		_ = os.RemoveAll(n.iconDir)
	}
}
