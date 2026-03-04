package library

import (
	"log/slog"
	"sync"
	"time"
)

const (
	maxFailures     = 3
	backoffDuration = 2 * time.Minute
	pingInterval    = 30 * time.Second
)

// ServerStatus represents the online/offline state of a server.
type ServerStatus struct {
	ServerID string
	Online   bool
}

type serverState struct {
	online       bool
	failures     int
	backoffUntil time.Time
}

// HealthMonitor periodically pings configured servers and tracks their status.
type HealthMonitor struct {
	db       *DB
	mu       sync.RWMutex
	statuses map[string]*serverState
	stop     chan struct{}
	done     chan struct{}
}

// NewHealthMonitor creates a health monitor backed by the given database.
func NewHealthMonitor(db *DB) *HealthMonitor {
	return &HealthMonitor{
		db:       db,
		statuses: make(map[string]*serverState),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start launches the background ping loop.
func (h *HealthMonitor) Start() {
	go h.loop()
}

// Stop signals the ping loop to exit and waits for it to finish.
func (h *HealthMonitor) Stop() {
	close(h.stop)
	<-h.done
}

// IsOnline returns whether the given server is considered online.
// Unknown servers are assumed online.
func (h *HealthMonitor) IsOnline(serverID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	st, ok := h.statuses[serverID]
	if !ok {
		return true
	}
	return st.online
}

// Statuses returns the current status of all known servers.
func (h *HealthMonitor) Statuses() []ServerStatus {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make([]ServerStatus, 0, len(h.statuses))
	for id, st := range h.statuses {
		result = append(result, ServerStatus{ServerID: id, Online: st.online})
	}
	return result
}

func (h *HealthMonitor) loop() {
	defer close(h.done)

	// Ping immediately on start.
	h.pingAll()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			h.pingAll()
		case <-h.stop:
			return
		}
	}
}

func (h *HealthMonitor) pingAll() {
	servers, err := h.db.GetServers()
	if err != nil {
		slog.Warn("health: get servers", "err", err)
		return
	}

	now := time.Now()
	for _, srv := range servers {
		h.mu.RLock()
		st := h.statuses[srv.ID]
		h.mu.RUnlock()

		// Skip if in backoff.
		if st != nil && now.Before(st.backoffUntil) {
			continue
		}

		provider, err := newProvider(srv)
		if err != nil {
			slog.Warn("health: create provider", "server", srv.Name, "err", err)
			continue
		}

		pingErr := provider.Ping()

		h.mu.Lock()
		if h.statuses[srv.ID] == nil {
			h.statuses[srv.ID] = &serverState{online: true}
		}
		s := h.statuses[srv.ID]

		if pingErr != nil {
			s.failures++
			if s.failures >= maxFailures {
				s.online = false
				s.backoffUntil = now.Add(backoffDuration)
				slog.Info("health: server offline, backing off", "server", srv.Name)
			} else {
				slog.Debug("health: ping failed", "server", srv.Name, "failures", s.failures, "err", pingErr)
			}
		} else {
			if !s.online {
				slog.Info("health: server back online", "server", srv.Name)
			}
			s.online = true
			s.failures = 0
			s.backoffUntil = time.Time{}
		}
		h.mu.Unlock()
	}
}
