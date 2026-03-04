package subsonic

import (
	"encoding/hex"
	"testing"
)

func TestGenerateToken(t *testing.T) {
	// Known vector: md5("sesame" + "c19b2d") = "26719a1196d2a940705a59634eb18eab"
	got := generateToken("sesame", "c19b2d")
	want := "26719a1196d2a940705a59634eb18eab"
	if got != want {
		t.Errorf("generateToken = %q, want %q", got, want)
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := generateSalt(8)
	if err != nil {
		t.Fatalf("generateSalt: %v", err)
	}

	// 8 bytes = 16 hex characters.
	if len(salt) != 16 {
		t.Errorf("salt length = %d, want 16", len(salt))
	}

	// Must be valid hex.
	if _, err := hex.DecodeString(salt); err != nil {
		t.Errorf("salt is not valid hex: %v", err)
	}
}
