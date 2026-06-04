package main

import (
	"reflect"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type fakeTrayPlayback struct {
	state string
	calls []string
}

type fakeTrayWindow struct {
	visible   bool
	minimised bool
	calls     []string
}

func (f *fakeTrayPlayback) State() string {
	return f.state
}

func (f *fakeTrayPlayback) Pause() {
	f.calls = append(f.calls, "pause")
}

func (f *fakeTrayPlayback) Resume() {
	f.calls = append(f.calls, "resume")
}

func (f *fakeTrayPlayback) Stop() {
	f.calls = append(f.calls, "stop")
}

func (f *fakeTrayPlayback) Next() {
	f.calls = append(f.calls, "next")
}

func (f *fakeTrayPlayback) Previous() {
	f.calls = append(f.calls, "previous")
}

func (f *fakeTrayWindow) IsVisible() bool {
	return f.visible
}

func (f *fakeTrayWindow) IsMinimised() bool {
	return f.minimised
}

func (f *fakeTrayWindow) Hide() application.Window {
	f.calls = append(f.calls, "hide")
	return nil
}

func (f *fakeTrayWindow) Restore() {
	f.calls = append(f.calls, "restore")
}

func (f *fakeTrayWindow) Show() application.Window {
	f.calls = append(f.calls, "show")
	return nil
}

func (f *fakeTrayWindow) Focus() {
	f.calls = append(f.calls, "focus")
}

func TestForteTrayMenuShape(t *testing.T) {
	menu := buildForteTrayMenu(application.NewMenu(), trayMenuActions{
		playback:     &fakeTrayPlayback{},
		toggleWindow: func() {},
		quit:         func() {},
	})

	var got []string
	for i := 0; ; i++ {
		item := menu.ItemAt(i)
		if item == nil {
			break
		}
		if item.IsSeparator() {
			got = append(got, "-")
			continue
		}
		got = append(got, item.Label())
	}

	want := []string{
		"Play/Pause",
		"Stop",
		"Next",
		"Previous",
		"-",
		"Show/Hide Window",
		"-",
		"Quit",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("tray menu labels = %#v, want %#v", got, want)
	}
}

func TestForteTrayMenuActions(t *testing.T) {
	playback := &fakeTrayPlayback{state: "playing"}
	toggled := 0
	quit := 0
	entries := forteTrayMenuEntries(trayMenuActions{
		playback: playback,
		toggleWindow: func() {
			toggled++
		},
		quit: func() {
			quit++
		},
	})

	runEntry(t, entries, "Play/Pause")
	playback.state = "paused"
	runEntry(t, entries, "Play/Pause")
	runEntry(t, entries, "Stop")
	runEntry(t, entries, "Next")
	runEntry(t, entries, "Previous")
	runEntry(t, entries, "Show/Hide Window")
	runEntry(t, entries, "Quit")

	if want := []string{"pause", "resume", "stop", "next", "previous"}; !reflect.DeepEqual(playback.calls, want) {
		t.Fatalf("playback calls = %#v, want %#v", playback.calls, want)
	}
	if toggled != 1 {
		t.Fatalf("toggleWindow called %d times, want 1", toggled)
	}
	if quit != 1 {
		t.Fatalf("quit called %d times, want 1", quit)
	}
}

func TestToggleForteWindow(t *testing.T) {
	tests := []struct {
		name      string
		window    *fakeTrayWindow
		wantCalls []string
	}{
		{
			name:      "hides visible window",
			window:    &fakeTrayWindow{visible: true},
			wantCalls: []string{"hide"},
		},
		{
			name:      "shows hidden window",
			window:    &fakeTrayWindow{},
			wantCalls: []string{"restore", "show", "focus"},
		},
		{
			name:      "restores minimised visible window",
			window:    &fakeTrayWindow{visible: true, minimised: true},
			wantCalls: []string{"restore", "show", "focus"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toggleForteWindow(tt.window)
			if !reflect.DeepEqual(tt.window.calls, tt.wantCalls) {
				t.Fatalf("calls = %#v, want %#v", tt.window.calls, tt.wantCalls)
			}
		})
	}
}

func runEntry(t *testing.T, entries []trayMenuEntry, label string) {
	t.Helper()
	for _, entry := range entries {
		if entry.label == label {
			if entry.action == nil {
				t.Fatalf("tray menu entry %q has no action", label)
			}
			entry.action()
			return
		}
	}
	t.Fatalf("tray menu entry %q not found", label)
}
