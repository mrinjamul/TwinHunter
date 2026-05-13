package core

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrinjamul/twinhunter/models"
)

func createTestReport(t *testing.T) models.Report {
	t.Helper()

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	return models.Report{
		ScanPath:    "/test/path",
		TotalFiles:  5,
		TotalSize:   10000,
		UniqueFiles: 3,
		DupGroups: []models.DuplicateGroup{
			{
				Hash: "abc123def456",
				Size: 2000,
				Files: []models.FileInfo{
					{Path: "/test/path/file1.jpg", Name: "file1.jpg", Size: 2000, ModTime: base},
					{Path: "/test/path/file2.jpg", Name: "file2.jpg", Size: 2000, ModTime: base.Add(1 * time.Hour)},
				},
			},
			{
				Hash: "xyz789abc012",
				Size: 1500,
				Files: []models.FileInfo{
					{Path: "/test/path/doc1.txt", Name: "doc1.txt", Size: 1500, ModTime: base},
					{Path: "/test/path/doc2.txt", Name: "doc2.txt", Size: 1500, ModTime: base.Add(2 * time.Hour)},
					{Path: "/test/path/doc3.txt", Name: "doc3.txt", Size: 1500, ModTime: base.Add(3 * time.Hour)},
				},
			},
		},
		DupFiles:    3,
		WastedSpace: 5000,
		ScannedAt:   base,
	}
}

func TestExportJSON(t *testing.T) {
	report := createTestReport(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	if err := ExportJSON(report, path); err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read exported file: %v", err)
	}

	var imported models.Report
	if err := json.Unmarshal(data, &imported); err != nil {
		t.Fatalf("exported JSON is invalid: %v", err)
	}

	if imported.ScanPath != report.ScanPath {
		t.Errorf("ScanPath mismatch: got %q, want %q", imported.ScanPath, report.ScanPath)
	}
	if imported.TotalFiles != report.TotalFiles {
		t.Errorf("TotalFiles mismatch: got %d, want %d", imported.TotalFiles, report.TotalFiles)
	}
	if imported.DupFiles != report.DupFiles {
		t.Errorf("DupFiles mismatch: got %d, want %d", imported.DupFiles, report.DupFiles)
	}
	if len(imported.DupGroups) != len(report.DupGroups) {
		t.Errorf("DupGroups count mismatch: got %d, want %d", len(imported.DupGroups), len(report.DupGroups))
	}
}

