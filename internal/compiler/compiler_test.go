package compiler

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func put(t *testing.T, root, path, data string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	put(t, root, ".promptherder/agent/rules/base.md", "# Rules\nUse clear names.\n")
	put(t, root, ".promptherder/agent/skills/example/SKILL.md", "---\nname: example\ndescription: Use for an example task.\n---\nSECRET_SKILL_BODY\nRead [guide](references/guide.md).\n")
	put(t, root, ".promptherder/agent/skills/example/references/guide.md", "Detailed reference\n")
	put(t, root, ".promptherder/agent/skills/example/scripts/check.sh", "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(root, ".promptherder/agent/skills/example/scripts/check.sh"), 0755); err != nil {
		t.Fatal(err)
	}
	return root
}
func plan(t *testing.T, root string, o Options) *Plan {
	t.Helper()
	p, err := Build(context.Background(), root, o)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func artifact(t *testing.T, p *Plan, path string) Artifact {
	t.Helper()
	for _, a := range p.Artifacts {
		if a.Path == path {
			return a
		}
	}
	t.Fatalf("missing artifact %s", path)
	return Artifact{}
}
func code(p *Plan, c string) bool {
	for _, d := range p.Diagnostics {
		if d.Code == c {
			return true
		}
	}
	return false
}
func apply(t *testing.T, p *Plan) {
	t.Helper()
	if err := p.Apply(context.Background()); err != nil {
		t.Fatalf("%v: %+v", err, p.Diagnostics)
	}
}

func TestNativeBundlesAndPurePlan(t *testing.T) {
	root := fixture(t)
	o := Options{Targets: []string{"codex", "claude"}}
	p := plan(t, root, o)
	if p.HasErrors() {
		t.Fatal(p.Diagnostics)
	}
	if strings.Contains(string(artifact(t, p, "AGENTS.md").Content), "SECRET_SKILL_BODY") {
		t.Fatal("skill was inlined")
	}
	if !strings.Contains(string(artifact(t, p, "CLAUDE.md").Content), "@AGENTS.md") {
		t.Fatal("missing Claude import")
	}
	for _, dir := range []string{".agents/skills/example/", ".claude/skills/example/"} {
		a := artifact(t, p, dir+"SKILL.md")
		d, err := parse(Source{Data: a.Content})
		if err != nil || str(d.Meta, "name") != "example" {
			t.Fatalf("invalid output frontmatter: %v", err)
		}
		if artifact(t, p, dir+"scripts/check.sh").Mode != 0755 {
			t.Fatal("lost executable mode")
		}
		artifact(t, p, dir+"references/guide.md")
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("planning wrote files")
	}
	apply(t, p)
	first, err := os.ReadFile(filepath.Join(root, ".promptherder/manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	apply(t, plan(t, root, o))
	second, _ := os.ReadFile(filepath.Join(root, ".promptherder/manifest.json"))
	if string(first) != string(second) {
		t.Fatal("non-idempotent manifest")
	}
}

func TestScopedAndManualRules(t *testing.T) {
	root := fixture(t)
	put(t, root, ".promptherder/agent/rules/shell.md", "---\napplyTo: '**/*.sh'\n---\nSHELL_ONLY\n")
	put(t, root, ".promptherder/agent/rules/browser.md", "---\ntrigger: manual\n---\nBROWSER_ONLY\n")
	p := plan(t, root, Options{Targets: []string{"codex", "claude", "cursor", "copilot", "cline", "windsurf", "antigravity"}})
	if p.HasErrors() {
		t.Fatal(p.Diagnostics)
	}
	if !code(p, "model-mediated-paths") {
		t.Fatal("missing glob fallback warning")
	}
	if strings.Contains(string(artifact(t, p, "AGENTS.md").Content), "BROWSER_ONLY") {
		t.Fatal("manual rule became always on")
	}
	artifact(t, p, ".cursor/rules/shell.mdc")
	artifact(t, p, ".claude/rules/shell.md")
	artifact(t, p, ".github/instructions/shell.instructions.md")
	d, err := parse(Source{Data: artifact(t, p, ".claude/skills/rule-browser/SKILL.md").Content})
	if err != nil || d.Meta["disable-model-invocation"] != true {
		t.Fatal("manual invocation metadata lost")
	}
	if !strings.Contains(string(artifact(t, p, ".agents/skills/rule-browser/agents/openai.yaml").Content), "allow_implicit_invocation: false") {
		t.Fatal("Codex manual policy missing")
	}
	strict := plan(t, root, Options{Targets: []string{"codex"}, Strict: true})
	if !strict.HasErrors() {
		t.Fatal("strict silently accepted fallback")
	}
}

func TestVariantLeakFailsBeforeWrites(t *testing.T) {
	root := fixture(t)
	put(t, root, ".promptherder/agent/skills/example/CLAUDE.md", "---\nname: example\ndescription: Example\n---\nClaude-only instructions\n")
	p := plan(t, root, Options{Targets: []string{"claude", "codex", "copilot"}})
	if !code(p, "variant-leak") {
		t.Fatal(p.Diagnostics)
	}
	if err := p.Apply(context.Background()); err == nil {
		t.Fatal("applied conflicting plan")
	}
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("partial write")
	}
}

func TestConflictsAndStaleOwnership(t *testing.T) {
	root := fixture(t)
	o := Options{Targets: []string{"codex", "claude"}}
	apply(t, plan(t, root, o))
	put(t, root, ".agents/skills/example/SKILL.md", "user changed this\n")
	p := plan(t, root, o)
	if !code(p, "local-conflict") {
		t.Fatal(p.Diagnostics)
	}
	p = plan(t, root, Options{Targets: []string{"claude"}})
	if !code(p, "stale-conflict") {
		t.Fatal("edited stale output would be deleted")
	}
}

func TestRemoveOneConsumerKeepsSharedArtifact(t *testing.T) {
	root := fixture(t)
	apply(t, plan(t, root, Options{Targets: []string{"codex", "cursor"}}))
	p := plan(t, root, Options{Targets: []string{"cursor"}})
	apply(t, p)
	if _, err := os.Stat(filepath.Join(root, ".agents/skills/example/SKILL.md")); err != nil {
		t.Fatal(err)
	}
	apply(t, plan(t, root, Options{}))
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("last target output retained")
	}
}

func TestAdoptionAndConcurrentEdits(t *testing.T) {
	root := fixture(t)
	put(t, root, "AGENTS.md", "My existing instructions\n")
	p := plan(t, root, Options{Targets: []string{"codex"}})
	if !p.HasErrors() {
		t.Fatal("unmanaged file would be overwritten")
	}
	o := Options{Targets: []string{"codex"}, Adopt: true}
	p = plan(t, root, o)
	apply(t, p)
	put(t, root, ".promptherder/agent/rules/base.md", "Updated shared rule\n")
	p = plan(t, root, o)
	if !strings.Contains(string(artifact(t, p, "AGENTS.md").Content), "My existing instructions") {
		t.Fatal("adopted text lost")
	}
	apply(t, p)
	p = plan(t, root, o)
	put(t, root, "AGENTS.md", "edit after plan\n")
	if err := p.Apply(context.Background()); err == nil {
		t.Fatal("overwrote edit after planning")
	}
}

func TestHerdPlanningLockAndOverrides(t *testing.T) {
	root := t.TempDir()
	put(t, root, ".promptherder/herds/test/herd.json", `{"name":"test","version":"1.0.0"}`)
	put(t, root, ".promptherder/herds/test/rules/rule.md", "FIRST\n")
	o := Options{Targets: []string{"codex"}}
	p := plan(t, root, o)
	if !strings.Contains(string(artifact(t, p, "AGENTS.md").Content), "FIRST") {
		t.Fatal("fresh herd missing from dry run")
	}
	apply(t, p)
	put(t, root, ".promptherder/herds/test/rules/rule.md", "SECOND\n")
	p = plan(t, root, o)
	if !code(p, "lock-mismatch") {
		t.Fatal("changed herd bypassed lock")
	}
	o.UpdateLock = true
	apply(t, plan(t, root, o))
	put(t, root, ".promptherder/agent/rules/rule.md", "LOCAL\n")
	if _, err := Build(context.Background(), root, o); err == nil {
		t.Fatal("implicit local override")
	}
	o.Overrides = []string{"rules/rule.md"}
	p = plan(t, root, o)
	if !strings.Contains(string(artifact(t, p, "AGENTS.md").Content), "LOCAL") {
		t.Fatal("override not resolved")
	}
}

func TestRecoveryProtectsNewEdits(t *testing.T) {
	root := t.TempDir()
	j := journal{Before: map[string]Snapshot{"AGENTS.md": {true, []byte("old"), 0644}}, After: map[string]Snapshot{"AGENTS.md": {true, []byte("new"), 0644}}}
	data, _ := json.Marshal(j)
	put(t, root, ".promptherder/apply-journal.json", string(data))
	put(t, root, "AGENTS.md", "new")
	if err := Recover(root); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if string(got) != "old" {
		t.Fatal("rollback failed")
	}
	put(t, root, ".promptherder/apply-journal.json", string(data))
	put(t, root, "AGENTS.md", "user edit")
	if err := Recover(root); err == nil {
		t.Fatal("recovery overwrote user edit")
	}
}

func TestSymlinkAndCorruptManifestRejected(t *testing.T) {
	root := fixture(t)
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, ".agents")); err != nil {
		t.Skip(err)
	}
	if _, err := Build(context.Background(), root, Options{Targets: []string{"codex"}}); err == nil {
		t.Fatal("followed output symlink")
	}
	root = fixture(t)
	put(t, root, ".promptherder/manifest.json", "{bad")
	if _, err := Build(context.Background(), root, Options{}); err == nil {
		t.Fatal("ignored corrupt ownership")
	}
}

