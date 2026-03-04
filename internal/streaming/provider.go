// Package streaming defines the interface and domain types for music streaming providers.
package streaming

// Provider is the interface that all streaming backends must implement.
type Provider interface {
	Ping() error
	GetArtists() ([]Artist, error)
	GetAlbums(sortBy string, offset, count int) ([]Album, error)
	GetAlbum(id string) (Album, []Track, error)
	StreamURL(trackID string) string
	CoverArtURL(coverArtID string) string
	Search(query string) (SearchResults, error)
}

// Artist represents a music artist from a streaming server.
type Artist struct {
	ID         string
	Name       string
	AlbumCount int
}

// Album represents a music album from a streaming server.
type Album struct {
	ID         string
	Title      string
	Artist     string
	ArtistID   string
	Year       int
	TrackCount int
	CoverArtID string
}

// Track represents a single track from a streaming server.
type Track struct {
	ID          string
	Title       string
	Artist      string
	ArtistID    string
	Album       string
	AlbumID     string
	CoverArtID  string
	DurationMs  int
	TrackNumber int
	DiscNumber  int
	Genre       string
	ContentType string
	Size        int64
}

// SearchResults holds the results of a search query across artists, albums, and tracks.
type SearchResults struct {
	Artists []Artist
	Albums  []Album
	Tracks  []Track
}
