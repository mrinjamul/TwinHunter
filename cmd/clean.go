package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrinjamul/twinhunter/core"
	"github.com/mrinjamul/twinhunter/models"
	"github.com/spf13/cobra"
)

var (
	cleanKeep      string
	cleanDelete    bool
	cleanLink      string
	cleanBackupDir string
	cleanDryRun    bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean [report]",
	Short: "Clean duplicates from a saved report",
	Long:  `Apply delete/link/backup actions to duplicates listed in a previously exported report (JSON, CSV, or HTML).`,
	Args:  cobra.ExactArgs(1),
	RunE:  runClean,
}

func init() {
	rootCmd.AddCommand(cleanCmd)

	cleanCmd.Flags().StringVarP(&cleanKeep, "keep", "k", "oldest", "keep strategy: oldest, newest, shortest")
	cleanCmd.Flags().BoolVarP(&cleanDelete, "delete", "d", false, "delete duplicate files")
	cleanCmd.Flags().StringVarP(&cleanLink, "link", "l", "", "replace duplicates with links: hard, soft")
	cleanCmd.Flags().StringVar(&cleanBackupDir, "backup-dir", "", "move duplicates to backup directory")
	cleanCmd.Flags().BoolVarP(&cleanDryRun, "dry-run", "n", false, "preview without making changes")
}

func runClean(cmd *cobra.Command, args []string) error {
	ext := strings.ToLower(filepath.Ext(args[0]))
	var report models.Report
	var err error
	switch ext {
	case ".json":
		report, err = core.ImportJSON(args[0])
	case ".csv":
		report, err = core.ImportCSV(args[0])
	case ".html", ".htm":
		report, err = core.ImportHTML(args[0])
	default:
		return fmt.Errorf("unsupported report format: %q (supported: .json, .csv, .html)", ext)
	}
	if err != nil {
		return fmt.Errorf("failed to read report: %w", err)
	}

	if len(report.DupGroups) == 0 {
		fmt.Fprintln(os.Stderr, "No duplicates found in report.")
		return nil
	}

	switch cleanKeep {
	case "oldest", "newest", "shortest":
	default:
		return fmt.Errorf("invalid keep strategy: %q (valid: oldest, newest, shortest)", cleanKeep)
	}

	flagsSet := 0
	if cleanDelete {
		flagsSet++
	}
	if cleanLink != "" {
		flagsSet++
	}
	if cleanBackupDir != "" {
		flagsSet++
	}
	if flagsSet > 1 {
		return fmt.Errorf("--delete, --link, and --backup-dir are mutually exclusive")
	}

	var action core.Action
	switch {
	case cleanDelete:
		action = core.ActionDelete
	case cleanLink == "hard":
		action = core.ActionHardLink
	case cleanLink == "soft":
		action = core.ActionSoftLink
	case cleanBackupDir != "":
		action = core.ActionBackup
	default:
		return fmt.Errorf("no action specified. Use --delete, --link, or --backup-dir")
	}

	var errorCount int
	processed := 0
	totalActions := report.DupFiles
	for _, g := range report.DupGroups {
		keep, toRemove := core.ApplyKeepStrategy(g, cleanKeep)
		for _, dup := range toRemove {
			processed++
			if cleanDryRun {
				fmt.Fprintf(os.Stderr, "[%d/%d] [DRY RUN] Would remove: %s\n", processed, totalActions, dup.Path)
				continue
			}
			fmt.Fprintf(os.Stderr, "[%d/%d] Processing: %s\n", processed, totalActions, dup.Path)
			if err := core.ApplyAction(action, keep, dup, cleanBackupDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				errorCount++
			}
		}
	}

	if !cleanDryRun {
		fmt.Fprintf(os.Stderr, "Done. Cleaned %d duplicate files. %d errors.\n", report.DupFiles, errorCount)
	} else {
		fmt.Fprintf(os.Stderr, "[DRY RUN] Would clean %d duplicate files.\n", report.DupFiles)
	}

	if errorCount > 0 {
		return fmt.Errorf("%d actions failed", errorCount)
	}

	return nil
}
