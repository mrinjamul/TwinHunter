package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for TwinHunter.

To load completions in your current shell session:
  Bash:       source <(twinhunter completion bash)
  Zsh:        source <(twinhunter completion zsh)
  Fish:       twinhunter completion fish | source
  PowerShell: twinhunter completion powershell | Out-String | Invoke-Expression

To load completions for every new session, run once:
  Bash:       twinhunter completion bash > /etc/bash_completion.d/twinhunter
  Zsh:        twinhunter completion zsh > "${fpath[1]}/_twinhunter"
  Fish:       twinhunter completion fish > ~/.config/fish/completions/twinhunter.fish
  PowerShell: twinhunter completion powershell > $PROFILE`,
	Args:      cobra.ExactArgs(1),
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			return cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			return cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			return cmd.Root().GenPowerShellCompletion(os.Stdout)
		default:
			return fmt.Errorf("unsupported shell: %s", args[0])
		}
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
