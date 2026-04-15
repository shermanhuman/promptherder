package app

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildConcatOutput_WithRules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create source dir with two rules.
	srcDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, srcDir)
	mustWrite(t, filepath.Join(srcDir, "rule-a.md"), "# Rule A\nDo stuff.\n")
	mustWrite(t, filepath.Join(srcDir, "rule-b.md"), "# Rule B\nMore stuff.\n")

	header := "<!-- test -->\n"
	content, names, err := buildConcatOutput(dir, defaultSourceDir, nil, header)
	if err != nil {
		t.Fatal(err)
	}
	if content == nil {
		t.Fatal("expected content, got nil")
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(names))
	}

	s := string(content)
	if !strings.Contains(s, "<!-- test -->") {
		t.Error("expected header in output")
	}
	if !strings.Contains(s, "# Rule A") {
		t.Error("expected rule A in output")
	}
	if !strings.Contains(s, "# Rule B") {
		t.Error("expected rule B in output")
	}
}

func TestBuildConcatOutput_WithHardRules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create source dir with one rule.
	srcDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, srcDir)
	mustWrite(t, filepath.Join(srcDir, "rule.md"), "# Rule\n")

	// Create hard-rules.
	phDir := filepath.Join(dir, ".promptherder")
	mustWrite(t, filepath.Join(phDir, "hard-rules.md"), "---\ntrigger: always_on\n---\n\n# Hard Rules\n\n- Never do bad things.\n")

	content, names, err := buildConcatOutput(dir, defaultSourceDir, nil, "<!-- test -->\n")
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "Never do bad things") {
		t.Error("expected hard-rules content")
	}
	if names[0] != "hard-rules" {
		t.Errorf("expected first source to be hard-rules, got %s", names[0])
	}
}

func TestBuildConcatOutput_NoSources(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content, names, err := buildConcatOutput(dir, defaultSourceDir, nil, "<!-- test -->\n")
	if err != nil {
		t.Fatal(err)
	}
	if content != nil {
		t.Errorf("expected nil content, got %d bytes", len(content))
	}
	if names != nil {
		t.Errorf("expected nil names, got %v", names)
	}
}

func TestBuildConcatOutput_SkipsApplyToRules(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	srcDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, srcDir)
	mustWrite(t, filepath.Join(srcDir, "global.md"), "# Global\n")
	mustWrite(t, filepath.Join(srcDir, "scoped.md"), "---\napplyTo: \"**/*.go\"\n---\n# Scoped\n")

	content, names, err := buildConcatOutput(dir, defaultSourceDir, nil, "<!-- test -->\n")
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "# Global") {
		t.Error("expected global rule")
	}
	if strings.Contains(s, "# Scoped") {
		t.Error("scoped rules (applyTo) should be excluded from concatenation")
	}
	if len(names) != 1 {
		t.Errorf("expected 1 source, got %d: %v", len(names), names)
	}
}
