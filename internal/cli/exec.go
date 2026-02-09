package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/cli/tui"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
)

// NewExecCmd creates the exec subcommand.
func NewExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [command] [args...]",
		Short: "Run a command in submodules",
		Long:  "Execute an arbitrary command in selected submodules.",
		Example: `  ssu exec git status
  ssu exec --auto npm install
  ssu exec ls -la`,
		Args: cobra.MinimumNArgs(1),
		RunE: runExec,
	}

	return cmd
}

func runExec(cmd *cobra.Command, args []string) error {
	cfg := config.FromContext(cmd.Context())

	// Build scan options from config.
	var scanOpts engine.ScanOpts
	if cfg != nil {
		scanOpts = engine.ScanOpts{
			SkipList:    cfg.Git.Skip,
			Concurrency: cfg.Git.ParallelJobs,
			BranchOpts: git.BranchDetectOpts{
				PriorityBranches: cfg.Branches.Priority,
				Override:         cfg.Branches.Override,
			},
		}
	} else {
		scanOpts = engine.ScanOpts{
			Concurrency: 8,
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	scanOpts.RootDir = cwd

	if cmd.Flags().Changed("jobs") {
		jobs, _ := cmd.Flags().GetInt("jobs")
		scanOpts.Concurrency = jobs
	}

	// Set up cancellation.
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
		}
	}()
	defer signal.Stop(sigCh)

	eng := engine.New(git.NewExecGit())
	pr := output.NewPrinter(cmd.OutOrStdout())

	// Scan submodules to get the full list.
	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	// Exclude root -- exec only runs in submodules, not root.
	subs := result.Submodules
	if len(subs) == 0 {
		pr.Info("No submodules found.")
		return nil
	}

	// Filter out skipped submodules for auto mode targets.
	var nonSkipped []*engine.SubmoduleInfo
	for _, sm := range subs {
		if !sm.HasStatus(git.StatusSkipped) {
			nonSkipped = append(nonSkipped, sm)
		}
	}

	autoMode, _ := cmd.Flags().GetBool("auto")
	isTTY := output.IsTTY()

	var targets []*engine.SubmoduleInfo

	if !isTTY || autoMode {
		// Auto mode: run in all non-skipped submodules.
		targets = nonSkipped
	} else {
		// Interactive mode: TUI selector with ALL non-skipped submodules.
		items := tui.SubmoduleItems(nonSkipped)
		cmdLabel := args[0]
		if len(args) > 1 {
			cmdLabel = args[0] + " " + args[1]
			if len(args) > 2 {
				cmdLabel += " ..."
			}
		}

		selModel := tui.NewSelectorModel(items, tui.SelectorOpts{
			Title:     fmt.Sprintf("Select submodules for: %s", cmdLabel),
			Operation: "exec",
		})

		p := tea.NewProgram(selModel)
		finalModel, err := p.Run()
		if err != nil {
			return fmt.Errorf("TUI selector: %w", err)
		}

		sm := finalModel.(tui.SelectorModel)
		if sm.Cancelled() {
			pr.Info("Exec cancelled.")
			return nil
		}
		if !sm.Confirmed() {
			pr.Info("No submodules selected.")
			return nil
		}

		targets = sm.Selected()
		if len(targets) == 0 {
			pr.Info("No submodules selected.")
			return nil
		}
	}

	slog.Info("exec: running command", "command", args, "targets", len(targets))

	// Execute command in each target submodule sequentially.
	var failedPaths []string
	for _, sub := range targets {
		if ctx.Err() != nil {
			break
		}

		subDir := filepath.Join(cwd, sub.Path)

		// Print separator header.
		output.Bold.Fprintf(cmd.OutOrStdout(), "==> %s\n", sub.Path)

		// Create and run the command.
		c := exec.CommandContext(ctx, args[0], args[1:]...)
		c.Dir = subDir
		c.Stdout = cmd.OutOrStdout()
		c.Stderr = cmd.ErrOrStderr()

		if err := c.Run(); err != nil {
			pr.Errorf("%s: %v", sub.Path, err)
			failedPaths = append(failedPaths, sub.Path)
			// Continue to next submodule on error.
		}
	}

	// Summary.
	total := len(targets)
	failCount := len(failedPaths)
	fmt.Fprintf(cmd.OutOrStdout(), "\nRan in %d submodule(s), %d failed\n", total, failCount)

	if failCount > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}
