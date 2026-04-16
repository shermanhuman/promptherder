package app

import (
	"context"
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
	content, names, err := buildConcatOutput(context.Background(), dir, defaultSourceDir, nil, header)
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

	content, names, err := buildConcatOutput(context.Background(), dir, defaultSourceDir, nil, "<!-- test -->\n")
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

	content, names, err := buildConcatOutput(context.Background(), dir, defaultSourceDir, nil, "<!-- test -->\n")
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

	content, names, err := buildConcatOutput(context.Background(), dir, defaultSourceDir, nil, "<!-- test -->\n")
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

// --- buildConcatSkillsSection tests ---

func TestBuildConcatSkillsSection_WithSkills(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Create two skill directories with SKILL.md.
	skill1 := filepath.Join(dir, ".promptherder", "agent", "skills", "alpha-skill")
	mustMkdir(t, skill1)
	mustWrite(t, filepath.Join(skill1, "SKILL.md"), "---\nname: alpha\n---\n\n# Alpha Skill\n\nDo alpha things.\n")

	skill2 := filepath.Join(dir, ".promptherder", "agent", "skills", "beta-skill")
	mustMkdir(t, skill2)
	mustWrite(t, filepath.Join(skill2, "SKILL.md"), "# Beta Skill\n\nDo beta things.\n")

	content, names, err := buildConcatSkillsSection(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if content == nil {
		t.Fatal("expected content, got nil")
	}

	s := string(content)
	if !strings.Contains(s, "## Skill: alpha-skill") {
		t.Error("expected alpha-skill section header")
	}
	if !strings.Contains(s, "Do alpha things") {
		t.Error("expected alpha skill content (frontmatter stripped)")
	}
	if !strings.Contains(s, "## Skill: beta-skill") {
		t.Error("expected beta-skill section header")
	}
	if len(names) != 2 {
		t.Errorf("expected 2 skill names, got %d", len(names))
	}
}

func TestBuildConcatSkillsSection_PrefersVariant(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	skill := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skill)
	mustWrite(t, filepath.Join(skill, "SKILL.md"), "# Generic\n")
	mustWrite(t, filepath.Join(skill, "CLAUDE.md"), "# Claude-specific\n")

	content, _, err := buildConcatSkillsSection(dir, "claude")
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "Claude-specific") {
		t.Error("expected claude variant content")
	}
	if strings.Contains(s, "# Generic") {
		t.Error("generic SKILL.md should not be included when variant exists")
	}
}

func TestBuildConcatSkillsSection_FallsBackToGeneric(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	skill := filepath.Join(dir, ".promptherder", "agent", "skills", "my-skill")
	mustMkdir(t, skill)
	mustWrite(t, filepath.Join(skill, "SKILL.md"), "# Generic Skill\n")

	content, _, err := buildConcatSkillsSection(dir, "claude")
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if !strings.Contains(s, "# Generic Skill") {
		t.Error("expected generic SKILL.md content when no variant exists")
	}
}

func TestBuildConcatSkillsSection_EmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content, names, err := buildConcatSkillsSection(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if content != nil {
		t.Error("expected nil content for missing skills dir")
	}
	if names != nil {
		t.Error("expected nil names for missing skills dir")
	}
}

func TestBuildConcatSkillsSection_SkipsDirWithoutSkillFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// A skill dir with no SKILL.md or variant.
	skill := filepath.Join(dir, ".promptherder", "agent", "skills", "empty-skill")
	mustMkdir(t, skill)
	mustWrite(t, filepath.Join(skill, "README.md"), "# Not a skill\n")

	content, _, err := buildConcatSkillsSection(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if content != nil {
		t.Error("expected nil content when no SKILL.md exists")
	}
}

// --- buildConcatWorkflowsSection tests ---

func TestBuildConcatWorkflowsSection_WithWorkflows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wfDir := filepath.Join(dir, ".promptherder", "agent", "workflows")
	mustMkdir(t, wfDir)
	mustWrite(t, filepath.Join(wfDir, "plan.md"), "---\ndescription: Plan things\n---\n\n# Plan\n\nDo planning.\n")
	mustWrite(t, filepath.Join(wfDir, "review.md"), "# Review\n\nDo reviewing.\n")

	content, names, err := buildConcatWorkflowsSection(dir)
	if err != nil {
		t.Fatal(err)
	}
	if content == nil {
		t.Fatal("expected content, got nil")
	}

	s := string(content)
	if !strings.Contains(s, "## Workflow: plan") {
		t.Error("expected plan workflow section header")
	}
	if !strings.Contains(s, "Do planning") {
		t.Error("expected plan workflow content")
	}
	if !strings.Contains(s, "## Workflow: review") {
		t.Error("expected review workflow section header")
	}
	if len(names) != 2 {
		t.Errorf("expected 2 workflow names, got %d", len(names))
	}
}

