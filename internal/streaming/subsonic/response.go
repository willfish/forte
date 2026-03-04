package subsonic

import (
	"fmt"

	"github.com/willfish/forte/internal/streaming"
)

// subsonicResponse is the outer JSON envelope for all Subsonic API responses.
type subsonicResponse struct {
	Response responseBody `json:"subsonic-response"`
}

// responseBody is the inner body of a Subsonic API response.
type responseBody struct {
	Status  string `json:"status"`
	Version string `json:"version"`

	Error *apiError `json:"error,omitempty"`

	// Endpoint-specific fields.
	Artists    *artistIndex   `json:"artists,omitempty"`
	AlbumList *albumList     `json:"albumList2,omitempty"`
	Album     *albumDetail   `json:"album,omitempty"`
	Search    *searchResult3 `json:"searchResult3,omitempty"`
}

// apiError represents an error returned by the Subsonic API.
type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *apiError) Error() string {
	return fmt.Sprintf("subsonic error %d: %s", e.Code, e.Message)
}

// --- JSON structures matching the Subsonic API ---

type artistIndex struct {
	Index []indexEntry `json:"index"`
}

type indexEntry struct {
	Artists []artistJSON `json:"artist"`
}

type artistJSON struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	AlbumCount int    `json:"albumCount"`
}

type albumList struct {
	Albums []albumJSON `json:"album"`
}

type albumDetail struct {
	albumJSON
	Songs []songJSON `json:"song"`
}

type albumJSON struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	ArtistID   string `json:"artistId"`
	Year       int    `json:"year"`
	SongCount  int    `json:"songCount"`
	CoverArt   string `json:"coverArt"`
}

type songJSON struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	ArtistID    string `json:"artistId"`
	Album       string `json:"album"`
	AlbumID     string `json:"albumId"`
	CoverArt    string `json:"coverArt"`
	Duration    int    `json:"duration"` // seconds
	Track       int    `json:"track"`
	DiscNumber  int    `json:"discNumber"`
	Genre       string `json:"genre"`
	ContentType string `json:"contentType"`
	Size        int64  `json:"size"`
}

type searchResult3 struct {
	Artists []artistJSON `json:"artist"`
	Albums  []albumJSON  `json:"album"`
	Songs   []songJSON   `json:"song"`
}

// --- Conversion functions ---

func convertArtist(a artistJSON) streaming.Artist {
	return streaming.Artist{
		ID:         a.ID,
		Name:       a.Name,
		AlbumCount: a.AlbumCount,
	}
}

func convertAlbum(a albumJSON) streaming.Album {
	return streaming.Album{
		ID:         a.ID,
		Title:      a.Title,
		Artist:     a.Artist,
		ArtistID:   a.ArtistID,
		Year:       a.Year,
		TrackCount: a.SongCount,
		CoverArtID: a.CoverArt,
	}
}

func convertTrack(s songJSON) streaming.Track {
	return streaming.Track{
		ID:          s.ID,
		Title:       s.Title,
		Artist:      s.Artist,
		ArtistID:    s.ArtistID,
		Album:       s.Album,
		AlbumID:     s.AlbumID,
		CoverArtID:  s.CoverArt,
		DurationMs:  s.Duration * 1000,
		TrackNumber: s.Track,
		DiscNumber:  s.DiscNumber,
		Genre:       s.Genre,
		ContentType: s.ContentType,
		Size:        s.Size,
	}
}
