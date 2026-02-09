package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
)

// UIHandler is a slog.Handler that produces pretty, human-readable output
// for CLI use. It replaces the default slog.TextHandler for normal operation.
//
// Output format:
//
//	✓ synced .agent/rules/compound-v.md ← rules/compound-v.md (compound-v)
//	→ installing target: antigravity
//	⚠ no herds found — run `promptherder pull <url>` to install one
//	✗ failed to resolve path
type UIHandler struct {
	w     io.Writer
	level slog.Level
	mu    sync.Mutex
}

// NewUIHandler creates a handler that writes pretty output to w.
func NewUIHandler(w io.Writer, level slog.Level) *UIHandler {
	return &UIHandler{w: w, level: level}
}

func (h *UIHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *UIHandler) Handle(_ context.Context, r slog.Record) error {
	icon := levelIcon(r.Level)
	msg := r.Message

	// Build a details string from attributes.
	var parts []string
	r.Attrs(func(a slog.Attr) bool {
		parts = append(parts, formatAttr(a))
		return true
	})

	var line string
	if len(parts) > 0 {
		line = fmt.Sprintf("  %s %s %s\n", icon, msg, strings.Join(parts, " "))
	} else {
		line = fmt.Sprintf("  %s %s\n", icon, msg)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.w, line)
	return err
}

func (h *UIHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *UIHandler) WithGroup(_ string) slog.Handler       { return h }

func levelIcon(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "✗"
	case l >= slog.LevelWarn:
		return "⚠"
	case l >= slog.LevelInfo:
		return "✓"
	default:
		return "·"
	}
}

func formatAttr(a slog.Attr) string {
	switch a.Key {
	case "target":
		return a.Value.String()
	case "source":
		return "← " + a.Value.String()
	case "herd":
		return "(" + a.Value.String() + ")"
	case "name":
		return a.Value.String()
	case "file":
		return a.Value.String()
	case "dir":
		return a.Value.String()
	case "error":
		return "— " + a.Value.String()
	case "url":
		return a.Value.String()
	case "path":
		return a.Value.String()
	case "herds":
		return a.Value.String()
	default:
		return a.Key + "=" + a.Value.String()
	}
}
