package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shermanhuman/promptherder/internal/app"
)

func TestConfigureTargets(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		args                         []string
		input                        string
		interactive, dryRun, wantErr bool
		want                         []string
	}{
		{name: "interactive explicit selection", input: "codex, claude codex\n", interactive: true, want: []string{"codex", "claude"}},
		{name: "interactive none", input: "none\n", interactive: true, want: []string{}},
		{name: "automation", args: []string{"codex", "claude"}, want: []string{"codex", "claude"}},
		{name: "explicit none", args: []string{"none"}, want: []string{}},
		{name: "dry run", args: []string{"codex"}, dryRun: true, want: []string{"codex"}},
		{name: "no terminal", wantErr: true},
		{name: "dry run does not prompt", interactive: true, dryRun: true, input: "codex\n", wantErr: true},
		{name: "cancel", interactive: true, wantErr: true},
		{name: "blank is not approval", interactive: true, input: "\n", wantErr: true},
		{name: "invalid target", args: []string{"codex", "typo"}, wantErr: true},
		{name: "none cannot be combined", args: []string{"none", "codex"}, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			original := app.Settings{Agents: []string{"cursor"}, CommandPrefix: "v-", CommandPrefixEnabled: true}
			if err := app.SaveSettings(dir, original); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, ".promptherder", "settings.json")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			got, err := configureTargets(dir, original, tc.args, tc.dryRun, tc.interactive, strings.NewReader(tc.input), &out)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			after, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if tc.wantErr || tc.dryRun {
				if !bytes.Equal(before, after) {
					t.Fatal("settings changed on error or dry-run")
				}
			} else {
				loaded, err := app.LoadSettings(dir)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(loaded.Agents, tc.want) || !loaded.TargetsConfigured() {
					t.Fatalf("saved selection: %#v", loaded.Agents)
				}
				if loaded.CommandPrefix != "v-" || !loaded.CommandPrefixEnabled {
					t.Fatal("setup lost prefix settings")
				}
			}
			if !tc.wantErr && !reflect.DeepEqual(got.Agents, tc.want) {
				t.Fatalf("selection = %#v, want %#v", got.Agents, tc.want)
			}
		})
	}
}

func TestChangeTargets(t *testing.T) {
	for _, tc := range []struct {
		name, action        string
		current, args, want []string
		wantErr             bool
	}{
		{name: "add from none", action: "add", args: []string{"codex", "claude"}, want: []string{"codex", "claude"}},
		{name: "deduplicate", action: "add", current: []string{"codex"}, args: []string{"claude", "codex", "claude"}, want: []string{"codex", "claude"}},
		{name: "remove last", action: "remove", current: []string{"codex"}, args: []string{"codex"}, want: []string{}},
		{name: "remove missing", action: "remove", current: []string{"codex"}, args: []string{"claude"}, want: []string{"codex"}},
		{name: "invalid batch", action: "add", current: []string{"codex"}, args: []string{"claude", "typo"}, wantErr: true},
		{name: "missing arguments", action: "remove", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := append([]string(nil), tc.current...)
			got, err := changeTargets(tc.current, tc.action, tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v", err)
			}
			if !tc.wantErr && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
			if !reflect.DeepEqual(before, tc.current) {
				t.Fatal("mutated original settings")
			}
		})
	}
}
