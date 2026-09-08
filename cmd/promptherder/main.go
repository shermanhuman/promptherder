package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/shermanhuman/promptherder/internal/app"
	"github.com/shermanhuman/promptherder/internal/compiler"
)

// Set via ldflags at build time (goreleaser).
// Falls back to Go module info for `go install`.
var (
	Version   = ""
	Commit    = ""
	BuildDate = ""
)

func init() {
	if Version != "" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		Version = "dev"
		return
	}
	Version = info.Main.Version
	if Version == "" || Version == "(devel)" {
		Version = "dev"
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				Commit = s.Value[:7]
			}
		case "vcs.time":
			BuildDate = s.Value
		}
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Extract subcommand (first non-flag argument).
	subcommand, args := extractSubcommand(os.Args[1:])

	// Separate flags from positional args so flags work regardless of position
	// (e.g. "pull https://url -dry-run" works the same as "pull -dry-run https://url").
	flagArgs, positionalArgs := splitFlagsAndArgs(args)

	// Parse flags.
	fs := flag.NewFlagSet("promptherder", flag.ExitOnError)
	var (
		scope                                         string
		includeCSV                                    string
		dryRun                                        bool
		verbose                                       bool
		showVersion                                   bool
		targetsCSV                                    string
		strict, adopt, updateLock, jsonOutput, locked bool
	)
	fs.StringVar(&includeCSV, "include", "", "Comma-separated glob patterns to include (default: all)")
	fs.BoolVar(&dryRun, "dry-run", false, "Show actions without writing files")
	fs.BoolVar(&verbose, "v", false, "Verbose logging")
	fs.BoolVar(&showVersion, "version", false, "Print version and exit")
	fs.StringVar(&scope, "scope", "repository", "Installation scope: repository or user (Codex/Claude)")
	fs.StringVar(&targetsCSV, "targets", "", "Targets for plan/check/doctor/explain (comma-separated)")
	fs.BoolVar(&locked, "locked", false, "Pull the exact revision from lock.json")
	fs.BoolVar(&strict, "strict", false, "Treat compatibility warnings as errors")
	fs.BoolVar(&adopt, "adopt", false, "Adopt legacy output; preserve unmanaged baseline instructions")
	fs.BoolVar(&updateLock, "update-lock", false, "Accept reviewed changes to herd content")
	fs.BoolVar(&jsonOutput, "json", false, "Print the build plan as JSON")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `promptherder — sync agent configuration across AI coding tools

Usage:
  promptherder [flags]              Sync all enabled targets
  promptherder <target> [flags]     Sync a single target
  promptherder install [name...]   Select targets (use none to disable all)
  promptherder pull <name|git-url>  Install a herd (by alias or URL)
  promptherder list                 Show available herd aliases
  promptherder target list          Show available/enabled targets
  promptherder target add <name>    Enable targets
  promptherder target remove <name> Disable targets
  promptherder agent ...            Alias for target ...
  promptherder plan                 Preview native output and compatibility diagnostics
  promptherder check                Validate sources and planned output (no writes)
  promptherder doctor               Inspect planned and installed instruction discovery
  promptherder explain <skill-id>   Trace a skill or rule to its outputs
  promptherder recover              Roll back an interrupted sync

Targets:
  codex         AGENTS.md + .agents/skills/
  claude        CLAUDE.md + .claude/rules/ + .claude/skills/
  copilot       AGENTS.md + .github/instructions/ + native skills
  antigravity   .agents/rules/ + .agents/skills/
  cursor        AGENTS.md + .cursor/rules/*.mdc + native skills
  windsurf      .windsurf/rules/ + .windsurf/skills/
  cline         AGENTS.md + .clinerules/ + .cline/skills/

Flags:
  -dry-run     Show actions without writing files
  -include     Comma-separated glob patterns to include (default: all)
  -v           Verbose logging (structured output to stderr)
  -version     Print version and exit
  -scope       repository (default) or user (Codex/Claude)
  -targets     Comma-separated targets for read-only plan/diagnostic commands
  -strict      Treat compatibility warnings as errors
  -adopt       Adopt legacy files or merge an unmanaged baseline (preview with plan)
  -locked      Pull the exact revision and verify its content digest
  -update-lock Accept reviewed herd changes
  -json        Print a machine-readable plan

Examples:
  promptherder                                Sync all enabled targets
  promptherder install codex claude           Select Codex and Claude Code
  promptherder target add cursor              Enable Cursor
  promptherder target remove cursor           Disable Cursor
  promptherder pull compound-v                Pull by alias
  promptherder copilot -dry-run               Preview copilot sync
`)
	}

	_ = fs.Parse(flagArgs)

	// Merge any remaining flag parse args with positional args.
	allPositional := append(fs.Args(), positionalArgs...)

	if showVersion {
		if Commit != "" {
			fmt.Printf("promptherder %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
		} else {
			fmt.Printf("promptherder %s\n", Version)
		}
		return
	}

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	var logger *slog.Logger
	if verbose {
		// Verbose: structured slog output to stderr (for debugging).
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
	} else {
		// Normal: pretty output to stdout.
		logger = slog.New(app.NewUIHandler(os.Stderr, level))
	}

	// Always use current working directory as repo root.
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("failed to get working directory", "error", err)
		os.Exit(1)
	}

	if scope != "repository" && scope != "user" {
		logger.Error("scope must be repository or user")
		os.Exit(2)
	}
	if scope == "user" {
		cwd, err = os.UserHomeDir()
		if err != nil {
			logger.Error("cannot resolve user home", "error", err)
			os.Exit(1)
		}
		if custom := os.Getenv("CODEX_HOME"); custom != "" && custom != cwd+"/.codex" {
			logger.Error("user scope requires the default CODEX_HOME; custom layouts are not supported")
			os.Exit(2)
		}
	}
	cfg := app.Config{
		RepoPath: cwd,
		Include:  parseIncludePatterns(includeCSV),
		DryRun:   dryRun,
		Logger:   logger,
	}

	// Load settings for agent filtering.
	settings, settingsErr := app.LoadSettings(cwd)
	if settingsErr != nil {
		logger.Error("failed to load settings", "error", settingsErr)
		os.Exit(1)
	}

	var runErr error
	build := func(targets []string, write bool, explain string) error {
		prefix := ""
		if settings.CommandPrefixEnabled {
			prefix = settings.CommandPrefix
		}
		p, err := compiler.Build(ctx, cwd, compiler.Options{Scope: scope, Targets: targets, Include: cfg.Include, Overrides: settings.Overrides, Skills: settings.Skills, CommandPrefix: prefix, Strict: strict, Adopt: adopt, UpdateLock: updateLock, ProjectDocMaxBytes: settings.ProjectDocMaxBytes})
		if err != nil {
			return err
		}
		if subcommand == "doctor" {
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			if err := p.Doctor(home, os.Getenv("CODEX_HOME")); err != nil {
				return err
			}
		}
		if explain != "" && !p.Explains(explain) {
			return fmt.Errorf("unknown or unselected content ID %q", explain)
		}
		if jsonOutput {
			data, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		} else {
			p.Print(os.Stdout, explain)
		}
		if p.HasErrors() {
			return fmt.Errorf("resolve the reported plan errors before syncing")
		}
		if write && !dryRun {
			return p.Apply(ctx)
		}
		return nil
	}
	if locked && subcommand != "pull" {
		logger.Error("--locked is only supported for pull")
		os.Exit(2)
	}
	if targetsCSV != "" && subcommand != "plan" && subcommand != "check" && subcommand != "doctor" && subcommand != "explain" {
		logger.Error("--targets is only supported for plan/check/doctor/explain; use install to change enabled targets")
		os.Exit(2)
	}
	switch subcommand {
	case "":
		if len(allPositional) != 0 {
			runErr = fmt.Errorf("unexpected arguments: %s", strings.Join(allPositional, " "))
			break
		}
		if !settings.TargetsConfigured() {
			settings, runErr = configureTargets(cwd, settings, nil, dryRun, stdinInteractive(), os.Stdin, os.Stdout, scope)
			if runErr != nil {
				break
			}
		}
		runErr = build(settings.EnabledAgents(), true, "")

	case "copilot", "antigravity", "claude", "codex", "cursor", "windsurf", "cline":
		var targets []string
		targets, runErr = compiler.InstalledTargets(cwd)
		if runErr == nil {
			targets, runErr = changeTargets(targets, "add", []string{subcommand})
		}
		if runErr == nil {
			runErr = build(targets, true, "")
		}

	case "plan", "check", "doctor", "explain":
		targets := settings.EnabledAgents()
		if targetsCSV != "" {
			targets = parseIncludePatterns(targetsCSV)
		}
		id := ""
		if subcommand == "explain" {
			if len(allPositional) != 1 {
				runErr = fmt.Errorf("usage: promptherder explain <skill-or-rule-id>")
				break
			}
			id = allPositional[0]
		}
		runErr = build(targets, false, id)

	case "recover":
		if dryRun {
			runErr = fmt.Errorf("recover does not support --dry-run; use plan to inspect pending recovery")
			break
		}
		runErr = compiler.Recover(cwd)

	case "install":
		_, runErr = configureTargets(cwd, settings, allPositional, dryRun, stdinInteractive(), os.Stdin, os.Stdout, scope)

	case "pull":
		var gitURL string
		if len(allPositional) > 0 {
			gitURL = allPositional[0]
		}
		if gitURL == "" {
			logger.Error("missing URL argument")
			fmt.Fprintf(os.Stderr, "Usage: promptherder pull <name|git-url>\n")
			os.Exit(2)
		}
		if !settings.TargetsConfigured() {
			settings, runErr = configureTargets(cwd, settings, nil, dryRun, stdinInteractive(), os.Stdin, os.Stdout, scope)
			if runErr != nil {
				break
			}
		}
		runErr = app.ResolveAndPull(ctx, gitURL, app.PullConfig{Locked: locked,
			RepoPath: cwd,
			DryRun:   dryRun,
			Logger:   logger,
		})

	case "list":
		aliases, source, aliasErr := app.LoadAliases()
		if aliasErr != nil {
			logger.Error("failed to load aliases", "error", aliasErr)
			os.Exit(1)
		}

		// Auto-scaffold config on first list.
		if !dryRun {
			if path := app.EnsureAliasesConfig(source, nil); path != "" {
				fmt.Fprintf(os.Stderr, "Created %s\n\n", path)
			}
		}

		fmt.Println("\nAvailable herds:")
		fmt.Println()
		names := app.SortedAliasNames(aliases)
		for _, name := range names {
			a := aliases[name]
			fmt.Printf("  %-14s %s\n", name, a.Description)
		}
		fmt.Println()
		if path, err := app.AliasesConfigPath(); err == nil {
			fmt.Printf("Config: %s\n", path)
		}
		fmt.Println("Pull:   promptherder pull <name>")

	case "agent", "target":
		if len(allPositional) == 0 {
			fmt.Fprintf(os.Stderr, "Usage: promptherder target <list|add|remove> [name...]\n")
			os.Exit(2)
		}
		agentCmd := allPositional[0]
		agentArgs := allPositional[1:]

		switch agentCmd {
		case "list":
			fmt.Println("\nAvailable targets:")
			fmt.Println()
			enabled := settings.EnabledAgents()
			enabledSet := make(map[string]bool, len(enabled))
			for _, a := range enabled {
				enabledSet[a] = true
			}
			for _, name := range app.AllAgents {
				if scope == "user" && name != "codex" && name != "claude" {
					continue
				}
				check := " "
				suffix := ""
				if enabledSet[name] {
					check = "✓"
					suffix = " (enabled)"
				}
				fmt.Printf("  %s %-14s%s\n", check, name, suffix)
			}
			fmt.Println()
			if !settings.TargetsConfigured() {
				fmt.Println("No targets selected. Run 'promptherder install' to set up this repository.")
			} else if len(enabled) == 0 {
				fmt.Println("No targets enabled.")
			}
			fmt.Println("Enable:  promptherder target add <name>")
			fmt.Println("Disable: promptherder target remove <name>")

		case "add", "remove":
			var selected []string
			selected, runErr = changeTargets(settings.EnabledAgents(), agentCmd, agentArgs)
			if runErr == nil && scope == "user" {
				for _, name := range selected {
					if name != "codex" && name != "claude" {
						runErr = fmt.Errorf("user scope supports codex and claude only")
					}
				}
			}
			if runErr == nil {
				settings.Agents = selected
				runErr = persistTargets(cwd, settings, dryRun, os.Stdout)
			}

		default:
			fmt.Fprintf(os.Stderr, "Unknown agent command: %s\nUsage: promptherder target <list|add|remove> [name...]\n", agentCmd)
			os.Exit(2)
		}

	default:
		logger.Error("unknown subcommand", "subcommand", subcommand)
		fmt.Fprintf(os.Stderr, "Usage: promptherder [<target>|install|pull|list|target] [flags]\n")
		os.Exit(2)
	}

	if runErr != nil {
		if errors.Is(runErr, app.ErrValidation) {
			logger.Error("validation error", "error", runErr)
			os.Exit(2)
		}
		logger.Error("failed", "error", runErr)
		os.Exit(1)
	}
}

