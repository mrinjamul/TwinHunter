package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrinjamul/twinhunter/core"
	"github.com/mrinjamul/twinhunter/models"
)

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

func TestFindAndCleanWorkflow(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"dir_a/photo.jpg":        "duplicate content",
		"dir_b/photo.jpg":        "duplicate content",
		"dir_b/backup/photo.jpg": "duplicate content",
		"dir_a/unique.txt":       "unique one",
		"dir_b/readme.md":        "another unique",
	})
	reportPath := filepath.Join(t.TempDir(), "test_report_workflow.json")

	files, err := core.Scan(core.ScanConfig{
		Path:      dir,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(files) != 5 {
		t.Errorf("expected 5 files, got %d", len(files))
	}

	groups := core.FindDuplicates(files, 2, nil)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	report := core.BuildReport(files, groups, dir)
	if report.DupFiles != 2 {
		t.Errorf("expected 2 dup files, got %d", report.DupFiles)
	}

	if err := core.ExportJSON(report, reportPath); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	imported, err := core.ImportJSON(reportPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if imported.DupFiles != report.DupFiles {
		t.Errorf("imported dup files %d != report dup files %d", imported.DupFiles, report.DupFiles)
	}
}

func TestFindWithFilters(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"large.dat":      strings.Repeat("X", 200000),
		"large_copy.dat": strings.Repeat("X", 200000),
		"medium.bin":     strings.Repeat("Y", 200000),
		"small.txt":      "t",
	})

	tests := []struct {
		name       string
		cfg        core.ScanConfig
		wantFiles  int
		wantGroups int
	}{
		{
			name:       "all files",
			cfg:        core.ScanConfig{Path: dir, Recursive: true},
			wantFiles:  4,
			wantGroups: 1,
		},
		{
			name:       "min size 100k",
			cfg:        core.ScanConfig{Path: dir, Recursive: true, MinSize: 1024 * 100},
			wantFiles:  3,
			wantGroups: 1,
		},
		{
			name:       "max size 1k",
			cfg:        core.ScanConfig{Path: dir, Recursive: true, MaxSize: 1024},
			wantFiles:  1,
			wantGroups: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := core.Scan(tt.cfg)
			if err != nil {
				t.Fatalf("Scan error: %v", err)
			}
			if len(files) != tt.wantFiles {
				t.Errorf("got %d files, want %d", len(files), tt.wantFiles)
			}

			groups := core.FindDuplicates(files, 2, nil)
			if len(groups) != tt.wantGroups {
				t.Errorf("got %d groups, want %d", len(groups), tt.wantGroups)
			}
		})
	}
}

func TestExcludeDirs(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		".git/config":               "git config",
		"node_modules/pkg/index.js": "npm stuff",
		"source/main.go":            "package main",
	})

	files, err := core.Scan(core.ScanConfig{
		Path:      dir,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file (main.go), got %d", len(files))
	}
}

func TestEmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	files, err := core.Scan(core.ScanConfig{
		Path:      dir,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestNonRecursive(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"dir_a/photo.jpg": "duplicate content",
		"dir_b/photo.jpg": "duplicate content",
	})

	files, err := core.Scan(core.ScanConfig{
		Path:      dir,
		Recursive: false,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files non-recursive, got %d", len(files))
	}
}

func TestSortGroups(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"dir_a/photo.jpg":         "duplicate content",
		"dir_b/photo.jpg":         "duplicate content",
		"dir_b/backup/photo.jpg":  "duplicate content",
		"dir_a/unique.txt":        "unique one",
		"dir_b/readme.md":         "another unique",
		"large.dat":               strings.Repeat("X", 200000),
		"large_copy.dat":          strings.Repeat("X", 200000),
	})

	files, err := core.Scan(core.ScanConfig{
		Path:      dir,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	groups := core.FindDuplicates(files, 2, nil)

	core.SortGroupsBySize(groups)
	for i := 0; i < len(groups)-1; i++ {
		if groups[i].Size < groups[i+1].Size {
			t.Error("groups not sorted by size")
		}
	}

	core.SortGroupsByCount(groups)
	for i := 0; i < len(groups)-1; i++ {
		if len(groups[i].Files) < len(groups[i+1].Files) {
			t.Error("groups not sorted by count")
		}
	}
}

func TestReportExportImport(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"dir_a/photo.jpg":        "duplicate content",
		"dir_b/photo.jpg":        "duplicate content",
		"dir_b/backup/photo.jpg": "duplicate content",
		"dir_a/unique.txt":       "unique one",
		"dir_b/readme.md":        "another unique",
	})
	reportPath := filepath.Join(t.TempDir(), "test_report_io.json")

	files, _ := core.Scan(core.ScanConfig{Path: dir, Recursive: true})
	groups := core.FindDuplicates(files, 2, nil)
	report := core.BuildReport(files, groups, dir)

	if err := core.ExportJSON(report, reportPath); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	info, err := os.Stat(reportPath)
	if err != nil {
		t.Fatalf("Report file not created: %v", err)
	}
	if info.Size() == 0 {
		t.Error("Report file is empty")
	}

	imported, err := core.ImportJSON(reportPath)
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if imported.ScanPath != report.ScanPath {
		t.Errorf("scan path mismatch: got %s, want %s", imported.ScanPath, report.ScanPath)
	}
	if len(imported.DupGroups) != len(report.DupGroups) {
		t.Errorf("group count mismatch: got %d, want %d", len(imported.DupGroups), len(report.DupGroups))
	}
}

