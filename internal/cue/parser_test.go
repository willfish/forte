package cue

import (
	"strings"
	"testing"
)

const singleFileCue = `PERFORMER "Test Artist"
TITLE "Test Album"
FILE "album.flac" FLAC
  TRACK 01 AUDIO
    PERFORMER "Test Artist"
    TITLE "Track One"
    INDEX 01 00:00:00
  TRACK 02 AUDIO
    TITLE "Track Two"
    INDEX 01 05:30:00
  TRACK 03 AUDIO
    TITLE "Track Three"
    INDEX 01 12:15:45
`

const multiFileCue = `PERFORMER "Test Artist"
TITLE "Multi-File Album"
FILE "disc1.wav" WAVE
  TRACK 01 AUDIO
    TITLE "Disc 1 Track 1"
    INDEX 01 00:00:00
  TRACK 02 AUDIO
    TITLE "Disc 1 Track 2"
    INDEX 01 05:30:00
FILE "disc2.wav" WAVE
  TRACK 03 AUDIO
    TITLE "Disc 2 Track 1"
    INDEX 01 00:00:00
  TRACK 04 AUDIO
    TITLE "Disc 2 Track 2"
    INDEX 01 06:15:30
`

func TestParseSingleFile(t *testing.T) {
	sheet, err := Parse(strings.NewReader(singleFileCue))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if sheet.Performer != "Test Artist" {
		t.Errorf("Performer = %q, want %q", sheet.Performer, "Test Artist")
	}
	if sheet.Title != "Test Album" {
		t.Errorf("Title = %q, want %q", sheet.Title, "Test Album")
	}

	if len(sheet.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(sheet.Files))
	}

	f := sheet.Files[0]
	if f.Name != "album.flac" {
		t.Errorf("File.Name = %q, want %q", f.Name, "album.flac")
	}
	if f.Format != "FLAC" {
		t.Errorf("File.Format = %q, want %q", f.Format, "FLAC")
	}
	if len(f.Tracks) != 3 {
		t.Fatalf("len(Tracks) = %d, want 3", len(f.Tracks))
	}

	// Track 1.
	if f.Tracks[0].Number != 1 {
		t.Errorf("Track[0].Number = %d, want 1", f.Tracks[0].Number)
	}
	if f.Tracks[0].Title != "Track One" {
		t.Errorf("Track[0].Title = %q, want %q", f.Tracks[0].Title, "Track One")
	}
	if f.Tracks[0].Performer != "Test Artist" {
		t.Errorf("Track[0].Performer = %q, want %q", f.Tracks[0].Performer, "Test Artist")
	}
	if f.Tracks[0].StartMs != 0 {
		t.Errorf("Track[0].StartMs = %d, want 0", f.Tracks[0].StartMs)
	}

	// Track 2: 05:30:00 = 5*60*1000 + 30*1000 = 330000ms.
	if f.Tracks[1].StartMs != 330000 {
		t.Errorf("Track[1].StartMs = %d, want 330000", f.Tracks[1].StartMs)
	}

	// Track 3: 12:15:45 = 12*60*1000 + 15*1000 + 45*1000/75 = 735600ms.
	if f.Tracks[2].StartMs != 735600 {
		t.Errorf("Track[2].StartMs = %d, want 735600", f.Tracks[2].StartMs)
	}
}

func TestParseMultiFile(t *testing.T) {
	sheet, err := Parse(strings.NewReader(multiFileCue))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}

	if len(sheet.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(sheet.Files))
	}

	if sheet.Files[0].Name != "disc1.wav" {
		t.Errorf("Files[0].Name = %q, want %q", sheet.Files[0].Name, "disc1.wav")
	}
	if len(sheet.Files[0].Tracks) != 2 {
		t.Fatalf("len(Files[0].Tracks) = %d, want 2", len(sheet.Files[0].Tracks))
	}

	if sheet.Files[1].Name != "disc2.wav" {
		t.Errorf("Files[1].Name = %q, want %q", sheet.Files[1].Name, "disc2.wav")
	}
	if len(sheet.Files[1].Tracks) != 2 {
		t.Fatalf("len(Files[1].Tracks) = %d, want 2", len(sheet.Files[1].Tracks))
	}

	// Disc 2 tracks should restart from 00:00:00.
	if sheet.Files[1].Tracks[0].StartMs != 0 {
		t.Errorf("Files[1].Tracks[0].StartMs = %d, want 0", sheet.Files[1].Tracks[0].StartMs)
	}

	// 06:15:30 = 6*60*1000 + 15*1000 + 30*1000/75 = 375400ms.
	if sheet.Files[1].Tracks[1].StartMs != 375400 {
		t.Errorf("Files[1].Tracks[1].StartMs = %d, want 375400", sheet.Files[1].Tracks[1].StartMs)
	}
}

func TestParseWithBOM(t *testing.T) {
	cue := "\xef\xbb\xbf" + singleFileCue
	sheet, err := Parse(strings.NewReader(cue))
	if err != nil {
		t.Fatalf("Parse() with BOM error: %v", err)
	}
	if sheet.Performer != "Test Artist" {
		t.Errorf("Performer = %q, want %q", sheet.Performer, "Test Artist")
	}
}

func TestParseWithREM(t *testing.T) {
	cue := `REM GENRE "Rock"
REM DATE 2024
PERFORMER "Test"
TITLE "Album"
FILE "test.flac" FLAC
  TRACK 01 AUDIO
    TITLE "Song"
    INDEX 01 00:00:00
`
	sheet, err := Parse(strings.NewReader(cue))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(sheet.Files[0].Tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(sheet.Files[0].Tracks))
	}
}

func TestParseWithPregap(t *testing.T) {
	cue := `PERFORMER "Test"
TITLE "Album"
FILE "test.flac" FLAC
  TRACK 01 AUDIO
    TITLE "Song"
    PREGAP 00:02:00
    INDEX 00 00:00:00
    INDEX 01 00:02:00
`
	sheet, err := Parse(strings.NewReader(cue))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	// INDEX 01 is at 00:02:00 = 2000ms.
	if sheet.Files[0].Tracks[0].StartMs != 2000 {
		t.Errorf("StartMs = %d, want 2000", sheet.Files[0].Tracks[0].StartMs)
	}
}

func TestParseTimeEdgeCases(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"00:00:00", 0},
		{"00:01:00", 1000},
		{"01:00:00", 60000},
		{"99:59:74", 5999986},
		{"00:00:01", 13},   // 1 frame = 13.33ms, truncated to 13
		{"00:00:75", 1000}, // 75 frames = 1 second
	}
	for _, tt := range tests {
		got, err := parseTime(tt.input)
		if err != nil {
			t.Errorf("parseTime(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseTime(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestParseTimeInvalid(t *testing.T) {
	invalid := []string{"", "00:00", "ab:cd:ef", "00:00:00:00"}
	for _, s := range invalid {
		_, err := parseTime(s)
		if err == nil {
			t.Errorf("parseTime(%q) expected error", s)
		}
	}
}

func TestParseEmpty(t *testing.T) {
	sheet, err := Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(sheet.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(sheet.Files))
	}
}
