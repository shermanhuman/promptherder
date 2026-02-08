package main

import (
	"testing"
)

// --- extractSubcommand tests ---

func TestExtractSubcommand(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		wantSubcommand   string
		wantRemainingLen int
	}{
		{
			name:             "no subcommand",
			args:             []string{"-dry-run", "-v"},
			wantSubcommand:   "",
			wantRemainingLen: 2,
		},
		{
			name:             "copilot subcommand",
			args:             []string{"copilot", "-dry-run"},
			wantSubcommand:   "copilot",
			wantRemainingLen: 1,
		},
		{
			name:             "antigravity subcommand",
			args:             []string{"antigravity", "-v"},
			wantSubcommand:   "antigravity",
			wantRemainingLen: 1,
		},
		{
			name:             "compound-v subcommand",
			args:             []string{"compound-v"},
			wantSubcommand:   "compound-v",
			wantRemainingLen: 0,
		},
		{
			name:             "unknown subcommand",
			args:             []string{"unknown", "-v"},
			wantSubcommand:   "",
			wantRemainingLen: 2,
		},
		{
			name:             "empty args",
			args:             []string{},
			wantSubcommand:   "",
			wantRemainingLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	tests := []struct {
		name string
		csv  string
		want []string
	}{
		{
			name: "empty string",
			csv:  "",
			want: nil,
		},
		{
			name: "single pattern",
			csv:  "**/*.md",
			want: []string{"**/*.md"},
		},
		{
			name: "multiple patterns",
			csv:  "**/*.md,**/*.yaml,*.txt",
			want: []string{"**/*.md", "**/*.yaml", "*.txt"},
		},
		{
			name: "with whitespace",
			csv:  " **/*.md , **/*.yaml , *.txt ",
			want: []string{"**/*.md", "**/*.yaml", "*.txt"},
		},
		{
			name: "empty parts ignored",
			csv:  "**/*.md,,**/*.yaml",
			want: []string{"**/*.md", "**/*.yaml"},
		},
		{
			name: "only whitespace",
			csv:  "   ",
			want: nil,
		},
		{
			name: "only commas",
			csv:  ",,,",
			want: []string{}, // empty slice, not nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIncludePatterns(tt.csv)

			// Handle nil vs empty slice comparison
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
