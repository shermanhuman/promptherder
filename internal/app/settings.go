package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const settingsFile = "settings.json"

// Settings holds user-configurable options from .promptherder/settings.json.
type Settings struct {
	// CommandPrefix is prepended to workflow/prompt output filenames.
	// e.g. "v-" turns plan.md into v-plan.md.
	CommandPrefix string `json:"command_prefix"`

	// CommandPrefixEnabled toggles prefix application. Default false.
	CommandPrefixEnabled bool `json:"command_prefix_enabled"`
}

// DefaultSettings returns the zero-value settings (all off).
func DefaultSettings() Settings {
	return Settings{}
}

// LoadSettings reads .promptherder/settings.json from the repo root.
// Returns defaults if the file is missing or empty.
func LoadSettings(repoPath string) (Settings, error) {
	path := filepath.Join(repoPath, manifestDir, settingsFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultSettings(), nil
		}
		return Settings{}, fmt.Errorf("read settings: %w", err)
	}

	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Settings{}, fmt.Errorf("parse settings %s: %w", path, err)
	}

	// Validate: empty prefix + enabled = treat as disabled.
	if s.CommandPrefixEnabled && s.CommandPrefix == "" {
		s.CommandPrefixEnabled = false
	}

	return s, nil
}

// PrefixCommand returns the prefixed filename if prefix is enabled,
// otherwise returns the original filename unchanged.
func (s Settings) PrefixCommand(filename string) string {
	if !s.CommandPrefixEnabled {
		return filename
	}
	return s.CommandPrefix + filename
}
