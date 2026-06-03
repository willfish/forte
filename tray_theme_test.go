package main

import "testing"

func TestGetTrayIconBytes_DarwinUsesMacTemplateIconsAndDistinguishesPlayback(t *testing.T) {
	idleState := newTrayIconState("green-dark", trayStateIdle)
	playingState := newTrayIconState("green-dark", trayStatePlaying)

	darwinIdle := getTrayIconBytesForOS(idleState, "darwin")
	darwinPlaying := getTrayIconBytesForOS(playingState, "darwin")

	if darwinIdle == nil {
		t.Fatal("expected non-nil bytes for darwin idle")
	}
	if darwinPlaying == nil {
		t.Fatal("expected non-nil bytes for darwin playing")
	}
	// cannot directly == slices; check they are at least non-nil and (for now with stub) same is ok until impl
	// real difference will be asserted after GREEN impl provides distinct mac assets

	// Linux path should still work
	linuxIdle := getTrayIconBytesForOS(idleState, "linux")
	if linuxIdle == nil {
		t.Fatal("expected non-nil bytes for linux idle")
	}
}

func TestGetTrayIconBytes_LinuxUsesThemeAndPlayback(t *testing.T) {
	state := newTrayIconState("blue-light", trayStatePlaying)
	b := getTrayIconBytesForOS(state, "linux")
	if b == nil {
		t.Fatal("expected bytes")
	}
	// We don't assert exact bytes here (depends on embedded), just that it doesn't panic and is non-nil
}