// extractSubcommand pulls the first non-flag argument from args.
// Returns the subcommand (or "" if none) and remaining args for flag parsing.
func extractSubcommand(args []string) (string, []string) {
	known := map[string]bool{
		"copilot":     true,
		"antigravity": true,
		"claude":      true,
		"codex":       true,
		"cursor":      true,
		"windsurf":    true,
		"cline":       true,
		"pull":        true,
		"list":        true,
		"agent":       true,
		"target":      true,
		"install":     true,
		"plan":        true, "check": true, "doctor": true, "explain": true, "recover": true,
	}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			name := strings.TrimLeft(arg, "-")
			if name == "include" || name == "targets" || name == "scope" {
				i++
			}
			continue
		}
		if known[arg] {
			remaining := append([]string{}, args[:i]...)
			remaining = append(remaining, args[i+1:]...)
			return arg, remaining
		}
		break
	}
	return "", args
}

func parseIncludePatterns(csv string) []string {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	parts := strings.Split(csv, ",")
	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		patterns = append(patterns, p)
	}
	return patterns
}

// splitFlagsAndArgs separates flag arguments (starting with -) from positional
// arguments. This allows flags to appear before or after positional args
// (e.g. "pull https://url -dry-run" works the same as "pull -dry-run https://url").
func splitFlagsAndArgs(args []string) (flags, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			// If it's a flag that takes a value (contains = or next arg is value),
			// consume the next arg too if it doesn't start with -.
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Check if this flag looks like it takes a value (e.g. -include).
				name := strings.TrimLeft(a, "-")
				if name == "include" || name == "targets" || name == "scope" { // known value flags
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			positional = append(positional, a)
		}
	}
	return flags, positional
}

// Setup is offered only on an interactive first run; automation must select
// targets explicitly rather than inheriting an implicit default.
func stdinInteractive() bool {
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	// /dev/null is also a character device, but cannot answer a setup prompt.
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(info, null)
}
