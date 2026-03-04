// Package library handles the music library database, scanning, and search.
package library

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps a SQLite connection for the music library.
type DB struct {
	*sql.DB
}

// OpenDB opens (or creates) the SQLite database at path.
// It enables WAL mode and runs any pending migrations.
func OpenDB(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Single connection avoids locking issues with modernc.org/sqlite.
	conn.SetMaxOpenConns(1)

	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	if _, err := conn.Exec("PRAGMA foreign_keys=ON"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	db := &DB{conn}
	if err := db.migrate(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

// migrate runs all pending schema migrations.
func (db *DB) migrate() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	for _, m := range migrations {
		var exists bool
		err := db.QueryRow("SELECT 1 FROM schema_migrations WHERE version = ?", m.version).Scan(&exists)
		if err == nil {
			continue // already applied
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", m.version, err)
		}

		if _, err := tx.Exec(m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d: %w", m.version, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", m.version, err)
		}
	}

	return nil
}

type migration struct {
	version int
	sql     string
}

var migrations = []migration{
	{version: 1, sql: migration001},
	{version: 2, sql: migration002},
	{version: 3, sql: migration003},
	{version: 4, sql: migration004},
	{version: 5, sql: migration005},
	{version: 6, sql: migration006},
	{version: 7, sql: migration007},
	{version: 8, sql: migration008},
}
