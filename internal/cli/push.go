package cli

import "github.com/spf13/cobra"

// NewPushCmd creates the push subcommand.
func NewPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push ahead submodules",
		Long:  "Push unpushed commits in submodules to their remote tracking branches.",
		Example: `  ssu push
  ssu push --auto`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("push: not implemented yet")
			return nil
		},
	}

	return cmd
}
