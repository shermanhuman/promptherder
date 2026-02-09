package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSettings_MissingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	s, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.CommandPrefix != "" {
		t.Errorf("expected empty prefix, got %q", s.CommandPrefix)
	}
	if s.CommandPrefixEnabled {
		t.Error("expected prefix disabled by default")
	}
}

func TestLoadSettings_ValidFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, manifestDir)
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `{
		"command_prefix": "v-",
		"command_prefix_enabled": true
	}`
	if err := os.WriteFile(filepath.Join(settingsDir, settingsFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.CommandPrefix != "v-" {
		t.Errorf("expected prefix %q, got %q", "v-", s.CommandPrefix)
	}
	if !s.CommandPrefixEnabled {
		t.Error("expected prefix enabled")
	}
}

func TestLoadSettings_PartialFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, manifestDir)
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Only set prefix, others should default.
	content := `{"command_prefix": "x-"}`
	if err := os.WriteFile(filepath.Join(settingsDir, settingsFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.CommandPrefix != "x-" {
		t.Errorf("expected prefix %q, got %q", "x-", s.CommandPrefix)
	}
	if s.CommandPrefixEnabled {
		t.Error("expected prefix disabled (not set in file)")
	}
}

func TestLoadSettings_EmptyPrefixEnabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, manifestDir)
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Empty prefix + enabled should be treated as disabled.
	content := `{"command_prefix": "", "command_prefix_enabled": true}`
	if err := os.WriteFile(filepath.Join(settingsDir, settingsFile), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s, err := LoadSettings(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.CommandPrefixEnabled {
		t.Error("expected prefix disabled when prefix is empty")
	}
}

func TestLoadSettings_InvalidJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	settingsDir := filepath.Join(dir, manifestDir)
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(settingsDir, settingsFile), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadSettings(dir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestPrefixCommand_Enabled(t *testing.T) {
	t.Parallel()
	s := Settings{CommandPrefix: "v-", CommandPrefixEnabled: true}

	if got := s.PrefixCommand("plan.md"); got != "v-plan.md" {
		t.Errorf("expected %q, got %q", "v-plan.md", got)
	}
}

func TestPrefixCommand_Disabled(t *testing.T) {
	t.Parallel()
	s := Settings{CommandPrefix: "v-", CommandPrefixEnabled: false}

	if got := s.PrefixCommand("plan.md"); got != "plan.md" {
		t.Errorf("expected %q, got %q", "plan.md", got)
	}
}
