package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrinjamul/twinhunter/core"
	"github.com/mrinjamul/twinhunter/models"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var (
	findRecursive  bool
	findMinSize    string
	findMaxSize    string
	findExclude    string
	findExcludeRe  string
	findExcludeDir string
	findWorkers    int
	findOutput     string
	findFormat     string
	findKeep       string
	findLink       string
	findDelete     bool
	findDryRun     bool
	findYes        bool
	findVerbose    bool
	findSort       string
)

var findCmd = &cobra.Command{
	Use:   "find [path]",
	Short: "Find duplicate files",
	Long: `Scan a directory for duplicate files.
If no path is given, the current directory is used.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runFind,
}

func init() {
	rootCmd.AddCommand(findCmd)

	findCmd.Flags().BoolVarP(&findRecursive, "recursive", "r", false, "scan subdirectories recursively")
	findCmd.Flags().StringVarP(&findMinSize, "min-size", "m", "0", "minimum file size (e.g. 1M, 100K)")
	findCmd.Flags().StringVarP(&findMaxSize, "max-size", "M", "0", "maximum file size (e.g. 10M)")
	findCmd.Flags().StringVarP(&findExclude, "exclude", "x", "", "comma-separated glob patterns to exclude")
	findCmd.Flags().StringVar(&findExcludeRe, "exclude-regex", "", "comma-separated regex patterns to exclude")
	findCmd.Flags().StringVar(&findExcludeDir, "exclude-dir", "", "comma-separated directory names to skip (default: .git,node_modules,.svn,__pycache__)")
	findCmd.Flags().IntVarP(&findWorkers, "workers", "w", 0, "number of parallel hashing workers (0 = auto)")
	findCmd.Flags().StringVarP(&findOutput, "output", "o", "", "export report to file (format auto-detected from extension: .json, .csv, .html)")
	findCmd.Flags().StringVarP(&findFormat, "format", "f", "pretty", "terminal output: pretty, json, csv, silent")
	findCmd.Flags().BoolVarP(&findVerbose, "verbose", "v", false, "show detailed per-group output")
	findCmd.Flags().StringVar(&findSort, "sort", "size", "sort groups by: size, count, path")
	findCmd.Flags().BoolVarP(&findYes, "yes", "y", false, "skip confirmation prompts")
	findCmd.Flags().StringVarP(&findKeep, "keep", "k", "", "auto-keep strategy: oldest (default), newest, shortest")
	findCmd.Flags().StringVarP(&findLink, "link", "l", "", "replace duplicates with links: hard, soft")
	findCmd.Flags().BoolVarP(&findDelete, "delete", "d", false, "delete duplicate files")
	findCmd.Flags().BoolVarP(&findDryRun, "dry-run", "n", false, "preview without making changes")
}

func runFind(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("path does not exist: %s", absPath)
	}

	return runFindCLI(cmd, absPath)
}

func runFindCLI(cmd *cobra.Command, path string) error {
	minSize := parseSize(findMinSize)
	maxSize := parseSize(findMaxSize)
	exclude := splitCSV(findExclude)
	excludeRe := splitCSV(findExcludeRe)
	excludeDir := splitCSV(findExcludeDir)

	if findKeep != "" {
		switch findKeep {
		case "oldest", "newest", "shortest":
		default:
			return fmt.Errorf("invalid keep strategy: %q (valid: oldest, newest, shortest)", findKeep)
		}
	}

	cfg := core.ScanConfig{
		Path:         path,
		Recursive:    findRecursive,
		MinSize:      minSize,
		MaxSize:      maxSize,
		Exclude:      exclude,
		ExcludeRegex: excludeRe,
		ExcludeDir:   excludeDir,
		Workers:      findWorkers,
	}

	var bar *progressbar.ProgressBar
	if findFormat == "pretty" {
		fmt.Printf("Scanning: %s\n", path)
		if findRecursive {
			fmt.Println("Mode: recursive")
		}
		fmt.Println("Discovering files...")
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("Scanning files..."),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowCount(),
			progressbar.OptionSetTheme(progressbar.Theme{Saucer: "█", SaucerPadding: " ", BarStart: "[", BarEnd: "]"}),
		)
		cfg.OnProgress = func(stats models.ScanStats) {
			bar.Set(stats.FilesScanned)
		}
	}

	allFiles, err := core.Scan(cfg)
	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	if findFormat == "pretty" {
		bar.Finish()
		fmt.Fprintln(os.Stderr)
		hardLinks := core.CountHardLinks(allFiles)
		fmt.Printf("Found %d files (%s)", len(allFiles), core.FormatSize(totalSize(allFiles)))
		if hardLinks > 0 {
			fmt.Printf(" (%d hard links)", hardLinks)
		}
		fmt.Println(". Finding duplicates...")
	}

	groups := core.FindDuplicates(allFiles, findWorkers)

	if findSort == "count" {
		core.SortGroupsByCount(groups)
	} else if findSort == "path" {
		core.SortGroupsByPath(groups)
	} else {
		core.SortGroupsBySize(groups)
	}

	report := core.BuildReport(allFiles, groups, path)

	if findFormat == "json" {
		data, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	if findFormat == "csv" {
		printCSV(report)
		return nil
	}

	if findFormat == "silent" {
		return nil
	}

	printReport(report)

	if len(groups) == 0 {
		fmt.Println("\nNo duplicates found.")
		return nil
	}

	if findOutput != "" {
		ext := strings.ToLower(filepath.Ext(findOutput))
		var exportErr error
		switch ext {
		case ".csv":
			exportErr = core.ExportCSV(report, findOutput)
		case ".html", ".htm":
			exportErr = core.ExportHTML(report, findOutput)
		default:
			exportErr = core.ExportJSON(report, findOutput)
		}
		if exportErr != nil {
			fmt.Fprintf(os.Stderr, "Error writing report: %v\n", exportErr)
		} else {
			fmt.Printf("\nReport saved to: %s\n", findOutput)
		}
	}

	if findDryRun {
		fmt.Println("\n[DRY RUN] No changes made.")
		return nil
	}

	if findDelete || findLink != "" {
		if findKeep == "" {
			findKeep = "oldest"
		}
		return applyActionsCLI(groups)
	}

	if findFormat == "pretty" && !findYes {
		promptActionCLI(groups)
	}

	return nil
}

func applyActionsCLI(groups []models.DuplicateGroup) error {
	var action core.Action
	actionName := ""
	switch {
	case findDelete:
		action = core.ActionDelete
		actionName = "delete"
	case findLink == "hard":
		action = core.ActionHardLink
		actionName = "hard-link"
	case findLink == "soft":
		action = core.ActionSoftLink
		actionName = "soft-link"
	default:
		return nil
	}

	totalActions := 0
	for _, g := range groups {
		_, toRemove := core.ApplyKeepStrategy(g, findKeep)
		totalActions += len(toRemove)
	}

	if findFormat == "pretty" {
		fmt.Printf("\nAction: %s %d duplicate files\n", actionName, totalActions)
		fmt.Printf("Strategy: keep %s\n", findKeep)
	}

	if !findYes && !findDryRun {
		var response string
		fmt.Print("\nProceed? (y/N): ")
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	var successCount, errorCount int
	for _, g := range groups {
		keep, toRemove := core.ApplyKeepStrategy(g, findKeep)
		for _, dup := range toRemove {
			if findDryRun {
				if findFormat == "pretty" {
					fmt.Printf("Would remove: %s\n", dup.Path)
				}
				successCount++
				continue
			}

			if findFormat == "pretty" || findVerbose {
				if findDelete {
					fmt.Printf("Deleting: %s\n", dup.Path)
				} else {
					fmt.Printf("Linking: %s -> %s\n", dup.Path, keep.Path)
				}
			}

			if err := core.ApplyAction(action, keep, dup, ""); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				errorCount++
			} else {
				successCount++
			}
		}
	}

	if findFormat == "pretty" && !findDryRun {
		fmt.Printf("\nDone. %d succeeded, %d failed.\n", successCount, errorCount)
	} else if findDryRun {
		fmt.Printf("\n[DRY RUN] Would process %d files.\n", successCount)
	}

	return nil
}

func promptActionCLI(groups []models.DuplicateGroup) error {
	if len(groups) == 0 || findYes {
		return nil
	}

	var totalActions int
	for _, g := range groups {
		_, toRemove := core.ApplyKeepStrategy(g, "oldest")
		totalActions += len(toRemove)
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println("Select action:")
	fmt.Println("  1) Delete duplicates")
	fmt.Println("  2) Replace with hard links")
	fmt.Println("  3) Replace with soft links")
	fmt.Println("  4) Skip")
	fmt.Print("\nChoice [1-4, Enter=4]: ")

	actionChosen := false
	for {
		if !scanner.Scan() {
			return nil
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "" || input == "4" {
			return nil
		}
		if input == "1" {
			findDelete = true
			actionChosen = true
			break
		}
		if input == "2" {
			findLink = "hard"
			actionChosen = true
			break
		}
		if input == "3" {
			findLink = "soft"
			actionChosen = true
			break
		}
		fmt.Print("Invalid choice. Enter 1-4 [Enter=4]: ")
	}

	if !actionChosen {
		return nil
	}

	if findKeep == "" {
		fmt.Println()
		fmt.Println("Keep strategy:")
		fmt.Println("  1) Oldest")
		fmt.Println("  2) Newest")
		fmt.Println("  3) Shortest path")
		fmt.Print("\nChoice [1-3, Enter=1]: ")

		for {
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			if input == "" || input == "1" {
				findKeep = "oldest"
				break
			}
			if input == "2" {
				findKeep = "newest"
				break
			}
			if input == "3" {
				findKeep = "shortest"
				break
			}
			fmt.Print("Invalid choice. Enter 1-3 [Enter=1]: ")
		}
	}

	fmt.Println()
	fmt.Printf("Will %s %d duplicate files (keep %s)\n", getActionLabel(), totalActions, findKeep)
	fmt.Print("\nProceed? [y/N]: ")

	if !scanner.Scan() {
		return nil
	}
	confirm := strings.TrimSpace(strings.ToLower(scanner.Text()))
	if confirm != "y" && confirm != "yes" {
		fmt.Println("Cancelled.")
		return nil
	}

	findYes = true
	return applyActionsCLI(groups)
}

func getActionLabel() string {
	if findDelete {
		return "delete"
	}
	if findLink == "hard" {
		return "hard-link"
	}
	return "soft-link"
}

func printReport(report models.Report) {
	fmt.Println()
	fmt.Printf("Duplicate Groups: %d\n", len(report.DupGroups))
	fmt.Printf("Duplicate Files:  %d\n", report.DupFiles)
	fmt.Printf("Wasted Space:     %s\n", core.FormatSize(report.WastedSpace))
	if report.TotalSize > 0 {
		pct := float64(report.WastedSpace) / float64(report.TotalSize) * 100
		fmt.Printf("Recoverable:      %.1f%%\n", pct)
	}
	fmt.Println()

	if findVerbose {
		for i, g := range report.DupGroups {
			fmt.Printf("Group %d — %d copies, %s each (hash: %s)\n", i+1, len(g.Files), core.FormatSize(g.Size), g.Hash[:16]+"...")
			for j, f := range g.Files {
				prefix := "  dup"
				if j == 0 {
					prefix = "  keep"
				}
				modTime := f.ModTime.Format("2006-01-02 15:04:05")
				fmt.Printf("  [%s] %s  (%s, %s)\n", prefix, f.Path, core.FormatSize(f.Size), modTime)
			}
			fmt.Println()
		}
	} else {
		showCount := len(report.DupGroups)
		if showCount > 20 {
			showCount = 20
		}
		for i := 0; i < showCount; i++ {
			g := report.DupGroups[i]
			fmt.Printf("Group %d — %d copies, %s each\n", i+1, len(g.Files), core.FormatSize(g.Size))
			for j, f := range g.Files {
				prefix := "  dup"
				if j == 0 {
					prefix = "  keep"
				}
				fmt.Printf("  [%s] %s\n", prefix, f.Path)
			}
			fmt.Println()
		}
		if len(report.DupGroups) > 20 {
			fmt.Printf("... and %d more groups. Use -v for full output.\n", len(report.DupGroups)-20)
			fmt.Println()
		}
	}
}

func parseSize(s string) int64 {
	if s == "" || s == "0" {
		return 0
	}
	s = strings.TrimSpace(s)
	last := strings.ToLower(s[len(s)-1:])

	switch last {
	case "k":
		var n int64
		fmt.Sscanf(s[:len(s)-1], "%d", &n)
		return n * 1024
	case "m":
		var n int64
		fmt.Sscanf(s[:len(s)-1], "%d", &n)
		return n * 1024 * 1024
	case "g":
		var n int64
		fmt.Sscanf(s[:len(s)-1], "%d", &n)
		return n * 1024 * 1024 * 1024
	default:
		var n int64
		fmt.Sscanf(s, "%d", &n)
		return n
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func totalSize(files []models.FileInfo) int64 {
	var total int64
	for _, f := range files {
		total += f.Size
	}
	return total
}

func printCSV(report models.Report) {
	fmt.Printf("group,hash,size,path,is_duplicate,mod_time\n")
	for gi, g := range report.DupGroups {
		for fi, f := range g.Files {
			fmt.Printf("%d,%s,%d,%s,%t,%s\n", gi+1, g.Hash, f.Size, f.Path, fi > 0, f.ModTime.Format("2006-01-02 15:04:05"))
		}
	}
}
