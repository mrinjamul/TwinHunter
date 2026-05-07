package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mrinjamul/twinhunter/core"
	"github.com/mrinjamul/twinhunter/models"
)

func testRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..")
}

func TestFindAndCleanWorkflow(t *testing.T) {
	root := testRoot()
	reportPath := filepath.Join(root, "test_report_workflow.json")
	defer os.Remove(reportPath)

	dupDir := filepath.Join(root, "test_data", "duplicates")

	files, err := core.Scan(core.ScanConfig{
		Path:      dupDir,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	if len(files) != 5 {
		t.Errorf("expected 5 files, got %d", len(files))
	}

	groups := core.FindDuplicates(files, 2)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}

	report := core.BuildReport(files, groups, dupDir)
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
	root := testRoot()
	mixedDir := filepath.Join(root, "test_data", "mixed_sizes")

	tests := []struct {
		name       string
		cfg        core.ScanConfig
		wantFiles  int
		wantGroups int
	}{
		{
			name:       "all files",
			cfg:        core.ScanConfig{Path: mixedDir, Recursive: true},
			wantFiles:  4,
			wantGroups: 1,
		},
		{
			name:       "min size 100k",
			cfg:        core.ScanConfig{Path: mixedDir, Recursive: true, MinSize: 1024 * 100},
			wantFiles:  3,
			wantGroups: 1,
		},
		{
			name:       "max size 1k",
			cfg:        core.ScanConfig{Path: mixedDir, Recursive: true, MaxSize: 1024},
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

			groups := core.FindDuplicates(files, 2)
			if len(groups) != tt.wantGroups {
				t.Errorf("got %d groups, want %d", len(groups), tt.wantGroups)
			}
		})
	}
}

func TestExcludeDirs(t *testing.T) {
	root := testRoot()
	excludeDir := filepath.Join(root, "test_data", "exclude_test")

	files, err := core.Scan(core.ScanConfig{
		Path:      excludeDir,
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
	root := testRoot()
	emptyDir := filepath.Join(root, "test_data", "empty_dir")

	files, err := core.Scan(core.ScanConfig{
		Path:      emptyDir,
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
	root := testRoot()
	dupDir := filepath.Join(root, "test_data", "duplicates")

	files, err := core.Scan(core.ScanConfig{
		Path:      dupDir,
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
	root := testRoot()
	testDir := filepath.Join(root, "test_data")

	files, err := core.Scan(core.ScanConfig{
		Path:      testDir,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	groups := core.FindDuplicates(files, 2)

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
	root := testRoot()
	reportPath := filepath.Join(root, "test_report_io.json")
	defer os.Remove(reportPath)

	dupDir := filepath.Join(root, "test_data", "duplicates")
	files, _ := core.Scan(core.ScanConfig{Path: dupDir, Recursive: true})
	groups := core.FindDuplicates(files, 2)
	report := core.BuildReport(files, groups, dupDir)

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
	root := testRoot()
	dupDir := filepath.Join(root, "test_data", "duplicates")

	files, err := core.Scan(core.ScanConfig{
		Path:      dupDir,
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	groups := core.FindDuplicates(files, 2)
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
