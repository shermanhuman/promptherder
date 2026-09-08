// Package compiler plans native host artifacts without changing source files.
package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"gopkg.in/yaml.v3"
)

type Source struct {
	Path   string      `json:"path"`
	Origin string      `json:"origin"`
	Data   []byte      `json:"-"`
	Mode   fs.FileMode `json:"mode"`
}

type Document struct {
	Source Source
	Meta   map[string]any
	Body   string
}

type Rule struct {
	Document
	ID         string
	Activation string
	Paths      []string
	Required   bool
}

type Skill struct {
	ID string
	Document
	Files    map[string]Source
	Variants map[string]Document
	Manual   bool
	Workflow bool
	Requires []string
}

type SkillDeclaration struct {
	Requires   []string          `json:"requires,omitempty"`
	Invocation string            `json:"invocation,omitempty"`
	Overlays   map[string]string `json:"overlays,omitempty"`
}

type Content struct {
	Rules  []Rule
	Skills []Skill
	Locks  map[string]HerdLock
}

type HerdLock struct {
	Repository string `json:"repository,omitempty"`
	Revision   string `json:"revision,omitempty"`
	Version    string `json:"version,omitempty"`
	Digest     string `json:"digest"`
}

func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

// HerdDigest hashes compiler inputs and executable modes, excluding download provenance.
func HerdDigest(root string) (string, error) {
	all, err := readTree(root, root)
	if err != nil {
		return "", err
	}
	var sum bytes.Buffer
	for _, k := range keys(all) {
		if k != "herd.json" && !strings.HasPrefix(k, "rules/") && !strings.HasPrefix(k, "skills/") && !strings.HasPrefix(k, "workflows/") && !strings.HasPrefix(k, "overlays/") {
			continue
		}
		src := all[k]
		fmt.Fprintf(&sum, "%s\x00%o\x00%s\x00", k, src.Mode, digest(src.Data))
	}
	return digest(sum.Bytes()), nil
}

func parse(src Source) (Document, error) {
	d := Document{Source: src, Meta: map[string]any{}, Body: string(src.Data)}
	lines := strings.Split(strings.ReplaceAll(d.Body, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return d, nil
	}
	end := 1
	for end < len(lines) && strings.TrimSpace(lines[end]) != "---" {
		end++
	}
	if end == len(lines) {
		return d, fmt.Errorf("%s: unterminated YAML frontmatter", src.Origin)
	}
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &d.Meta); err != nil {
		return d, fmt.Errorf("%s: %w", src.Origin, err)
	}
	if d.Meta == nil {
		d.Meta = map[string]any{}
	}
	for _, key := range []string{"disable-model-invocation", "user-invocable", "required"} {
		if v, exists := d.Meta[key]; exists {
			if _, ok := v.(bool); !ok {
				return d, fmt.Errorf("%s: %s must be a boolean", src.Origin, key)
			}
		}
	}
	for _, key := range []string{"name", "description", "activation", "trigger", "id"} {
		if v, exists := d.Meta[key]; exists {
			if _, ok := v.(string); !ok {
				return d, fmt.Errorf("%s: %s must be a string", src.Origin, key)
			}
		}
	}
	d.Body = strings.TrimSpace(strings.Join(lines[end+1:], "\n")) + "\n"
	return d, nil
}

func str(m map[string]any, key string) string { v, _ := m[key].(string); return v }
func list(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	if s, ok := v.(string); ok {
		return []string{s}, nil
	} // A comma may occur inside a brace glob.
	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a string or string list")
	}
	result := []string{}
	for _, item := range items {
		s, ok := item.(string)
		if !ok || s == "" {
			return nil, fmt.Errorf("expected nonempty path strings")
		}
		result = append(result, s)
	}
	return result, nil
}

