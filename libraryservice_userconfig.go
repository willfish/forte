package main

import (
	"fmt"

	"github.com/willfish/forte/internal/logx"
	"github.com/willfish/forte/internal/userconfig"
)

// UserConfigImportResultJSON reports the outcome of a config import.
type UserConfigImportResultJSON struct {
	Path            string   `json:"path"`
	SectionsApplied []string `json:"sectionsApplied"`
	SectionsSkipped []string `json:"sectionsSkipped"`
	Warnings        []string `json:"warnings"`
}

func importResultToJSON(r userconfig.ImportResult) UserConfigImportResultJSON {
	return UserConfigImportResultJSON{
		Path:            r.Path,
		SectionsApplied: r.SectionsApplied,
		SectionsSkipped: r.SectionsSkipped,
		Warnings:        r.Warnings,
	}
}

// GetUserConfigPath returns the canonical config file path (~/.config/forte/config.toml).
func (s *LibraryService) GetUserConfigPath() (string, error) {
	return userconfig.DefaultPath()
}

// ExportUserConfig writes the canonical config file from current database state.
func (s *LibraryService) ExportUserConfig() (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("library not initialised")
	}
	return userconfig.ExportToDBPath(s.db)
}

// ExportUserConfigToPath writes user configuration to the given path.
func (s *LibraryService) ExportUserConfigToPath(path string) error {
	if s.db == nil {
		return fmt.Errorf("library not initialised")
	}
	cfg, err := userconfig.BuildFromDB(s.db)
	if err != nil {
		return err
	}
	return userconfig.WriteFile(path, cfg)
}

// ImportUserConfig loads and applies the canonical config file.
func (s *LibraryService) ImportUserConfig() (UserConfigImportResultJSON, error) {
	if s.db == nil {
		return UserConfigImportResultJSON{}, fmt.Errorf("library not initialised")
	}
	path, err := userconfig.DefaultPath()
	if err != nil {
		return UserConfigImportResultJSON{}, err
	}
	return s.ImportUserConfigFromPath(path)
}

// ImportUserConfigFromPath loads and applies a config file from path.
func (s *LibraryService) ImportUserConfigFromPath(path string) (UserConfigImportResultJSON, error) {
	if s.db == nil {
		return UserConfigImportResultJSON{}, fmt.Errorf("library not initialised")
	}
	result, err := userconfig.ImportFile(s.db, path)
	if err != nil {
		return importResultToJSON(result), err
	}
	jsonResult := importResultToJSON(result)
	if err := s.applyImportedAppPreferences(); err != nil {
		return jsonResult, err
	}
	return jsonResult, nil
}

func (s *LibraryService) applyImportedAppPreferences() error {
	prefs, err := s.db.GetAppPreferences()
	if err != nil {
		return err
	}
	return s.SaveAppPreferences(AppPreferencesJSON{
		LibraryEnabled:   prefs.LibraryEnabled,
		StartLastStation: prefs.StartLastStation,
		AutoReconnect:    prefs.AutoReconnect,
		ShowTitlebar:     prefs.ShowTitlebar,
		LogLevel:         prefs.LogLevel,
		LogFilePath:      logx.Path(),
	})
}
