package core

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mrinjamul/twinhunter/models"
)

var now = time.Now()

func testDir(path ...string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..")
	parts := append([]string{root, "test_data"}, path...)
	return filepath.Join(parts...)
}

func TestScanDuplicates(t *testing.T) {
	dir := testDir("duplicates")
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

	groups := FindDuplicates(files, 2)
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
	dir := testDir("exclude_test")
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
	dir := testDir("mixed_sizes")
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
	dir := testDir("mixed_sizes")
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
	dir := testDir("empty_dir")
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
	dir := testDir("duplicates")
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
	dir := testDir("mixed_sizes")
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
	dir := testDir("mixed_sizes")
	cfg := ScanConfig{
		Path:      dir,
		Recursive: true,
		MinSize:   1024 * 100,
	}

	files, err := Scan(cfg)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	groups := FindDuplicates(files, 2)

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


