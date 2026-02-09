package app

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestUIHandler_LevelIcons(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := NewUIHandler(&buf, slog.LevelDebug)
	logger := slog.New(h)

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d: %q", len(lines), output)
	}

	wantIcons := []string{"·", "✓", "⚠", "✗"}
	for i, want := range wantIcons {
		if !strings.Contains(lines[i], want) {
			t.Errorf("line %d: expected icon %q, got %q", i, want, lines[i])
		}
	}
}

func TestUIHandler_FormatsAttrs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := NewUIHandler(&buf, slog.LevelInfo)
	logger := slog.New(h)

	logger.Info("synced", "target", ".agent/rules/foo.md", "source", "rules/foo.md", "herd", "compound-v")
	output := buf.String()

	if !strings.Contains(output, ".agent/rules/foo.md") {
		t.Errorf("missing target path in output: %q", output)
	}
	if !strings.Contains(output, "← rules/foo.md") {
		t.Errorf("missing source arrow in output: %q", output)
	}
	if !strings.Contains(output, "(compound-v)") {
		t.Errorf("missing herd parens in output: %q", output)
	}
}

func TestUIHandler_RespectsLevel(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	h := NewUIHandler(&buf, slog.LevelWarn)
	logger := slog.New(h)

	logger.Info("should not appear")
	logger.Warn("should appear")

	output := buf.String()
	if strings.Contains(output, "should not appear") {
		t.Error("info message should be filtered at warn level")
	}
	if !strings.Contains(output, "should appear") {
		t.Error("warn message should appear at warn level")
	}
}
