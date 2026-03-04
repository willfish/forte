package artistinfo

import "testing"

func TestParseMusicBrainzResponse(t *testing.T) {
	body := []byte(`{
		"artists": [
			{
				"id": "a74b1b7f-71a5-4011-9441-d0b5e4122711",
				"name": "Radiohead",
				"disambiguation": "English rock band",
				"type": "Group",
				"area": {"name": "United Kingdom"},
				"life-span": {"begin": "1985", "end": ""},
				"tags": [
					{"name": "rock"},
					{"name": "alternative rock"},
					{"name": "electronic"}
				]
			}
		]
	}`)

	info, err := parseMusicBrainzResponse(body)
	if err != nil {
		t.Fatalf("parseMusicBrainzResponse: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil result")
	}
	if info.ID != "a74b1b7f-71a5-4011-9441-d0b5e4122711" {
		t.Errorf("id = %q", info.ID)
	}
	if info.Type != "Group" {
		t.Errorf("type = %q", info.Type)
	}
	if info.Area != "United Kingdom" {
		t.Errorf("area = %q", info.Area)
	}
	if info.Begin != "1985" {
		t.Errorf("begin = %q", info.Begin)
	}
	if len(info.Tags) != 3 {
		t.Fatalf("tags count = %d, want 3", len(info.Tags))
	}
	if info.Tags[0] != "rock" {
		t.Errorf("tags[0] = %q", info.Tags[0])
	}
}

func TestParseMusicBrainzEmptyResults(t *testing.T) {
	body := []byte(`{"artists": []}`)

	info, err := parseMusicBrainzResponse(body)
	if err != nil {
		t.Fatalf("parseMusicBrainzResponse: %v", err)
	}
	if info != nil {
		t.Error("expected nil for empty results")
	}
}

func TestParseMusicBrainzMinimalArtist(t *testing.T) {
	body := []byte(`{
		"artists": [
			{
				"id": "abc123",
				"name": "Solo Artist"
			}
		]
	}`)

	info, err := parseMusicBrainzResponse(body)
	if err != nil {
		t.Fatalf("parseMusicBrainzResponse: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil result")
	}
	if info.ID != "abc123" {
		t.Errorf("id = %q", info.ID)
	}
	if info.Type != "" {
		t.Errorf("type = %q, want empty", info.Type)
	}
	if len(info.Tags) != 0 {
		t.Errorf("tags = %d, want 0", len(info.Tags))
	}
}
