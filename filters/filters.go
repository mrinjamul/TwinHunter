package filters

import (
	"path/filepath"
	"regexp"
	"strings"
)

var defaultExcludes = []string{
	".git",
	"node_modules",
	".svn",
	"__pycache__",
	".DS_Store",
	"Thumbs.db",
}

// DefaultExcludes returns the built-in exclusion patterns.
func DefaultExcludes() []string {
	return defaultExcludes
}

// MatchExclude checks if a path matches any of the given glob patterns.
func MatchExclude(path string, patterns []string) bool {
	base := filepath.Base(path)
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		matched, _ := filepath.Match(p, base)
		if matched {
			return true
		}
		matched, _ = filepath.Match(p, path)
		if matched {
			return true
		}
		if strings.Contains(path, p) {
			return true
		}
	}
	return false
}

// MatchExcludeRegex checks if a path matches any of the given regex patterns.
func MatchExcludeRegex(path string, patterns []string) bool {
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		if re.MatchString(path) {
			return true
		}
	}
	return false
}

// SizeFilter returns true if the file size is within the [min, max] range.
// A value of 0 means no limit on that side.
func SizeFilter(size, min, max int64) bool {
	if min > 0 && size < min {
		return false
	}
	if max > 0 && size > max {
		return false
	}
	return true
}
