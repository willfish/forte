package library

import (
	"fmt"
	"path/filepath"
)

// LocalDirectory is a configured root for local music scans.
type LocalDirectory struct {
	Path    string
	AddedAt string
}

// AddLocalDirectory stores a local music directory root.
func (db *DB) AddLocalDirectory(path string) error {
	normalised, err := normaliseLocalDirectory(path)
	if err != nil {
		return err
	}
	_, err = db.Exec("INSERT OR IGNORE INTO local_directories (path) VALUES (?)", normalised)
	if err != nil {
		return fmt.Errorf("add local directory: %w", err)
	}
	return nil
}

// GetLocalDirectories returns all configured local music directories.
func (db *DB) GetLocalDirectories() ([]LocalDirectory, error) {
	rows, err := db.Query("SELECT path, added_at FROM local_directories ORDER BY path")
	if err != nil {
		return nil, fmt.Errorf("get local directories: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var dirs []LocalDirectory
	for rows.Next() {
		var dir LocalDirectory
		if err := rows.Scan(&dir.Path, &dir.AddedAt); err != nil {
			return nil, fmt.Errorf("scan local directory: %w", err)
		}
		dirs = append(dirs, dir)
	}
	return dirs, rows.Err()
}

// RemoveLocalDirectory removes a configured local music directory root.
func (db *DB) RemoveLocalDirectory(path string) error {
	normalised, err := normaliseLocalDirectory(path)
	if err != nil {
		return err
	}
	if _, err := db.Exec("DELETE FROM local_directories WHERE path = ?", normalised); err != nil {
		return fmt.Errorf("remove local directory: %w", err)
	}
	return nil
}

func normaliseLocalDirectory(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("local directory path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("normalise local directory: %w", err)
	}
	return filepath.Clean(abs), nil
}
