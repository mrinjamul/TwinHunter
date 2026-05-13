package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mrinjamul/twinhunter/models"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	return path
}

func TestHashBlake3(t *testing.T) {
	dir := setupTestDir(t)
	p := writeFile(t, dir, "test.txt", "hello world")

	hash, err := HashBlake3(p)
	if err != nil {
		t.Fatalf("HashBlake3 error: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hash, got %d", len(hash))
	}

	hash2, err := HashBlake3(p)
	if err != nil {
		t.Fatalf("HashBlake3 error: %v", err)
	}
	if hash != hash2 {
		t.Error("same file should produce same hash")
	}
}

func TestHashSHA256(t *testing.T) {
	dir := setupTestDir(t)
	p := writeFile(t, dir, "test.txt", "hello world")

	hash, err := HashSHA256(p)
	if err != nil {
		t.Fatalf("HashSHA256 error: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("expected 64-char hash, got %d", len(hash))
	}
}

func TestHashBlake3NonExistent(t *testing.T) {
	_, err := HashBlake3("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestDeleteFile(t *testing.T) {
	dir := setupTestDir(t)
	p := writeFile(t, dir, "todelete.txt", "content")

	if err := DeleteFile(p); err != nil {
		t.Fatalf("DeleteFile error: %v", err)
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Error("file should not exist after delete")
	}
}

func TestDeleteFileNonExistent(t *testing.T) {
	err := DeleteFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReplaceWithHardLink(t *testing.T) {
	dir := setupTestDir(t)
	orig := writeFile(t, dir, "original.txt", "original content")
	dup := writeFile(t, dir, "duplicate.txt", "duplicate content")

	if err := ReplaceWithHardLink(orig, dup); err != nil {
		t.Fatalf("ReplaceWithHardLink error: %v", err)
	}

	origInfo, _ := os.Stat(orig)
	dupInfo, _ := os.Stat(dup)

	if !os.SameFile(origInfo, dupInfo) {
		t.Error("files should share the same inode after hard link")
	}
}

func TestReplaceWithSoftLink(t *testing.T) {
	dir := setupTestDir(t)
	orig := writeFile(t, dir, "original.txt", "original content")
	dup := writeFile(t, dir, "duplicate.txt", "duplicate content")

	if err := ReplaceWithSoftLink(orig, dup); err != nil {
		t.Fatalf("ReplaceWithSoftLink error: %v", err)
	}

	linkTarget, err := os.Readlink(dup)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if filepath.Base(linkTarget) != "original.txt" {
		t.Errorf("symlink should point to original, got %s", linkTarget)
	}
}

func TestReplaceWithLinkToSelf(t *testing.T) {
	dir := setupTestDir(t)
	p := writeFile(t, dir, "file.txt", "content")

	err := ReplaceWithHardLink(p, p)
	if err == nil {
		t.Error("expected error when linking file to itself")
	}

	err = ReplaceWithSoftLink(p, p)
	if err == nil {
		t.Error("expected error when linking file to itself")
	}
}

func TestGroupBySize(t *testing.T) {
	files := []models.FileInfo{
		{Path: "/a", Size: 10},
		{Path: "/b", Size: 10},
		{Path: "/c", Size: 20},
		{Path: "/d", Size: 30},
		{Path: "/e", Size: 30},
		{Path: "/f", Size: 30},
	}

	groups := GroupBySize(files)

	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
	if len(groups[10]) != 2 {
		t.Errorf("expected 2 files in size-10 group, got %d", len(groups[10]))
	}
	if len(groups[30]) != 3 {
		t.Errorf("expected 3 files in size-30 group, got %d", len(groups[30]))
	}
	if _, ok := groups[20]; ok {
		t.Error("size-20 should not be a group (only 1 file)")
	}
}

func TestApplyKeepStrategyOldest(t *testing.T) {
	now := time.Now()
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/new", ModTime: now.Add(time.Hour)},
			{Path: "/old", ModTime: now.Add(-time.Hour)},
			{Path: "/middle", ModTime: now},
		},
	}

	keep, remove := ApplyKeepStrategy(group, "oldest")
	if keep.Path != "/old" {
		t.Errorf("expected oldest file '/old', got '%s'", keep.Path)
	}
	if len(remove) != 2 {
		t.Errorf("expected 2 files to remove, got %d", len(remove))
	}
}

func TestApplyKeepStrategyNewest(t *testing.T) {
	now := time.Now()
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/new", ModTime: now.Add(time.Hour)},
			{Path: "/old", ModTime: now.Add(-time.Hour)},
		},
	}

	keep, _ := ApplyKeepStrategy(group, "newest")
	if keep.Path != "/new" {
		t.Errorf("expected newest file '/new', got '%s'", keep.Path)
	}
}

func TestApplyKeepStrategyShortest(t *testing.T) {
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/very/long/path/to/file.txt"},
			{Path: "/short.txt"},
		},
	}

	keep, _ := ApplyKeepStrategy(group, "shortest")
	if keep.Path != "/short.txt" {
		t.Errorf("expected shortest path '/short.txt', got '%s'", keep.Path)
	}
}

