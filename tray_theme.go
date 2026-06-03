package main

import "sync"

type trayPlaybackState string

const (
	trayStateIdle    trayPlaybackState = "idle"
	trayStatePlaying trayPlaybackState = "playing"
)

type trayThemeIconSet struct {
	idle    []byte
	playing []byte
}

type trayIconState struct {
	mu            sync.Mutex
	theme         string
	playbackState trayPlaybackState
}

func newTrayIconState(theme string, playbackState trayPlaybackState) *trayIconState {
	return &trayIconState{
		theme:         theme,
		playbackState: playbackState,
	}
}

func (s *trayIconState) current() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return trayThemeIcons(s.theme).iconFor(s.playbackState)
}

func (s *trayIconState) setTheme(theme string) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.theme = theme
	return trayThemeIcons(s.theme).iconFor(s.playbackState)
}

func (s *trayIconState) setPlaybackState(playbackState trayPlaybackState) []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.playbackState = playbackState
	return trayThemeIcons(s.theme).iconFor(s.playbackState)
}

func trayThemeIcons(theme string) trayThemeIconSet {
	switch theme {
	case "green-light":
		return trayThemeIconSet{idle: trayIconGreenLightIdle, playing: trayIconGreenLightPlaying}
	case "blue-dark":
		return trayThemeIconSet{idle: trayIconBlueDarkIdle, playing: trayIconBlueDarkPlaying}
	case "blue-light":
		return trayThemeIconSet{idle: trayIconBlueLightIdle, playing: trayIconBlueLightPlaying}
	case "financial-times-dark":
		return trayThemeIconSet{idle: trayIconFinancialTimesDarkIdle, playing: trayIconFinancialTimesDarkPlaying}
	case "financial-times-light":
		return trayThemeIconSet{idle: trayIconFinancialTimesLightIdle, playing: trayIconFinancialTimesLightPlaying}
	default:
		return trayThemeIconSet{idle: trayIconGreenDarkIdle, playing: trayIconGreenDarkPlaying}
	}
}

func (set trayThemeIconSet) iconFor(state trayPlaybackState) []byte {
	if state == trayStatePlaying {
		return set.playing
	}
	return set.idle
}

// getTrayIconBytesForOS is the platform-aware icon selector (extracted for testability).
// On darwin we return template bytes (for SetTemplateIcon); on other OS we use the
// existing theme+state logic. This will be called from main.go tray setup.
func getTrayIconBytesForOS(s *trayIconState, goos string) []byte {
	if goos == "darwin" {
		if s.playbackState == trayStatePlaying {
			return trayIconMacPlaying
		}
		return trayIconMacIdle
	}
	return s.current()
}