func TestKeepStrategyIntegration(t *testing.T) {
	dir := createTestDir(t, map[string]string{
		"dir_a/photo.jpg":        "duplicate content",
		"dir_b/photo.jpg":        "duplicate content",
		"dir_b/backup/photo.jpg": "duplicate content",
	})

	files, err := core.Scan(core.ScanConfig{
		Path:      dir,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	groups := core.FindDuplicates(files, 2, nil)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	g := groups[0]

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	g.Files = []models.FileInfo{
		{Path: "dir_a/newest.jpg", ModTime: base.Add(2 * time.Hour)},
		{Path: "dir_b/old.jpg", ModTime: base},
		{Path: "dir_b/backup/middle.jpg", ModTime: base.Add(1 * time.Hour)},
	}

	tests := []struct {
		strategy string
		wantKeep string
		wantDrop int
	}{
		{"oldest", "dir_b/old.jpg", 2},
		{"newest", "dir_a/newest.jpg", 2},
		{"shortest", "dir_b/old.jpg", 2},
	}

	for _, tt := range tests {
		keep, toRemove := core.ApplyKeepStrategy(g, tt.strategy)
		if keep.Path != tt.wantKeep {
			t.Errorf("%s: want keep %s, got %s", tt.strategy, tt.wantKeep, keep.Path)
		}
		if len(toRemove) != tt.wantDrop {
			t.Errorf("%s: want %d to drop, got %d", tt.strategy, tt.wantDrop, len(toRemove))
		}
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"", 0, false},
		{"0", 0, false},
		{"100", 100, false},
		{"500", 500, false},
		{"1K", 1024, false},
		{"1k", 1024, false},
		{"2K", 2 * 1024, false},
		{"1M", 1024 * 1024, false},
		{"1m", 1024 * 1024, false},
		{"2M", 2 * 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"1g", 1024 * 1024 * 1024, false},
		{"1T", 1024 * 1024 * 1024 * 1024, false},
		{"1t", 1024 * 1024 * 1024 * 1024, false},
		{"0K", 0, false},
		{"0M", 0, false},
		{"0G", 0, false},
		{"0T", 0, false},
		{"  2M  ", 2 * 1024 * 1024, false},
		{"abc", 0, true},
		{"1X", 0, true},
		{"-1", 0, true},
	}

	for _, tt := range tests {
		got, err := parseSize(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSize(%q) expected error, got %d", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSize(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.expected {
			t.Errorf("parseSize(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"a,,c", []string{"a", "c"}},
		{"a, ,c", []string{"a", "c"}},
		{",,,", nil},
	}

	for _, tt := range tests {
		got := splitCSV(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("splitCSV(%q) = %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for i, v := range tt.expected {
			if got[i] != v {
				t.Errorf("splitCSV(%q)[%d] = %q, want %q", tt.input, i, got[i], v)
			}
		}
	}
}

func TestTotalSize(t *testing.T) {
	files := []models.FileInfo{
		{Size: 100},
		{Size: 200},
		{Size: 300},
	}

	got := totalSize(files)
	if got != 600 {
		t.Errorf("totalSize = %d, want 600", got)
	}

	gotEmpty := totalSize([]models.FileInfo{})
	if gotEmpty != 0 {
		t.Errorf("totalSize(empty) = %d, want 0", gotEmpty)
	}

	gotNil := totalSize(nil)
	if gotNil != 0 {
		t.Errorf("totalSize(nil) = %d, want 0", gotNil)
	}
}
