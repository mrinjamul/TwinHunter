package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "twinhunter [path]",
	Short: "TwinHunter — a fast duplicate file finder",
	Long: `TwinHunter finds duplicate files quickly using a two-stage
hashing pipeline (Blake3 + SHA256).

Usage:
  twinhunter                        # scan current directory
  twinhunter /path/to/scan          # scan specific path (non-recursive)
  twinhunter find /path -r          # recursive scan
  twinhunter find /path -r -d -k shortest   # find and delete, keep shortest path
  twinhunter clean report.json      # remove duplicates from saved report`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return findCmd.RunE(cmd, args)
	},
}

func Execute() error {
	return rootCmd.Execute()
}
