// Package cue parses CUE sheet files into structured track data.
package cue

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Sheet represents a parsed CUE sheet.
type Sheet struct {
	Performer string
	Title     string
	Files     []File
}

// File represents a FILE entry in a CUE sheet.
type File struct {
	Name   string
	Format string
	Tracks []Track
}

// Track represents a TRACK entry in a CUE sheet.
type Track struct {
	Number    int
	Performer string
	Title     string
	StartMs   int // INDEX 01 time in milliseconds
}

// Parse reads a CUE sheet from r and returns the parsed structure.
func Parse(r io.Reader) (*Sheet, error) {
	scanner := bufio.NewScanner(r)
	sheet := &Sheet{}

	var currentFile *File
	var currentTrack *Track

	for scanner.Scan() {
		line := stripBOM(strings.TrimSpace(scanner.Text()))
		if line == "" || strings.HasPrefix(line, "REM ") {
			continue
		}

		keyword, rest := splitFirst(line)

		switch strings.ToUpper(keyword) {
		case "PERFORMER":
			val := unquote(rest)
			if currentTrack != nil {
				currentTrack.Performer = val
			} else {
				sheet.Performer = val
			}

		case "TITLE":
			val := unquote(rest)
			if currentTrack != nil {
				currentTrack.Title = val
			} else {
				sheet.Title = val
			}

		case "FILE":
			// Flush the current track before switching files.
			if currentTrack != nil && currentFile != nil {
				currentFile.Tracks = append(currentFile.Tracks, *currentTrack)
				currentTrack = nil
			}

			name, format := parseFileArgs(rest)
			f := File{Name: name, Format: format}
			sheet.Files = append(sheet.Files, f)
			currentFile = &sheet.Files[len(sheet.Files)-1]

		case "TRACK":
			// Flush the previous track.
			if currentTrack != nil && currentFile != nil {
				currentFile.Tracks = append(currentFile.Tracks, *currentTrack)
			}

			num, err := parseTrackNumber(rest)
			if err != nil {
				return nil, fmt.Errorf("line %q: %w", line, err)
			}
			currentTrack = &Track{Number: num}

		case "INDEX":
			if currentTrack == nil {
				continue
			}
			idx, timeStr := splitFirst(rest)
			if idx != "01" {
				continue // We only care about INDEX 01 (track start).
			}
			ms, err := parseTime(timeStr)
			if err != nil {
				return nil, fmt.Errorf("line %q: %w", line, err)
			}
			currentTrack.StartMs = ms

		case "CATALOG", "ISRC", "FLAGS", "PREGAP", "POSTGAP":
			// Recognised but not needed for playback.
		}
	}

	// Flush the last track.
	if currentTrack != nil && currentFile != nil {
		currentFile.Tracks = append(currentFile.Tracks, *currentTrack)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading cue sheet: %w", err)
	}

	return sheet, nil
}

// parseTime converts MM:SS:FF to milliseconds.
// FF is frames at 75fps (1 frame = 1000/75 = 13.333ms).
func parseTime(s string) (int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid time %q: expected MM:SS:FF", s)
	}

	mm, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes in %q: %w", s, err)
	}
	ss, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid seconds in %q: %w", s, err)
	}
	ff, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid frames in %q: %w", s, err)
	}

	return mm*60*1000 + ss*1000 + ff*1000/75, nil
}

// splitFirst splits on the first whitespace.
func splitFirst(s string) (string, string) {
	i := strings.IndexAny(s, " \t")
	if i < 0 {
		return s, ""
	}
	return s[:i], strings.TrimSpace(s[i+1:])
}

// unquote removes surrounding double quotes.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// parseFileArgs extracts the filename and format from a FILE line's arguments.
// e.g., `"album.wav" WAVE` -> ("album.wav", "WAVE")
func parseFileArgs(s string) (string, string) {
	if len(s) > 0 && s[0] == '"' {
		end := strings.Index(s[1:], "\"")
		if end >= 0 {
			name := s[1 : end+1]
			rest := strings.TrimSpace(s[end+2:])
			return name, rest
		}
	}
	// Unquoted filename.
	name, format := splitFirst(s)
	return name, format
}

// parseTrackNumber extracts the track number from "NN AUDIO".
func parseTrackNumber(s string) (int, error) {
	numStr, _ := splitFirst(s)
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid track number %q: %w", numStr, err)
	}
	return n, nil
}

// stripBOM removes a UTF-8 BOM prefix if present.
func stripBOM(s string) string {
	return strings.TrimPrefix(s, "\xef\xbb\xbf")
}
