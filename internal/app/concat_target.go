package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// buildConcatOutput reads rules + hard-rules, concatenates them into a single
// markdown document. Used by targets that output a single file (claude, codex,
// cursor, windsurf, cline).
//
// Returns the concatenated content, source names list, and any error.
func buildConcatOutput(repoPath, srcDir string, include []string, header string) ([]byte, []string, error) {
	sources, err := readSources(repoPath, srcDir, include)
	if err != nil {
		return nil, nil, err
	}

	// Inject hard-rules as the first source.
	hardRulesPath := filepath.Join(repoPath, filepath.FromSlash(hardRulesFile))
	if data, readErr := os.ReadFile(hardRulesPath); readErr == nil {
		_, body := parseFrontmatter(data)
		hardRule := sourceFile{
			Path:    hardRulesPath,
			Name:    "hard-rules",
			ApplyTo: "",
			Body:    body,
		}
		sources = append([]sourceFile{hardRule}, sources...)
	}

	if len(sources) == 0 {
		return nil, nil, nil
	}

	// Collect only repo-wide rules (no applyTo).
	var parts [][]byte
	var names []string
	for _, s := range sources {
		if s.ApplyTo == "" {
			parts = append(parts, s.Body)
			names = append(names, s.Name)
		}
	}

	if len(parts) == 0 {
		return nil, nil, nil
	}

	content := concatWithHeader(header, parts)
	return content, names, nil
}

// buildConcatSkillsSection reads all skills and returns a concatenated summary
// that can be appended to a single-file target.
func buildConcatSkillsSection(repoPath, variantName string) ([]byte, []string, error) {
	skillsRoot := filepath.Join(repoPath, filepath.FromSlash(skillSourceDir))

	if _, err := os.Stat(skillsRoot); os.IsNotExist(err) {
		return nil, nil, nil
	}

	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("read skills dir: %w", err)
	}

	var buf bytes.Buffer
	var names []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Prefer target-specific variant, then generic SKILL.md.
		skillFile := ""
		if variantName != "" {
			candidate := filepath.Join(skillsRoot, entry.Name(), strings.ToUpper(variantName)+".md")
			if _, err := os.Stat(candidate); err == nil {
				skillFile = candidate
			}
		}
		if skillFile == "" {
			candidate := filepath.Join(skillsRoot, entry.Name(), "SKILL.md")
			if _, err := os.Stat(candidate); err == nil {
				skillFile = candidate
			}
		}
		if skillFile == "" {
			continue
		}

		data, err := os.ReadFile(skillFile)
		if err != nil {
			return nil, nil, fmt.Errorf("read skill %s: %w", entry.Name(), err)
		}

		_, body := parseFrontmatter(data)
		buf.WriteString(fmt.Sprintf("\n## Skill: %s\n\n", entry.Name()))
		buf.Write(bytes.TrimSpace(body))
		buf.WriteByte('\n')
		names = append(names, entry.Name())
	}

	if buf.Len() == 0 {
		return nil, nil, nil
	}
	return buf.Bytes(), names, nil
}

// buildConcatWorkflowsSection reads all workflows and returns concatenated content.
func buildConcatWorkflowsSection(repoPath string) ([]byte, []string, error) {
	wfRoot := filepath.Join(repoPath, filepath.FromSlash(workflowSourceDir))

	if _, err := os.Stat(wfRoot); os.IsNotExist(err) {
		return nil, nil, nil
	}

	entries, err := os.ReadDir(wfRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("read workflows dir: %w", err)
	}

	var buf bytes.Buffer
	var names []string

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(wfRoot, entry.Name()))
		if err != nil {
			return nil, nil, fmt.Errorf("read workflow %s: %w", entry.Name(), err)
		}

		_, body := parseFrontmatter(data)
		body = stripAntigravityAnnotations(body)
		stem := strings.TrimSuffix(entry.Name(), ".md")

		buf.WriteString(fmt.Sprintf("\n## Workflow: %s\n\n", stem))
		buf.Write(bytes.TrimSpace(body))
		buf.WriteByte('\n')
		names = append(names, stem)
	}

	if buf.Len() == 0 {
		return nil, nil, nil
	}
	return buf.Bytes(), names, nil
}

// buildFullConcatTarget builds a complete single-file target output including
// rules, skills, and workflows.
func buildFullConcatTarget(repoPath, srcDir string, include []string, header, variantName string) ([]byte, []string, error) {
	content, names, err := buildConcatOutput(repoPath, srcDir, include, header)
	if err != nil {
		return nil, nil, err
	}

	var buf bytes.Buffer
	if content != nil {
		buf.Write(content)
	}

	// Append skills.
	skillContent, skillNames, err := buildConcatSkillsSection(repoPath, variantName)
	if err != nil {
		return nil, nil, err
	}
	if skillContent != nil {
		buf.Write(skillContent)
		names = append(names, skillNames...)
	}

	// Append workflows.
	wfContent, wfNames, err := buildConcatWorkflowsSection(repoPath)
	if err != nil {
		return nil, nil, err
	}
	if wfContent != nil {
		buf.Write(wfContent)
		names = append(names, wfNames...)
	}

	if buf.Len() == 0 {
		return nil, nil, nil
	}

	return buf.Bytes(), names, nil
}
