package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/spf13/cobra"
)

// NewRootCmd creates the root cobra command with global flags and all subcommands.
func NewRootCmd(version, commit, date string) *cobra.Command {
	root := &cobra.Command{
		Use:   "ssu",
		Short: "Smart Submodule Updater",
		Long: `SSU intelligently manages git submodules with smart branch detection,
automatic backups, and conflict handling.

Run without arguments for an interactive menu, or use a subcommand directly.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !output.IsTTY() {
				return cmd.Help()
			}
			return showInteractiveMenu(cmd)
		},
	}

	// Global persistent flags
	root.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	root.PersistentFlags().BoolP("dry-run", "n", false, "Preview changes without modifying anything")
	root.PersistentFlags().BoolP("auto", "a", false, "Automatic mode (no prompts, for CI/CD)")
	root.PersistentFlags().IntP("jobs", "j", 8, "Number of parallel fetch jobs")

	// Add subcommands
	root.AddCommand(
		NewStatusCmd(),
		NewUpdateCmd(),
		NewPushCmd(),
		NewRollbackCmd(),
		NewBackupCmd(),
		NewVersionCmd(version, commit, date),
		NewCompletionCmd(),
	)

	return root
}

// showInteractiveMenu displays a simple numbered menu for TTY sessions.
// This is a Phase 1 placeholder -- Bubble Tea replaces it in Phase 5.
func showInteractiveMenu(cmd *cobra.Command) error {
	type menuItem struct {
		name string
		desc string
	}
	items := []menuItem{
		{"status", "Show submodule status"},
		{"update", "Update submodules"},
		{"push", "Push ahead submodules"},
		{"rollback", "Rollback from backup"},
		{"backup", "Manage backups"},
		{"help", "Show full help"},
	}

	fmt.Fprintln(cmd.OutOrStdout(), "What would you like to do?")
	fmt.Fprintln(cmd.OutOrStdout())
	for i, item := range items {
		fmt.Fprintf(cmd.OutOrStdout(), "  %d) %-10s - %s\n", i+1, item.name, item.desc)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprint(cmd.OutOrStdout(), "Choose [1-6]: ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil
	}
	choice := strings.TrimSpace(scanner.Text())

	var subcmd string
	switch choice {
	case "1":
		subcmd = "status"
	case "2":
		subcmd = "update"
	case "3":
		subcmd = "push"
	case "4":
		subcmd = "rollback"
	case "5":
		subcmd = "backup"
	case "6":
		return cmd.Help()
	default:
		fmt.Fprintf(cmd.ErrOrStderr(), "Invalid choice: %s\n", choice)
		return nil
	}

	// Dispatch to the chosen subcommand
	cmd.SetArgs([]string{subcmd})
	return cmd.Execute()
}
