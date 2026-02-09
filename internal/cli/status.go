package cli

import "github.com/spf13/cobra"

// NewStatusCmd creates the status subcommand.
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show submodule status",
		Long:  "Display the status of all submodules including branch, commits behind, and modification state.",
		Example: `  ssu status
  ssu status --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("status: not implemented yet")
			return nil
		},
	}

	cmd.Flags().Bool("json", false, "Output status as JSON")

	return cmd
}