func TestApplyKeepStrategyDefault(t *testing.T) {
	group := models.DuplicateGroup{
		Files: []models.FileInfo{
			{Path: "/second"},
			{Path: "/first"},
		},
	}

	keep, remove := ApplyKeepStrategy(group, "unknown")
	if keep.Path != "/second" {
		t.Errorf("expected first file '/second', got '%s'", keep.Path)
	}
	if len(remove) != 1 {
		t.Errorf("expected 1 file to remove, got %d", len(remove))
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes  int64
		expect string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		got := FormatSize(tt.bytes)
		if got != tt.expect {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.expect)
		}
	}
}

func TestBuildReport(t *testing.T) {
	files := []models.FileInfo{
		{Path: "/a", Size: 100},
		{Path: "/b", Size: 100},
		{Path: "/c", Size: 100},
		{Path: "/d", Size: 200},
	}

	groups := []models.DuplicateGroup{
		{Hash: "abc", Size: 100, Files: files[:3]},
	}

	report := BuildReport(files, groups, "/test")

	if report.TotalFiles != 4 {
		t.Errorf("expected 4 total files, got %d", report.TotalFiles)
	}
	if report.DupFiles != 2 {
		t.Errorf("expected 2 dup files, got %d", report.DupFiles)
	}
	if report.WastedSpace != 200 {
		t.Errorf("expected 200 wasted space, got %d", report.WastedSpace)
	}
	if report.ScanPath != "/test" {
		t.Errorf("expected scan path '/test', got '%s'", report.ScanPath)
	}
}

func TestMoveToBackup(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup")

	srcFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(srcFile, []byte("test content"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := MoveToBackup(srcFile, backupDir); err != nil {
		t.Fatalf("MoveToBackup failed: %v", err)
	}

	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Error("source file should not exist after backup")
	}

	relPathFromRoot, _ := filepath.Rel("/", srcFile)
	expectedDest := filepath.Join(backupDir, relPathFromRoot)
	
	if _, err := os.Stat(expectedDest); err != nil {
		t.Errorf("backup file should exist at %s: %v", expectedDest, err)
	}

	data, _ := os.ReadFile(expectedDest)
	if string(data) != "test content" {
		t.Errorf("backup content mismatch: got %q", string(data))
	}
}

func TestApplyAction(t *testing.T) {
	dir := t.TempDir()

	keepFile := filepath.Join(dir, "keep.txt")
	dupFile := filepath.Join(dir, "dup.txt")

	if err := os.WriteFile(keepFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dupFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	keep := models.FileInfo{Path: keepFile}
	dup := models.FileInfo{Path: dupFile}

	if err := ApplyAction(ActionDelete, keep, dup, ""); err != nil {
		t.Fatalf("ApplyAction delete failed: %v", err)
	}

	if _, err := os.Stat(dupFile); !os.IsNotExist(err) {
		t.Error("dup file should be deleted")
	}
}

func TestApplyAction_None(t *testing.T) {
	keep := models.FileInfo{Path: "/keep"}
	dup := models.FileInfo{Path: "/dup"}

	err := ApplyAction(ActionNone, keep, dup, "")
	if err != nil {
		t.Errorf("ActionNone should return nil, got %v", err)
	}
}

func TestCountHardLinks(t *testing.T) {
	files := []models.FileInfo{
		{Path: "/a", IsHardLink: false},
		{Path: "/b", IsHardLink: true},
		{Path: "/c", IsHardLink: true},
		{Path: "/d", IsHardLink: false},
	}

	count := CountHardLinks(files)
	if count != 2 {
		t.Errorf("CountHardLinks = %d, want 2", count)
	}

	countEmpty := CountHardLinks([]models.FileInfo{})
	if countEmpty != 0 {
		t.Errorf("CountHardLinks(empty) = %d, want 0", countEmpty)
	}
}

func TestPlatformName(t *testing.T) {
	name := PlatformName()
	if name == "" {
		t.Error("PlatformName should not be empty")
	}
}

func TestFormatSize_TB(t *testing.T) {
	oneTB := int64(1024) * 1024 * 1024 * 1024
	got := FormatSize(oneTB)
	if got != "1.0 TB" {
		t.Errorf("FormatSize(1TB) = %q, want \"1.0 TB\"", got)
	}
}

func TestHashPipeline_WorkerEdgeCases(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "test1.txt")
	file2 := filepath.Join(dir, "test2.txt")

	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	results := HashPipeline([]string{file1, file2}, "blake3", 0, nil)
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	results2 := HashPipeline([]string{file1, file2}, "blake3", 10, nil)
	if len(results2) != 2 {
		t.Errorf("expected 2 results with workers=10, got %d", len(results2))
	}

	results3 := HashPipeline([]string{file1, file2}, "blake3", -1, nil)
	if len(results3) != 2 {
		t.Errorf("expected 2 results with workers=-1, got %d", len(results3))
	}
}

func TestHashSHA256_NonExistent(t *testing.T) {
	_, err := HashSHA256("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
