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

	promptherder "github.com/shermanhuman/promptherder"
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

	// Parse flags from remaining args.
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
	_ = fs.Parse(args)

	if showVersion {
		fmt.Printf("promptherder %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
		return
	}

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

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
	compoundV := app.CompoundVTarget{FS: promptherder.CompoundVFS}

	allTargets := []app.Target{copilot, antigravity, compoundV}

	var runErr error
	switch subcommand {
	case "":
		// Bare promptherder — run all targets.
		runErr = app.RunAll(ctx, allTargets, cfg)
	case "copilot":
		runErr = app.RunTarget(ctx, copilot, cfg)
	case "antigravity":
		runErr = app.RunTarget(ctx, antigravity, cfg)
	case "compound-v":
		runErr = app.RunTarget(ctx, compoundV, cfg)
	default:
		logger.Error("unknown subcommand", "subcommand", subcommand)
		fmt.Fprintf(os.Stderr, "Usage: promptherder [copilot|antigravity|compound-v] [flags]\n")
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
		"compound-v":  true,
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
