package main

import (
	"testing"
)

// --- extractSubcommand tests ---

func TestExtractSubcommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		args             []string
		wantSubcommand   string
		wantRemainingLen int
	}{
		{"no subcommand", []string{"-dry-run", "-v"}, "", 2},
		{"copilot subcommand", []string{"copilot", "-dry-run"}, "copilot", 1},
		{"antigravity subcommand", []string{"antigravity", "-v"}, "antigravity", 1},
		{"compound-v subcommand", []string{"compound-v"}, "compound-v", 0},
		{"unknown subcommand", []string{"unknown", "-v"}, "", 2},
		{"empty args", []string{}, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotSubcommand, gotRemaining := extractSubcommand(tt.args)
			if gotSubcommand != tt.wantSubcommand {
				t.Errorf("subcommand = %q, want %q", gotSubcommand, tt.wantSubcommand)
			}
			if len(gotRemaining) != tt.wantRemainingLen {
				t.Errorf("remaining args length = %d, want %d", len(gotRemaining), tt.wantRemainingLen)
			}
		})
	}
}

// --- parseIncludePatterns tests ---

func TestParseIncludePatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		csv  string
		want []string
	}{
		{"empty string", "", nil},
		{"single pattern", "**/*.md", []string{"**/*.md"}},
		{"multiple patterns", "**/*.md,**/*.yaml,*.txt", []string{"**/*.md", "**/*.yaml", "*.txt"}},
		{"with whitespace", " **/*.md , **/*.yaml , *.txt ", []string{"**/*.md", "**/*.yaml", "*.txt"}},
		{"empty parts ignored", "**/*.md,,**/*.yaml", []string{"**/*.md", "**/*.yaml"}},
		{"only whitespace", "   ", nil},
		{"only commas", ",,,", []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := parseIncludePatterns(tt.csv)

			if tt.want == nil {
				if got != nil {
					t.Errorf("parseIncludePatterns() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("parseIncludePatterns() length = %d, want %d", len(got), len(tt.want))
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseIncludePatterns()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
