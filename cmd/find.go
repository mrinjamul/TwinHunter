package cmd

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mrinjamul/twinhunter/core"
	"github.com/mrinjamul/twinhunter/models"
	"github.com/spf13/cobra"
)

var (
	findRecursive  bool
	findMinSize    string
	findMaxSize    string
	findExclude    string
	findExcludeRe  string
	findExcludeDir string
	findExcludeExt string
	findWorkers    int
	findOutput     string
	findFormat     string
	findKeep       string
	findLink       string
	findDelete     bool
	findBackupDir  string
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
	findCmd.Flags().StringVar(&findExcludeExt, "exclude-ext", "", "comma-separated file extensions to exclude (e.g. log,tmp)")
	findCmd.Flags().IntVarP(&findWorkers, "workers", "w", 0, "number of parallel hashing workers (0 = auto)")
	findCmd.Flags().StringVarP(&findOutput, "output", "o", "", "export report to file (format auto-detected from extension: .json, .csv, .html)")
	findCmd.Flags().StringVarP(&findFormat, "format", "f", "pretty", "terminal output: pretty, json, csv, silent")
	findCmd.Flags().BoolVarP(&findVerbose, "verbose", "v", false, "show detailed per-group output")
	findCmd.Flags().StringVar(&findSort, "sort", "size", "sort groups by: size, count, path")
	findCmd.Flags().BoolVarP(&findYes, "yes", "y", false, "skip confirmation prompts")
	findCmd.Flags().StringVarP(&findKeep, "keep", "k", "", "auto-keep strategy: oldest (default), newest, shortest")
	findCmd.Flags().StringVarP(&findLink, "link", "l", "", "replace duplicates with links: hard, soft")
	findCmd.Flags().BoolVarP(&findDelete, "delete", "d", false, "delete duplicate files")
	findCmd.Flags().StringVar(&findBackupDir, "backup-dir", "", "move duplicates to backup directory")
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
	var err error
	minSize, err := parseSize(findMinSize)
	if err != nil {
		return fmt.Errorf("invalid min-size: %w", err)
	}
	maxSize, err := parseSize(findMaxSize)
	if err != nil {
		return fmt.Errorf("invalid max-size: %w", err)
	}
	exclude := splitCSV(findExclude)
	excludeRe := splitCSV(findExcludeRe)
	excludeDir := splitCSV(findExcludeDir)
	// If --exclude-dir was not explicitly set, nil means "use defaults" in Scan.
	if !cmd.Flags().Changed("exclude-dir") {
		excludeDir = nil
	}

	if findKeep != "" {
		switch findKeep {
		case "oldest", "newest", "shortest":
		default:
			return fmt.Errorf("invalid keep strategy: %q (valid: oldest, newest, shortest)", findKeep)
		}
	}

	switch findFormat {
	case "pretty", "json", "csv", "silent":
	default:
		return fmt.Errorf("invalid format: %q (valid: pretty, json, csv, silent)", findFormat)
	}

	flagsSet := 0
	if findDelete {
		flagsSet++
	}
	if findLink != "" {
		flagsSet++
	}
	if findBackupDir != "" {
		flagsSet++
	}
	if flagsSet > 1 {
		return fmt.Errorf("--delete, --link, and --backup-dir are mutually exclusive")
	}

	if findWorkers < 0 {
		return fmt.Errorf("invalid workers: %d (minimum 0 for auto)", findWorkers)
	}

	excludeExt := splitCSV(findExcludeExt)
	for _, ext := range excludeExt {
		ext = strings.TrimPrefix(ext, ".")
		exclude = append(exclude, "*."+ext)
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

	if findFormat == "pretty" {
		fmt.Fprintf(os.Stderr, "Scanning: %s\n", path)
		if findRecursive {
			fmt.Fprintln(os.Stderr, "Mode: recursive")
		}
		cfg.OnProgress = func(stats models.ScanStats) {
			writeProgress("Scanning files", stats.FilesScanned, 0)
		}
	}

	allFiles, err := core.Scan(cfg)
	if err != nil {
		return fmt.Errorf("scan error: %w", err)
	}

	sizeGroups := core.GroupBySize(allFiles)
	var candidatesCount int
	for _, g := range sizeGroups {
		candidatesCount += len(g)
	}

	if findFormat == "pretty" {
		done()
		hardLinks := core.CountHardLinks(allFiles)
		fmt.Fprintf(os.Stderr, "Found %d files (%s)", len(allFiles), core.FormatSize(totalSize(allFiles)))
		if hardLinks > 0 {
			fmt.Fprintf(os.Stderr, " (%d hard links)", hardLinks)
		}

		if candidatesCount > 0 {
			fmt.Fprintln(os.Stderr, ". Checking for matches...")
		}
	}

	groups := core.FindDuplicates(allFiles, findWorkers, func(p core.HashProgress) {
		if findFormat == "pretty" && candidatesCount > 0 {
			switch p.Phase {
			case "blake3":
				writeProgress("Hashing files", p.Current, p.Total)
			case "sha256":
				writeProgress("Verifying matches", p.Current, p.Total)
			}
		}
	})

	if findFormat == "pretty" && candidatesCount > 0 {
		done()
	}

	if findSort == "count" {
		core.SortGroupsByCount(groups)
	} else if findSort == "path" {
		core.SortGroupsByPath(groups)
	} else {
		core.SortGroupsBySize(groups)
	}

	// Sort files within each group so the first file shown as [keep]
	// is always a meaningful default (oldest, tiebreak: shortest path).
	activeKeep := findKeep
	if activeKeep == "" {
		activeKeep = "oldest"
	}
	for i, g := range groups {
		keep, toRemove := core.ApplyKeepStrategy(g, activeKeep)
		groups[i].Files = append([]models.FileInfo{keep}, toRemove...)
	}

	report := core.BuildReport(allFiles, groups, path)

	// Export to file first (before display, so errors are surfaced).
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
			return fmt.Errorf("failed to write report: %w", exportErr)
		}
	}

	// Display to terminal. For JSON/CSV output, skip display when saving to file
	// to avoid duplicate output.
	displayJSON := findFormat == "json" && findOutput == ""
	displayCSV := findFormat == "csv" && findOutput == ""
	displayPretty := findFormat == "pretty"

	if displayJSON {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}
		fmt.Println(string(data))
	}

	if displayCSV {
		printCSV(report)
	}

	if displayPretty {
		printReport(report)
		if findOutput != "" {
			fmt.Printf("\nReport saved to: %s\n", findOutput)
		}
	}

	if len(groups) == 0 {
		if displayPretty {
			fmt.Println("\nNo duplicates found.")
		}
		return nil
	}

	if findDelete || findLink != "" || findBackupDir != "" {
		if findKeep == "" {
			findKeep = "oldest"
		}
		return applyActionsCLI(groups)
	}

	if findDryRun {
		if findFormat == "pretty" {
			fmt.Println("\n[DRY RUN] No changes made.")
		}
		return nil
	}

	if findFormat == "pretty" && !findYes {
		if !isTerminal() {
			fmt.Fprintln(os.Stderr, "Warning: stdin is not a terminal, skipping interactive prompt. Use -y to auto-confirm actions.")
			return nil
		}
		return promptActionCLI(groups)
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
	case findBackupDir != "":
		action = core.ActionBackup
		actionName = "backup"
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

	if findFormat == "pretty" && !findYes && !findDryRun {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("\nProceed? (y/N): ")
		if !scanner.Scan() {
			fmt.Println("Cancelled.")
			return nil
		}
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if response != "y" && response != "yes" {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	var successCount, errorCount int
	var wastedSpace int64
	processed := 0
	for _, g := range groups {
		keep, toRemove := core.ApplyKeepStrategy(g, findKeep)
		wastedSpace += g.Size * int64(len(toRemove))
		for _, dup := range toRemove {
			processed++
		if findDryRun {
			if findFormat == "pretty" {
				switch {
				case findDelete:
					fmt.Printf("[%d/%d] Would delete: %s\n", processed, totalActions, dup.Path)
				case findBackupDir != "":
					fmt.Printf("[%d/%d] Would backup: %s\n", processed, totalActions, dup.Path)
				default:
					fmt.Printf("[%d/%d] Would link: %s\n", processed, totalActions, dup.Path)
				}
			}
			successCount++
			continue
		}

			if findFormat == "pretty" || findVerbose {
				switch {
				case findDelete:
					fmt.Printf("[%d/%d] Deleting: %s\n", processed, totalActions, dup.Path)
				case findBackupDir != "":
					fmt.Printf("[%d/%d] Backing up: %s\n", processed, totalActions, dup.Path)
				default:
					fmt.Printf("[%d/%d] Linking: %s -> %s\n", processed, totalActions, dup.Path, keep.Path)
				}
			}

			if err := core.ApplyAction(action, keep, dup, findBackupDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				errorCount++
			} else {
				successCount++
			}
		}
	}

	if findFormat == "pretty" && !findDryRun {
		fmt.Printf("\nDone. %d succeeded, %d failed.\n", successCount, errorCount)
		fmt.Printf("Targeted space recovered: %s\n", core.FormatSize(wastedSpace))
	} else if findDryRun {
		fmt.Printf("\n[DRY RUN] Would process %d files.\n", successCount)
	}

	if errorCount > 0 {
		return fmt.Errorf("%d actions failed", errorCount)
	}

	return nil
}

func promptActionCLI(groups []models.DuplicateGroup) error {
	if len(groups) == 0 || findYes {
		return nil
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println()
	fmt.Println("Select action:")
	fmt.Println("  1) Delete duplicates")
	fmt.Println("  2) Replace with hard links")
	fmt.Println("  3) Replace with soft links")
	fmt.Println("  4) Backup duplicates")
	fmt.Println("  5) Skip")
	fmt.Print("\nChoice [1-5, Enter=5]: ")

	actionChosen := false
	for {
		if !scanner.Scan() {
			return nil
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "" || input == "5" {
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
		if input == "4" {
			fmt.Print("\nBackup directory path: ")
			if !scanner.Scan() {
				return nil
			}
			findBackupDir = strings.TrimSpace(scanner.Text())
			if findBackupDir == "" {
				fmt.Println("Cancelled.")
				return nil
			}
			actionChosen = true
			break
		}
		fmt.Print("Invalid choice. Enter 1-5 [Enter=5]: ")
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

	var totalActions int
	for _, g := range groups {
		_, toRemove := core.ApplyKeepStrategy(g, findKeep)
		totalActions += len(toRemove)
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
	switch {
	case findDelete:
		return "delete"
	case findLink == "hard":
		return "hard-link"
	case findLink == "soft":
		return "soft-link"
	case findBackupDir != "":
		return "backup"
	default:
		return "soft-link"
	}
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
			hashDisplay := g.Hash
			if len(hashDisplay) > 16 {
				hashDisplay = hashDisplay[:16] + "..."
			}
			fmt.Printf("Group %d — %d copies, %s each (hash: %s)\n", i+1, len(g.Files), core.FormatSize(g.Size), hashDisplay)
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
			fmt.Printf("... and %d more groups. Use -v (verbose) to show all groups with full details.\n", len(report.DupGroups)-20)
			fmt.Println()
		}
	}
}

func parseSize(s string) (int64, error) {
	if s == "" || s == "0" {
		return 0, nil
	}
	s = strings.TrimSpace(s)
	last := strings.ToLower(s[len(s)-1:])

	switch last {
	case "k":
		n, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid size: %s", s)
		}
		return n * 1024, nil
	case "m":
		n, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid size: %s", s)
		}
		return n * 1024 * 1024, nil
	case "g":
		n, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid size: %s", s)
		}
		return n * 1024 * 1024 * 1024, nil
	case "t":
		n, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
		if err != nil || n < 0 {
			return 0, fmt.Errorf("invalid size: %s", s)
		}
		return n * 1024 * 1024 * 1024 * 1024, nil
	default:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n <= 0 {
			return 0, fmt.Errorf("invalid size: %s (valid suffixes: K, M, G, T)", s)
		}
		return n, nil
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return []string{}
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

const barWidth = 30

func writeProgress(prefix string, current, total int) {
	var line string
	if total <= 0 {
		line = fmt.Sprintf("%s... (%d)", prefix, current)
	} else {
		if current > total {
			current = total
		}
		ratio := float64(current) / float64(total)
		filled := int(ratio * float64(barWidth))
		var bar string
		if filled >= barWidth {
			bar = "[" + strings.Repeat("=", barWidth) + "]"
		} else {
			bar = "[" + strings.Repeat("=", filled) + ">" + strings.Repeat("-", barWidth-filled-1) + "]"
		}
		line = fmt.Sprintf("%s %s %d/%d", prefix, bar, current, total)
	}
	fmt.Fprintf(os.Stderr, "\r%-80s", line)
}

func done() {
	fmt.Fprintf(os.Stderr, "\r%s\r", strings.Repeat(" ", 80))
}

func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func printCSV(report models.Report) {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

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
}
