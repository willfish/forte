package main

import "testing"

func TestPauseKeepsRadioMode(t *testing.T) {
	p := &PlayerService{}
	p.radioMode = true
	p.radioName = "Test Radio"
	p.radioStreamURL = "https://example.com/stream"

	p.Pause()

	if !p.radioMode {
		t.Fatal("Pause() should keep radio mode active")
	}
	if p.radioName != "Test Radio" || p.radioStreamURL != "https://example.com/stream" {
		t.Fatalf("radio metadata should be preserved: name=%q stream=%q", p.radioName, p.radioStreamURL)
	}
}

func TestStopStopsRadioMode(t *testing.T) {
	p := &PlayerService{}
	p.radioMode = true
	p.radioName = "Test Radio"
	p.radioStreamURL = "https://example.com/stream"

	p.Stop()

	if p.radioMode {
		t.Fatal("Stop() should stop radio mode")
	}
	if p.radioName != "" || p.radioStreamURL != "" {
		t.Fatalf("radio metadata was not cleared: name=%q stream=%q", p.radioName, p.radioStreamURL)
	}
}

func TestPlayerServiceShutdownIsIdempotent(t *testing.T) {
	p := &PlayerService{}

	if err := p.ServiceShutdown(); err != nil {
		t.Fatalf("first shutdown: %v", err)
	}
	if err := p.ServiceShutdown(); err != nil {
		t.Fatalf("second shutdown: %v", err)
	}
}
