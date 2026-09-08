package compiler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MarshalJSON includes reviewable UTF-8 artifact content rather than base64.
func (a Artifact) MarshalJSON() ([]byte, error) {
	type alias Artifact
	return json.Marshal(struct {
		alias
		Content string `json:"content"`
	}{alias(a), string(a.Content)})
}

// Doctor supplements repository validation with read-only personal discovery.
// It does not inspect credentials or execute any installed host.
func (p *Plan) Doctor(home, codexHome string) error {
	if codexHome == "" {
		codexHome = filepath.Join(home, ".codex")
	}
	if contains(p.Targets, "codex") {
		for _, name := range []string{"AGENTS.override.md", "AGENTS.md"} {
			path := filepath.Join(codexHome, name)
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			p.note("info", "personal-instructions", path, fmt.Sprintf("Codex discovers %d bytes of personal instructions before project instructions", len(data)))
			break
		}
	}
	if contains(p.Targets, "claude") {
		path := filepath.Join(home, ".claude/CLAUDE.md")
		if info, err := os.Stat(path); err == nil {
			p.note("info", "personal-instructions", path, fmt.Sprintf("Claude Code also loads %d bytes of personal instructions", info.Size()))
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	for _, host := range p.Targets {
		var roots []string
		switch host {
		case "codex":
			roots = []string{filepath.Join(home, ".agents/skills"), filepath.Join(codexHome, "skills")}
		case "claude":
			roots = []string{filepath.Join(home, ".claude/skills")}
		default:
			continue
		}
		for _, root := range roots {
			entries, err := os.ReadDir(root)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return err
			}
			for _, entry := range entries {
				path := filepath.Join(root, entry.Name(), "SKILL.md")
				data, err := os.ReadFile(path)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return err
				}
				d, err := parse(Source{Origin: path, Data: data})
				if err != nil {
					p.note("warning", "personal-skill-invalid", path, err.Error())
					continue
				}
				for _, a := range p.Artifacts {
					if filepath.Clean(filepath.Join(p.Root, a.Path)) == filepath.Clean(path) {
						continue
					}
					if a.Kind == "skill" && a.ID == str(d.Meta, "name") && visible(host, a.Path) {
						p.note("warning", "personal-skill-overlap", path, host+" also discovers this name; personal precedence may change the selected skill")
						break
					}
				}
			}
		}
	}
	p.note("info", "runtime-unverified", "", "Host versions, enterprise policies, plugins, nested instructions, and actual model behavior are not verified by this static check.")
	if p.opts.Strict {
		for i := range p.Diagnostics {
			if p.Diagnostics[i].Severity == "warning" {
				p.Diagnostics[i].Severity = "error"
			}
		}
	}
	return nil
}

func (p *Plan) Explains(id string) bool {
	for _, a := range p.Artifacts {
		if a.ID == id {
			return true
		}
	}
	return false
}

// Workflow aliases remain intelligible after migration to namespaced skills.
func workflowAliases(skills []Skill) string {
	var aliases []string
	for _, s := range skills {
		if !s.Workflow {
			continue
		}
		name := strings.TrimSuffix(filepath.Base(s.Source.Path), ".md")
		aliases = append(aliases, fmt.Sprintf("`/%s` refers to the `%s` skill", name, s.ID))
	}
	if len(aliases) == 0 {
		return ""
	}
	return "Legacy workflow references: " + strings.Join(aliases, "; ") + ". These mappings do not authorize starting another workflow.\n\n"
}