func TestBuildConcatWorkflowsSection_StripsTurboAnnotations(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wfDir := filepath.Join(dir, ".promptherder", "agent", "workflows")
	mustMkdir(t, wfDir)
	mustWrite(t, filepath.Join(wfDir, "execute.md"), "// turbo-all\n\n# Execute\n\n1. Do step one.\n// turbo\n2. Do step two.\n")

	content, _, err := buildConcatWorkflowsSection(dir)
	if err != nil {
		t.Fatal(err)
	}

	s := string(content)
	if strings.Contains(s, "// turbo") {
		t.Error("turbo annotations should be stripped from workflow content")
	}
	if !strings.Contains(s, "Do step one") {
		t.Error("expected workflow content preserved after stripping")
	}
}

func TestBuildConcatWorkflowsSection_EmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	content, names, err := buildConcatWorkflowsSection(dir)
	if err != nil {
		t.Fatal(err)
	}
	if content != nil {
		t.Error("expected nil content for missing workflows dir")
	}
	if names != nil {
		t.Error("expected nil names for missing workflows dir")
	}
}

func TestBuildConcatWorkflowsSection_SkipsNonMarkdown(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	wfDir := filepath.Join(dir, ".promptherder", "agent", "workflows")
	mustMkdir(t, wfDir)
	mustWrite(t, filepath.Join(wfDir, "plan.md"), "# Plan\n")
	mustWrite(t, filepath.Join(wfDir, "notes.txt"), "not a workflow")

	_, names, err := buildConcatWorkflowsSection(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "plan" {
		t.Errorf("expected [plan], got %v", names)
	}
}

// --- buildFullConcatTarget integration test ---

func TestBuildFullConcatTarget_Integration(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	// Rules.
	rulesDir := filepath.Join(dir, ".promptherder", "agent", "rules")
	mustMkdir(t, rulesDir)
	mustWrite(t, filepath.Join(rulesDir, "project.md"), "# Project Rules\n\n- Use Go.\n")

	// Hard rules.
	mustWrite(t, filepath.Join(dir, ".promptherder", "hard-rules.md"), "---\ntrigger: always_on\n---\n\n# Hard Rules\n\n- No secrets.\n")

	// Skill.
	skillDir := filepath.Join(dir, ".promptherder", "agent", "skills", "tdd")
	mustMkdir(t, skillDir)
	mustWrite(t, filepath.Join(skillDir, "SKILL.md"), "# TDD Skill\n\nWrite tests first.\n")

	// Workflow.
	wfDir := filepath.Join(dir, ".promptherder", "agent", "workflows")
	mustMkdir(t, wfDir)
	mustWrite(t, filepath.Join(wfDir, "plan.md"), "# Plan Workflow\n\nPlan before coding.\n")

	content, names, err := buildFullConcatTarget(context.Background(), dir, defaultSourceDir, nil, "<!-- header -->\n", "")
	if err != nil {
		t.Fatal(err)
	}
	if content == nil {
		t.Fatal("expected content, got nil")
	}

	s := string(content)

	// Verify all sections present.
	if !strings.Contains(s, "<!-- header -->") {
		t.Error("expected header")
	}
	if !strings.Contains(s, "No secrets") {
		t.Error("expected hard-rules content")
	}
	if !strings.Contains(s, "Use Go") {
		t.Error("expected project rules content")
	}
	if !strings.Contains(s, "## Skill: tdd") {
		t.Error("expected skill section")
	}
	if !strings.Contains(s, "Write tests first") {
		t.Error("expected skill content")
	}
	if !strings.Contains(s, "## Workflow: plan") {
		t.Error("expected workflow section")
	}
	if !strings.Contains(s, "Plan before coding") {
		t.Error("expected workflow content")
	}

	// Verify all sources tracked.
	if len(names) < 4 {
		t.Errorf("expected at least 4 sources (hard-rules, project, tdd, plan), got %d: %v", len(names), names)
	}
}
