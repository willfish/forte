package userconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dir returns the Forte config directory (e.g. ~/.config/forte).
func Dir() (string, error) {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("config dir: %w", err)
	}
	return filepath.Join(dataDir, "forte"), nil
}

// DefaultPath returns the canonical config file path (…/forte/config.toml).
func DefaultPath() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, FileName), nil
}
