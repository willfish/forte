package jellyfin

import "github.com/willfish/forte/internal/streaming"

// itemsResponse is the standard list wrapper for Jellyfin API responses.
type itemsResponse struct {
	Items            []itemJSON `json:"Items"`
	TotalRecordCount int        `json:"TotalRecordCount"`
}

// itemJSON is the unified item struct used across Jellyfin API responses.
// Artists, albums, and tracks are all represented as Items with different Type values.
type itemJSON struct {
	Name              string        `json:"Name"`
	ID                string        `json:"Id"`
	Type              string        `json:"Type"`
	AlbumArtist       string        `json:"AlbumArtist"`
	AlbumArtists      []nameIDPair  `json:"AlbumArtists"`
	Artists           []string      `json:"Artists"`
	Album             string        `json:"Album"`
	AlbumID           string        `json:"AlbumId"`
	ProductionYear    int           `json:"ProductionYear"`
	ChildCount        int           `json:"ChildCount"`
	RunTimeTicks      int64         `json:"RunTimeTicks"`
	IndexNumber       int           `json:"IndexNumber"`
	ParentIndexNumber int           `json:"ParentIndexNumber"`
	MediaSources      []mediaSource `json:"MediaSources"`
	ImageTags         map[string]string `json:"ImageTags"`
	Genres            []string      `json:"Genres"`
}

// nameIDPair is used in AlbumArtists and ArtistItems arrays.
type nameIDPair struct {
	Name string `json:"Name"`
	ID   string `json:"Id"`
}

// mediaSource holds container, size, and bitrate information for a track.
type mediaSource struct {
	Container string `json:"Container"`
	Size      int64  `json:"Size"`
	Bitrate   int    `json:"Bitrate"`
}

// authResponse is the response from /Users/AuthenticateByName.
type authResponse struct {
	AccessToken string   `json:"AccessToken"`
	User        authUser `json:"User"`
}

// authUser holds the user ID from the auth response.
type authUser struct {
	ID string `json:"Id"`
}

// ticksToMs converts .NET ticks (100ns intervals) to milliseconds.
func ticksToMs(ticks int64) int {
	return int(ticks / 10_000)
}

func convertArtist(item itemJSON) streaming.Artist {
	return streaming.Artist{
		ID:         item.ID,
		Name:       item.Name,
		AlbumCount: item.ChildCount,
	}
}

func convertAlbum(item itemJSON) streaming.Album {
	artist := item.AlbumArtist
	var artistID string
	if len(item.AlbumArtists) > 0 {
		artistID = item.AlbumArtists[0].ID
		if artist == "" {
			artist = item.AlbumArtists[0].Name
		}
	}

	return streaming.Album{
		ID:         item.ID,
		Title:      item.Name,
		Artist:     artist,
		ArtistID:   artistID,
		Year:       item.ProductionYear,
		TrackCount: item.ChildCount,
		CoverArtID: item.ID,
	}
}

func convertTrack(item itemJSON) streaming.Track {
	artist := item.AlbumArtist
	var artistID string
	if len(item.AlbumArtists) > 0 {
		artistID = item.AlbumArtists[0].ID
		if artist == "" {
			artist = item.AlbumArtists[0].Name
		}
	}
	if artist == "" && len(item.Artists) > 0 {
		artist = item.Artists[0]
	}

	var genre string
	if len(item.Genres) > 0 {
		genre = item.Genres[0]
	}

	var contentType string
	var size int64
	if len(item.MediaSources) > 0 {
		contentType = "audio/" + item.MediaSources[0].Container
		size = item.MediaSources[0].Size
	}

	return streaming.Track{
		ID:          item.ID,
		Title:       item.Name,
		Artist:      artist,
		ArtistID:    artistID,
		Album:       item.Album,
		AlbumID:     item.AlbumID,
		CoverArtID:  item.ID,
		DurationMs:  ticksToMs(item.RunTimeTicks),
		TrackNumber: item.IndexNumber,
		DiscNumber:  item.ParentIndexNumber,
		Genre:       genre,
		ContentType: contentType,
		Size:        size,
	}
}
