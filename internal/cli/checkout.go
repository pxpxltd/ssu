package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/cli/tui"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
)

// NewCheckoutCmd creates the checkout subcommand.
func NewCheckoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checkout",
		Short: "Resolve detached HEAD in submodules",
		Long: `Re-attach HEAD to the correct branch in submodules with detached HEAD.

Finds which branches point at the current commit and checks out the best match.
Safe: only checks out branches whose tip equals HEAD (no commits gained or lost).

Branch priority: feature branch > develop > master > main`,
		Example: `  ssu checkout
  ssu checkout --auto
  ssu checkout --dry-run`,
		RunE: runCheckout,
	}

	cmd.Flags().Bool("reset", false, "Reset all submodules to match root project's recorded commits")

	return cmd
}

func runCheckout(cmd *cobra.Command, _ []string) error {
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

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	autoMode, _ := cmd.Flags().GetBool("auto")
	isTTY := output.IsTTY()

	// Build checkout branch opts from config.
	branchOpts := git.BranchCheckoutOpts{DefaultRemote: "origin"}
	if cfg != nil && len(cfg.Branches.Priority) > 0 {
		branchOpts.PriorityBranches = cfg.Branches.Priority
	}

	resetMode, _ := cmd.Flags().GetBool("reset")
	if resetMode {
		gitSvc := git.NewExecGit()
		if dryRun {
			return runResetDryRun(ctx, eng, gitSvc, scanOpts, branchOpts, pr, cmd)
		}
		if !isTTY || autoMode {
			return runResetAuto(ctx, eng, gitSvc, scanOpts, branchOpts, pr, cmd)
		}
		return runResetInteractive(ctx, eng, gitSvc, scanOpts, branchOpts, pr, cmd)
	}

	if dryRun {
		return runCheckoutDryRun(ctx, eng, scanOpts, branchOpts, pr, cmd)
	}

	if !isTTY || autoMode {
		return runCheckoutAuto(ctx, eng, scanOpts, branchOpts, pr, cmd)
	}
	return runCheckoutInteractive(ctx, eng, scanOpts, branchOpts, pr, cmd)
}

// ---------------------------------------------------------------------------
// Dry-run mode
// ---------------------------------------------------------------------------

func runCheckoutDryRun(ctx context.Context, eng *engine.Engine, scanOpts engine.ScanOpts, branchOpts git.BranchCheckoutOpts, pr *output.Printer, cmd *cobra.Command) error {
	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	detached := filterDetached(result.Submodules)
	if len(detached) == 0 {
		pr.Info("No submodules have detached HEAD.")
		return nil
	}

	// Resolve target branches for display.
	gitSvc := git.NewExecGit()
	t := table.New().
		Headers("Path", "HEAD SHA", "Target Branch", "Source").
		Border(lipgloss.NormalBorder()).
		BorderHeader(true).
		BorderColumn(true).
		Width(100)

	resolved := 0
	for _, sub := range detached {
		subDir := sub.Path
		if scanOpts.RootDir != "" {
			subDir = scanOpts.RootDir + "/" + sub.Path
		}

		sha, _ := gitSvc.CurrentSHA(ctx, subDir)
		if len(sha) > 7 {
			sha = sha[:7]
		}

		branch, isLocal, _ := git.ResolveBranchForCheckout(ctx, gitSvc, subDir, branchOpts)
		source := "remote"
		if isLocal {
			source = "local"
		}
		if branch == "" {
			branch = "(no match)"
			source = "-"
		} else {
			resolved++
		}

		t.Row(sub.Path, sha, branch, source)
	}

	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return tui.HeaderStyle
		}
		return lipgloss.NewStyle()
	})

	fmt.Fprintln(cmd.OutOrStdout(), t.Render())
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d of %d detached submodule(s) would be checked out (dry-run)\n", resolved, len(detached))
	return nil
}

// ---------------------------------------------------------------------------
// Auto / non-TTY mode
// ---------------------------------------------------------------------------

