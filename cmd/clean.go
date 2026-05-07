package cmd

import (
	"fmt"
	"os"

	"github.com/mrinjamul/twinhunter/core"
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
	Use:   "clean [report.json]",
	Short: "Clean duplicates from a saved report",
	Long:  `Apply delete/link/backup actions to duplicates listed in a previously exported JSON report.`,
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
	report, err := core.ImportJSON(args[0])
	if err != nil {
		return fmt.Errorf("failed to read report: %w", err)
	}

	if len(report.DupGroups) == 0 {
		fmt.Println("No duplicates found in report.")
		return nil
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

	for _, g := range report.DupGroups {
		keep, toRemove := core.ApplyKeepStrategy(g, cleanKeep)
		for _, dup := range toRemove {
			if cleanDryRun {
				fmt.Printf("[DRY RUN] Would remove: %s\n", dup.Path)
				continue
			}
			fmt.Printf("Processing: %s\n", dup.Path)
			if err := core.ApplyAction(action, keep, dup, cleanBackupDir); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
		}
	}

	if !cleanDryRun {
		fmt.Printf("Done. Cleaned %d duplicate files.\n", report.DupFiles)
	} else {
		fmt.Printf("[DRY RUN] Would clean %d duplicate files.\n", report.DupFiles)
	}

	return nil
}
