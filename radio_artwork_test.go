package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func resetRadioArtworkTestState(t *testing.T) {
	t.Helper()
	oldClient := radioArtworkClient
	radioArtworkClient = &http.Client{Timeout: time.Second}
	radioArtworkCache.Lock()
	radioArtworkCache.m = nil
	radioArtworkCache.Unlock()
	t.Cleanup(func() {
		radioArtworkClient = oldClient
		radioArtworkCache.Lock()
		radioArtworkCache.m = nil
		radioArtworkCache.Unlock()
	})
}

func TestResolveRadioArtworkKeepsExistingFavicon(t *testing.T) {
	resetRadioArtworkTestState(t)

	got := resolveRadioArtwork("https://example.com/icon.png", "https://example.com")
	if got != "https://example.com/icon.png" {
		t.Fatalf("resolveRadioArtwork = %q, want existing favicon", got)
	}
}

func TestResolveRadioArtworkUsesHomepageMetadata(t *testing.T) {
	resetRadioArtworkTestState(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><meta property="og:image" content="/station.png"></head></html>`))
	}))
	defer server.Close()

	got := resolveRadioArtwork("", server.URL)
	want := server.URL + "/station.png"
	if got != want {
		t.Fatalf("resolveRadioArtwork = %q, want %q", got, want)
	}
}

func TestResolveRadioArtworkUsesIconWhenMetadataMissing(t *testing.T) {
	resetRadioArtworkTestState(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><link rel="apple-touch-icon" href="touch.png"></head></html>`))
	}))
	defer server.Close()

	got := resolveRadioArtwork("", server.URL+"/listen")
	want := server.URL + "/touch.png"
	if got != want {
		t.Fatalf("resolveRadioArtwork = %q, want %q", got, want)
	}
}

func TestResolveRadioArtworkFallsBackToFaviconPath(t *testing.T) {
	resetRadioArtworkTestState(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Station</title></head></html>`))
	}))
	defer server.Close()

	got := resolveRadioArtwork("", server.URL+"/radio")
	want := server.URL + "/favicon.ico"
	if got != want {
		t.Fatalf("resolveRadioArtwork = %q, want %q", got, want)
	}
}
