package app

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestMappingsFor_Antigravity(t *testing.T) {
	m, err := mappingsFor("antigravity")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(m))
	}

	assertMapping(t, m[0], "rules", ".antigravity/rules", ".agent/rules")
	assertMapping(t, m[1], "skills", ".antigravity/skills", ".agent/skills")
}

func TestMappingsFor_Agent(t *testing.T) {
	m, err := mappingsFor("agent")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 2 {
		t.Fatalf("expected 2 mappings, got %d", len(m))
	}

	assertMapping(t, m[0], "rules", ".agent/rules", ".antigravity/rules")
	assertMapping(t, m[1], "skills", ".agent/skills", ".antigravity/skills")
}

func TestMappingsFor_CaseInsensitive(t *testing.T) {
	for _, name := range []string{"Antigravity", "ANTIGRAVITY", " antigravity "} {
		m, err := mappingsFor(name)
		if err != nil {
			t.Errorf("mappingsFor(%q) unexpected error: %v", name, err)
			continue
		}
		if len(m) != 2 {
			t.Errorf("mappingsFor(%q) expected 2 mappings, got %d", name, len(m))
		}
	}
}

func TestMappingsFor_Unknown(t *testing.T) {
	_, err := mappingsFor("copilot")
	if err == nil {
		t.Fatal("expected error for unknown source")
	}
}

func TestBuildPlan_Basic(t *testing.T) {
	dir := t.TempDir()

	// Create source structure: .antigravity/rules/00-foo.md
	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "00-foo.md"), []byte("# Foo"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := buildPlan(dir, "antigravity", nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(plan) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan))
	}

	want := filepath.Join(dir, ".agent", "rules", "00-foo.md")
	if plan[0].Target != want {
		t.Errorf("target = %q, want %q", plan[0].Target, want)
	}
}

func TestBuildPlan_MultipleMappings(t *testing.T) {
	dir := t.TempDir()

	// Rules
	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "rule.md"), []byte("rule"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Skills
	skillsDir := filepath.Join(dir, ".antigravity", "skills", "deploy")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte("skill"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := buildPlan(dir, "antigravity", nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(plan) != 2 {
		t.Fatalf("expected 2 plan items, got %d", len(plan))
	}

	targets := make(map[string]bool, len(plan))
	for _, item := range plan {
		targets[item.Target] = true
	}

	wantRule := filepath.Join(dir, ".agent", "rules", "rule.md")
	wantSkill := filepath.Join(dir, ".agent", "skills", "deploy", "SKILL.md")

	if !targets[wantRule] {
		t.Errorf("missing target %q", wantRule)
	}
	if !targets[wantSkill] {
		t.Errorf("missing target %q", wantSkill)
	}
}

func TestBuildPlan_IncludeFilter(t *testing.T) {
	dir := t.TempDir()

	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "keep.md"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "skip.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := buildPlan(dir, "antigravity", []string{"**/*.md"})
	if err != nil {
		t.Fatal(err)
	}

	if len(plan) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan))
	}
	wantTarget := filepath.Join(dir, ".agent", "rules", "keep.md")
	if plan[0].Target != wantTarget {
		t.Errorf("target = %q, want %q", plan[0].Target, wantTarget)
	}
}

func TestBuildPlan_MissingSource(t *testing.T) {
	dir := t.TempDir()
	// No .antigravity/ directory exists — should return empty plan, not error
	plan, err := buildPlan(dir, "antigravity", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan) != 0 {
		t.Fatalf("expected 0 plan items, got %d", len(plan))
	}
}

func TestBuildPlan_ReverseDirection(t *testing.T) {
	dir := t.TempDir()

	agentDir := filepath.Join(dir, ".agent", "rules")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "rule.md"), []byte("rule"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := buildPlan(dir, "agent", nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(plan) != 1 {
		t.Fatalf("expected 1 plan item, got %d", len(plan))
	}

	want := filepath.Join(dir, ".antigravity", "rules", "rule.md")
	if plan[0].Target != want {
		t.Errorf("target = %q, want %q", plan[0].Target, want)
	}
}

func TestDedupePlan_PreservesOrder(t *testing.T) {
	items := []planItem{
		{Source: "a", Target: "x"},
		{Source: "b", Target: "y"},
		{Source: "c", Target: "x"}, // duplicate target, should overwrite first
	}

	result := dedupePlan(items)

	if len(result) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result))
	}
	// Order preserved: "x" first, "y" second. Source for "x" is "c" (last write wins).
	if result[0].Target != "x" || result[0].Source != "c" {
		t.Errorf("result[0] = %+v, want {Source:c, Target:x}", result[0])
	}
	if result[1].Target != "y" || result[1].Source != "b" {
		t.Errorf("result[1] = %+v, want {Source:b, Target:y}", result[1])
	}
}

func TestRun_DryRun(t *testing.T) {
	dir := t.TempDir()

	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "00-test.md"), []byte("# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		RepoPath: dir,
		Source:   "antigravity",
		DryRun:   true,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	// Target should NOT exist in dry-run mode
	target := filepath.Join(dir, ".agent", "rules", "00-test.md")
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Error("dry-run should not create target files")
	}
}

func TestRun_ActualSync(t *testing.T) {
	dir := t.TempDir()

	rulesDir := filepath.Join(dir, ".antigravity", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte("# Test Rule\n\nDo the thing.\n")
	if err := os.WriteFile(filepath.Join(rulesDir, "00-test.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		RepoPath: dir,
		Source:   "antigravity",
		DryRun:   false,
		Logger:   slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}

	if err := Run(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, ".agent", "rules", "00-test.md")
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("target not created: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestRun_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"empty repo path", Config{RepoPath: "", Source: "antigravity"}},
		{"empty source", Config{RepoPath: "/tmp", Source: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.cfg.Logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
			err := Run(context.Background(), tt.cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func assertMapping(t *testing.T, m mapping, name, source, target string) {
	t.Helper()
	if m.Name != name {
		t.Errorf("name = %q, want %q", m.Name, name)
	}
	if m.SourceRoot != source {
		t.Errorf("source = %q, want %q", m.SourceRoot, source)
	}
	if m.TargetRoot != target {
		t.Errorf("target = %q, want %q", m.TargetRoot, target)
	}
}
