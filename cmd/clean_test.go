package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrinjamul/twinhunter/core"
	"github.com/mrinjamul/twinhunter/models"
	"github.com/spf13/cobra"
)

func createTestCleanEnv(t *testing.T) (dir string, reportPath string) {
	t.Helper()

	dir = t.TempDir()

	content := "duplicate content for testing"

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")
	file3 := filepath.Join(dir, "file3.txt")

	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file3, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	report := models.Report{
		ScanPath:   dir,
		TotalFiles: 3,
		TotalSize:  int64(len(content) * 3),
		DupGroups: []models.DuplicateGroup{
			{
				Hash: "abc123",
				Size: int64(len(content)),
				Files: []models.FileInfo{
					{Path: file1, Name: "file1.txt", Size: int64(len(content)), ModTime: base},
					{Path: file2, Name: "file2.txt", Size: int64(len(content)), ModTime: base.Add(1 * time.Hour)},
					{Path: file3, Name: "file3.txt", Size: int64(len(content)), ModTime: base.Add(2 * time.Hour)},
				},
			},
		},
		DupFiles: 2,
	}

	reportPath = filepath.Join(t.TempDir(), "clean_test_report.json")
	if err := core.ExportJSON(report, reportPath); err != nil {
		t.Fatal(err)
	}

	return dir, reportPath
}

func saveCleanGlobals() (keep string, delete bool, link string, backup string, dryrun bool) {
	return cleanKeep, cleanDelete, cleanLink, cleanBackupDir, cleanDryRun
}

func restoreCleanGlobals(keep string, delete bool, link string, backup string, dryrun bool) {
	cleanKeep = keep
	cleanDelete = delete
	cleanLink = link
	cleanBackupDir = backup
	cleanDryRun = dryrun
}

func TestClean_DryRun(t *testing.T) {
	oldKeep, oldDelete, oldLink, oldBackup, oldDryRun := saveCleanGlobals()
	defer restoreCleanGlobals(oldKeep, oldDelete, oldLink, oldBackup, oldDryRun)

	dir, reportPath := createTestCleanEnv(t)

	cleanKeep = "oldest"
	cleanDelete = true
	cleanDryRun = true

	cmd := &cobra.Command{}
	args := []string{reportPath}

	if err := runClean(cmd, args); err != nil {
		t.Fatalf("runClean failed: %v", err)
	}

	files, _ := os.ReadDir(dir)
	if len(files) != 3 {
		t.Errorf("expected 3 files after dry-run, got %d", len(files))
	}
}

func TestClean_InvalidKeepStrategy(t *testing.T) {
	oldKeep, oldDelete, oldLink, oldBackup, oldDryRun := saveCleanGlobals()
	defer restoreCleanGlobals(oldKeep, oldDelete, oldLink, oldBackup, oldDryRun)

	_, reportPath := createTestCleanEnv(t)

	cleanKeep = "invalid"
	cleanDelete = true

	cmd := &cobra.Command{}
	args := []string{reportPath}

	err := runClean(cmd, args)
	if err == nil {
		t.Error("expected error for invalid keep strategy")
	}
}

func TestClean_NoAction(t *testing.T) {
	oldKeep, oldDelete, oldLink, oldBackup, oldDryRun := saveCleanGlobals()
	defer restoreCleanGlobals(oldKeep, oldDelete, oldLink, oldBackup, oldDryRun)

	_, reportPath := createTestCleanEnv(t)

	cleanKeep = "oldest"
	cleanDelete = false
	cleanLink = ""
	cleanBackupDir = ""

	cmd := &cobra.Command{}
	args := []string{reportPath}

	err := runClean(cmd, args)
	if err == nil {
		t.Error("expected error when no action specified")
	}
}

func TestClean_NonExistentReport(t *testing.T) {
	oldKeep, oldDelete, oldLink, oldBackup, oldDryRun := saveCleanGlobals()
	defer restoreCleanGlobals(oldKeep, oldDelete, oldLink, oldBackup, oldDryRun)

	cleanKeep = "oldest"
	cleanDelete = true

	cmd := &cobra.Command{}
	args := []string{"/nonexistent/path/report.json"}

	err := runClean(cmd, args)
	if err == nil {
		t.Error("expected error for nonexistent report file")
	}
}

func TestClean_EmptyReport(t *testing.T) {
	oldKeep, oldDelete, oldLink, oldBackup, oldDryRun := saveCleanGlobals()
	defer restoreCleanGlobals(oldKeep, oldDelete, oldLink, oldBackup, oldDryRun)

	report := models.Report{
		ScanPath:  "/empty",
		DupGroups: []models.DuplicateGroup{},
	}

	reportPath := filepath.Join(t.TempDir(), "empty_report.json")
	if err := core.ExportJSON(report, reportPath); err != nil {
		t.Fatal(err)
	}

	cleanKeep = "oldest"
	cleanDelete = true

	cmd := &cobra.Command{}
	args := []string{reportPath}

	if err := runClean(cmd, args); err != nil {
		t.Fatalf("runClean should handle empty report: %v", err)
	}
}
