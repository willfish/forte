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

//go:embed build/tray-green-dark-idle-32.png
var trayIconGreenDarkIdle []byte

//go:embed build/tray-green-dark-playing-32.png
var trayIconGreenDarkPlaying []byte

//go:embed build/tray-green-light-idle-32.png
var trayIconGreenLightIdle []byte

//go:embed build/tray-green-light-playing-32.png
var trayIconGreenLightPlaying []byte

//go:embed build/tray-blue-dark-idle-32.png
var trayIconBlueDarkIdle []byte

//go:embed build/tray-blue-dark-playing-32.png
var trayIconBlueDarkPlaying []byte

//go:embed build/tray-blue-light-idle-32.png
var trayIconBlueLightIdle []byte

//go:embed build/tray-blue-light-playing-32.png
var trayIconBlueLightPlaying []byte

//go:embed build/tray-financial-times-dark-idle-32.png
var trayIconFinancialTimesDarkIdle []byte

//go:embed build/tray-financial-times-dark-playing-32.png
var trayIconFinancialTimesDarkPlaying []byte

//go:embed build/tray-financial-times-light-idle-32.png
var trayIconFinancialTimesLightIdle []byte

//go:embed build/tray-financial-times-light-playing-32.png
var trayIconFinancialTimesLightPlaying []byte

const appID = "io.github.willfish.forte"

// setupCrashLog directs Go runtime crash output (SIGSEGV, etc.) to a file
// so crashes from the installed program can be diagnosed. It also writes
// /proc/self/maps at startup so library addresses can be resolved from the
// faulting PC in the crash trace.
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

	// Write process memory maps so we can resolve library addresses on crash.
	maps, err := os.ReadFile("/proc/self/maps")
	if err == nil {
		_, _ = f.WriteString("=== /proc/self/maps at startup ===\n")
		_, _ = f.Write(maps)
		_, _ = f.WriteString("=== end maps ===\n\n")
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
		Linux: application.LinuxOptions{
			ProgramName: appID,
		},
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
	tray.SetLabel("Forte")

	menu := buildForteTrayMenu(app.NewMenu(), trayMenuActions{
		playback:     ps,
		toggleWindow: tray.ToggleWindow,
		quit: func() {
			app.Quit()
		},
	})

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Forte",
		Width:            1200,
		Height:           800,
		Frameless:        true,
		MinWidth:         700,
		MinHeight:        600,
		BackgroundType:   application.BackgroundTypeTransparent,
		BackgroundColour: application.NewRGBA(27, 38, 54, 0),
		URL:              "/",
	})

	// Close to tray: hide the window instead of quitting.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	trayIconState := newTrayIconState("green-dark", trayStateIdle)
	tray.SetIcon(trayIconState.current()).SetMenu(menu).AttachWindow(window)
	tray.SetTooltip("Forte")

	ls.onThemeChange = func(theme string) {
		tray.SetIcon(trayIconState.setTheme(theme))
	}

	// Left-click toggles window, right-click opens menu.
	tray.OnClick(func() {
		tray.ToggleWindow()
	})

	// Update tooltip and icon when track changes.
	ps.onTrayUpdate = func(title, artist string) {
		if title == "" {
			tray.SetTooltip("Forte")
			tray.SetIcon(trayIconState.setPlaybackState(trayStateIdle))
		} else if artist != "" {
			tray.SetTooltip(title + " - " + artist)
			tray.SetIcon(trayIconState.setPlaybackState(trayStatePlaying))
		} else {
			tray.SetTooltip(title)
			tray.SetIcon(trayIconState.setPlaybackState(trayStatePlaying))
		}
	}

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
