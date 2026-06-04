package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/willfish/forte/internal/logx"
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

// macOS menubar template icons (monochrome for SetTemplateIcon / native tinting).
//
//go:embed build/tray-macos-idle.png
var trayIconMacIdle []byte

//go:embed build/tray-macos-playing.png
var trayIconMacPlaying []byte

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
	// Linux only (/proc not present or different on darwin).
	if runtime.GOOS == "linux" {
		maps, err := os.ReadFile("/proc/self/maps")
		if err == nil {
			_, _ = f.WriteString("=== /proc/self/maps at startup ===\n")
			_, _ = f.Write(maps)
			_, _ = f.WriteString("=== end maps ===\n\n")
		}
	}

	return f
}

func main() {
	if f := setupCrashLog(); f != nil {
		defer func() { _ = f.Close() }()
	}

	initLogging(logx.StartupLevel())

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
		Logger:      logx.Logger(),
		LogLevel:    logx.SlogLevel(),
		Linux: application.LinuxOptions{
			ProgramName: appID,
		},
		Mac: application.MacOptions{
			ActivationPolicy: application.ActivationPolicyRegular,
		},
		Services: []application.Service{
			application.NewService(ps),
			application.NewService(ls),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})
	forteApp = app

	app.SetIcon(appIcon)

	// System tray with playback controls.
	tray := app.SystemTray.New()
	tray.SetLabel("Forte")

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

	menu := buildForteTrayMenu(app.NewMenu(), trayMenuActions{
		playback:     ps,
		toggleWindow: func() { toggleForteWindow(window) },
		quit: func() {
			app.Quit()
		},
	})

	trayIconState := newTrayIconState("green-dark", trayStateIdle)

	tray.SetMenu(menu).AttachWindow(window)
	tray.SetTooltip("Forte")

	if runtime.GOOS == "darwin" {
		// Use template icon for native macOS menubar appearance (tints for light/dark).
		// Per design: primary click shows menu (B choice); no OnClick toggle here.
		// Playback state still drives icon change for parity with Linux tray.
		tray.SetTemplateIcon(getTrayIconBytesForOS(trayIconState, "darwin"))
	} else {
		tray.SetIcon(trayIconState.current())
		// Left-click toggles window (Linux behaviour preserved).
		tray.OnClick(func() {
			toggleForteWindow(window)
		})
	}

	ls.onThemeChange = func(theme string) {
		if runtime.GOOS == "darwin" {
			// macOS menubar uses fixed template icons (theme is Linux tray panel matching only).
			return
		}
		tray.SetIcon(trayIconState.setTheme(theme))
	}

	// Update tooltip and icon when track changes.
	// Tooltip is no-op on darwin.
	ps.onTrayUpdate = func(title, artist string) {
		if title == "" {
			tray.SetTooltip("Forte")
			trayIconState.setPlaybackState(trayStateIdle)
		} else if artist != "" {
			tray.SetTooltip(title + " - " + artist)
			trayIconState.setPlaybackState(trayStatePlaying)
		} else {
			tray.SetTooltip(title)
			trayIconState.setPlaybackState(trayStatePlaying)
		}
		if runtime.GOOS == "darwin" {
			tray.SetTemplateIcon(getTrayIconBytesForOS(trayIconState, "darwin"))
		} else {
			tray.SetIcon(getTrayIconBytesForOS(trayIconState, "linux"))
		}
	}

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
