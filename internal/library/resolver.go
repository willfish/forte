package library

import (
	"fmt"
	"strings"
	"sync"

	"github.com/willfish/forte/internal/streaming"
	"github.com/willfish/forte/internal/streaming/jellyfin"
	"github.com/willfish/forte/internal/streaming/subsonic"
)

const serverPathPrefix = "server://"

// PathResolver translates server:// file paths into streaming URLs.
type PathResolver struct {
	db        *DB
	mu        sync.RWMutex
	providers map[string]streaming.Provider
}

// NewPathResolver creates a resolver backed by the given database.
func NewPathResolver(db *DB) *PathResolver {
	return &PathResolver{
		db:        db,
		providers: make(map[string]streaming.Provider),
	}
}

// Resolve returns a playable path. Local paths are returned unchanged.
// Server paths (server://{serverID}/{remoteID}) are resolved to streaming URLs.
func (r *PathResolver) Resolve(filePath string) (string, error) {
	if !IsServerPath(filePath) {
		return filePath, nil
	}

	serverID, remoteID, err := ParseServerPath(filePath)
	if err != nil {
		return "", err
	}

	provider, err := r.getProvider(serverID)
	if err != nil {
		return "", fmt.Errorf("resolve: %w", err)
	}

	return provider.StreamURL(remoteID), nil
}

// IsServerPath returns true if the path uses the server:// scheme.
func IsServerPath(filePath string) bool {
	return strings.HasPrefix(filePath, serverPathPrefix)
}

// ParseServerPath extracts server ID and remote ID from a server:// path.
func ParseServerPath(filePath string) (serverID, remoteID string, err error) {
	rest := strings.TrimPrefix(filePath, serverPathPrefix)
	idx := strings.IndexByte(rest, '/')
	if idx < 0 {
		return "", "", fmt.Errorf("invalid server path: %s", filePath)
	}
	return rest[:idx], rest[idx+1:], nil
}

func (r *PathResolver) getProvider(serverID string) (streaming.Provider, error) {
	r.mu.RLock()
	p, ok := r.providers[serverID]
	r.mu.RUnlock()
	if ok {
		return p, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Double-check after acquiring write lock.
	if p, ok := r.providers[serverID]; ok {
		return p, nil
	}

	srv, err := r.db.GetServer(serverID)
	if err != nil {
		return nil, fmt.Errorf("get server %s: %w", serverID, err)
	}

	var provider streaming.Provider
	switch srv.Type {
	case "subsonic":
		provider = subsonic.New(srv.URL, srv.Username, srv.Password)
	case "jellyfin":
		provider = jellyfin.New(srv.URL, srv.Username, srv.Password)
	default:
		return nil, fmt.Errorf("unknown server type: %s", srv.Type)
	}

	r.providers[serverID] = provider
	return provider, nil
}
