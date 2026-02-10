package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/backup"
	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/cli/tui"
	"github.com/pxpxltd/ssu/internal/git"
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
	autoMode, _ := cmd.Flags().GetBool("auto")

	// If no backup file specified, show available backups
	if len(args) == 0 {
		if autoMode {
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

	// Resolve project root
	projectRoot, err := detectProjectRoot()
	if err != nil {
		return fmt.Errorf("detecting project root: %w", err)
	}

	// Create git service
	gitSvc := git.NewExecGit()
	ctx := context.Background()

	// Collect previous SHAs for results table
	previousSHAs := make(map[string]string)
	for path := range b.Submodules {
		subDir := filepath.Join(projectRoot, path)
		sha, shaErr := gitSvc.CurrentSHA(ctx, subDir)
		if shaErr == nil {
			previousSHAs[path] = sha
		}
	}

	// Interactive confirmation (TTY only, not auto mode)
	if output.IsTTY() && !autoMode {
		fmt.Fprintf(cmd.OutOrStdout(), "Restore %d submodule(s) from this backup? [y/N]: ", len(b.Submodules))
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return nil
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			p.Info("Cancelled.")
			return nil
		}
	}

	// Build closure callbacks that adapt GitService to backup function signatures
	getCurrentStates := func(root string, paths []string) (map[string]backup.SubmoduleState, error) {
		states := make(map[string]backup.SubmoduleState)
		for _, path := range paths {
			subDir := filepath.Join(projectRoot, path)
			sha, shaErr := gitSvc.CurrentSHA(ctx, subDir)
			if shaErr != nil {
				continue
			}
			br, brErr := gitSvc.CurrentBranch(ctx, subDir)
			branch := ""
			if brErr == nil {
				branch = br.Name
			}
			states[path] = backup.SubmoduleState{
				SHA:    sha,
				Branch: branch,
			}
		}
		return states, nil
	}

	gitCheckout := func(dir, branch string) error {
		_, err := gitSvc.Checkout(ctx, filepath.Join(projectRoot, dir), branch)
		return err
	}

	gitResetHard := func(dir, sha string) error {
		return gitSvc.ResetHard(ctx, filepath.Join(projectRoot, dir), sha)
	}

	// Resolve backup directory for safety backup
	bkDir, bkErr := resolveBackupDir(cmd)
	if bkErr != nil {
		bkDir = "" // safety backup will be skipped
	}

	// Call backup.Rollback with real git callbacks
	result, err := backup.Rollback(
		backup.RollbackOpts{
			BackupPath:  backupPath,
			ProjectRoot: projectRoot,
			BackupDir:   bkDir,
		},
		getCurrentStates,
		gitCheckout,
		gitResetHard,
	)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	// Display safety backup info
	if result.SafetyBackupFile != "" {
		p.Infof("Safety backup: %s", result.SafetyBackupFile)
	}

	// Build results table
	successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))

	t := table.New().
		Headers("Path", "Branch", "Previous", "Restored", "Status").
		Border(lipgloss.NormalBorder()).
		BorderHeader(true).
		BorderColumn(true).
		Width(120)

	for _, sub := range result.Submodules {
		prevSHA := previousSHAs[sub.Path]
		if len(prevSHA) > 7 {
			prevSHA = prevSHA[:7]
		}
		targetSHA := sub.SHA
		if len(targetSHA) > 7 {
			targetSHA = targetSHA[:7]
		}
		branch := sub.Branch
		status := "restored"
		if sub.Error != nil {
			status = sub.Error.Error()
		}
		t.Row(sub.Path, branch, prevSHA, targetSHA, status)
	}

	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return tui.HeaderStyle
		}
		if col == 4 && row >= 0 && row < len(result.Submodules) {
			if result.Submodules[row].Error != nil {
				return errorStyle
			}
			return successStyle
		}
		return lipgloss.NewStyle()
	})

	fmt.Fprintln(cmd.OutOrStdout(), t.Render())

	// Summary line
	total := len(result.Submodules)
	if result.RestoredCount == total {
		p.Successf("Restored %d/%d submodule(s)", result.RestoredCount, total)
	} else {
		p.Warningf("Restored %d/%d submodule(s)", result.RestoredCount, total)
	}

	return nil
}

// minInt returns the smaller of a and b.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
