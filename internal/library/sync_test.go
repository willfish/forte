package library

import "testing"

func TestServerFilePath(t *testing.T) {
	got := serverFilePath("srv-1", "track-42")
	want := "server://srv-1/track-42"
	if got != want {
		t.Errorf("serverFilePath = %q, want %q", got, want)
	}
}

func TestNewProviderSubsonic(t *testing.T) {
	srv := Server{ID: "1", Name: "Test", Type: "subsonic", URL: "http://localhost", Username: "u", Password: "p"}
	p, err := newProvider(srv)
	if err != nil {
		t.Fatalf("newProvider subsonic: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewProviderJellyfin(t *testing.T) {
	srv := Server{ID: "1", Name: "Test", Type: "jellyfin", URL: "http://localhost", Username: "u", Password: "p"}
	p, err := newProvider(srv)
	if err != nil {
		t.Fatalf("newProvider jellyfin: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
}

func TestNewProviderUnknown(t *testing.T) {
	srv := Server{ID: "1", Name: "Test", Type: "unknown", URL: "http://localhost"}
	_, err := newProvider(srv)
	if err == nil {
		t.Error("expected error for unknown server type")
	}
}
