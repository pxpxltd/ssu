package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pxpxltd/ssu/internal/backup"
	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/spf13/cobra"
)

// NewBackupCmd creates the backup command with list and clean subcommands.
func NewBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage backups",
		Long:  "List, inspect, or clean up submodule backup files.",
		Example: `  ssu backup
  ssu backup list
  ssu backup clean --keep 5
  ssu backup clean --keep 7d`,
		// Default to list when run without subcommand
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupList(cmd)
		},
	}

	cmd.AddCommand(newBackupListCmd())
	cmd.AddCommand(newBackupCleanCmd())

	return cmd
}

func newBackupListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available backups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupList(cmd)
		},
	}
}

func newBackupCleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove old backups",
		Long: `Remove old backups keeping the newest ones.
Use --keep with a number to keep N most recent, or with "Nd" for time-based (e.g. 7d = 7 days).`,
		Example: `  ssu backup clean --keep 5
  ssu backup clean --keep 7d`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBackupClean(cmd)
		},
	}

	cmd.Flags().String("keep", "", "Number of backups to keep (e.g. \"5\") or age limit (e.g. \"7d\")")
	cmd.MarkFlagRequired("keep")

	return cmd
}

// resolveBackupDir determines the backup directory from config or auto-detection.
func resolveBackupDir(cmd *cobra.Command) (string, error) {
	cfg := config.FromContext(cmd.Context())

	// Detect project root
	projectRoot, err := detectProjectRoot()
	if err != nil {
		// Fallback to cwd
		projectRoot, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot determine project root: %w", err)
		}
	}

	projectName := backup.ProjectName(projectRoot)

	// Use config if available (backup.max_backups is set but no custom dir in config)
	_ = cfg // config is used for future backup.dir setting

	dir, err := backup.BackupDir(projectName)
	if err != nil {
		return "", err
	}
	return dir, nil
}

func runBackupList(cmd *cobra.Command) error {
	p := output.NewPrinter(cmd.OutOrStdout())
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
		fmt.Fprintf(cmd.OutOrStdout(), "  Backup directory: %s\n", backupDir)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Found %d backup(s):\n\n", len(infos))

	for _, info := range infos {
		ts := info.Timestamp.Format("2006-01-02 15:04:05")

		// Get file size
		sizeStr := "?"
		if fi, statErr := os.Stat(info.Path); statErr == nil {
			sizeStr = formatSize(fi.Size())
		}

		typeStr := "go-era"
		if info.IsBashEra {
			typeStr = "bash-era"
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %s  %-40s  %s  %s\n",
			ts, info.Filename, typeStr, sizeStr)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nBackup directory: %s\n", backupDir)
	if parentDir := filepath.Dir(backupDir); parentDir != backupDir {
		fmt.Fprintf(cmd.OutOrStdout(), "Legacy directory: %s\n", parentDir)
	}

	return nil
}

func runBackupClean(cmd *cobra.Command) error {
	p := output.NewPrinter(cmd.OutOrStdout())
	backupDir, err := resolveBackupDir(cmd)
	if err != nil {
		return err
	}

	keep, _ := cmd.Flags().GetString("keep")

	removed, err := backup.Clean(backupDir, keep)
	if err != nil {
		return err
	}

	if removed == 0 {
		p.Info("No backups to remove")
	} else {
		p.Successf("Removed %d old backup(s)", removed)
	}

	return nil
}

// formatSize returns a human-readable size string.
func formatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
