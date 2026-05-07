package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	AppName   = "twinhunter"
	Version   = "dev"
	GitCommit = "none"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		short := GitCommit
		if len(short) > 7 {
			short = short[:7]
		}
		fmt.Printf("%s %s-%s\n", AppName, Version, short)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
