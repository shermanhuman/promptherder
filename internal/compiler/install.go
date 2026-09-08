package compiler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shermanhuman/promptherder/internal/files"
)

type Record struct {
	Hash    string      `json:"hash"`
	Mode    fs.FileMode `json:"mode"`
	ID      string      `json:"id"`
	Kind    string      `json:"kind"`
	Origins []string    `json:"origins"`
	Targets []string    `json:"targets"`
	Prefix  string      `json:"preserved_prefix,omitempty"`
}
type Manifest struct {
	Scope     string              `json:"scope,omitempty"`
	Version   int                 `json:"version"`
	Targets   map[string][]string `json:"targets"`
	Artifacts map[string]Record   `json:"artifacts,omitempty"`
	Generated []string            `json:"generated,omitempty"`
	Files     []string            `json:"files,omitempty"`
}
type Snapshot struct {
	Exists bool        `json:"exists"`
	Data   []byte      `json:"data,omitempty"`
	Mode   fs.FileMode `json:"mode,omitempty"`
}
type journal struct {
	Before map[string]Snapshot `json:"before"`
	After  map[string]Snapshot `json:"after"`
}

func InstalledTargets(root string) ([]string, error) {
	m, err := readManifest(root)
	if err != nil {
		return nil, err
	}
	var result []string
	for _, t := range keys(m.Targets) {
		if _, ok := Capabilities[t]; ok {
			result = append(result, t)
		}
	}
	return result, nil
}

func readManifest(root string) (Manifest, error) {
	m := Manifest{Targets: map[string][]string{}, Artifacts: map[string]Record{}}
	data, err := os.ReadFile(filepath.Join(root, ".promptherder/manifest.json"))
	if os.IsNotExist(err) {
		return m, nil
	}
	if err != nil {
		return m, err
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("invalid manifest; refusing to discard ownership: %w", err)
	}
	if m.Version > 3 {
		return m, fmt.Errorf("manifest version %d is newer than this compiler", m.Version)
	}
	if m.Targets == nil {
		m.Targets = map[string][]string{}
	}
	if m.Artifacts == nil {
		m.Artifacts = map[string]Record{}
	}
	return m, nil
}

// safePath checks every existing ancestor as well as the leaf, preventing an
// instruction directory symlink from redirecting writes outside the repository.
func safePath(root, rel string) (string, error) {
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, "\\") || filepath.ToSlash(filepath.Clean(rel)) != rel || strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("unsafe artifact path %q", rel)
	}
	path := root
	for _, part := range strings.Split(rel, "/") {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("refusing symlink path %s", path)
		}
	}
	return path, nil
}

func snapshot(root, rel string) (Snapshot, error) {
	path, err := safePath(root, rel)
	if err != nil {
		return Snapshot{}, err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return Snapshot{}, nil
	}
	if err != nil {
		return Snapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return Snapshot{}, fmt.Errorf("not a regular file: %s", rel)
	}
	data, err := os.ReadFile(path)
	return Snapshot{true, data, info.Mode().Perm()}, err
}
func same(a, b Snapshot) bool {
	return a.Exists == b.Exists && a.Mode == b.Mode && bytes.Equal(a.Data, b.Data)
}

