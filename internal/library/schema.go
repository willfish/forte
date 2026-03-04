package library

const migration001 = `
CREATE TABLE artists (
	id         INTEGER PRIMARY KEY,
	name       TEXT NOT NULL,
	sort_name  TEXT NOT NULL DEFAULT '',
	image_url  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_artists_name ON artists (name);

CREATE TABLE albums (
	id            INTEGER PRIMARY KEY,
	artist_id     INTEGER NOT NULL REFERENCES artists(id),
	title         TEXT NOT NULL,
	year          INTEGER NOT NULL DEFAULT 0,
	track_count   INTEGER NOT NULL DEFAULT 0,
	artwork_blob  BLOB,
	created_at    TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at    TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_albums_artist ON albums (artist_id);
CREATE INDEX idx_albums_title  ON albums (title);

CREATE TABLE tracks (
	id             INTEGER PRIMARY KEY,
	album_id       INTEGER REFERENCES albums(id),
	artist_id      INTEGER NOT NULL REFERENCES artists(id),
	title          TEXT NOT NULL,
	track_number   INTEGER NOT NULL DEFAULT 0,
	disc_number    INTEGER NOT NULL DEFAULT 1,
	duration_ms    INTEGER NOT NULL DEFAULT 0,
	file_path      TEXT NOT NULL UNIQUE,
	file_size      INTEGER NOT NULL DEFAULT 0,
	file_mod_time  TEXT NOT NULL DEFAULT '',
	format         TEXT NOT NULL DEFAULT '',
	bitrate        INTEGER NOT NULL DEFAULT 0,
	cue_file_path  TEXT NOT NULL DEFAULT '',
	start_ms       INTEGER NOT NULL DEFAULT 0,
	end_ms         INTEGER NOT NULL DEFAULT 0,
	server_id      TEXT NOT NULL DEFAULT '',
	remote_id      TEXT NOT NULL DEFAULT '',
	created_at     TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at     TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_tracks_album    ON tracks (album_id);
CREATE INDEX idx_tracks_artist   ON tracks (artist_id);
CREATE INDEX idx_tracks_path     ON tracks (file_path);

CREATE TABLE genres (
	id   INTEGER PRIMARY KEY,
	name TEXT NOT NULL UNIQUE
);

CREATE TABLE track_genres (
	track_id INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
	genre_id INTEGER NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
	PRIMARY KEY (track_id, genre_id)
);

CREATE VIRTUAL TABLE fts_tracks USING fts5 (
	title,
	artist,
	album,
	genre,
	content='',
	content_rowid='rowid'
);

CREATE TABLE playlists (
	id         INTEGER PRIMARY KEY,
	name       TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE playlist_tracks (
	playlist_id INTEGER NOT NULL REFERENCES playlists(id) ON DELETE CASCADE,
	track_id    INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
	position    INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (playlist_id, track_id)
);
`

const migration002 = `
CREATE TABLE playback_state (
	id                 INTEGER PRIMARY KEY CHECK (id = 1),
	queue_json         TEXT NOT NULL DEFAULT '[]',
	position           INTEGER NOT NULL DEFAULT -1,
	track_position_ms  INTEGER NOT NULL DEFAULT 0,
	volume             INTEGER NOT NULL DEFAULT 100,
	shuffle            INTEGER NOT NULL DEFAULT 0,
	repeat_mode        TEXT NOT NULL DEFAULT 'off'
);
INSERT INTO playback_state (id) VALUES (1);
`

const migration003 = `
CREATE TABLE servers (
	id       TEXT PRIMARY KEY,
	name     TEXT NOT NULL,
	type     TEXT NOT NULL DEFAULT 'subsonic',
	url      TEXT NOT NULL,
	username TEXT NOT NULL,
	password TEXT NOT NULL DEFAULT ''
);
`

const migration004 = `
ALTER TABLE albums ADD COLUMN server_id TEXT NOT NULL DEFAULT '';
ALTER TABLE albums ADD COLUMN remote_id TEXT NOT NULL DEFAULT '';
CREATE INDEX idx_albums_server ON albums (server_id);
CREATE INDEX idx_tracks_server ON tracks (server_id);
`

const migration005 = `
CREATE TABLE scrobble_config (
	id          INTEGER PRIMARY KEY CHECK (id = 1),
	api_key     TEXT NOT NULL DEFAULT '',
	api_secret  TEXT NOT NULL DEFAULT '',
	session_key TEXT NOT NULL DEFAULT '',
	username    TEXT NOT NULL DEFAULT '',
	enabled     INTEGER NOT NULL DEFAULT 0
);
INSERT INTO scrobble_config (id) VALUES (1);
`

const migration006 = `
CREATE TABLE listenbrainz_config (
	id         INTEGER PRIMARY KEY CHECK (id = 1),
	user_token TEXT NOT NULL DEFAULT '',
	username   TEXT NOT NULL DEFAULT '',
	enabled    INTEGER NOT NULL DEFAULT 0
);
INSERT INTO listenbrainz_config (id) VALUES (1);
`

const migration007 = `
CREATE TABLE scrobble_queue (
	id              INTEGER PRIMARY KEY,
	service         TEXT NOT NULL,
	track_json      TEXT NOT NULL,
	timestamp       INTEGER NOT NULL,
	attempts        INTEGER NOT NULL DEFAULT 0,
	last_attempt_at TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_scrobble_queue_service ON scrobble_queue (service);
`

const migration008 = `
CREATE TABLE play_history (
	id                 INTEGER PRIMARY KEY,
	track_id           INTEGER NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
	played_at          TEXT NOT NULL DEFAULT (datetime('now')),
	duration_played_ms INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_play_history_played_at ON play_history (played_at);
CREATE INDEX idx_play_history_track ON play_history (track_id);
`

const migration009 = `
CREATE TABLE artist_metadata (
	artist_id    INTEGER PRIMARY KEY REFERENCES artists(id) ON DELETE CASCADE,
	bio          TEXT NOT NULL DEFAULT '',
	image_url    TEXT NOT NULL DEFAULT '',
	similar_json TEXT NOT NULL DEFAULT '[]',
	mb_id        TEXT NOT NULL DEFAULT '',
	mb_area      TEXT NOT NULL DEFAULT '',
	mb_type      TEXT NOT NULL DEFAULT '',
	mb_begin     TEXT NOT NULL DEFAULT '',
	mb_end       TEXT NOT NULL DEFAULT '',
	mb_tags      TEXT NOT NULL DEFAULT '',
	fetched_at   TEXT NOT NULL DEFAULT (datetime('now'))
);
`
