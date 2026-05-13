package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrinjamul/twinhunter/models"
)

var now = time.Now()

func createTestDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestScanDuplicates(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"dir_a/photo.jpg":         "duplicate content",
		"dir_b/photo.jpg":         "duplicate content",
		"dir_b/backup/photo.jpg":  "duplicate content",
		"dir_a/unique.txt":        "unique one",
		"dir_b/readme.md":         "another unique",
	})
	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 5 {
		t.Errorf("expected 5 files, got %d", len(files))
	}

	groups := FindDuplicates(files, 2, nil)
	if len(groups) != 1 {
		t.Fatalf("expected 1 duplicate group, got %d", len(groups))
	}

	g := groups[0]
	if len(g.Files) != 3 {
		t.Errorf("expected 3 files in group, got %d", len(g.Files))
	}
	if g.Size == 0 {
		t.Error("expected non-zero group size")
	}
}

func TestScanExcludeDirs(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		".git/config":                  "git config",
		"node_modules/pkg/index.js":    "npm stuff",
		"source/main.go":               "package main",
	})
	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file (main.go), got %d: %v", len(files), files)
	}
}

func TestScanMinSize(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"large.dat":      strings.Repeat("X", 200000),
		"large_copy.dat": strings.Repeat("X", 200000),
		"medium.bin":     strings.Repeat("Y", 200000),
		"small.txt":      "t",
	})
	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
		MinSize:   1024 * 100,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 3 {
		t.Errorf("expected 3 files >= 100KB, got %d", len(files))
	}
}

func TestScanMaxSize(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"large.dat":      strings.Repeat("X", 200000),
		"large_copy.dat": strings.Repeat("X", 200000),
		"medium.bin":     strings.Repeat("Y", 200000),
		"small.txt":      "t",
	})
	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
		MaxSize:   1024,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file <= 1KB, got %d", len(files))
	}
}

func TestScanEmptyDir(t *testing.T) {
	dir := t.TempDir()
	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(files))
	}
}

func TestScanNonRecursive(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"dir_a/photo.jpg": "duplicate content",
		"dir_b/photo.jpg": "duplicate content",
	})
	cfg := ScanConfig{
		Path:      dir,
		Recursive: false,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files (non-recursive, no files in root), got %d", len(files))
	}
}

func TestScanWithExcludePatterns(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"large.dat":      strings.Repeat("X", 200000),
		"large_copy.dat": strings.Repeat("X", 200000),
		"medium.bin":     strings.Repeat("Y", 200000),
		"small.txt":      "t",
	})
	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
		Exclude:   []string{"*.dat"},
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	for _, f := range files {
		if filepath.Ext(f.Path) == ".dat" {
			t.Errorf("should have excluded .dat files, got %s", f.Path)
		}
	}
}

func TestFindDuplicatesSameSizeDifferentContent(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"large.dat":      strings.Repeat("X", 200000),
		"large_copy.dat": strings.Repeat("X", 200000),
		"medium.bin":     strings.Repeat("Y", 200000),
	})
	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
		MinSize:   1,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	groups := FindDuplicates(files, 2, nil)

	found := false
	for _, g := range groups {
		for _, f := range g.Files {
			if filepath.Ext(f.Path) == ".dat" {
				found = true
			}
		}
	}

	if !found {
		t.Error("expected .dat duplicates to be found")
	}
}

func TestApplyKeepStrategyWithTestFiles(t *testing.T) {
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/very/long/path/to/file.jpg", ModTime: now},
			{Path: "/short.jpg", ModTime: now.Add(-1 * time.Hour)},
			{Path: "/medium/path.jpg", ModTime: now.Add(1 * time.Hour)},
		},
	}

	tests := []struct {
		strategy string
		wantKeep string
	}{
		{"shortest", "/short.jpg"},
		{"oldest", "/short.jpg"},
		{"newest", "/medium/path.jpg"},
	}

	for _, tt := range tests {
		keep, _ := ApplyKeepStrategy(group, tt.strategy)
		if keep.Path != tt.wantKeep {
			t.Errorf("strategy %s: want %s, got %s", tt.strategy, tt.wantKeep, keep.Path)
		}
	}
}

