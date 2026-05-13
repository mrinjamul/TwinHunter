package filters

import "testing"

func TestDefaultExcludes(t *testing.T) {
	excludes := DefaultExcludes()
	if len(excludes) == 0 {
		t.Error("default excludes should not be empty")
	}

	expected := []string{".git", "node_modules", ".svn", "__pycache__"}
	for i, e := range expected {
		if excludes[i] != e {
			t.Errorf("expected default exclude[%d] = %q, got %q", i, e, excludes[i])
		}
	}
}

func TestMatchExclude(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		expect   bool
	}{
		{"/path/to/.git/config", []string{".git"}, true},
		{"/path/to/node_modules/pkg", []string{"node_modules"}, true},
		{"/path/to/file.txt", []string{"file.txt"}, true},
		{"/path/to/file.txt", []string{".git", "node_modules"}, false},
		{"/path/to/file.txt", []string{".txt"}, false},
		{"/path/to/file.py", []string{"*.py"}, true},
		{"/path/to/file.txt", []string{""}, false},
		{"/path/to/file.txt", nil, false},
	}

	for _, tt := range tests {
		got := MatchExclude(tt.path, tt.patterns)
		if got != tt.expect {
			t.Errorf("MatchExclude(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.expect)
		}
	}
}

func TestMatchExcludeRegex(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		expect   bool
	}{
		{"/tmp/cache/data", []string{`/cache/`}, true},
		{"/tmp/data/file", []string{`/cache/`}, false},
		{"/path/to/.hidden/file", []string{`\.\w+/`}, true},
		{"/valid/path", []string{`\.\w+/`}, false},
		{"/path/file", []string{`[invalid`}, false},
		{"/path/file", []string{""}, false},
		{"/path/file", nil, false},
	}

	for _, tt := range tests {
		got := MatchExcludeRegex(tt.path, tt.patterns)
		if got != tt.expect {
			t.Errorf("MatchExcludeRegex(%q, %v) = %v, want %v", tt.path, tt.patterns, got, tt.expect)
		}
	}
}

func TestSizeFilter(t *testing.T) {
	tests := []struct {
		size   int64
		min    int64
		max    int64
		expect bool
	}{
		{100, 0, 0, true},
		{100, 50, 0, true},
		{100, 200, 0, false},
		{100, 0, 200, true},
		{100, 0, 50, false},
		{100, 50, 200, true},
		{100, 100, 100, true},
		{0, 1, 0, false},
		{0, 0, 0, true},
	}

	for _, tt := range tests {
		got := SizeFilter(tt.size, tt.min, tt.max)
		if got != tt.expect {
			t.Errorf("SizeFilter(%d, %d, %d) = %v, want %v", tt.size, tt.min, tt.max, got, tt.expect)
		}
	}
}
