//go:build !linux

package system

// Notifier is a no-op desktop notification integration on platforms without
// Freedesktop notifications.
type Notifier struct {
	enabled bool
}

// NewNotifier returns a no-op notifier.
func NewNotifier() (*Notifier, error) {
	return &Notifier{enabled: true}, nil
}

// Notify records no platform state on unsupported platforms.
func (n *Notifier) Notify(_, _ string, _ []byte) {}

// SetEnabled enables or disables notifications.
func (n *Notifier) SetEnabled(enabled bool) {
	n.enabled = enabled
}

// Enabled returns whether notifications are enabled.
func (n *Notifier) Enabled() bool {
	return n.enabled
}

// Close releases the no-op notifier.
func (n *Notifier) Close() {}
