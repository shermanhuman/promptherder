package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAliases_MissingFile_ReturnsDefaults(t *testing.T) {
	// Not parallel: uses t.Setenv.

	// Set config dir to a temp dir with no aliases file.
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	aliases, source, err := LoadAliases()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "default" {
		t.Errorf("expected source 'default', got %q", source)
	}
	if len(aliases) == 0 {
		t.Fatal("expected embedded defaults, got empty map")
	}
	// Verify a known default exists.
	if _, ok := aliases["compound-v"]; !ok {
		t.Error("expected 'compound-v' in default aliases")
	}
}

func TestLoadAliases_UserFile_Overrides(t *testing.T) {
	// Not parallel: uses t.Setenv.

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	// Write a custom aliases file.
	configDir := filepath.Join(tmpDir, aliasesSubdir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	custom := AliasConfig{
		"my-herd": {
			URLs:        []string{"https://github.com/someone/my-herd"},
			Description: "My custom herd",
		},
	}
	data, _ := json.Marshal(custom)
	if err := os.WriteFile(filepath.Join(configDir, aliasesFile), data, 0644); err != nil {
		t.Fatal(err)
	}

	aliases, source, err := LoadAliases()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "file" {
		t.Errorf("expected source 'file', got %q", source)
	}
	if _, ok := aliases["my-herd"]; !ok {
		t.Error("expected 'my-herd' in loaded aliases")
	}
	// Defaults should NOT be present when user file exists.
	if _, ok := aliases["compound-v"]; ok {
		t.Error("expected user file to fully replace defaults, but 'compound-v' is still present")
	}
}

func TestLoadAliases_MalformedJSON_ReturnsError(t *testing.T) {
	// Not parallel: uses t.Setenv.

	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	configDir := filepath.Join(tmpDir, aliasesSubdir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, aliasesFile), []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, _, err := LoadAliases()
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestResolveAlias_Found(t *testing.T) {
	t.Parallel()

	aliases := AliasConfig{
		"oh": {
			URLs:        []string{"https://github.com/shermanhuman/oh"},
			Description: "test",
		},
	}

	urls := ResolveAlias("oh", aliases)
	if len(urls) != 1 || urls[0] != "https://github.com/shermanhuman/oh" {
		t.Errorf("unexpected URLs: %v", urls)
	}
}

func TestResolveAlias_NotFound(t *testing.T) {
	t.Parallel()

	aliases := AliasConfig{}
	urls := ResolveAlias("nonexistent", aliases)
	if urls != nil {
		t.Errorf("expected nil for unknown alias, got %v", urls)
	}
}

func TestResolveAlias_MultipleURLs(t *testing.T) {
	t.Parallel()

	aliases := AliasConfig{
		"bundle": {
			URLs: []string{
				"https://github.com/shermanhuman/compound-v",
				"https://github.com/shermanhuman/oh",
			},
			Description: "Multi-herd bundle",
		},
	}

	urls := ResolveAlias("bundle", aliases)
	if len(urls) != 2 {
		t.Fatalf("expected 2 URLs, got %d", len(urls))
	}
}

func TestWriteDefaultAliases(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sub", "deep", aliasesFile)

	if err := WriteDefaultAliases(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	var aliases AliasConfig
	if err := json.Unmarshal(data, &aliases); err != nil {
		t.Fatalf("written file has invalid JSON: %v", err)
	}

	if _, ok := aliases["compound-v"]; !ok {
		t.Error("expected 'compound-v' in written defaults")
	}
}

func TestSortedAliasNames(t *testing.T) {
	t.Parallel()

	aliases := AliasConfig{
		"zulu":  {Description: "z"},
		"alpha": {Description: "a"},
		"bravo": {Description: "b"},
	}

	names := SortedAliasNames(aliases)
	if len(names) != 3 || names[0] != "alpha" || names[1] != "bravo" || names[2] != "zulu" {
		t.Errorf("expected sorted names, got %v", names)
	}
}

func TestIsURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected bool
	}{
		{"https://github.com/user/repo", true},
		{"http://example.com/repo", true},
		{"git@github.com:user/repo.git", true},
		{"compound-v", false},
		{"oh", false},
		{"my-herd", false},
	}

	for _, tt := range tests {
		if got := isURL(tt.input); got != tt.expected {
			t.Errorf("isURL(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
