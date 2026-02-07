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

	"github.com/breakdown-analytics/promptherder/internal/app"
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

	var (
		repoPath    string
		sourceName  string
		includeCSV  string
		dryRun      bool
		verbose     bool
		showVersion bool
	)

	flag.StringVar(&repoPath, "repo", ".", "Path to repo root (default: current directory)")
	flag.StringVar(&sourceName, "source", "antigravity", "Source of truth: antigravity|agent")
	flag.StringVar(&includeCSV, "include", "", "Comma-separated glob patterns to include (default: all)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show actions without writing files")
	flag.BoolVar(&verbose, "v", false, "Verbose logging")
	flag.BoolVar(&showVersion, "version", false, "Print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("promptherder %s (commit: %s, built: %s)\n", Version, Commit, BuildDate)
		return
	}

	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))

	include := parseIncludePatterns(includeCSV)
	cfg := app.Config{
		RepoPath: repoPath,
		Source:   sourceName,
		Include:  include,
		DryRun:   dryRun,
		Logger:   logger,
	}

	if err := app.Run(ctx, cfg); err != nil {
		if errors.Is(err, app.ErrValidation) {
			logger.Error("validation error", "error", err)
			os.Exit(2)
		}
		logger.Error("failed", "error", err)
		os.Exit(1)
	}
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
