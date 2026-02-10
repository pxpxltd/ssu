package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/cli/tui"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
)

// NewPushCmd creates the push subcommand.
func NewPushCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push ahead submodules",
		Long:  "Push unpushed commits in submodules to their remote tracking branches.",
		Example: `  ssu push
  ssu push --auto`,
		RunE: runPush,
	}

	return cmd
}

func runPush(cmd *cobra.Command, _ []string) error {
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

	autoMode, _ := cmd.Flags().GetBool("auto")
	isTTY := output.IsTTY()

	if !isTTY || autoMode {
		return runPushAuto(ctx, cmd, eng, pr, scanOpts)
	}
	return runPushInteractive(ctx, cmd, eng, pr, scanOpts)
}

// runPushAuto scans and pushes all ahead submodules without prompts.
func runPushAuto(ctx context.Context, cmd *cobra.Command, eng *engine.Engine, pr *output.Printer, scanOpts engine.ScanOpts) error {
	// Scan with text progress.
	scanOpts.OnProgress = func(evt engine.ProgressEvent) {
		if evt.Type == engine.EventStarted {
			tui.PrintProgressLine(cmd.OutOrStdout(), evt.Done, evt.Total, evt.Path)
		}
	}

	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	// Filter to ahead submodules only.
	var targets []*engine.SubmoduleInfo
	for _, sm := range result.Submodules {
		if sm.HasStatus(git.StatusAhead) {
			targets = append(targets, sm)
		}
	}

	if len(targets) == 0 {
		pr.Info("No submodules have unpushed commits.")
		return nil
	}

	slog.Info("push: auto mode", "targets", len(targets))

	// Push all ahead submodules.
	pushOpts := engine.PushOpts{
		RootDir:     scanOpts.RootDir,
		Concurrency: scanOpts.Concurrency,
	}

	pushed := 0
	failed := 0
	pushOpts.OnProgress = func(evt engine.ProgressEvent) {
		if evt.Type == engine.EventCompleted {
			pushed++
		} else if evt.Type == engine.EventFailed {
			failed++
		}
	}

	pushResult, err := eng.Push(ctx, targets, pushOpts)
	if err != nil {
		return fmt.Errorf("pushing submodules: %w", err)
	}

	// Stream results.
	printPushResults(pr, pushResult)

	// Summary.
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d pushed, %d failed\n", pushed, failed)

	if failed > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}

// runPushInteractive runs the 3-phase TUI flow: scan -> select -> push.
func runPushInteractive(ctx context.Context, cmd *cobra.Command, eng *engine.Engine, pr *output.Printer, scanOpts engine.ScanOpts) error {
	// Phase A: Scan with progress bar.
	total := 0
	scanOpts.OnProgress = func(evt engine.ProgressEvent) {
		if evt.Total > total {
			total = evt.Total
		}
	}

	// Do a quick pre-scan to get the total count for the progress model.
	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	// Filter to ahead submodules only.
	var targets []*engine.SubmoduleInfo
	for _, sm := range result.Submodules {
		if sm.HasStatus(git.StatusAhead) {
			targets = append(targets, sm)
		}
	}

	if len(targets) == 0 {
		pr.Info("No submodules have unpushed commits.")
		return nil
	}

	// Phase B: TUI selector filtered to ahead submodules.
	items := tui.SubmoduleItems(result.Submodules)
	selectorModel := tui.NewSelectorModel(items, tui.SelectorOpts{
		Title:      "Select submodules to push",
		Subtitle:   fmt.Sprintf("%d ahead of %d submodules scanned", len(targets), len(result.Submodules)),
		ShowDetail: false,
		FilterFn:   tui.FilterAhead(),
		Operation:  "push",
	})

	p := tea.NewProgram(selectorModel)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI selector: %w", err)
	}

	sm := finalModel.(tui.SelectorModel)
	if sm.Cancelled() {
		pr.Info("Push cancelled.")
		return nil
	}
	if !sm.Confirmed() {
		pr.Info("No submodules selected.")
		return nil
	}

	selected := sm.Selected()
	if len(selected) == 0 {
		pr.Info("No submodules selected.")
		return nil
	}

	slog.Info("push: interactive mode", "selected", len(selected))

	// Phase C: Push selected submodules with result streaming.
	pushOpts := engine.PushOpts{
		RootDir:     scanOpts.RootDir,
		Concurrency: scanOpts.Concurrency,
	}

	pushed := 0
	failed := 0
	pushOpts.OnProgress = func(evt engine.ProgressEvent) {
		if evt.Type == engine.EventCompleted {
			pushed++
		} else if evt.Type == engine.EventFailed {
			failed++
		}
	}

	pushResult, err := eng.Push(ctx, selected, pushOpts)
	if err != nil {
		return fmt.Errorf("pushing submodules: %w", err)
	}

	// Stream results.
	printPushResults(pr, pushResult)

	// Summary banner.
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d pushed, %d failed\n", pushed, failed)

	if failed > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}

// printPushResults streams individual push action results to the printer.
func printPushResults(pr *output.Printer, result *engine.PushResult) {
	for _, action := range result.Actions {
		if action.Error != nil {
			pr.Errorf("%s -- %s", action.Path, action.Error)
		} else if strings.Contains(action.Action, "skipped") {
			pr.Warningf("%s (detached HEAD -- cannot push without a branch)", action.Path)
		} else {
			pr.Successf("%s %s", action.Path, action.Action)
		}
	}
}