func TestSortGroupsBySize(t *testing.T) {
	groups := []models.DuplicateGroup{
		{Size: 100},
		{Size: 500},
		{Size: 200},
	}

	SortGroupsBySize(groups)

	if groups[0].Size != 500 || groups[1].Size != 200 || groups[2].Size != 100 {
		t.Errorf("expected [500, 200, 100], got %v", groups)
	}
}

func TestSortGroupsByCount(t *testing.T) {
	groups := []models.DuplicateGroup{
		{Files: []models.FileInfo{{}, {}}},
		{Files: []models.FileInfo{{}, {}, {}, {}}},
		{Files: []models.FileInfo{{}, {}, {}}},
	}

	SortGroupsByCount(groups)

	if len(groups[0].Files) != 4 || len(groups[1].Files) != 3 || len(groups[2].Files) != 2 {
		t.Errorf("expected [4, 3, 2] files, got [%d, %d, %d]",
			len(groups[0].Files), len(groups[1].Files), len(groups[2].Files))
	}
}

func TestScan_CustomExcludeDir(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"custom_cache/data.txt":  "cache content",
		"source/main.go":         "package main",
		"source/other.txt":       "other content",
	})

	cfg := ScanConfig{
		Path:        dir,
		Recursive:   true,
		ExcludeDir:  []string{"custom_cache"},
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files (custom_cache excluded), got %d", len(files))
	}

	for _, f := range files {
		if strings.Contains(f.Path, "custom_cache") {
			t.Errorf("custom_cache should be excluded, but found: %s", f.Path)
		}
	}
}

func TestScan_ExcludeRegex(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"file.txt":         "content",
		"image.jpg":        "image",
		"document.pdf":     "pdf",
		"data.log":         "log",
	})

	cfg := ScanConfig{
		Path:         dir,
		Recursive:    true,
		ExcludeRegex: []string{`\.log$`, `\.pdf$`},
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files (.log and .pdf excluded), got %d", len(files))
	}
}

func TestScan_SymlinkSkipped(t *testing.T) {
	dir := t.TempDir()

	realFile := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(realFile, []byte("real content"), 0644); err != nil {
		t.Fatal(err)
	}

	symlinkFile := filepath.Join(dir, "link.txt")
	if err := os.Symlink(realFile, symlinkFile); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("expected 1 file (symlink skipped), got %d", len(files))
	}

	if files[0].Path != realFile {
		t.Errorf("expected real file, got %s", files[0].Path)
	}
}

func TestScan_HardLinkDetection(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	file2 := filepath.Join(dir, "file2.txt")
	if err := os.Link(file1, file2); err != nil {
		t.Skip("hard links not supported on this filesystem")
	}

	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(files) != 2 {
		t.Errorf("expected 2 files, got %d", len(files))
	}

	hardLinkCount := CountHardLinks(files)
	if hardLinkCount < 1 {
		t.Errorf("expected at least 1 hard link detected, got %d", hardLinkCount)
	}
}

func TestScan_OnProgressCallback(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"file1.txt": "content1",
		"file2.txt": "content2",
		"file3.txt": "content3",
	})

	callbackCalled := false
	var maxFilesScanned int

	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
		OnProgress: func(stats models.ScanStats) {
			callbackCalled = true
			if stats.FilesScanned > maxFilesScanned {
				maxFilesScanned = stats.FilesScanned
			}
		},
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if !callbackCalled {
		t.Error("OnProgress callback should have been called")
	}

	if maxFilesScanned != len(files) {
		t.Errorf("expected progress to report %d files, got %d", len(files), maxFilesScanned)
	}
}

func TestScan_OnProgressNilSafety(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"file1.txt": "content1",
	})

	cfg := ScanConfig{
		Path:       dir,
		Recursive:  true,
		OnProgress: nil,
	}

	_, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan with nil OnProgress should not panic: %v", err)
	}
}
