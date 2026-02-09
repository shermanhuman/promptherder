package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/shermanhuman/promptherder/internal/app"
)

// Set via ldflags at build time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

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
  promptherder [flags]              Sync all targets
  promptherder <target> [flags]     Sync a single target (copilot, antigravity)
  promptherder pull <git-url>       Install a herd from a Git repository

Flags:
  -dry-run     Show actions without writing files
  -include     Comma-separated glob patterns to include (default: all)
  -v           Verbose logging (structured output to stderr)
  -version     Print version and exit

Settings (.promptherder/settings.json):
  command_prefix           Prefix for command filenames, e.g. "v-" (default: "")
  command_prefix_enabled   Enable the prefix (default: false)

  Example:
    {
      "command_prefix": "v-",
      "command_prefix_enabled": true
    }

Examples:
  promptherder                                Sync all targets
  promptherder pull https://github.com/user/herd
  promptherder copilot -dry-run               Preview copilot sync
  promptherder antigravity                    Sync antigravity only
`)
	}

	_ = fs.Parse(flagArgs)

	// Merge any remaining flag parse args with positional args.
	allPositional := append(fs.Args(), positionalArgs...)

	if showVersion {
		fmt.Printf("promptherder %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
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

	// Build the targets registry.
	copilot := app.CopilotTarget{Include: cfg.Include}
	antigravity := app.AntigravityTarget{}

	allTargets := []app.Target{copilot, antigravity}

	var runErr error
	switch subcommand {
	case "":
		// Bare promptherder — discover herds, merge, then fan out to targets.
		runErr = app.RunAll(ctx, allTargets, cfg)
	case "copilot":
		runErr = app.RunTarget(ctx, copilot, cfg)
	case "antigravity":
		runErr = app.RunTarget(ctx, antigravity, cfg)
	case "pull":
		var gitURL string
		if len(allPositional) > 0 {
			gitURL = allPositional[0]
		}
		if gitURL == "" {
			logger.Error("missing URL argument")
			fmt.Fprintf(os.Stderr, "Usage: promptherder pull <git-url>\n")
			os.Exit(2)
		}
		runErr = app.Pull(ctx, gitURL, app.PullConfig{
			RepoPath: cwd,
			DryRun:   dryRun,
			Logger:   logger,
		})
	default:
		logger.Error("unknown subcommand", "subcommand", subcommand)
		fmt.Fprintf(os.Stderr, "Usage: promptherder [copilot|antigravity|pull] [flags]\n")
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
		"pull":        true,
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
