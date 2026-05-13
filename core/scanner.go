package core

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mrinjamul/twinhunter/filters"
	"github.com/mrinjamul/twinhunter/models"
)

// ScanConfig holds options for the file scanner.
type ScanConfig struct {
	Path         string
	Recursive    bool
	MinSize      int64
	MaxSize      int64
	Exclude      []string
	ExcludeRegex []string
	ExcludeDir   []string
	Workers      int
	OnProgress   func(stats models.ScanStats)
}

// Scan discovers files using filepath.WalkDir for better performance,
// applies filters, and returns a list of FileInfo.
func Scan(cfg ScanConfig) ([]models.FileInfo, error) {
	// nil means "not set" — apply defaults.
	// non-nil empty slice means "explicitly set to empty" — no defaults.
	if cfg.ExcludeDir == nil {
		cfg.ExcludeDir = filters.DefaultExcludes()
	}

	var allFiles []models.FileInfo

	err := walkDir(cfg.Path, cfg.Recursive, cfg.ExcludeDir, cfg.Exclude, cfg.ExcludeRegex, cfg.MinSize, cfg.MaxSize, func(fullPath string, info fs.FileInfo) {
		isHardLink := detectHardLink(info)

		allFiles = append(allFiles, models.FileInfo{
			Path:       fullPath,
			Name:       info.Name(),
			Size:       info.Size(),
			ModTime:    info.ModTime(),
			IsHardLink: isHardLink,
		})

		if cfg.OnProgress != nil {
			cfg.OnProgress(models.ScanStats{
				FilesScanned: len(allFiles),
				CurrentPath:  fullPath,
			})
		}
	})
	if err != nil {
		return nil, err
	}

	if cfg.OnProgress != nil {
		cfg.OnProgress(models.ScanStats{
			FilesScanned: len(allFiles),
			TotalFiles:   len(allFiles),
			CurrentPath:  cfg.Path,
		})
	}

	return allFiles, nil
}
func walkDir(root string, recursive bool, excludeDirs, excludePatterns, excludeRegex []string, minSize, maxSize int64, fn func(string, fs.FileInfo)) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error accessing %s: %v\n", path, err)
			return nil
		}

		if d.IsDir() {
			if path != root && !recursive {
				return filepath.SkipDir
			}
			name := d.Name()
			for _, pattern := range excludeDirs {
				if name == pattern {
					return filepath.SkipDir
				}
			}
			return nil
		}

		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		if filters.MatchExclude(d.Name(), excludePatterns) {
			return nil
		}

		for _, pattern := range excludeRegex {
			if filters.MatchExcludeRegex(path, []string{pattern}) {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read file info for %s: %v\n", path, err)
			return nil
		}

		size := info.Size()
		if minSize > 0 && size < minSize {
			return nil
		}
		if maxSize > 0 && size > maxSize {
			return nil
		}

		fn(path, info)
		return nil
	})
}

// CountHardLinks returns the number of files that are already hard-linked.
func CountHardLinks(files []models.FileInfo) int {
	n := 0
	for _, f := range files {
		if f.IsHardLink {
			n++
		}
	}
	return n
}

// PlatformName returns a human-readable name for the current OS/arch.
func PlatformName() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
