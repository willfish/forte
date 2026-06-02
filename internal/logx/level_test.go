package logx

import (
	"log/slog"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{LevelOff, slog.LevelError + 4},
		{LevelError, slog.LevelError},
		{LevelWarn, slog.LevelWarn},
		{LevelInfo, slog.LevelInfo},
		{LevelDebug, slog.LevelDebug},
		{"", slog.LevelWarn},
		{"LOUD", slog.LevelWarn},
	}
	for _, tc := range tests {
		if got := ParseLevel(tc.in); got != tc.want {
			t.Fatalf("ParseLevel(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}