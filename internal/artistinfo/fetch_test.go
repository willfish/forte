package artistinfo

import "testing"

func TestFetchNoAPIKey(t *testing.T) {
	// With empty API key, Last.fm is skipped but MusicBrainz is still called.
	// We can't test the actual HTTP call, but we can verify it doesn't panic
	// and returns a result even when both services are unreachable.
	result, err := Fetch("", "Nonexistent Artist 12345")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// Bio should be empty since no API key was provided.
	if result.Bio != "" {
		t.Errorf("Bio = %q, expected empty without API key", result.Bio)
	}
}

func TestResultFieldsZeroValue(t *testing.T) {
	r := &Result{}
	if r.Bio != "" || r.ImageURL != "" || r.MbID != "" {
		t.Error("expected zero-value Result to have empty strings")
	}
	if r.Similar != nil {
		t.Error("expected nil Similar slice")
	}
}
