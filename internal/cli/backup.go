package cli

import "github.com/spf13/cobra"

// NewBackupCmd creates the backup subcommand.
func NewBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage backups",
		Long:  "List, inspect, or clean up submodule backup files.",
		Example: `  ssu backup
  ssu backup list
  ssu backup clean`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("backup: not implemented yet")
			return nil
		},
	}

	return cmd
}
