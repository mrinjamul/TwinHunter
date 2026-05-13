package core

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mrinjamul/twinhunter/models"
)

// ExportJSON writes a Report to a JSON file.
func ExportJSON(report models.Report, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ImportJSON reads a Report from a JSON file.
func ImportJSON(path string) (models.Report, error) {
	var report models.Report
	data, err := os.ReadFile(path)
	if err != nil {
		return report, err
	}
	err = json.Unmarshal(data, &report)
	return report, err
}

// importCSV parses CSV data from a reader into a Report.
func importCSV(r io.Reader) (models.Report, error) {
	cr := csv.NewReader(r)
	records, err := cr.ReadAll()
	if err != nil {
		return models.Report{}, fmt.Errorf("failed to parse CSV: %w", err)
	}

	if len(records) < 2 {
		return models.Report{}, nil
	}

	expectedHeader := []string{"group", "hash", "size", "path", "is_duplicate", "mod_time"}
	for i, h := range expectedHeader {
		if len(records[0]) <= i || records[0][i] != h {
			return models.Report{}, fmt.Errorf("invalid CSV header: expected %q at column %d, got %q", h, i, records[0][i])
		}
	}

	type csvRow struct {
		group       int
		hash        string
		size        int64
		path        string
		isDuplicate bool
		modTime     time.Time
	}

	var rows []csvRow
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 6 {
			return models.Report{}, fmt.Errorf("row %d: expected 6 columns, got %d", i+1, len(record))
		}

		group, err := strconv.Atoi(record[0])
		if err != nil {
			return models.Report{}, fmt.Errorf("row %d: invalid group number: %q", i+1, record[0])
		}

		size, err := strconv.ParseInt(record[2], 10, 64)
		if err != nil {
			return models.Report{}, fmt.Errorf("row %d: invalid size: %q", i+1, record[2])
		}

		isDup, err := strconv.ParseBool(record[4])
		if err != nil {
			return models.Report{}, fmt.Errorf("row %d: invalid is_duplicate: %q", i+1, record[4])
		}

		modTime, err := time.Parse("2006-01-02 15:04:05", record[5])
		if err != nil {
			return models.Report{}, fmt.Errorf("row %d: invalid mod_time: %q", i+1, record[5])
		}

		rows = append(rows, csvRow{
			group:       group,
			hash:        record[1],
			size:        size,
			path:        record[3],
			isDuplicate: isDup,
			modTime:     modTime,
		})
	}

	groupMap := make(map[int][]csvRow)
	for _, row := range rows {
		groupMap[row.group] = append(groupMap[row.group], row)
	}

	var groups []models.DuplicateGroup
	for _, grp := range groupMap {
		if len(grp) < 2 {
			continue
		}

		var keepFiles, dupFiles []models.FileInfo
		for _, row := range grp {
			fi := models.FileInfo{
				Path:    row.path,
				Name:    filepath.Base(row.path),
				Size:    row.size,
				ModTime: row.modTime,
			}
			if row.isDuplicate {
				dupFiles = append(dupFiles, fi)
			} else {
				keepFiles = append(keepFiles, fi)
			}
		}

		groups = append(groups, models.DuplicateGroup{
			Hash:  grp[0].hash,
			Size:  grp[0].size,
			Files: append(keepFiles, dupFiles...),
		})
	}

	var dupCount int
	var wastedSpace int64
	for _, g := range groups {
		dupCount += len(g.Files) - 1
		wastedSpace += g.Size * int64(len(g.Files)-1)
	}

	return models.Report{
		DupGroups:   groups,
		DupFiles:    dupCount,
		WastedSpace: wastedSpace,
		ScannedAt:   time.Now(),
	}, nil
}

// ImportCSV reads a Report from a CSV file.
func ImportCSV(path string) (models.Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return models.Report{}, err
	}
	defer f.Close()
	return importCSV(f)
}

// formatCSV returns the CSV representation of a Report as a string.
func formatCSV(report models.Report) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{"group", "hash", "size", "path", "is_duplicate", "mod_time"})
	for gi, g := range report.DupGroups {
		for fi, f := range g.Files {
			w.Write([]string{
				fmt.Sprintf("%d", gi+1),
				g.Hash,
				fmt.Sprintf("%d", f.Size),
				f.Path,
				fmt.Sprintf("%t", fi > 0),
				f.ModTime.Format("2006-01-02 15:04:05"),
			})
		}
	}
	w.Flush()
	return buf.String()
}

