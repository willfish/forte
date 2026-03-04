package library

// ScrobbleConfig holds Last.fm scrobbling configuration.
type ScrobbleConfig struct {
	APIKey     string
	APISecret  string
	SessionKey string
	Username   string
	Enabled    bool
}

// SaveScrobbleConfig upserts the single scrobble config row.
func (db *DB) SaveScrobbleConfig(cfg ScrobbleConfig) error {
	enabled := 0
	if cfg.Enabled {
		enabled = 1
	}
	_, err := db.Exec(`
		UPDATE scrobble_config SET
			api_key = ?,
			api_secret = ?,
			session_key = ?,
			username = ?,
			enabled = ?
		WHERE id = 1`,
		cfg.APIKey, cfg.APISecret, cfg.SessionKey, cfg.Username, enabled,
	)
	return err
}

// LoadScrobbleConfig reads the persisted scrobble configuration.
// Returns a zero-value config if no row exists.
func (db *DB) LoadScrobbleConfig() (ScrobbleConfig, error) {
	var cfg ScrobbleConfig
	var enabled int
	err := db.QueryRow(`
		SELECT api_key, api_secret, session_key, username, enabled
		FROM scrobble_config WHERE id = 1`,
	).Scan(&cfg.APIKey, &cfg.APISecret, &cfg.SessionKey, &cfg.Username, &enabled)
	if err != nil {
		return ScrobbleConfig{}, err
	}
	cfg.Enabled = enabled != 0
	return cfg, nil
}
