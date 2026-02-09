package cli

import "github.com/spf13/cobra"

// NewRollbackCmd creates the rollback subcommand.
func NewRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback [backup-file]",
		Short: "Rollback from backup",
		Long:  "Restore submodules to a previous state using a backup file.",
		Example: `  ssu rollback
  ssu rollback ~/.ssu/myproject/.submodule-backup-20260209-103000.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("rollback: not implemented yet")
			return nil
		},
	}

	return cmd
}