func TestMalformedAndMissingSkillMetadata(t *testing.T) {
	for _, text := range []string{"---\nname: example\nname: duplicate\n---\nbody", "---\nname: wrong\ndescription: example\n---\nbody", "---\nname: example\n---\nbody"} {
		root := fixture(t)
		put(t, root, ".promptherder/agent/skills/example/SKILL.md", text)
		if _, err := Build(context.Background(), root, Options{Targets: []string{"codex"}}); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
}

func TestRemovingTargetPreservesAdoptedInstructions(t *testing.T) {
	root := fixture(t)
	put(t, root, "AGENTS.md", "User-owned policy.\n")
	apply(t, plan(t, root, Options{Targets: []string{"codex"}, Adopt: true}))
	apply(t, plan(t, root, Options{Targets: []string{}}))
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil || !strings.Contains(string(data), "User-owned policy.") || strings.Contains(string(data), "Use clear names") {
		t.Fatal("adopted instructions were not restored", string(data), err)
	}
	apply(t, plan(t, root, Options{Targets: []string{"codex"}}))
	data, _ = os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if strings.Count(string(data), "User-owned policy.") != 1 {
		t.Fatal("preserved policy duplicated")
	}
}

func TestRecoveryCannotRaceLiveApply(t *testing.T) {
	root := fixture(t)
	release, err := acquireLock(root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := Recover(root); err == nil {
		t.Fatal("recovery acquired active sync lock")
	}
}

func TestDeclarationsSelectDependenciesAndOverlay(t *testing.T) {
	root := t.TempDir()
	base := ".promptherder/herds/demo/"
	put(t, root, base+"herd.json", `{"name":"demo","skills":{"entry":{"requires":["helper"],"invocation":"manual","overlays":{"claude":"overlays/claude/entry.md"}}}}`)
	put(t, root, base+"skills/entry/SKILL.md", "---\nname: entry\ndescription: Entry\n---\nEntry body\n")
	put(t, root, base+"skills/helper/SKILL.md", "---\nname: helper\ndescription: Helper\n---\nHelper body\n")
	put(t, root, base+"overlays/claude/entry.md", "---\nname: entry\ndescription: Claude entry\n---\nClaude body\n")
	p := plan(t, root, Options{Targets: []string{"claude", "codex"}, Skills: []string{"entry"}})
	if p.HasErrors() {
		t.Fatal(p.Diagnostics)
	}
	artifact(t, p, ".agents/skills/helper/SKILL.md")
	claude := artifact(t, p, ".claude/skills/entry/SKILL.md")
	if !strings.Contains(string(claude.Content), "Claude body") || !strings.Contains(string(claude.Content), "disable-model-invocation: true") {
		t.Fatal(string(claude.Content))
	}
	codex := artifact(t, p, ".agents/skills/entry/agents/openai.yaml")
	if !strings.Contains(string(codex.Content), "allow_implicit_invocation: false") {
		t.Fatal(string(codex.Content))
	}
	apply(t, p)
	put(t, root, base+"overlays/claude/entry.md", "---\nname: entry\ndescription: Changed\n---\nChanged\n")
	if !code(plan(t, root, Options{Targets: []string{"claude"}}), "lock-mismatch") {
		t.Fatal("overlay changes escaped lock")
	}
}

func TestDoctorReportsPersonalOverlap(t *testing.T) {
	root := fixture(t)
	home := t.TempDir()
	put(t, home, ".claude/skills/example/SKILL.md", "---\nname: example\ndescription: Personal variant\n---\nBody\n")
	p := plan(t, root, Options{Targets: []string{"claude"}, Strict: true})
	if err := p.Doctor(home, ""); err != nil {
		t.Fatal(err)
	}
	if !code(p, "personal-skill-overlap") || !p.HasErrors() {
		t.Fatal(p.Diagnostics)
	}
}

func TestExplicitUserScope(t *testing.T) {
	root := fixture(t)
	p := plan(t, root, Options{Scope: "user", Targets: []string{"codex", "claude"}})
	artifact(t, p, ".codex/AGENTS.md")
	bridge := artifact(t, p, ".claude/CLAUDE.md")
	if !strings.Contains(string(bridge.Content), "@../.codex/AGENTS.md") {
		t.Fatal("wrong personal import")
	}
	apply(t, p)
	if _, err := os.Stat(filepath.Join(root, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("wrote repository baseline in home")
	}
	if _, err := Build(context.Background(), root, Options{Targets: []string{"codex"}}); err == nil {
		t.Fatal("scope mismatch accepted")
	}
	apply(t, plan(t, root, Options{Scope: "user", Targets: []string{"claude"}}))
	if _, err := os.Stat(filepath.Join(root, ".codex/AGENTS.md")); !os.IsNotExist(err) {
		t.Fatal("disabled Codex retains personal baseline")
	}
	data, _ := os.ReadFile(filepath.Join(root, ".claude/CLAUDE.md"))
	if !strings.Contains(string(data), "Use clear names") || strings.Contains(string(data), "@../.codex") {
		t.Fatal("Claude baseline was not retained")
	}
	if _, err := Build(context.Background(), t.TempDir(), Options{Scope: "user", Targets: []string{"cursor"}}); err == nil {
		t.Fatal("unverified user profile accepted")
	}
}

func TestRecoveryReclaimsExitedProcess(t *testing.T) {
	root := fixture(t)
	put(t, root, ".promptherder/sync.lock", "2147483647\n")
	if err := Recover(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".promptherder/sync.lock")); !os.IsNotExist(err) {
		t.Fatal("stale lock retained")
	}
}

func TestWorkflowDeclarationsHonorCommandPrefix(t *testing.T) {
	root := t.TempDir()
	base := ".promptherder/herds/demo/"
	put(t, root, base+"herd.json", `{"name":"demo","skills":{"workflow-plan":{"requires":["helper"]}}}`)
	put(t, root, base+"workflows/plan.md", "---\ndescription: Plan a task\n---\nUse helper.\n")
	put(t, root, base+"skills/helper/SKILL.md", "---\nname: helper\ndescription: Help\n---\nBody\n")
	put(t, root, base+"skills/helper/scripts/__pycache__/helper.pyc", "temporary interpreter cache")
	p := plan(t, root, Options{Targets: []string{"codex", "claude"}, CommandPrefix: "cv-", Skills: []string{"cv-workflow-plan"}})
	if p.HasErrors() {
		t.Fatal(p.Diagnostics)
	}
	artifact(t, p, ".agents/skills/helper/SKILL.md")
	artifact(t, p, ".claude/skills/cv-workflow-plan/SKILL.md")
	for _, a := range p.Artifacts {
		if strings.Contains(a.Path, "__pycache__") {
			t.Fatal("bundled interpreter cache")
		}
	}
}
