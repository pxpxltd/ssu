package cli

import (
	"os"

	"github.com/spf13/cobra"
)

// NewCompletionCmd creates the completion subcommand for shell completion generation.
func NewCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate a shell completion script for ssu.

To load completions:

Bash:
  $ source <(ssu completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ ssu completion bash > /etc/bash_completion.d/ssu
  # macOS:
  $ ssu completion bash > $(brew --prefix)/etc/bash_completion.d/ssu

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it. Execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ ssu completion zsh > "${fpath[1]}/_ssu"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ ssu completion fish | source

  # To load completions for each session, execute once:
  $ ssu completion fish > ~/.config/fish/completions/ssu.fish

PowerShell:
  PS> ssu completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, add the output to your profile:
  PS> ssu completion powershell >> $PROFILE`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
}
