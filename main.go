package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var appIcon []byte

//go:embed build/tray-idle-32.png
var trayIconIdle []byte

//go:embed build/tray-playing-32.png
var trayIconPlaying []byte

// setupCrashLog directs Go runtime crash output (SIGSEGV, etc.) to a file
// so crashes from the installed program can be diagnosed.
func setupCrashLog() *os.File {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil
	}
	logDir := filepath.Join(dir, "forte")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil
	}
	f, err := os.Create(filepath.Join(logDir, "crash.log"))
	if err != nil {
		return nil
	}
	if err := debug.SetCrashOutput(f, debug.CrashOptions{}); err != nil {
		_ = f.Close()
		return nil
	}
	return f
}

func main() {
	if f := setupCrashLog(); f != nil {
		defer func() { _ = f.Close() }()
	}

	ps := &PlayerService{}
	ls := &LibraryService{}

	// Wire server health check into the player service.
	ps.isServerOnline = func(serverID string) bool {
		if ls.health == nil {
			return true
		}
		return ls.health.IsOnline(serverID)
	}

	app := application.New(application.Options{
		Name:        "Forte",
		Description: "A modern music player",
		Services: []application.Service{
			application.NewService(ps),
			application.NewService(ls),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	app.SetIcon(appIcon)

	// System tray with playback controls.
	tray := app.SystemTray.New()

	menu := app.NewMenu()
	menu.Add("Play/Pause").OnClick(func(_ *application.Context) {
		if ps.State() == "playing" {
			ps.Pause()
		} else {
			ps.Resume()
		}
	})
	menu.Add("Next").OnClick(func(_ *application.Context) {
		ps.Next()
	})
	menu.Add("Previous").OnClick(func(_ *application.Context) {
		ps.Previous()
	})
	menu.AddSeparator()
	menu.Add("Show/Hide Window").OnClick(func(_ *application.Context) {
		tray.ToggleWindow()
	})
	menu.AddSeparator()
	menu.Add("Quit").OnClick(func(_ *application.Context) {
		app.Quit()
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Forte",
		Width:            1200,
		Height:           800,
		MinWidth:         700,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
	})

	// Close to tray: hide the window instead of quitting.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	tray.SetIcon(trayIconIdle).SetMenu(menu).AttachWindow(window)
	tray.SetTooltip("Forte")

	// Left-click toggles window, right-click opens menu.
	tray.OnClick(func() {
		tray.ToggleWindow()
	})

	// Update tooltip and icon when track changes.
	ps.onTrayUpdate = func(title, artist string) {
		if title == "" {
			tray.SetTooltip("Forte")
			tray.SetIcon(trayIconIdle)
		} else if artist != "" {
			tray.SetTooltip(title + " - " + artist)
			tray.SetIcon(trayIconPlaying)
		} else {
			tray.SetTooltip(title)
			tray.SetIcon(trayIconPlaying)
		}
	}

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
