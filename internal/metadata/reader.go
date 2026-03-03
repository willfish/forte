// Package metadata reads audio file tags and properties.
package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.senan.xyz/taglib"
)

// TrackMeta holds metadata extracted from an audio file.
type TrackMeta struct {
	Title       string
	Artist      string
	AlbumArtist string
	Album       string
	TrackNumber int
	DiscNumber  int
	Year        int
	Genre       string
	Duration    time.Duration
	Bitrate     int
	SampleRate  int
	Channels    int
	HasArtwork  bool
}

// ReadTags reads metadata from the audio file at path.
// Falls back to filename parsing when tags are missing.
func ReadTags(path string) (TrackMeta, error) {
	tags, err := taglib.ReadTags(path)
	if err != nil {
		return TrackMeta{}, err
	}

	props, err := taglib.ReadProperties(path)
	if err != nil {
		return TrackMeta{}, err
	}

	meta := TrackMeta{
		Title:       first(tags[taglib.Title]),
		Artist:      first(tags[taglib.Artist]),
		AlbumArtist: first(tags[taglib.AlbumArtist]),
		Album:       first(tags[taglib.Album]),
		TrackNumber: parseNum(first(tags[taglib.TrackNumber])),
		DiscNumber:  parseNum(first(tags[taglib.DiscNumber])),
		Year:        parseNum(first(tags[taglib.Date])),
		Genre:       first(tags[taglib.Genre]),
		Duration:    props.Length,
		Bitrate:     int(props.Bitrate),
		SampleRate:  int(props.SampleRate),
		Channels:    int(props.Channels),
		HasArtwork:  len(props.Images) > 0,
	}

	// Fall back to filename if title is missing.
	if meta.Title == "" {
		meta.Title, meta.TrackNumber = parseFilename(path)
	}
	// Fall back to parent directory name if album is missing.
	if meta.Album == "" {
		meta.Album = parentDirName(path)
	}

	return meta, nil
}

// ReadArtwork returns the embedded front cover image bytes and MIME type.
// Returns empty values if no artwork is embedded.
func ReadArtwork(path string) ([]byte, string, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, "", fmt.Errorf("artwork: %w", err)
	}

	props, err := taglib.ReadProperties(path)
	if err != nil {
		return nil, "", err
	}

	if len(props.Images) == 0 {
		return nil, "", nil
	}

	data, err := taglib.ReadImage(path)
	if err != nil {
		return nil, "", err
	}

	return data, props.Images[0].MIMEType, nil
}

// first returns the first element of a string slice, or "" if empty.
func first(ss []string) string {
	if len(ss) > 0 {
		return ss[0]
	}
	return ""
}

// parseNum parses a string to int, handling "3/12" track number formats.
func parseNum(s string) int {
	if s == "" {
		return 0
	}
	// Handle "3/12" format (track 3 of 12).
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}

// parseFilename extracts a title and optional track number from a filename.
// Patterns: "01 - Track Name.flac", "01 Track Name.flac", "Track Name.flac"
func parseFilename(path string) (title string, trackNum int) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if name == "" {
		return "", 0
	}

	// Try "NN - Title" or "NN. Title" patterns.
	for _, sep := range []string{" - ", ". ", " "} {
		if i := strings.Index(name, sep); i > 0 {
			prefix := name[:i]
			if n, err := strconv.Atoi(prefix); err == nil {
				return strings.TrimSpace(name[i+len(sep):]), n
			}
		}
	}

	return name, 0
}

// parentDirName returns the name of the parent directory.
func parentDirName(path string) string {
	return filepath.Base(filepath.Dir(path))
}
