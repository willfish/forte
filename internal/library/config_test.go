package library

import "testing"

func TestSaveAndLoadScrobbleConfig(t *testing.T) {
	db := openTestDB(t)

	cfg := ScrobbleConfig{
		APIKey:     "key123",
		APISecret:  "secret456",
		SessionKey: "session789",
		Username:   "testuser",
		Enabled:    true,
	}

	if err := db.SaveScrobbleConfig(cfg); err != nil {
		t.Fatalf("SaveScrobbleConfig: %v", err)
	}

	got, err := db.LoadScrobbleConfig()
	if err != nil {
		t.Fatalf("LoadScrobbleConfig: %v", err)
	}

	if got.APIKey != "key123" {
		t.Errorf("APIKey = %q", got.APIKey)
	}
	if got.APISecret != "secret456" {
		t.Errorf("APISecret = %q", got.APISecret)
	}
	if got.SessionKey != "session789" {
		t.Errorf("SessionKey = %q", got.SessionKey)
	}
	if got.Username != "testuser" {
		t.Errorf("Username = %q", got.Username)
	}
	if !got.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestLoadScrobbleConfigDefault(t *testing.T) {
	db := openTestDB(t)

	got, err := db.LoadScrobbleConfig()
	if err != nil {
		t.Fatalf("LoadScrobbleConfig: %v", err)
	}
	if got.APIKey != "" {
		t.Errorf("default APIKey = %q, want empty", got.APIKey)
	}
	if got.Enabled {
		t.Error("default Enabled = true, want false")
	}
}

func TestScrobbleConfigDisable(t *testing.T) {
	db := openTestDB(t)

	_ = db.SaveScrobbleConfig(ScrobbleConfig{Enabled: true})
	_ = db.SaveScrobbleConfig(ScrobbleConfig{Enabled: false})

	got, _ := db.LoadScrobbleConfig()
	if got.Enabled {
		t.Error("Enabled = true after disabling")
	}
}

func TestSaveAndLoadListenBrainzConfig(t *testing.T) {
	db := openTestDB(t)

	cfg := ListenBrainzConfig{
		UserToken: "token-abc",
		Username:  "lbuser",
		Enabled:   true,
	}

	if err := db.SaveListenBrainzConfig(cfg); err != nil {
		t.Fatalf("SaveListenBrainzConfig: %v", err)
	}

	got, err := db.LoadListenBrainzConfig()
	if err != nil {
		t.Fatalf("LoadListenBrainzConfig: %v", err)
	}

	if got.UserToken != "token-abc" {
		t.Errorf("UserToken = %q", got.UserToken)
	}
	if got.Username != "lbuser" {
		t.Errorf("Username = %q", got.Username)
	}
	if !got.Enabled {
		t.Error("Enabled = false, want true")
	}
}

func TestLoadListenBrainzConfigDefault(t *testing.T) {
	db := openTestDB(t)

	got, err := db.LoadListenBrainzConfig()
	if err != nil {
		t.Fatalf("LoadListenBrainzConfig: %v", err)
	}
	if got.UserToken != "" {
		t.Errorf("default UserToken = %q", got.UserToken)
	}
	if got.Enabled {
		t.Error("default Enabled = true, want false")
	}
}