func (p *Plan) inspectExisting() error {
	if _, err := os.Stat(filepath.Join(p.Root, ".promptherder/apply-journal.json")); err == nil {
		p.note("error", "interrupted-apply", ".promptherder/apply-journal.json", "run 'promptherder recover' before syncing")
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	wanted := map[string]bool{}
	for i := range p.Artifacts {
		a := &p.Artifacts[i]
		wanted[a.Path] = true
		snap, err := snapshot(p.Root, a.Path)
		if err != nil {
			return err
		}
		p.Before[a.Path] = snap
		old, owned := p.Previous.Artifacts[a.Path]
		if old.Prefix != "" {
			a.Content = append([]byte(old.Prefix), a.Content...)
			a.Hash = digest(a.Content)
			a.Bytes = len(a.Content)
		}
		if !snap.Exists {
			continue
		}
		if owned && old.Hash == digest(snap.Data) && old.Mode == snap.Mode {
			continue
		}
		if a.Hash == digest(snap.Data) && a.Mode == snap.Mode {
			continue
		}
		if !owned && legacyOutput(snap.Data) {
			if !p.opts.Adopt {
				p.note("error", "legacy-adoption", a.Path, "legacy output has no recorded hash; review 'plan --adopt', then sync with --adopt")
				continue
			}
			p.note("info", "legacy-adoption", a.Path, "replace legacy generated content with native output")
			continue
		}
		if !owned && p.opts.Adopt && (a.Kind == "baseline" || a.Kind == "bridge") {
			prefix := string(snap.Data) + "\n\n"
			a.Content = append([]byte(prefix), a.Content...)
			a.Hash = digest(a.Content)
			a.Bytes = len(a.Content)
			old.Prefix = prefix
			p.Previous.Artifacts[a.Path] = old
			p.note("info", "adoption-preview", a.Path, "preserve existing instructions and append:\n"+string(a.Content[len(prefix):]))
			continue
		}
		p.note("error", "local-conflict", a.Path, "existing content differs from the last installed hash; preserve or reconcile your changes before syncing")
	}
	oldPaths := map[string]bool{}
	for path := range p.Previous.Artifacts {
		oldPaths[path] = true
	}
	for _, paths := range p.Previous.Targets {
		for _, path := range paths {
			oldPaths[path] = true
		}
	}
	for _, path := range p.Previous.Files {
		oldPaths[path] = true
	}
	for _, path := range keys(oldPaths) {
		if wanted[path] {
			continue
		}
		// Legacy staging remains source material; never delete it during migration.
		if strings.HasPrefix(path, ".promptherder/agent/") || strings.HasPrefix(path, ".promptherder/agents/") {
			continue
		}
		if contains(p.Previous.Generated, filepath.Base(path)) {
			continue
		}
		snap, err := snapshot(p.Root, path)
		if err != nil {
			return err
		}
		p.Before[path] = snap
		if !snap.Exists {
			continue
		}
		old, ok := p.Previous.Artifacts[path]
		if ok && old.Hash == digest(snap.Data) && old.Mode == snap.Mode {
			if old.Prefix != "" {
				p.add(Artifact{Path: path, Content: []byte(old.Prefix), Mode: old.Mode, Kind: "preserved", ID: old.ID})
			} else {
				p.Delete = append(p.Delete, path)
			}
		} else if !ok && p.opts.Adopt && legacyOutput(snap.Data) {
			p.Delete = append(p.Delete, path)
		} else {
			p.note("error", "stale-conflict", path, "stale output has local edits or unverified legacy ownership; reconcile before deletion")
		}
	}
	// Inspect unowned skill definitions in every documented discovery root.
	for _, dir := range []string{".agents/skills", ".claude/skills", ".github/skills", ".cursor/skills", ".cline/skills", ".clinerules/skills", ".windsurf/skills"} {
		full, err := safePath(p.Root, dir)
		if err != nil {
			return err
		}
		err = filepath.WalkDir(full, func(path string, e fs.DirEntry, err error) error {
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil {
				return err
			}
			if e.Type()&os.ModeSymlink != 0 {
				p.note("warning", "external-symlink", path, "discovery through this symlink is not inspected")
				return nil
			}
			if e.IsDir() || e.Name() != "SKILL.md" {
				return nil
			}
			rel, _ := filepath.Rel(p.Root, path)
			rel = filepath.ToSlash(rel)
			if wanted[rel] || contains(p.Delete, rel) {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			d, err := parse(Source{Origin: rel, Data: data})
			if err != nil {
				return err
			}
			for _, host := range p.Targets {
				if !visible(host, rel) {
					continue
				}
				for _, a := range p.Artifacts {
					if a.Kind == "skill" && a.ID == str(d.Meta, "name") && visible(host, a.Path) {
						p.note("error", "unmanaged-skill-shadow", rel, host+" can discover this skill as well as "+a.Path)
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Plan) Apply(ctx context.Context) error {
	if p.HasErrors() {
		return fmt.Errorf("plan has errors; no files written")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	release, err := acquireLock(p.Root, false)
	if err != nil {
		return err
	}
	defer release()
	for rel, before := range p.Before {
		current, err := snapshot(p.Root, rel)
		if err != nil {
			return err
		}
		if !same(before, current) {
			return fmt.Errorf("%s changed after planning; rebuild the plan", rel)
		}
	}
	m := Manifest{Scope: p.Scope, Version: 3, Targets: map[string][]string{}, Artifacts: map[string]Record{}, Generated: p.Previous.Generated}
	// Keep migration provenance so old cached herd sources stay distinguishable.
	if paths := p.Previous.Targets["herds"]; len(paths) > 0 {
		m.Targets["herds"] = paths
	}
	j := journal{Before: map[string]Snapshot{}, After: map[string]Snapshot{}}
	for _, a := range p.Artifacts {
		m.Artifacts[a.Path] = Record{a.Hash, a.Mode, a.ID, a.Kind, a.Origins, a.Targets, p.Previous.Artifacts[a.Path].Prefix}
		for _, t := range a.Targets {
			m.Targets[t] = append(m.Targets[t], a.Path)
		}
		j.After[a.Path] = Snapshot{true, a.Content, a.Mode}
	}
	for _, t := range p.Targets {
		if m.Targets[t] == nil {
			m.Targets[t] = []string{}
		}
	}
	for _, rel := range p.Delete {
		j.After[rel] = Snapshot{}
	}
	manifestData, _ := json.MarshalIndent(m, "", "  ")
	lockData, _ := json.MarshalIndent(p.Locks, "", "  ")
	j.After[".promptherder/manifest.json"] = Snapshot{true, append(manifestData, '\n'), 0644}
	j.After[".promptherder/lock.json"] = Snapshot{true, append(lockData, '\n'), 0644}
	for rel, after := range j.After {
		before, err := snapshot(p.Root, rel)
		if err != nil {
			return err
		}
		if same(before, after) {
			delete(j.After, rel)
			continue
		}
		j.Before[rel] = before
	}
	if len(j.After) == 0 {
		return nil
	}
	journalPath, err := safePath(p.Root, ".promptherder/apply-journal.json")
	if err != nil {
		return err
	}
	if _, err := os.Stat(journalPath); err == nil {
		return fmt.Errorf("pending recovery journal; run promptherder recover")
	}
	data, _ := json.MarshalIndent(j, "", "  ")
	if err := (files.AtomicWriter{Path: journalPath, Perm: 0600}).Write(data); err != nil {
		return err
	}
	// Commit metadata last. If interrupted, the journal contains both sides and
	// recovery refuses to overwrite any subsequent user edits.
	order := keys(j.After)
	for _, rel := range order {
		if rel == ".promptherder/manifest.json" || rel == ".promptherder/lock.json" {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := restore(p.Root, rel, j.After[rel]); err != nil {
			return err
		}
	}
	for _, rel := range []string{".promptherder/lock.json", ".promptherder/manifest.json"} {
		if s, ok := j.After[rel]; ok {
			if err := restore(p.Root, rel, s); err != nil {
				return err
			}
		}
	}
	return os.Remove(journalPath)
}

func restore(root, rel string, s Snapshot) error {
	path, err := safePath(root, rel)
	if err != nil {
		return err
	}
	if !s.Exists {
		err = os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return (files.AtomicWriter{Path: path, Perm: s.Mode}).Write(s.Data)
}

// Recover rolls an interrupted apply back without overwriting edits made since.
func Recover(root string) error {
	release, err := acquireLock(root, true)
	if err != nil {
		return err
	}
	defer release()
	path, err := safePath(root, ".promptherder/apply-journal.json")
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var j journal
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	for rel, before := range j.Before {
		current, err := snapshot(root, rel)
		if err != nil {
			return err
		}
		after, ok := j.After[rel]
		if !ok {
			return fmt.Errorf("invalid journal entry %s", rel)
		}
		if !same(current, before) && !same(current, after) {
			return fmt.Errorf("%s has edits made after interruption; reconcile before recovery", rel)
		}
	}
	for _, rel := range keys(j.Before) {
		if err := restore(root, rel, j.Before[rel]); err != nil {
			return err
		}
	}
	return os.Remove(path)
}

// PID ownership prevents recovery from racing an active apply. Only explicit
// recovery may reclaim a lock whose owner is known to have exited.
func acquireLock(root string, recovery bool) (func(), error) {
	path, err := safePath(root, ".promptherder/sync.lock")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	if recovery {
		data, err := os.ReadFile(path)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
			if err != nil || pid <= 0 {
				return nil, fmt.Errorf("unrecognized sync.lock; verify no sync is running before removing it")
			}
			alive, err := processAlive(pid)
			if err != nil || alive {
				return nil, fmt.Errorf("sync.lock belongs to a live or inaccessible process %d", pid)
			}
			if err := os.Remove(path); err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("sync is locked; if interrupted, run promptherder recover: %w", err)
	}
	_, writeErr := fmt.Fprintf(f, "%d\n", os.Getpid())
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return nil, errors.Join(writeErr, closeErr)
	}
	return func() { _ = os.Remove(path) }, nil
}

func legacyOutput(data []byte) bool {
	d, err := parse(Source{Data: data})
	if err != nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(d.Body), "<!-- Auto-generated by promptherder")
}
