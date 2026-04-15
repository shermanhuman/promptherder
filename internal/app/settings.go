package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const settingsFile = "settings.json"

// AllAgents lists every supported target name.
var AllAgents = []string{
	"copilot",
	"antigravity",
	"claude",
	"codex",
	"cursor",
	"windsurf",
	"cline",
}

// DefaultAgents are enabled when no explicit agents list is configured.
var DefaultAgents = []string{"copilot", "antigravity"}

// Settings holds user-configurable options from .promptherder/settings.json.
type Settings struct {
	// CommandPrefix is prepended to workflow/prompt output filenames.
	// e.g. "v-" turns plan.md into v-plan.md.
	CommandPrefix string `json:"command_prefix"`

	// CommandPrefixEnabled toggles prefix application. Default false.
	CommandPrefixEnabled bool `json:"command_prefix_enabled"`

	// Agents lists the enabled target agent names.
	// Empty means DefaultAgents (copilot, antigravity).
	Agents []string `json:"agents,omitempty"`
}

// DefaultSettings returns the zero-value settings (all off).
func DefaultSettings() Settings {
	return Settings{}
}

// EnabledAgents returns the list of agents to sync.
// Returns DefaultAgents if none are explicitly configured.
func (s Settings) EnabledAgents() []string {
	if len(s.Agents) == 0 {
		return DefaultAgents
	}
	return s.Agents
}

// IsAgentEnabled returns true if the given agent name is in the enabled list.
func (s Settings) IsAgentEnabled(name string) bool {
	for _, a := range s.EnabledAgents() {
		if a == name {
			return true
		}
	}
	return false
}

// IsValidAgent returns true if the name is a recognized agent.
func IsValidAgent(name string) bool {
	for _, a := range AllAgents {
		if a == name {
			return true
		}
	}
	return false
}

// SaveSettings writes settings back to .promptherder/settings.json.
func SaveSettings(repoPath string, s Settings) error {
	path := filepath.Join(repoPath, manifestDir, settingsFile)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
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
