package library

// ListenBrainzConfig holds ListenBrainz scrobbling configuration.
type ListenBrainzConfig struct {
	UserToken string
	Username  string
	Enabled   bool
}

// SaveListenBrainzConfig upserts the single ListenBrainz config row.
func (db *DB) SaveListenBrainzConfig(cfg ListenBrainzConfig) error {
	enabled := 0
	if cfg.Enabled {
		enabled = 1
	}
	_, err := db.Exec(`
		UPDATE listenbrainz_config SET
			user_token = ?,
			username = ?,
			enabled = ?
		WHERE id = 1`,
		cfg.UserToken, cfg.Username, enabled,
	)
	return err
}

// LoadListenBrainzConfig reads the persisted ListenBrainz configuration.
// Returns a zero-value config if no row exists.
func (db *DB) LoadListenBrainzConfig() (ListenBrainzConfig, error) {
	var cfg ListenBrainzConfig
	var enabled int
	err := db.QueryRow(`
		SELECT user_token, username, enabled
		FROM listenbrainz_config WHERE id = 1`,
	).Scan(&cfg.UserToken, &cfg.Username, &enabled)
	if err != nil {
		return ListenBrainzConfig{}, err
	}
	cfg.Enabled = enabled != 0
	return cfg, nil
}
