package library

import (
	"fmt"

	"github.com/willfish/forte/internal/streaming"
	"github.com/willfish/forte/internal/streaming/jellyfin"
	"github.com/willfish/forte/internal/streaming/subsonic"
)

// NewServerProvider creates the streaming provider adapter for a configured server.
func NewServerProvider(srv Server) (streaming.Provider, error) {
	switch srv.Type {
	case "subsonic":
		return subsonic.New(srv.URL, srv.Username, srv.Password), nil
	case "jellyfin":
		return jellyfin.New(srv.URL, srv.Username, srv.Password), nil
	default:
		return nil, fmt.Errorf("unknown server type: %s", srv.Type)
	}
}
