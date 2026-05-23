package main

import (
	"bytes"
	"testing"
)

func TestTrayIconStateTracksThemeAndPlayback(t *testing.T) {
	state := newTrayIconState("green-dark", trayStateIdle)

	if got, want := state.current(), trayIconGreenDarkIdle; !bytes.Equal(got, want) {
		t.Fatal("initial tray icon did not use green dark idle icon")
	}

	if got, want := state.setPlaybackState(trayStatePlaying), trayIconGreenDarkPlaying; !bytes.Equal(got, want) {
		t.Fatal("playing tray icon did not preserve current theme")
	}

	if got, want := state.setTheme("financial-times-light"), trayIconFinancialTimesLightPlaying; !bytes.Equal(got, want) {
		t.Fatal("theme change did not preserve current playback state")
	}

	if got, want := state.setPlaybackState(trayStateIdle), trayIconFinancialTimesLightIdle; !bytes.Equal(got, want) {
		t.Fatal("idle tray icon did not preserve current theme")
	}
}

func TestTrayThemeIconsFallsBackToGreenDark(t *testing.T) {
	got := trayThemeIcons("unknown")
	if !bytes.Equal(got.idle, trayIconGreenDarkIdle) {
		t.Fatal("unknown tray theme did not fall back to green dark")
	}
}
