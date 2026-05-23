package main

import "github.com/wailsapp/wails/v3/pkg/application"

type trayPlaybackController interface {
	State() string
	Pause()
	Resume()
	Next()
	Previous()
}

type trayMenuActions struct {
	playback     trayPlaybackController
	toggleWindow func()
	quit         func()
}

type trayMenuEntry struct {
	label     string
	separator bool
	action    func()
}

func forteTrayMenuEntries(actions trayMenuActions) []trayMenuEntry {
	return []trayMenuEntry{
		{
			label: "Play/Pause",
			action: func() {
				if actions.playback.State() == "playing" {
					actions.playback.Pause()
				} else {
					actions.playback.Resume()
				}
			},
		},
		{label: "Next", action: actions.playback.Next},
		{label: "Previous", action: actions.playback.Previous},
		{separator: true},
		{label: "Show/Hide Window", action: actions.toggleWindow},
		{separator: true},
		{label: "Quit", action: actions.quit},
	}
}

func buildForteTrayMenu(menu *application.Menu, actions trayMenuActions) *application.Menu {
	for _, entry := range forteTrayMenuEntries(actions) {
		if entry.separator {
			menu.AddSeparator()
			continue
		}

		action := entry.action
		menu.Add(entry.label).OnClick(func(_ *application.Context) {
			if action != nil {
				action()
			}
		})
	}
	return menu
}
