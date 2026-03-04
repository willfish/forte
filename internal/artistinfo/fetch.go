package artistinfo

import (
	"strings"
	"sync"
	"time"
)

// Result is the combined metadata from Last.fm and MusicBrainz.
type Result struct {
	Bio      string
	ImageURL string
	Similar  []SimilarArtist
	MbID     string
	MbArea   string
	MbType   string
	MbBegin  string
	MbEnd    string
	MbTags   string
}

var (
	mu             sync.Mutex
	lastLastFmCall time.Time
	lastMBCall     time.Time
)

const (
	lastFmMinInterval = 200 * time.Millisecond
	mbMinInterval     = 1 * time.Second
)

func throttleLastFm() {
	mu.Lock()
	defer mu.Unlock()
	if elapsed := time.Since(lastLastFmCall); elapsed < lastFmMinInterval {
		time.Sleep(lastFmMinInterval - elapsed)
	}
	lastLastFmCall = time.Now()
}

func throttleMB() {
	mu.Lock()
	defer mu.Unlock()
	if elapsed := time.Since(lastMBCall); elapsed < mbMinInterval {
		time.Sleep(mbMinInterval - elapsed)
	}
	lastMBCall = time.Now()
}

// Fetch retrieves artist metadata from Last.fm (primary) and MusicBrainz (supplementary).
func Fetch(apiKey, artistName string) (*Result, error) {
	result := &Result{}

	// Last.fm for bio, image, similar artists.
	if apiKey != "" {
		throttleLastFm()
		lfm, err := FetchLastFm(apiKey, artistName)
		if err == nil && lfm != nil {
			result.Bio = lfm.Bio
			result.ImageURL = lfm.ImageURL
			result.Similar = lfm.Similar
		}
	}

	// MusicBrainz for supplementary metadata.
	throttleMB()
	mb, err := FetchMusicBrainz(artistName)
	if err == nil && mb != nil {
		result.MbID = mb.ID
		result.MbArea = mb.Area
		result.MbType = mb.Type
		result.MbBegin = mb.Begin
		result.MbEnd = mb.End
		if len(mb.Tags) > 0 {
			result.MbTags = strings.Join(mb.Tags, ", ")
		}
	}

	return result, nil
}
