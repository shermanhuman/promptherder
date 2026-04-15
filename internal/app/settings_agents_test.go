package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnabledAgents_DefaultsToLegacy(t *testing.T) {
	t.Parallel()
	s := Settings{}
	agents := s.EnabledAgents()
	if len(agents) != 2 || agents[0] != "copilot" || agents[1] != "antigravity" {
		t.Errorf("expected default agents [copilot antigravity], got %v", agents)
	}
}

func TestEnabledAgents_CustomList(t *testing.T) {
	t.Parallel()
	s := Settings{Agents: []string{"claude", "cursor"}}
	agents := s.EnabledAgents()
	if len(agents) != 2 || agents[0] != "claude" || agents[1] != "cursor" {
		t.Errorf("expected [claude cursor], got %v", agents)
	}
}

func TestIsAgentEnabled(t *testing.T) {
	t.Parallel()
	s := Settings{Agents: []string{"claude", "cursor"}}
	if !s.IsAgentEnabled("claude") {
		t.Error("claude should be enabled")
	}
	if s.IsAgentEnabled("copilot") {
		t.Error("copilot should not be enabled")
	}
}

func TestIsValidAgent(t *testing.T) {
	t.Parallel()
	if !IsValidAgent("claude") {
		t.Error("claude should be valid")
	}
	if IsValidAgent("emacs") {
		t.Error("emacs should not be valid")
	}
}

func TestSaveSettings(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	phDir := filepath.Join(dir, ".promptherder")
	if err := os.MkdirAll(phDir, 0755); err != nil {
		t.Fatal(err)
	}

	s := Settings{
		Agents: []string{"claude", "cursor", "copilot"},
	}

	if err := SaveSettings(dir, s); err != nil {
		t.Fatal(err)
	}

	// Read back.
	loaded, err := LoadSettings(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(loaded.Agents) != 3 {
		t.Fatalf("expected 3 agents, got %d", len(loaded.Agents))
	}
	if loaded.Agents[0] != "claude" {
		t.Errorf("expected first agent claude, got %s", loaded.Agents[0])
	}
}

func TestLoadSettings_WithAgents(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	settingsDir := filepath.Join(dir, ".promptherder")
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		t.Fatal(err)
	}

	content := `{"agents": ["claude", "codex"], "command_prefix": "v-", "command_prefix_enabled": true}`
	if err := os.WriteFile(filepath.Join(settingsDir, settingsFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := LoadSettings(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(s.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(s.Agents))
	}
	if s.Agents[0] != "claude" || s.Agents[1] != "codex" {
		t.Errorf("expected [claude codex], got %v", s.Agents)
	}
	if !s.CommandPrefixEnabled {
		t.Error("command_prefix_enabled should be true")
	}
}
