package player

import "sync"

// Toast represents a notification message for the frontend.
type Toast struct {
	Message string `json:"message"`
	Type    string `json:"type"` // "info", "warn", "error"
}

// Notifications is a thread-safe queue of toast messages.
type Notifications struct {
	mu    sync.Mutex
	queue []Toast
}

// NewNotifications creates an empty notification queue.
func NewNotifications() *Notifications {
	return &Notifications{}
}

// Push adds a toast to the queue.
func (n *Notifications) Push(message, typ string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.queue = append(n.queue, Toast{Message: message, Type: typ})
}

// Drain returns all pending toasts and clears the queue.
func (n *Notifications) Drain() []Toast {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.queue) == 0 {
		return nil
	}
	out := n.queue
	n.queue = nil
	return out
}
