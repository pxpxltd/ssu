package cli

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/pxpxltd/ssu/internal/logging"
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

	// Load config before every subcommand
	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return loadConfig(cmd)
	}

	// Add subcommands
	root.AddCommand(
		NewStatusCmd(),
		NewUpdateCmd(),
		NewPushCmd(),
		NewExecCmd(),
		NewInitCmd(),
		NewRollbackCmd(),
		NewBackupCmd(),
		NewConfigCmd(),
		NewVersionCmd(version, commit, date),
		NewCompletionCmd(),
		NewClaudeCmd(),
	)

	return root
}

// loadConfig detects the project root, loads layered config, applies CLI flag
// overrides, and stores the result in the command context.
func loadConfig(cmd *cobra.Command) error {
	// Detect project root via git. If not in a git repo, use cwd.
	projectRoot, err := detectProjectRoot()
	if err != nil {
		// Not in a git repo -- use cwd (allows version/completion/help to work anywhere)
		projectRoot, _ = os.Getwd()
	}

	cfg, ac, err := config.Load(projectRoot)
	if err != nil {
		// Config errors are warnings, not fatal -- use defaults so ssu still works
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: config load error: %v (using defaults)\n", err)
		defaults := config.Defaults()
		cfg = &defaults
		ac = config.NewAnnotatedConfig()
	}

	// Apply CLI flag overrides (only when explicitly set by user)
	if cmd.Flags().Changed("jobs") {
		jobs, _ := cmd.Flags().GetInt("jobs")
		cfg.Git.ParallelJobs = jobs
		ac.Set("git.parallel_jobs", jobs, config.SourceFlag)
	}

	// Store config in context for subcommands
	ctx := config.WithConfig(cmd.Context(), cfg)
	ctx = config.WithAnnotated(ctx, ac)
	cmd.SetContext(ctx)

	// Initialize logger (skip for lightweight utility commands)
	switch cmd.Name() {
	case "version", "completion":
		// No file logging for utility commands
	default:
		initLogger(cmd, cfg, projectRoot)
	}

	return nil
}

// initLogger sets up file logging. Failure is non-fatal: if the log directory
// cannot be created, a warning is printed to stderr and the command continues.
func initLogger(cmd *cobra.Command, cfg *config.Config, projectRoot string) {
	projectName := filepath.Base(projectRoot)
	logDir := logging.LogDir(projectName)

	verbose, _ := cmd.Flags().GetBool("verbose")

	logger, err := logging.InitLogger(logDir, verbose, cfg.Log.MaxSizeMB, cfg.Log.MaxBackups)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not initialize logger: %v\n", err)
		return
	}

	slog.SetDefault(logger)
	slog.Info("SSU started", "project", projectName)
}

// detectProjectRoot returns the git repository root directory.
func detectProjectRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
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
		{"exec", "Run command in submodules"},
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
	fmt.Fprint(cmd.OutOrStdout(), "Choose [1-7]: ")

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
		subcmd = "exec"
	case "5":
		subcmd = "rollback"
	case "6":
		subcmd = "backup"
	case "7":
		return cmd.Help()
	default:
		fmt.Fprintf(cmd.ErrOrStderr(), "Invalid choice: %s\n", choice)
		return nil
	}

	// Dispatch to the chosen subcommand
	cmd.SetArgs([]string{subcmd})
	return cmd.Execute()
}
