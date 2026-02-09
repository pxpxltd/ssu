package cli

import (
	"fmt"

	"github.com/pxpxltd/ssu/internal/backup"
	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/spf13/cobra"
)

// NewRollbackCmd creates the rollback subcommand.
func NewRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback [backup-file]",
		Short: "Rollback from backup",
		Long: `Restore submodules to a previous state using a backup file.

If a backup file path is provided, it will be used directly. Otherwise,
recent backups are listed for reference.

A safety backup of the current state is automatically created before restoring.`,
		Example: `  ssu rollback backup-20260209-103000.json
  ssu rollback ~/.ssu/myproject/backups/backup-20260209-103000.json
  ssu rollback --dry-run backup-20260209-103000.json`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRollback(cmd, args)
		},
	}

	return cmd
}

func runRollback(cmd *cobra.Command, args []string) error {
	p := output.NewPrinter(cmd.OutOrStdout())
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// If no backup file specified, show available backups
	if len(args) == 0 {
		auto, _ := cmd.Flags().GetBool("auto")
		if auto {
			return fmt.Errorf("rollback requires a backup file path in auto mode")
		}

		backupDir, err := resolveBackupDir(cmd)
		if err != nil {
			return err
		}

		infos, err := backup.List(backupDir)
		if err != nil {
			return err
		}

		if len(infos) == 0 {
			p.Info("No backups found")
			return nil
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Available backups:")
		fmt.Fprintln(cmd.OutOrStdout())
		for _, info := range infos {
			ts := info.Timestamp.Format("2006-01-02 15:04:05")
			fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s\n", ts, info.Path)
		}
		fmt.Fprintln(cmd.OutOrStdout())
		p.Info("Run: ssu rollback <backup-file-path>")
		return nil
	}

	// Read the backup file to show what would be restored
	backupPath := args[0]
	b, err := backup.Read(backupPath)
	if err != nil {
		return fmt.Errorf("reading backup: %w", err)
	}

	versionLabel := "go-era"
	if b.Version == 1 {
		versionLabel = "bash-era"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Backup: %s (%s, v%d)\n", backupPath, versionLabel, b.Version)
	fmt.Fprintf(cmd.OutOrStdout(), "Timestamp: %s\n", b.Timestamp)
	fmt.Fprintf(cmd.OutOrStdout(), "Submodules: %d\n\n", len(b.Submodules))

	for path, state := range b.Submodules {
		fmt.Fprintf(cmd.OutOrStdout(), "  %-40s  %s @ %s\n", path, state.Branch, state.SHA[:minInt(7, len(state.SHA))])
	}
	fmt.Fprintln(cmd.OutOrStdout())

	if dryRun {
		p.Info("Dry run: no changes made")
		return nil
	}

	// Actual git operations are wired in Phase 5 when commands connect to the engine.
	// For now, we show the restoration plan and explain the deferral.
	p.Warning("Git operations not yet wired (Phase 5)")
	fmt.Fprintln(cmd.OutOrStdout(), "  The backup was parsed and validated successfully.")
	fmt.Fprintln(cmd.OutOrStdout(), "  Actual submodule checkout/reset will be available after Phase 5 integration.")

	return nil
}

// minInt returns the smaller of a and b.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
