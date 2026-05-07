package core

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"os"

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

// ExportCSV writes a Report to a CSV file, one row per file.
func ExportCSV(report models.Report, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{"group", "hash", "size", "path", "is_duplicate", "mod_time"}
	if err := w.Write(header); err != nil {
		return err
	}

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

	return w.Error()
}

// ExportHTML writes a Report to a styled HTML file.
func ExportHTML(report models.Report, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>TwinHunter Report — %s</title>
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
<span><b>Scan:</b> %s</span>
<span><b>Files:</b> %d</span>
<span><b>Dup Groups:</b> %d</span>
<span><b>Dup Files:</b> %d</span>
<span><b>Wasted:</b> %s</span>
</div>
`, html.EscapeString(report.ScanPath), html.EscapeString(report.ScanPath), report.TotalFiles, len(report.DupGroups), report.DupFiles, FormatSize(report.WastedSpace))

	for i, g := range report.DupGroups {
		fmt.Fprintf(f, `<div class="group"><h3>Group %d — %d copies, %s each</h3><table><tr><th></th><th>Path</th><th>Size</th><th>Modified</th></tr>`, i+1, len(g.Files), FormatSize(g.Size))
		for j, file := range g.Files {
			cls := "keep"
			tag := `<span class="tag tag-keep">keep</span>`
			if j > 0 {
				cls = "dup"
				tag = `<span class="tag tag-dup">dup</span>`
			}
			fmt.Fprintf(f, `<tr class="%s"><td>%s</td><td>%s</td><td>%s</td><td>%s</td></tr>`, cls, tag, html.EscapeString(file.Path), FormatSize(file.Size), file.ModTime.Format("2006-01-02 15:04"))
		}
		fmt.Fprint(f, `</table></div>`)
	}

	fmt.Fprintf(f, "</body></html>")
	return f.Close()
}
