package main

import "testing"

func TestPauseStopsRadioMode(t *testing.T) {
	p := &PlayerService{}
	p.radioMode = true
	p.radioName = "Test Radio"
	p.radioStreamURL = "https://example.com/stream"

	p.Pause()

	if p.radioMode {
		t.Fatal("Pause() should stop radio mode")
	}
	if p.radioName != "" || p.radioStreamURL != "" {
		t.Fatalf("radio metadata was not cleared: name=%q stream=%q", p.radioName, p.radioStreamURL)
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
