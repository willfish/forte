package userconfig

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// LoadFile reads and decodes a config file from disk.
func LoadFile(path string) (File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return File{}, fmt.Errorf("read config: %w", err)
	}
	var cfg File
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return File{}, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

// ValidateSchema returns an error when the file version is not supported.
func ValidateSchema(cfg File) error {
	if cfg.SchemaVersion == 0 {
		return fmt.Errorf("config missing schemaVersion (expected %d)", SchemaVersion)
	}
	if cfg.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported config schemaVersion %d (expected %d)", cfg.SchemaVersion, SchemaVersion)
	}
	return nil
}
