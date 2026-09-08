package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/shermanhuman/promptherder/internal/app"
)

// configureTargets replaces the selection. No target is preselected, including
// on an existing installation. Explicit arguments also work without a terminal.
func configureTargets(repo string, settings app.Settings, names []string, dryRun, interactive bool, in io.Reader, out io.Writer, scopes ...string) (app.Settings, error) {
	userScope := len(scopes) > 0 && scopes[0] == "user"
	if len(names) == 0 {
		if dryRun || !interactive {
			return settings, fmt.Errorf("select targets first: run 'promptherder install' interactively or 'promptherder install codex claude' (use 'install none' for no targets)")
		}
		fmt.Fprintln(out, "Select targets for this installation. Nothing is preselected.")
		fmt.Fprintln(out, "  codex         OpenAI Codex")
		fmt.Fprintln(out, "  claude        Claude Code")
		if !userScope {
			fmt.Fprintln(out, "  copilot       GitHub Copilot")
			fmt.Fprintln(out, "  antigravity   Google Antigravity")
			fmt.Fprintln(out, "  cursor        Cursor")
			fmt.Fprintln(out, "  windsurf      Windsurf/Cascade")
			fmt.Fprintln(out, "  cline         Cline")
		}
		fmt.Fprint(out, "Enter names separated by spaces or commas, or 'none': ")
		scanner := bufio.NewScanner(in)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return settings, fmt.Errorf("read target selection: %w", err)
			}
			return settings, fmt.Errorf("target selection cancelled; no settings changed")
		}
		names = strings.FieldsFunc(scanner.Text(), func(r rune) bool { return r == ',' || r == ' ' || r == '\t' })
		if len(names) == 0 {
			return settings, fmt.Errorf("no selection entered; use 'none' to explicitly disable all targets")
		}
	}
	if userScope {
		for _, name := range names {
			if name != "codex" && name != "claude" && name != "none" {
				return settings, fmt.Errorf("user scope supports codex and claude only")
			}
		}
	}
	selected := []string{}
	if len(names) != 1 || names[0] != "none" {
		var err error
		selected, err = changeTargets(nil, "add", names)
		if err != nil {
			return settings, err
		}
	}
	settings.Agents = selected
	if err := persistTargets(repo, settings, dryRun, out); err != nil {
		return settings, err
	}
	return settings, nil
}

func changeTargets(current []string, action string, names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("usage: promptherder target %s <name> [name...]", action)
	}
	requested := make(map[string]bool)
	for _, name := range names {
		if !app.IsValidAgent(name) {
			return nil, fmt.Errorf("unknown target %q; available: %s", name, strings.Join(app.AllAgents, ", "))
		}
		requested[name] = true
	}
	result := []string{}
	seen := make(map[string]bool)
	for _, name := range current {
		if (action != "remove" || !requested[name]) && !seen[name] {
			result = append(result, name)
			seen[name] = true
		}
	}
	if action == "add" {
		for _, name := range names {
			if !seen[name] {
				result = append(result, name)
				seen[name] = true
			}
		}
	}
	return result, nil
}

func persistTargets(repo string, settings app.Settings, dryRun bool, out io.Writer) error {
	verb := "Enabled targets"
	if dryRun {
		verb = "Would enable targets"
	} else if err := app.SaveSettings(repo, settings); err != nil {
		return err
	}
	selection := strings.Join(settings.Agents, ", ")
	if selection == "" {
		selection = "none"
	}
	fmt.Fprintf(out, "%s: %s\n", verb, selection)
	if !dryRun {
		fmt.Fprintln(out, "Run 'promptherder' to sync. Previously generated files for removed targets are cleaned up on sync.")
	}
	return nil
}