func readTree(root, origin string) (map[string]Source, error) {
	result := map[string]Source{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if os.IsNotExist(err) && path == root {
			return nil
		}
		if err != nil {
			return err
		}
		if entry.Name() == ".git" && entry.IsDir() {
			return filepath.SkipDir
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("source symlink is not portable: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported source: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		result[rel] = Source{Path: rel, Origin: origin + "/" + rel, Data: data, Mode: info.Mode().Perm()}
		return nil
	})
	return result, err
}

func keys[T any](m map[string]T) []string {
	a := make([]string, 0, len(m))
	for k := range m {
		a = append(a, k)
	}
	sort.Strings(a)
	return a
}
func contains(items []string, s string) bool {
	for _, item := range items {
		if item == s {
			return true
		}
	}
	return false
}

var skillName = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var variants = map[string]string{"CODEX.md": "codex", "CLAUDE.md": "claude", "COPILOT.md": "copilot", "CURSOR.md": "cursor", "WINDSURF.md": "windsurf", "CLINE.md": "cline", "ANTIGRAVITY.md": "antigravity"}

func loadContent(root string, opts Options, previous Manifest) (Content, error) {
	c := Content{Locks: map[string]HerdLock{}}
	merged := map[string]Source{}
	declarations := map[string]SkillDeclaration{}
	overlays := map[string]map[string]Document{}
	herdRoot := filepath.Join(root, ".promptherder/herds")
	entries, err := os.ReadDir(herdRoot)
	if err != nil && !os.IsNotExist(err) {
		return c, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		base := filepath.Join(herdRoot, e.Name())
		metaBytes, err := os.ReadFile(filepath.Join(base, "herd.json"))
		if err != nil {
			return c, err
		}
		var meta struct {
			Name    string                      `json:"name"`
			Version string                      `json:"version"`
			Skills  map[string]SkillDeclaration `json:"skills"`
		}
		if err := json.Unmarshal(metaBytes, &meta); err != nil {
			return c, err
		}
		all, err := readTree(base, ".promptherder/herds/"+e.Name())
		if err != nil {
			return c, err
		}
		for id, declaration := range meta.Skills {
			if _, exists := declarations[id]; exists {
				return c, fmt.Errorf("duplicate declaration for skill %s", id)
			}
			declarations[id] = declaration
			overlays[id] = map[string]Document{}
			for host, path := range declaration.Overlays {
				if _, ok := Capabilities[host]; !ok {
					return c, fmt.Errorf("unknown overlay host %q", host)
				}
				if !strings.HasPrefix(path, "overlays/") {
					return c, fmt.Errorf("overlay %s must be under overlays/", path)
				}
				source, ok := all[path]
				if !ok {
					return c, fmt.Errorf("missing overlay %s for %s", path, id)
				}
				doc, err := parse(source)
				if err != nil {
					return c, err
				}
				overlays[id][host] = doc
			}
		}
		var sum bytes.Buffer
		for _, k := range keys(all) {
			if k == ".source.json" {
				continue
			}
			if k != "herd.json" && !strings.HasPrefix(k, "rules/") && !strings.HasPrefix(k, "skills/") && !strings.HasPrefix(k, "workflows/") && !strings.HasPrefix(k, "overlays/") {
				continue
			}
			src := all[k]
			fmt.Fprintf(&sum, "%s\x00%o\x00%s\x00", k, src.Mode, digest(src.Data))
			if k == "herd.json" {
				continue
			}
			if old, ok := merged[k]; ok {
				return c, fmt.Errorf("source conflict: %s and %s; rename one source", old.Origin, src.Origin)
			}
			merged[k] = src
		}
		lock := HerdLock{Version: meta.Version, Digest: digest(sum.Bytes())}
		if src, ok := all[".source.json"]; ok {
			if err := json.Unmarshal(src.Data, &lock); err != nil {
				return c, err
			}
			lock.Version = meta.Version
			lock.Digest = digest(sum.Bytes())
		}
		c.Locks[e.Name()] = lock
	}
	for _, localRoot := range []string{".promptherder/agents", ".promptherder/agent"} {
		local, err := readTree(filepath.Join(root, localRoot), localRoot)
		if err != nil {
			return c, err
		}
		for _, k := range keys(local) {
			src := local[k]
			if old, ok := merged[k]; ok {
				// Old releases cached herd content in agent/. Identical copies are not overrides.
				if bytes.Equal(old.Data, src.Data) {
					continue
				}
				if contains(previous.Targets["herds"], src.Origin) && !contains(opts.Overrides, k) {
					return c, fmt.Errorf("edited legacy staging file %s conflicts with %s; declare override %q", src.Origin, old.Origin, k)
				}
				if !contains(opts.Overrides, k) {
					return c, fmt.Errorf("local source %s conflicts with %s; declare override %q", src.Origin, old.Origin, k)
				}
			} else if contains(previous.Targets["herds"], src.Origin) && len(entries) > 0 {
				continue
			}
			merged[k] = src
		}
	}
	if data, err := os.ReadFile(filepath.Join(root, ".promptherder/hard-rules.md")); err == nil {
		if old, exists := merged["rules/00-hard-rules.md"]; exists {
			return c, fmt.Errorf("hard-rules.md conflicts with %s; rename that source", old.Origin)
		}
		merged["rules/00-hard-rules.md"] = Source{Path: "rules/00-hard-rules.md", Origin: ".promptherder/hard-rules.md", Data: data, Mode: 0644}
	} else if !os.IsNotExist(err) {
		return c, err
	}
	skills := map[string]*Skill{}
	for _, k := range keys(merged) {
		src := merged[k]
		if strings.HasPrefix(k, "rules/") && strings.HasSuffix(k, ".md") {
			if len(opts.Include) > 0 {
				matched := false
				for _, pattern := range opts.Include {
					ok, err := doublestar.Match(pattern, strings.TrimPrefix(k, "rules/"))
					if err != nil {
						return c, err
					}
					matched = matched || ok
				}
				if !matched {
					continue
				}
			}
			d, err := parse(src)
			if err != nil {
				return c, err
			}
			r := Rule{Document: d, ID: strings.TrimSuffix(strings.TrimPrefix(k, "rules/"), ".md"), Activation: str(d.Meta, "activation")}
			if id := str(d.Meta, "id"); id != "" {
				r.ID = id
			}
			if r.Activation == "" {
				r.Activation = str(d.Meta, "trigger")
			}
			pv := d.Meta["paths"]
			if pv == nil {
				pv = d.Meta["applyTo"]
			}
			if pv == nil {
				pv = d.Meta["globs"]
			}
			r.Paths, err = list(pv)
			if err != nil {
				return c, fmt.Errorf("%s: %w", src.Origin, err)
			}
			if r.Activation == "" {
				if pv != nil {
					r.Activation = "paths"
				} else {
					r.Activation = "always"
				}
			}
			switch r.Activation {
			case "always_on":
				r.Activation = "always"
			case "glob":
				r.Activation = "paths"
			case "model_decision":
				r.Activation = "relevance"
			}
			if !contains([]string{"always", "paths", "manual", "relevance"}, r.Activation) {
				return c, fmt.Errorf("%s: unknown activation %q", src.Origin, r.Activation)
			}
			if r.Activation == "paths" && len(r.Paths) == 0 {
				return c, fmt.Errorf("%s: path activation requires paths", src.Origin)
			}
			for _, pattern := range r.Paths {
				if _, err := doublestar.Match(pattern, ""); err != nil {
					return c, fmt.Errorf("%s: invalid path pattern %q", src.Origin, pattern)
				}
			}
			r.Required, _ = d.Meta["required"].(bool)
			c.Rules = append(c.Rules, r)
		} else if strings.HasPrefix(k, "skills/") {
			parts := strings.SplitN(strings.TrimPrefix(k, "skills/"), "/", 2)
			if len(parts) != 2 {
				continue
			}
			id := parts[0]
			if skills[id] == nil {
				skills[id] = &Skill{ID: id, Files: map[string]Source{}, Variants: map[string]Document{}}
			}
			s := skills[id]
			name := parts[1]
			if name == "SKILL.md" {
				d, err := parse(src)
				if err != nil {
					return c, err
				}
				s.Document = d
				s.Manual, _ = d.Meta["disable-model-invocation"].(bool)
			} else if host, ok := variants[name]; ok {
				d, err := parse(src)
				if err != nil {
					return c, err
				}
				s.Variants[host] = d
			} else {
				s.Files[name] = src
			}
		} else if strings.HasPrefix(k, "workflows/") && strings.HasSuffix(k, ".md") {
			d, err := parse(src)
			if err != nil {
				return c, err
			}
			id := opts.CommandPrefix + "workflow-" + strings.TrimSuffix(strings.TrimPrefix(k, "workflows/"), ".md")
			if skills[id] != nil {
				return c, fmt.Errorf("workflow collides with skill %s", id)
			}
			d.Meta["name"] = id
			if str(d.Meta, "description") == "" {
				return c, fmt.Errorf("%s: workflow needs a description", src.Origin)
			}
			d.Body = stripTurbo(d.Body)
			skills[id] = &Skill{ID: id, Document: d, Files: map[string]Source{}, Variants: map[string]Document{}, Manual: true, Workflow: true}
		}
	}
	seen := map[string]bool{}
	for _, r := range c.Rules {
		if seen[r.ID] {
			return c, fmt.Errorf("duplicate rule ID %s", r.ID)
		}
		seen[r.ID] = true
	}
	for id, declaration := range declarations {
		s, ok := skills[id]
		if !ok {
			return c, fmt.Errorf("declaration refers to missing skill %s", id)
		}
		s.Requires = declaration.Requires
		switch declaration.Invocation {
		case "":
		case "manual":
			s.Manual = true
			s.Meta["disable-model-invocation"] = true
		case "automatic":
			s.Manual = false
			s.Meta["disable-model-invocation"] = false
		default:
			return c, fmt.Errorf("invalid invocation policy %q for %s", declaration.Invocation, id)
		}
		for host, doc := range overlays[id] {
			if _, exists := s.Variants[host]; exists {
				return c, fmt.Errorf("multiple %s overlays for %s", host, id)
			}
			s.Variants[host] = doc
		}
	}
	for id, s := range skills {
		for _, dependency := range s.Requires {
			if _, exists := skills[dependency]; !exists {
				return c, fmt.Errorf("skill %s requires missing skill %s", id, dependency)
			}
		}
	}
	for _, id := range keys(skills) {
		s := skills[id]
		if !skillName.MatchString(id) || len(id) > 64 {
			return c, fmt.Errorf("invalid skill name %q", id)
		}
		if s.Meta == nil {
			return c, fmt.Errorf("skill %s needs a generic SKILL.md", id)
		}
		if str(s.Meta, "name") != id || str(s.Meta, "description") == "" {
			return c, fmt.Errorf("%s: name must match %q and description is required", s.Source.Origin, id)
		}
		if len(str(s.Meta, "description")) > 1024 {
			return c, fmt.Errorf("%s: description exceeds 1024 bytes", id)
		}
		docs := []Document{s.Document}
		for _, d := range s.Variants {
			docs = append(docs, d)
		}
		for _, d := range docs {
			for _, match := range regexp.MustCompile(`\]\((?:\./)?((?:references|scripts|assets)/[^) #]+)(?:#[^)]*)?\)`).FindAllStringSubmatch(d.Body, -1) {
				if _, exists := s.Files[match[1]]; !exists {
					return c, fmt.Errorf("%s: missing bundled resource %s", d.Source.Origin, match[1])
				}
			}
		}
		c.Skills = append(c.Skills, *s)
	}
	return c, nil
}

func stripTurbo(body string) string {
	var lines []string
	for _, l := range strings.Split(body, "\n") {
		if strings.TrimSpace(l) != "// turbo" && strings.TrimSpace(l) != "// turbo-all" {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}