func TestImportJSON(t *testing.T) {
	report := createTestReport(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")

	if err := ExportJSON(report, path); err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	imported, err := ImportJSON(path)
	if err != nil {
		t.Fatalf("ImportJSON failed: %v", err)
	}

	if imported.ScanPath != report.ScanPath {
		t.Errorf("ScanPath mismatch: got %q, want %q", imported.ScanPath, report.ScanPath)
	}
	if len(imported.DupGroups) != len(report.DupGroups) {
		t.Errorf("DupGroups count mismatch: got %d, want %d", len(imported.DupGroups), len(report.DupGroups))
	}
}

func TestImportJSON_NonExistent(t *testing.T) {
	_, err := ImportJSON("/nonexistent/path/report.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestImportJSON_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.json")

	if err := os.WriteFile(path, []byte("not valid json{}{{"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ImportJSON(path)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestExportCSV(t *testing.T) {
	report := createTestReport(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.csv")

	if err := ExportCSV(report, path); err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to read CSV: %v", err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("failed to parse CSV: %v", err)
	}

	expectedRows := 1 + 2 + 3
	if len(records) != expectedRows {
		t.Errorf("expected %d rows (header + 2 groups), got %d", expectedRows, len(records))
	}

	expectedHeader := []string{"group", "hash", "size", "path", "is_duplicate", "mod_time"}
	for i, h := range expectedHeader {
		if records[0][i] != h {
			t.Errorf("header[%d] mismatch: got %q, want %q", i, records[0][i], h)
		}
	}
}

func TestExportCSV_EmptyReport(t *testing.T) {
	report := models.Report{
		ScanPath: "/empty",
		DupGroups: []models.DuplicateGroup{},
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")

	if err := ExportCSV(report, path); err != nil {
		t.Fatalf("ExportCSV failed for empty report: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "group,hash,size,path") {
		t.Error("CSV header should be present even in empty report")
	}
}

func TestExportHTML(t *testing.T) {
	report := createTestReport(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.html")

	if err := ExportHTML(report, path); err != nil {
		t.Fatalf("ExportHTML failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read HTML: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "<!DOCTYPE html>") {
		t.Error("HTML should contain DOCTYPE")
	}
	if !strings.Contains(content, "TwinHunter") {
		t.Error("HTML should contain TwinHunter title")
	}
	if !strings.Contains(content, "tag-keep") {
		t.Error("HTML should contain tag-keep CSS class")
	}
	if !strings.Contains(content, "tag-dup") {
		t.Error("HTML should contain tag-dup CSS class")
	}
	if !strings.Contains(content, report.ScanPath) {
		t.Error("HTML should contain scan path")
	}
}

func TestExportHTML_SpecialCharacters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.html")

	report := models.Report{
		ScanPath:  "/path/with <script>alert('xss')</script>",
		TotalFiles: 2,
		DupGroups: []models.DuplicateGroup{
			{
				Hash: "abc",
				Size: 100,
				Files: []models.FileInfo{
					{Path: "/file/with&special<chars>", Name: "test", Size: 100, ModTime: time.Now()},
					{Path: "/file2.jpg", Name: "test2", Size: 100, ModTime: time.Now()},
				},
			},
		},
	}

	if err := ExportHTML(report, path); err != nil {
		t.Fatalf("ExportHTML failed: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if strings.Contains(content, "<script>") {
		t.Error("HTML should escape <script> tags")
	}
	if strings.Contains(content, "alert('xss')") {
		t.Error("HTML should escape JavaScript")
	}
}

func TestImportCSV_FullRoundTrip(t *testing.T) {
	report := createTestReport(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.csv")

	if err := ExportCSV(report, path); err != nil {
		t.Fatalf("ExportCSV failed: %v", err)
	}

	imported, err := ImportCSV(path)
	if err != nil {
		t.Fatalf("ImportCSV failed: %v", err)
	}

	if len(imported.DupGroups) != len(report.DupGroups) {
		t.Errorf("DupGroups count mismatch: got %d, want %d", len(imported.DupGroups), len(report.DupGroups))
	}
	if imported.DupFiles != report.DupFiles {
		t.Errorf("DupFiles mismatch: got %d, want %d", imported.DupFiles, report.DupFiles)
	}
	if imported.WastedSpace != report.WastedSpace {
		t.Errorf("WastedSpace mismatch: got %d, want %d", imported.WastedSpace, report.WastedSpace)
	}
	if len(imported.DupGroups) > 0 {
		if imported.DupGroups[0].Size != report.DupGroups[0].Size {
			t.Errorf("first group Size mismatch: got %d, want %d", imported.DupGroups[0].Size, report.DupGroups[0].Size)
		}
		if imported.DupGroups[0].Hash != report.DupGroups[0].Hash {
			t.Errorf("first group Hash mismatch: got %s, want %s", imported.DupGroups[0].Hash, report.DupGroups[0].Hash)
		}
	}
}

func TestImportCSV_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.csv")

	if err := os.WriteFile(path, []byte("group,hash,size,path,is_duplicate,mod_time\n"), 0644); err != nil {
		t.Fatal(err)
	}

	imported, err := ImportCSV(path)
	if err != nil {
		t.Fatalf("ImportCSV should not error on empty CSV: %v", err)
	}
	if len(imported.DupGroups) != 0 {
		t.Errorf("expected 0 groups for empty CSV, got %d", len(imported.DupGroups))
	}
}

func TestImportCSV_NonExistent(t *testing.T) {
	_, err := ImportCSV("/nonexistent/path/report.csv")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestImportCSV_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.csv")

	if err := os.WriteFile(path, []byte("group,hash,size,path,is_duplicate,mod_time\nnot,a,valid,row"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ImportCSV(path)
	if err == nil {
		t.Error("expected error for malformed CSV")
	}
}

func TestExportJSON_ToReadOnlyDirectory(t *testing.T) {
	report := createTestReport(t)
	dir := t.TempDir()

	readOnlyDir := filepath.Join(dir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(readOnlyDir, "report.json")
	err := ExportJSON(report, path)

	if err == nil {
		t.Error("expected error when writing to read-only directory")
	}
}

func TestImportHTML_FullRoundTrip(t *testing.T) {
	report := createTestReport(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "report.html")

	if err := ExportHTML(report, path); err != nil {
		t.Fatalf("ExportHTML failed: %v", err)
	}

	imported, err := ImportHTML(path)
	if err != nil {
		t.Fatalf("ImportHTML failed: %v", err)
	}

	if len(imported.DupGroups) != len(report.DupGroups) {
		t.Errorf("DupGroups count mismatch: got %d, want %d", len(imported.DupGroups), len(report.DupGroups))
	}
	if imported.DupFiles != report.DupFiles {
		t.Errorf("DupFiles mismatch: got %d, want %d", imported.DupFiles, report.DupFiles)
	}
	if imported.WastedSpace != report.WastedSpace {
		t.Errorf("WastedSpace mismatch: got %d, want %d", imported.WastedSpace, report.WastedSpace)
	}
	if len(imported.DupGroups) > 0 {
		if imported.DupGroups[0].Size != report.DupGroups[0].Size {
			t.Errorf("first group Size mismatch: got %d, want %d", imported.DupGroups[0].Size, report.DupGroups[0].Size)
		}
		if imported.DupGroups[0].Hash != report.DupGroups[0].Hash {
			t.Errorf("first group Hash mismatch: got %s, want %s", imported.DupGroups[0].Hash, report.DupGroups[0].Hash)
		}
	}
}

func TestImportHTML_NoEmbeddedCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_csv.html")

	if err := os.WriteFile(path, []byte("<html><body>No CSV here</body></html>"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ImportHTML(path)
	if err == nil {
		t.Error("expected error for HTML without embedded CSV")
	}
}

func TestImportHTML_MalformedCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed_csv.html")

	content := `<html><body>Report</body></html>
<!--
TWINHUNTER_CSV:
group,hash,size,path,is_duplicate,mod_time
not,a,valid,row
-->`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ImportHTML(path)
	if err == nil {
		t.Error("expected error for malformed CSV in HTML")
	}
}

func TestImportHTML_NonExistent(t *testing.T) {
	_, err := ImportHTML("/nonexistent/path/report.html")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
