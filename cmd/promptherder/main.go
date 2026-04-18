package main

import (
	"context"
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
		includeCSV  string
		dryRun      bool
		verbose     bool
		showVersion bool
	)
	fs.StringVar(&includeCSV, "include", "", "Comma-separated glob patterns to include (default: all)")
	fs.BoolVar(&dryRun, "dry-run", false, "Show actions without writing files")
	fs.BoolVar(&verbose, "v", false, "Verbose logging")
	fs.BoolVar(&showVersion, "version", false, "Print version and exit")

	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `promptherder — sync agent configuration across AI coding tools

Usage:
  promptherder [flags]              Sync all enabled targets
  promptherder <target> [flags]     Sync a single target
  promptherder pull <name|git-url>  Install a herd (by alias or URL)
  promptherder list                 Show available herd aliases
  promptherder agent list           Show available/enabled targets
  promptherder agent add <name>     Enable a target
  promptherder agent remove <name>  Disable a target

Targets:
  copilot       .github/copilot-instructions.md + .github/prompts/
  antigravity   .agents/ (Gemini CLI)
  claude        CLAUDE.md (Claude Code)
  codex         AGENTS.md (OpenAI Codex)
  cursor        .cursor/rules/promptherder.md
  windsurf      .windsurf/rules/promptherder.md
  cline         .clinerules/promptherder.md

Flags:
  -dry-run     Show actions without writing files
  -include     Comma-separated glob patterns to include (default: all)
  -v           Verbose logging (structured output to stderr)
  -version     Print version and exit

Examples:
  promptherder                                Sync all enabled targets
  promptherder agent add claude cursor        Enable claude and cursor
  promptherder agent remove copilot           Disable copilot
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
		logger = slog.New(app.NewUIHandler(os.Stdout, level))
	}

	// Always use current working directory as repo root.
	cwd, err := os.Getwd()
	if err != nil {
		logger.Error("failed to get working directory", "error", err)
		os.Exit(1)
	}

	cfg := app.Config{
		RepoPath: cwd,
		Include:  parseIncludePatterns(includeCSV),
		DryRun:   dryRun,
		Logger:   logger,
	}

	// Build the full targets registry.
	targetRegistry := map[string]app.Target{
		"copilot":     app.CopilotTarget{Include: cfg.Include},
		"antigravity": app.AntigravityTarget{},
		"claude":      app.NewClaudeTarget(cfg.Include),
		"codex":       app.NewCodexTarget(cfg.Include),
		"cursor":      app.NewCursorTarget(cfg.Include),
		"windsurf":    app.NewWindsurfTarget(cfg.Include),
		"cline":       app.NewClineTarget(cfg.Include),
	}

	// Load settings for agent filtering.
	settings, settingsErr := app.LoadSettings(cwd)
	if settingsErr != nil {
		logger.Warn("failed to load settings, using defaults", "error", settingsErr)
		settings = app.DefaultSettings()
	}

	var runErr error
	switch subcommand {
	case "":
		// Bare promptherder — discover herds, merge, then fan out to enabled targets.
		var enabled []app.Target
		for _, name := range settings.EnabledAgents() {
			if t, ok := targetRegistry[name]; ok {
				enabled = append(enabled, t)
			}
		}
		runErr = app.RunAll(ctx, enabled, cfg)

	case "copilot", "antigravity", "claude", "codex", "cursor", "windsurf", "cline":
		runErr = app.RunTarget(ctx, targetRegistry[subcommand], cfg)

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
		runErr = app.ResolveAndPull(ctx, gitURL, app.PullConfig{
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
		if path := app.EnsureAliasesConfig(source, nil); path != "" {
			fmt.Fprintf(os.Stderr, "Created %s\n\n", path)
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

	case "agent":
		if len(allPositional) == 0 {
			fmt.Fprintf(os.Stderr, "Usage: promptherder agent <list|add|remove> [name...]\n")
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
				t := targetRegistry[name]
				check := " "
				suffix := ""
				if enabledSet[name] {
					check = "✓"
					suffix = " (enabled)"
				}
				_ = t // target exists in registry
				fmt.Printf("  %s %-14s%s\n", check, name, suffix)
			}
			fmt.Println()
			fmt.Println("Enable:  promptherder agent add <name>")
			fmt.Println("Disable: promptherder agent remove <name>")

		case "add":
			if len(agentArgs) == 0 {
				fmt.Fprintf(os.Stderr, "Usage: promptherder agent add <name> [name...]\n")
				os.Exit(2)
			}
			// Validate all names first.
			for _, name := range agentArgs {
				if !app.IsValidAgent(name) {
					fmt.Fprintf(os.Stderr, "Unknown agent: %s\nAvailable: %s\n", name, strings.Join(app.AllAgents, ", "))
					os.Exit(2)
				}
			}
			// Start from current enabled list.
			current := settings.EnabledAgents()
			have := make(map[string]bool, len(current))
			for _, a := range current {
				have[a] = true
			}
			for _, name := range agentArgs {
				if !have[name] {
					current = append(current, name)
					have[name] = true
					fmt.Printf("  ✓ added %s\n", name)
				} else {
					fmt.Printf("  · %s already enabled\n", name)
				}
			}
			settings.Agents = current
			if err := app.SaveSettings(cwd, settings); err != nil {
				logger.Error("failed to save settings", "error", err)
				os.Exit(1)
			}
			fmt.Println("\nRun 'promptherder' to sync.")

		case "remove":
			if len(agentArgs) == 0 {
				fmt.Fprintf(os.Stderr, "Usage: promptherder agent remove <name> [name...]\n")
				os.Exit(2)
			}
			removeSet := make(map[string]bool, len(agentArgs))
			for _, name := range agentArgs {
				if !app.IsValidAgent(name) {
					fmt.Fprintf(os.Stderr, "Unknown agent: %s\nAvailable: %s\n", name, strings.Join(app.AllAgents, ", "))
					os.Exit(2)
				}
				removeSet[name] = true
			}
			var remaining []string
			for _, a := range settings.EnabledAgents() {
				if !removeSet[a] {
					remaining = append(remaining, a)
				} else {
					fmt.Printf("  ✓ removed %s\n", a)
				}
			}
			settings.Agents = remaining
			if err := app.SaveSettings(cwd, settings); err != nil {
				logger.Error("failed to save settings", "error", err)
				os.Exit(1)
			}
			fmt.Println("\nRun 'promptherder' to sync.")

		default:
			fmt.Fprintf(os.Stderr, "Unknown agent command: %s\nUsage: promptherder agent <list|add|remove> [name...]\n", agentCmd)
			os.Exit(2)
		}

	default:
		logger.Error("unknown subcommand", "subcommand", subcommand)
		fmt.Fprintf(os.Stderr, "Usage: promptherder [<target>|pull|list|agent] [flags]\n")
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
	}
	if len(args) > 0 && known[args[0]] {
		return args[0], args[1:]
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
				if name == "include" { // known value flags
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