// ExportCSV writes a Report to a CSV file, one row per file.
func ExportCSV(report models.Report, path string) error {
	return os.WriteFile(path, []byte(formatCSV(report)), 0o644)
}

// errWriter wraps an io.Writer and tracks the first write error.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) write(s string) {
	if ew.err != nil {
		return
	}
	_, ew.err = io.WriteString(ew.w, s)
}

func (ew *errWriter) writef(format string, args ...interface{}) {
	if ew.err != nil {
		return
	}
	_, ew.err = fmt.Fprintf(ew.w, format, args...)
}

// ExportHTML writes a Report to a styled HTML file.
func ExportHTML(report models.Report, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := &errWriter{w: f}

	w.write(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>TwinHunter Report — `)
	w.write(html.EscapeString(report.ScanPath))
	w.write(`</title>
<style>
body{font-family:system-ui,sans-serif;margin:2em;background:#f5f5f5;color:#222}
h1{margin-bottom:.2em}
.summary{background:#fff;padding:1em;border-radius:8px;margin-bottom:1.5em}
.summary span{margin-right:1.5em}
.group{background:#fff;padding:1em;margin-bottom:1em;border-radius:8px;border-left:4px solid #e74c3c}
.group h3{margin:0 0 .5em;font-size:1em}
table{width:100%%;border-collapse:collapse}
th,td{text-align:left;padding:.4em .8em;border-bottom:1px solid #eee}
th{background:#fafafa;font-size:.85em;color:#666}
.dup{background:#fef0f0}
.keep{background:#f0fef0}
.tag{padding:2px 8px;border-radius:4px;font-size:.75em;font-weight:600}
.tag-keep{background:#d4edda;color:#155724}
.tag-dup{background:#f8d7da;color:#721c24}
</style></head><body>
<h1>TwinHunter Duplicate Report</h1>
<div class="summary">
<span><b>Scan:</b> `)
	w.write(html.EscapeString(report.ScanPath))
	w.writef(`</span>
<span><b>Files:</b> %d</span>
<span><b>Dup Groups:</b> %d</span>
<span><b>Dup Files:</b> %d</span>
<span><b>Wasted:</b> %s</span>
</div>
`, report.TotalFiles, len(report.DupGroups), report.DupFiles, FormatSize(report.WastedSpace))

	for i, g := range report.DupGroups {
		w.writef(`<div class="group"><h3>Group %d — %d copies, %s each</h3><table><tr><th></th><th>Path</th><th>Size</th><th>Modified</th></tr>`, i+1, len(g.Files), FormatSize(g.Size))
		for j, file := range g.Files {
			cls := "keep"
			tag := `<span class="tag tag-keep">keep</span>`
			if j > 0 {
				cls = "dup"
				tag = `<span class="tag tag-dup">dup</span>`
			}
			w.writef(`<tr class="%s"><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`, cls, tag, html.EscapeString(file.Path), FormatSize(file.Size), file.ModTime.Format("2006-01-02 15:04"))
		}
		w.write(`</table></div>`)
	}

	w.write("</body></html>\n")
	csvData := formatCSV(report)
	w.writef("<!--\nTWINHUNTER_CSV:\n%s-->\n", csvData)

	if w.err != nil {
		return fmt.Errorf("failed to write HTML report: %w", w.err)
	}
	return nil
}

// ImportHTML reads a Report from an HTML file with embedded CSV.
func ImportHTML(path string) (models.Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return models.Report{}, err
	}

	content := string(data)
	marker := "TWINHUNTER_CSV:\n"
	start := strings.Index(content, marker)
	if start < 0 {
		return models.Report{}, fmt.Errorf("no embedded CSV data found in HTML report")
	}
	start += len(marker)

	end := strings.Index(content[start:], "-->")
	if end < 0 {
		return models.Report{}, fmt.Errorf("no closing --> found for embedded CSV data")
	}

	csvText := content[start : start+end]
	return importCSV(strings.NewReader(csvText))
}
