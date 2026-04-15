package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const aliasesFile = "aliases.json"
const aliasesSubdir = "promptherder"

// Alias represents a named herd shortcut with one or more URLs.
type Alias struct {
	URLs        []string `json:"urls"`
	Description string   `json:"description"`
}

// AliasConfig maps short names to alias definitions.
type AliasConfig map[string]Alias

// defaultAliases are embedded in the binary and used when no user config exists.
var defaultAliases = AliasConfig{
	"compound-v": {
		URLs:        []string{"https://github.com/shermanhuman/compound-v"},
		Description: "Agent methodology — planning, execution, review",
	},
	"grugg": {
		URLs:        []string{"https://github.com/shermanhuman/grugg"},
		Description: "Token reduction prompt — ~75% fewer output tokens",
	},
	"oh": {
		URLs:        []string{"https://github.com/shermanhuman/oh"},
		Description: "Elixir/Go environment skills — starting template, customize for your stack",
	},
}

// AliasesConfigPath returns the full path to the aliases config file.
// Returns empty string and error if the config directory cannot be determined.
func AliasesConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(configDir, aliasesSubdir, aliasesFile), nil
}

// LoadAliases loads aliases from the user's config file.
// Falls back to embedded defaults if the file doesn't exist.
// Returns the config, the source ("file" or "default"), and any error.
func LoadAliases() (AliasConfig, string, error) {
	path, err := AliasesConfigPath()
	if err != nil {
		// Can't determine config dir — use embedded defaults only.
		return defaultAliases, "default", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultAliases, "default", nil
		}
		return nil, "", fmt.Errorf("read aliases: %w", err)
	}

	var aliases AliasConfig
	if err := json.Unmarshal(data, &aliases); err != nil {
		return nil, "", fmt.Errorf("parse %s: %w", path, err)
	}

	return aliases, "file", nil
}

// WriteDefaultAliases writes the embedded default aliases to disk.
// Creates parent directories as needed.
func WriteDefaultAliases(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := json.MarshalIndent(defaultAliases, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal defaults: %w", err)
	}

	// Append final newline.
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write aliases: %w", err)
	}
	return nil
}

// ResolveAlias looks up a short name in the alias config.
// Returns the URLs if found, nil if not found.
func ResolveAlias(name string, aliases AliasConfig) []string {
	if alias, ok := aliases[name]; ok {
		return alias.URLs
	}
	return nil
}

// SortedAliasNames returns alias names in alphabetical order.
func SortedAliasNames(aliases AliasConfig) []string {
	names := make([]string, 0, len(aliases))
	for name := range aliases {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// isURL returns true if the argument looks like a URL rather than a short name.
func isURL(arg string) bool {
	return strings.Contains(arg, "://") || strings.HasPrefix(arg, "git@")
}
