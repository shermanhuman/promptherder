package app

import (
	"testing"
)

func TestHerdNameFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{"https with .git", "https://github.com/shermanhuman/compound-v.git", "compound-v"},
		{"https without .git", "https://github.com/shermanhuman/compound-v", "compound-v"},
		{"trailing slash", "https://github.com/shermanhuman/compound-v/", "compound-v"},
		{"trailing slash and .git", "https://github.com/shermanhuman/compound-v.git/", "compound-v"},
		{"ssh url", "git@github.com:shermanhuman/compound-v.git", "compound-v"},
		{"simple name", "compound-v", "compound-v"},
		{"local path", "/home/user/herds/my-herd", "my-herd"},
		{"windows path", "C:\\Users\\s\\github\\compound-v", "compound-v"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := herdNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("herdNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
