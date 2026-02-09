package cli

import "github.com/spf13/cobra"

// NewUpdateCmd creates the update subcommand.
func NewUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update submodules",
		Long:  "Fetch and merge updates for selected submodules with automatic conflict handling.",
		Example: `  ssu update
  ssu update --auto
  ssu update --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("update: not implemented yet")
			return nil
		},
	}

	return cmd
}
