package artistinfo

import "testing"

func TestParseLastFmResponse(t *testing.T) {
	body := []byte(`{
		"artist": {
			"name": "Radiohead",
			"bio": {
				"summary": "Radiohead are an English rock band from <a href=\"https://last.fm\">Abingdon</a>, Oxfordshire."
			},
			"image": [
				{"#text": "https://img.fm/small.jpg", "size": "small"},
				{"#text": "https://img.fm/medium.jpg", "size": "medium"},
				{"#text": "https://img.fm/large.jpg", "size": "large"},
				{"#text": "", "size": "mega"}
			],
			"similar": {
				"artist": [
					{"name": "Muse"},
					{"name": "Thom Yorke"}
				]
			}
		}
	}`)

	info, err := parseLastFmResponse(body)
	if err != nil {
		t.Fatalf("parseLastFmResponse: %v", err)
	}
	if info.Bio != "Radiohead are an English rock band from Abingdon, Oxfordshire." {
		t.Errorf("bio = %q", info.Bio)
	}
	if info.ImageURL != "https://img.fm/large.jpg" {
		t.Errorf("image = %q, want large.jpg", info.ImageURL)
	}
	if len(info.Similar) != 2 {
		t.Fatalf("similar count = %d, want 2", len(info.Similar))
	}
	if info.Similar[0].Name != "Muse" {
		t.Errorf("similar[0] = %q", info.Similar[0].Name)
	}
}

func TestParseLastFmMissingFields(t *testing.T) {
	body := []byte(`{
		"artist": {
			"name": "Unknown",
			"bio": {"summary": ""},
			"image": [],
			"similar": {"artist": []}
		}
	}`)

	info, err := parseLastFmResponse(body)
	if err != nil {
		t.Fatalf("parseLastFmResponse: %v", err)
	}
	if info.Bio != "" {
		t.Errorf("bio = %q, want empty", info.Bio)
	}
	if info.ImageURL != "" {
		t.Errorf("image = %q, want empty", info.ImageURL)
	}
	if len(info.Similar) != 0 {
		t.Errorf("similar = %d, want 0", len(info.Similar))
	}
}

func TestParseLastFmApiError(t *testing.T) {
	body := []byte(`{"error": 6, "message": "The artist you supplied could not be found"}`)

	_, err := parseLastFmResponse(body)
	if err == nil {
		t.Error("expected error for API error response")
	}
}