func runCheckoutAuto(ctx context.Context, eng *engine.Engine, scanOpts engine.ScanOpts, branchOpts git.BranchCheckoutOpts, pr *output.Printer, cmd *cobra.Command) error {
	scanOpts.OnProgress = func(evt engine.ProgressEvent) {
		if evt.Type == engine.EventStarted {
			tui.PrintProgressLine(cmd.OutOrStdout(), evt.Done, evt.Total, evt.Path)
		}
	}

	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	detached := filterDetached(result.Submodules)
	if len(detached) == 0 {
		pr.Info("No submodules have detached HEAD.")
		return nil
	}

	slog.Info("checkout: auto mode", "targets", len(detached))

	checkoutOpts := engine.CheckoutOpts{
		RootDir:     scanOpts.RootDir,
		Concurrency: scanOpts.Concurrency,
		BranchOpts:  branchOpts,
	}

	checked := 0
	skipped := 0
	failed := 0
	checkoutOpts.OnProgress = func(evt engine.ProgressEvent) {
		switch evt.Type {
		case engine.EventCompleted:
			if strings.Contains(evt.Action, "skipped") {
				skipped++
			} else {
				checked++
			}
		case engine.EventFailed:
			failed++
		}
	}

	checkoutResult, err := eng.Checkout(ctx, detached, checkoutOpts)
	if err != nil {
		return fmt.Errorf("checking out submodules: %w", err)
	}

	printCheckoutResults(pr, checkoutResult)
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d checked out, %d skipped, %d failed\n", checked, skipped, failed)

	if failed > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Interactive TUI mode
// ---------------------------------------------------------------------------

func runCheckoutInteractive(ctx context.Context, eng *engine.Engine, scanOpts engine.ScanOpts, branchOpts git.BranchCheckoutOpts, pr *output.Printer, cmd *cobra.Command) error {
	// Phase A: Scan with spinner.
	result, err := runScanWithSpinner(ctx, eng, scanOpts)
	if err != nil {
		if err == context.Canceled {
			pr.Warning("Scan cancelled.")
			return nil
		}
		return fmt.Errorf("scanning submodules: %w", err)
	}

	detached := filterDetached(result.Submodules)
	if len(detached) == 0 {
		pr.Info("No submodules have detached HEAD.")
		return nil
	}

	// Phase B: TUI selector filtered to detached submodules.
	items := tui.SubmoduleItems(result.Submodules)
	selectorModel := tui.NewSelectorModel(items, tui.SelectorOpts{
		Title:      "Select submodules to checkout",
		Subtitle:   fmt.Sprintf("%d detached of %d submodules scanned", len(detached), len(result.Submodules)),
		ShowDetail: false,
		FilterFn:   tui.FilterDetached(),
		Operation:  "checkout",
		SelectAll:  true,
	})

	p := tea.NewProgram(selectorModel)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI selector: %w", err)
	}

	sm := finalModel.(tui.SelectorModel)
	if sm.Cancelled() {
		pr.Info("Checkout cancelled.")
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

	slog.Info("checkout: interactive mode", "selected", len(selected))

	// Phase C: Checkout with live multi-line status.
	paths := make([]string, len(selected))
	for i, s := range selected {
		paths[i] = s.Path
	}
	pm := tui.NewProcessModel(paths, "checkout")
	p3 := tea.NewProgram(pm)

	checkoutOpts := engine.CheckoutOpts{
		RootDir:     scanOpts.RootDir,
		Concurrency: scanOpts.Concurrency,
		BranchOpts:  branchOpts,
		OnProgress: func(evt engine.ProgressEvent) {
			p3.Send(tui.ProcessItemMsg{
				Type:   evt.Type,
				Path:   evt.Path,
				Action: evt.Action,
				Err:    evt.Error,
				Done:   evt.Done,
				Total:  evt.Total,
			})
		},
	}

	go func() {
		result, err := eng.Checkout(ctx, selected, checkoutOpts)
		p3.Send(tui.ProcessCompleteMsg{Result: result, Err: err})
	}()

	finalModel2, err := p3.Run()
	if err != nil {
		return fmt.Errorf("process TUI: %w", err)
	}

	pm2, ok := finalModel2.(tui.ProcessModel)
	if !ok {
		return fmt.Errorf("unexpected model type from process TUI")
	}

	if pm2.Err() != nil {
		if pm2.Err() == context.Canceled {
			pr.Warning("Checkout cancelled.")
			return &exitError{code: ExitError}
		}
		return pm2.Err()
	}

	checkoutResult, ok := pm2.Result().(*engine.CheckoutResult)
	if !ok {
		return fmt.Errorf("unexpected result type from process model")
	}

	// Count results for summary.
	checked := 0
	skipped := 0
	failed := 0
	for _, action := range checkoutResult.Actions {
		switch {
		case action.Error != nil:
			failed++
		case strings.Contains(action.Action, "skipped"):
			skipped++
		default:
			checked++
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n%d checked out, %d skipped, %d failed\n", checked, skipped, failed)

	if failed > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// filterDetached returns only submodules with detached HEAD.
func filterDetached(subs []*engine.SubmoduleInfo) []*engine.SubmoduleInfo {
	var detached []*engine.SubmoduleInfo
	for _, sub := range subs {
		if sub.DetachedHead {
			detached = append(detached, sub)
		}
	}
	return detached
}

// printCheckoutResults streams individual checkout action results to the printer.
func printCheckoutResults(pr *output.Printer, result *engine.CheckoutResult) {
	for _, action := range result.Actions {
		if action.Error != nil {
			pr.Errorf("%s -- %s", action.Path, action.Error)
		} else if strings.Contains(action.Action, "skipped") {
			reason := action.Action
			if strings.Contains(reason, "no matching branch") {
				pr.Warningf("%s (no branch points at HEAD)", action.Path)
			} else {
				pr.Infof("%s %s", action.Path, reason)
			}
		} else if action.Detached {
			pr.Warningf("%s %s", action.Path, action.Action)
		} else {
			pr.Successf("%s %s", action.Path, action.Action)
		}
	}
}

// ---------------------------------------------------------------------------
// Reset mode functions
// ---------------------------------------------------------------------------

func runResetDryRun(ctx context.Context, eng *engine.Engine, gitSvc *git.ExecGit, scanOpts engine.ScanOpts, branchOpts git.BranchCheckoutOpts, pr *output.Printer, cmd *cobra.Command) error {
	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	paths := submodulePaths(result.Submodules)
	recordedSHAs, err := gitSvc.SubmoduleRecordedSHAs(ctx, scanOpts.RootDir, paths)
	if err != nil {
		return fmt.Errorf("reading recorded SHAs: %w", err)
	}

	type resetTarget struct {
		info         *engine.SubmoduleInfo
		recordedSHA  string
		targetBranch string
		detached     bool
		resolveErr   error
	}

	var targets []resetTarget
	for _, sm := range result.Submodules {
		recorded, ok := recordedSHAs[sm.Path]
		if !ok {
			continue
		}
		if sm.CurrentSHA == recorded && !sm.DetachedHead {
			continue
		}
		dir := scanOpts.RootDir + "/" + sm.Path
		opts := branchOpts
		opts.TargetSHA = recorded
		branch, _, resolveErr := git.ResolveBranchForCheckout(ctx, gitSvc, dir, opts)
		targets = append(targets, resetTarget{
			info:         sm,
			recordedSHA:  recorded,
			targetBranch: branch,
			// Only call it detached when resolution actually found no branch --
			// a failed lookup is unknown, not detached.
			detached:   resolveErr == nil && branch == "",
			resolveErr: resolveErr,
		})
	}

	if len(targets) == 0 {
		pr.Info("All submodules match root project's recorded commits.")
		return nil
	}

	t := table.New().
		Headers("Path", "Current", "Recorded SHA", "Target Branch").
		Border(lipgloss.NormalBorder()).
		BorderHeader(true).
		BorderColumn(true).
		Width(120)

	for _, tgt := range targets {
		currentCol := tgt.info.CurrentBranch
		if tgt.info.DetachedHead {
			currentCol = shortSHA(tgt.info.CurrentSHA) + " (detached)"
		}
		recordedCol := shortSHA(tgt.recordedSHA)
		branchCol := tgt.targetBranch
		switch {
		case tgt.resolveErr != nil:
			branchCol = "(resolution failed)"
		case tgt.detached:
			branchCol = recordedCol + " (detached)"
		}
		t.Row(tgt.info.Path, currentCol, recordedCol, branchCol)
	}

	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return tui.HeaderStyle
		}
		return lipgloss.NewStyle()
	})

	fmt.Fprintln(cmd.OutOrStdout(), t.Render())
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d submodule(s) would be reset (dry-run)\n", len(targets))

	for _, tgt := range targets {
		if tgt.resolveErr != nil {
			pr.Warningf("%s -- could not resolve target branch: %v", tgt.info.Path, tgt.resolveErr)
		}
	}
	return nil
}

func runResetAuto(ctx context.Context, eng *engine.Engine, gitSvc *git.ExecGit, scanOpts engine.ScanOpts, branchOpts git.BranchCheckoutOpts, pr *output.Printer, cmd *cobra.Command) error {
	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	paths := submodulePaths(result.Submodules)
	recordedSHAs, err := gitSvc.SubmoduleRecordedSHAs(ctx, scanOpts.RootDir, paths)
	if err != nil {
		return fmt.Errorf("reading recorded SHAs: %w", err)
	}

	targets := filterResetTargets(result.Submodules, recordedSHAs)
	if len(targets) == 0 {
		pr.Info("All submodules match root project's recorded commits.")
		return nil
	}

	slog.Info("checkout: reset auto mode", "targets", len(targets))

	checkoutOpts := engine.CheckoutOpts{
		RootDir:      scanOpts.RootDir,
		Concurrency:  scanOpts.Concurrency,
		BranchOpts:   branchOpts,
		Reset:        true,
		RecordedSHAs: recordedSHAs,
	}

	checked := 0
	skipped := 0
	failed := 0
	checkoutOpts.OnProgress = func(evt engine.ProgressEvent) {
		switch evt.Type {
		case engine.EventCompleted:
			if strings.Contains(evt.Action, "skipped") {
				skipped++
			} else {
				checked++
			}
		case engine.EventFailed:
			failed++
		}
	}

	checkoutResult, err := eng.Checkout(ctx, targets, checkoutOpts)
	if err != nil {
		return fmt.Errorf("checking out submodules: %w", err)
	}

	printCheckoutResults(pr, checkoutResult)
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d checked out, %d skipped, %d failed\n", checked, skipped, failed)

	if failed > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}

func runResetInteractive(ctx context.Context, eng *engine.Engine, gitSvc *git.ExecGit, scanOpts engine.ScanOpts, branchOpts git.BranchCheckoutOpts, pr *output.Printer, cmd *cobra.Command) error {
	scanResult, err := runScanWithSpinner(ctx, eng, scanOpts)
	if err != nil {
		if err == context.Canceled {
			pr.Warning("Scan cancelled.")
			return nil
		}
		return fmt.Errorf("scanning submodules: %w", err)
	}

	paths := submodulePaths(scanResult.Submodules)
	recordedSHAs, err := gitSvc.SubmoduleRecordedSHAs(ctx, scanOpts.RootDir, paths)
	if err != nil {
		return fmt.Errorf("reading recorded SHAs: %w", err)
	}

	targets := filterResetTargets(scanResult.Submodules, recordedSHAs)
	if len(targets) == 0 {
		pr.Info("All submodules match root project's recorded commits.")
		return nil
	}

	items := tui.SubmoduleItems(targets)
	selModel := tui.NewSelectorModel(items, tui.SelectorOpts{
		Title:      "Select submodules to reset",
		Subtitle:   fmt.Sprintf("%d need reset of %d submodules scanned", len(targets), len(scanResult.Submodules)),
		ShowDetail: false,
		Operation:  "checkout",
		SelectAll:  true,
	})

	p := tea.NewProgram(selModel)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("selector: %w", err)
	}

	sm := finalModel.(tui.SelectorModel)
	if sm.Cancelled() || !sm.Confirmed() {
		pr.Info("Cancelled.")
		return nil
	}

	selected := sm.Selected()
	if len(selected) == 0 {
		pr.Info("No submodules selected.")
		return nil
	}

	slog.Info("checkout: reset interactive mode", "selected", len(selected))

	// Checkout with live multi-line status.
	pPaths := make([]string, len(selected))
	for i, s := range selected {
		pPaths[i] = s.Path
	}
	pm := tui.NewProcessModel(pPaths, "checkout")
	p3 := tea.NewProgram(pm)

	checkoutOpts := engine.CheckoutOpts{
		RootDir:      scanOpts.RootDir,
		Concurrency:  scanOpts.Concurrency,
		BranchOpts:   branchOpts,
		Reset:        true,
		RecordedSHAs: recordedSHAs,
		OnProgress: func(evt engine.ProgressEvent) {
			p3.Send(tui.ProcessItemMsg{
				Type:   evt.Type,
				Path:   evt.Path,
				Action: evt.Action,
				Err:    evt.Error,
				Done:   evt.Done,
				Total:  evt.Total,
			})
		},
	}

	go func() {
		result, err := eng.Checkout(ctx, selected, checkoutOpts)
		p3.Send(tui.ProcessCompleteMsg{Result: result, Err: err})
	}()

	finalModel2, err := p3.Run()
	if err != nil {
		return fmt.Errorf("process TUI: %w", err)
	}

	pm2, ok := finalModel2.(tui.ProcessModel)
	if !ok {
		return fmt.Errorf("unexpected model type from process TUI")
	}

	if pm2.Err() != nil {
		if pm2.Err() == context.Canceled {
			pr.Warning("Checkout cancelled.")
			return &exitError{code: ExitError}
		}
		return pm2.Err()
	}

	checkoutResult, ok := pm2.Result().(*engine.CheckoutResult)
	if !ok {
		return fmt.Errorf("unexpected result type from process model")
	}

	checked := 0
	skipped := 0
	failed := 0
	for _, action := range checkoutResult.Actions {
		switch {
		case action.Error != nil:
			failed++
		case strings.Contains(action.Action, "skipped"):
			skipped++
		default:
			checked++
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n%d checked out, %d skipped, %d failed\n", checked, skipped, failed)

	if failed > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Reset helpers
// ---------------------------------------------------------------------------

func submodulePaths(subs []*engine.SubmoduleInfo) []string {
	paths := make([]string, len(subs))
	for i, sm := range subs {
		paths[i] = sm.Path
	}
	return paths
}

func filterResetTargets(subs []*engine.SubmoduleInfo, recordedSHAs map[string]string) []*engine.SubmoduleInfo {
	var targets []*engine.SubmoduleInfo
	for _, sm := range subs {
		recorded, ok := recordedSHAs[sm.Path]
		if !ok {
			continue
		}
		if sm.CurrentSHA == recorded && !sm.DetachedHead {
			continue
		}
		targets = append(targets, sm)
	}
	return targets
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

