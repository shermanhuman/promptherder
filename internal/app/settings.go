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

// Settings holds user-configurable options from .promptherder/settings.json.
type Settings struct {
	// CommandPrefix is prepended to workflow/prompt output filenames.
	// e.g. "v-" turns plan.md into v-plan.md.
	CommandPrefix string `json:"command_prefix"`

	// CommandPrefixEnabled toggles prefix application. Default false.
	CommandPrefixEnabled bool `json:"command_prefix_enabled"`

	// Agents lists the enabled target agent names.
	// Nil means setup has not run; an empty list explicitly enables no targets.
	Agents []string `json:"agents"`
	// Explicit source-path overrides and optional skill selection for native builds.
	Overrides          []string `json:"overrides,omitempty"`
	Skills             []string `json:"skills"`
	ProjectDocMaxBytes int      `json:"project_doc_max_bytes,omitempty"`
}

// DefaultSettings returns the zero-value settings (all off).
func DefaultSettings() Settings {
	return Settings{}
}

// EnabledAgents returns the list of agents to sync.
func (s Settings) EnabledAgents() []string {
	return s.Agents
}

// TargetsConfigured distinguishes first-time setup from an explicit empty selection.
func (s Settings) TargetsConfigured() bool { return s.Agents != nil }

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
	if s.Agents == nil {
		s.Agents = []string{}
	}
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
	for _, name := range s.Agents {
		if !IsValidAgent(name) {
			return Settings{}, fmt.Errorf("unknown target %q in %s", name, path)
		}
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
