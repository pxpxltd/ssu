package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/backup"
	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/cli/tui"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
)

// NewUpdateCmd creates the update subcommand.
func NewUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update submodules",
		Long:  "Fetch and merge updates for selected submodules with automatic conflict handling.",
		Example: `  ssu update
  ssu update --auto
  ssu update --dry-run`,
		RunE: runUpdate,
	}

	return cmd
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	// Parse flags.
	autoMode, _ := cmd.Flags().GetBool("auto")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Get project root.
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Get config from context.
	cfg := config.FromContext(cmd.Context())

	// Build scan options.
	var scanOpts engine.ScanOpts
	if cfg != nil {
		scanOpts = engine.ScanOpts{
			RootDir:     rootDir,
			SkipList:    cfg.Git.Skip,
			Concurrency: cfg.Git.ParallelJobs,
			BranchOpts: git.BranchDetectOpts{
				PriorityBranches: cfg.Branches.Priority,
				Override:         cfg.Branches.Override,
			},
		}
	} else {
		scanOpts = engine.ScanOpts{
			RootDir:     rootDir,
			Concurrency: 8,
		}
	}

	// Respect --jobs flag override.
	if cmd.Flags().Changed("jobs") {
		jobs, _ := cmd.Flags().GetInt("jobs")
		scanOpts.Concurrency = jobs
	}

	// Create engine and git service.
	gitSvc := git.NewExecGit()
	eng := engine.New(gitSvc)

	// Create context with cancel for Ctrl+C handling.
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	pr := output.NewPrinter(cmd.OutOrStdout())

	// --- Branch: dry-run mode ---
	if dryRun {
		return runUpdateDryRun(ctx, eng, gitSvc, scanOpts, rootDir, pr, cmd)
	}

	// --- Branch: auto mode or non-TTY ---
	if autoMode || !output.IsTTY() {
		return runUpdateAuto(ctx, eng, gitSvc, scanOpts, rootDir, verbose, pr, cmd, cfg)
	}

	// --- Branch: interactive TUI mode ---
	return runUpdateInteractive(ctx, eng, gitSvc, scanOpts, rootDir, pr, cmd, cfg)
}

// ---------------------------------------------------------------------------
// Dry-run mode
// ---------------------------------------------------------------------------

func runUpdateDryRun(ctx context.Context, eng *engine.Engine, gitSvc *git.ExecGit, scanOpts engine.ScanOpts, rootDir string, pr *output.Printer, cmd *cobra.Command) error {
	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	// Report missing (uninitialized) submodules.
	missing := filterMissing(result.Submodules)
	if len(missing) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%d submodule(s) not initialized, would be initialized:\n", len(missing))
		for _, sub := range missing {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", sub.Path)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	pending := filterPending(result.Submodules)
	if len(pending) == 0 {
		pr.Info("All submodules are up to date.")
		return nil
	}

	// Build dry-run diff table.
	t := table.New().
		Headers("Path", "Current SHA", "Target SHA", "Behind").
		Border(lipgloss.NormalBorder()).
		BorderHeader(true).
		BorderColumn(true).
		Width(120)

	for _, sub := range pending {
		subDir := filepath.Join(rootDir, sub.Path)

		currentSHA := resolveRef(ctx, gitSvc, subDir, "HEAD")
		targetSHA := resolveRef(ctx, gitSvc, subDir, "origin/"+sub.TargetBranch)
		behind := strconv.Itoa(sub.CommitsBehind)

		t.Row(sub.Path, currentSHA, targetSHA, behind)
	}

	t.StyleFunc(func(row, col int) lipgloss.Style {
		if row == table.HeaderRow {
			return tui.HeaderStyle
		}
		return lipgloss.NewStyle()
	})

	fmt.Fprintln(cmd.OutOrStdout(), t.Render())
	fmt.Fprintf(cmd.OutOrStdout(), "\n%d submodule(s) would be updated (dry-run)\n", len(pending))
	return nil
}

// ---------------------------------------------------------------------------
// Auto / non-TTY mode
// ---------------------------------------------------------------------------

func runUpdateAuto(ctx context.Context, eng *engine.Engine, gitSvc *git.ExecGit, scanOpts engine.ScanOpts, rootDir string, verbose bool, pr *output.Printer, cmd *cobra.Command, cfg *config.Config) error {
	// Add progress callback for verbose mode.
	if verbose {
		scanOpts.OnProgress = func(evt engine.ProgressEvent) {
			if evt.Type == engine.EventCompleted {
				fmt.Fprintf(os.Stderr, "Scanned %d/%d %s\n", evt.Done, evt.Total, evt.Path)
			}
		}
	}

	result, err := eng.Scan(ctx, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	// Auto-init missing submodules.
	missing := filterMissing(result.Submodules)
	if len(missing) > 0 {
		paths := missingPaths(missing)
		pr.Infof("Initializing %d missing submodule(s)...", len(missing))
		if initErr := gitSvc.SubmoduleInit(ctx, rootDir, paths); initErr != nil {
			pr.Warningf("Init failed: %v (continuing with available submodules)", initErr)
		} else {
			pr.Successf("Initialized %d submodule(s)", len(missing))
			// Re-scan to pick up newly initialized submodules.
			result, err = eng.Scan(ctx, scanOpts)
			if err != nil {
				return fmt.Errorf("re-scanning submodules: %w", err)
			}
		}
	}

	pending := filterPending(result.Submodules)
	if len(pending) == 0 {
		pr.Info("All submodules are up to date.")
		return nil
	}

	// Create backup before updating.
	if err := createBackupIfEnabled(ctx, gitSvc, rootDir, pending, cfg); err != nil {
		pr.Warningf("Backup failed: %v (continuing with update)", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Updating %d submodule(s)...\n", len(pending))

	updateOpts := engine.UpdateOpts{
		RootDir:     rootDir,
		Concurrency: scanOpts.Concurrency,
	}

	if verbose {
		updateOpts.OnProgress = func(evt engine.ProgressEvent) {
			switch evt.Type {
			case engine.EventCompleted:
				fmt.Fprintf(os.Stderr, "Updated %d/%d %s\n", evt.Done, evt.Total, evt.Path)
			case engine.EventFailed:
				fmt.Fprintf(os.Stderr, "Failed %d/%d %s: %v\n", evt.Done, evt.Total, evt.Path, evt.Error)
			}
		}
	}

	updateResult, err := eng.Update(ctx, pending, updateOpts)
	if err != nil {
		return fmt.Errorf("updating submodules: %w", err)
	}

	return printUpdateSummary(cmd, pr, updateResult)
}

// ---------------------------------------------------------------------------
// Interactive TUI mode
// ---------------------------------------------------------------------------

func runUpdateInteractive(ctx context.Context, eng *engine.Engine, gitSvc *git.ExecGit, scanOpts engine.ScanOpts, rootDir string, pr *output.Printer, cmd *cobra.Command, cfg *config.Config) error {
	// Phase A: Scan with spinner.
	scanResult, err := runScanWithSpinner(ctx, eng, scanOpts)
	if err != nil {
		if err == context.Canceled {
			pr.Warning("Scan cancelled.")
			return nil
		}
		return fmt.Errorf("scanning submodules: %w", err)
	}

	// Auto-init missing submodules.
	missing := filterMissing(scanResult.Submodules)
	if len(missing) > 0 {
		paths := missingPaths(missing)
		pr.Infof("Initializing %d missing submodule(s)...", len(missing))
		if initErr := gitSvc.SubmoduleInit(ctx, rootDir, paths); initErr != nil {
			pr.Warningf("Init failed: %v (continuing with available submodules)", initErr)
		} else {
			pr.Successf("Initialized %d submodule(s)", len(missing))
			// Re-scan to pick up newly initialized submodules.
			scanResult, err = runScanWithSpinner(ctx, eng, scanOpts)
			if err != nil {
				if err == context.Canceled {
					pr.Warning("Re-scan cancelled.")
					return nil
				}
				return fmt.Errorf("re-scanning submodules: %w", err)
			}
		}
	}

	pending := filterPending(scanResult.Submodules)
	if len(pending) == 0 {
		pr.Info("All submodules are up to date.")
		return nil
	}

	// Phase B: Select submodules with TUI selector.
	items := tui.SubmoduleItems(pending)
	selModel := tui.NewSelectorModel(items, tui.SelectorOpts{
		Title:      "Select submodules to update",
		Subtitle:   fmt.Sprintf("%d pending of %d submodules scanned", len(pending), len(scanResult.Submodules)),
		ShowDetail: true,
		Operation:  "update",
	})

	p2 := tea.NewProgram(selModel)
	finalModel, err := p2.Run()
	if err != nil {
		return fmt.Errorf("selector: %w", err)
	}

	sm, ok := finalModel.(tui.SelectorModel)
	if !ok {
		return fmt.Errorf("unexpected model type from selector")
	}

	if sm.Cancelled() || !sm.Confirmed() {
		pr.Info("Cancelled.")
		return nil
	}

	targets := sm.Selected()
	if len(targets) == 0 {
		pr.Info("No submodules selected.")
		return nil
	}

	// Phase C: Update with live multi-line status.
	// Create backup before updating.
	if err := createBackupIfEnabled(ctx, gitSvc, rootDir, targets, cfg); err != nil {
		pr.Warningf("Backup failed: %v (continuing with update)", err)
	}

	// Build ProcessModel with target paths.
	paths := make([]string, len(targets))
	for i, t := range targets {
		paths[i] = t.Path
	}
	pm := tui.NewProcessModel(paths, "update")
	p3 := tea.NewProgram(pm)

	updateOpts := engine.UpdateOpts{
		RootDir:     rootDir,
		Concurrency: scanOpts.Concurrency,
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
		result, err := eng.Update(ctx, targets, updateOpts)
		p3.Send(tui.ProcessCompleteMsg{Result: result, Err: err})
	}()

	finalModel, err = p3.Run()
	if err != nil {
		return fmt.Errorf("process TUI: %w", err)
	}

	pm2, ok := finalModel.(tui.ProcessModel)
	if !ok {
		return fmt.Errorf("unexpected model type from process TUI")
	}

	if pm2.Err() != nil {
		if pm2.Err() == context.Canceled {
			if res, ok := pm2.Result().(*engine.UpdateResult); ok && res != nil {
				printPartialResults(cmd, pr, res, len(targets))
			}
			return &exitError{code: ExitError}
		}
		return pm2.Err()
	}

	updateResult, ok := pm2.Result().(*engine.UpdateResult)
	if !ok {
		return fmt.Errorf("unexpected result type from process model")
	}

	return printUpdateSummary(cmd, pr, updateResult)
}

// ---------------------------------------------------------------------------
// TUI scan with progress bar
// ---------------------------------------------------------------------------

func runScanWithSpinner(ctx context.Context, eng *engine.Engine, scanOpts engine.ScanOpts) (*engine.ScanResult, error) {
	sm := tui.NewSpinnerModel()
	p := tea.NewProgram(sm)

	// Wire scan progress to TUI.
	scanOpts.OnProgress = func(evt engine.ProgressEvent) {
		p.Send(tui.FetchProgressMsg{
			Path:  evt.Path,
			Done:  evt.Done,
			Total: evt.Total,
			Err:   evt.Error,
		})
	}

	// Run scan in goroutine, sending completion to TUI.
	go func() {
		result, err := eng.Scan(ctx, scanOpts)
		p.Send(tui.FetchCompleteMsg{Result: result, Err: err})
	}()

	// Run TUI -- blocks until FetchCompleteMsg arrives.
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("spinner: %w", err)
	}

	sm2, ok := finalModel.(tui.SpinnerModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type from spinner")
	}

	if sm2.Err() != nil {
		return nil, sm2.Err()
	}

	result, ok := sm2.Result().(*engine.ScanResult)
	if !ok {
		return nil, fmt.Errorf("unexpected result type from spinner model")
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// filterPending returns only submodules with StatusPending.
func filterPending(subs []*engine.SubmoduleInfo) []*engine.SubmoduleInfo {
	var pending []*engine.SubmoduleInfo
	for _, sub := range subs {
		if sub.HasStatus(git.StatusPending) {
			pending = append(pending, sub)
		}
	}
	return pending
}

// filterMissing returns only submodules with StatusMissing.
func filterMissing(subs []*engine.SubmoduleInfo) []*engine.SubmoduleInfo {
	var missing []*engine.SubmoduleInfo
	for _, sub := range subs {
		if sub.HasStatus(git.StatusMissing) {
			missing = append(missing, sub)
		}
	}
	return missing
}

// missingPaths extracts paths from missing submodules.
func missingPaths(subs []*engine.SubmoduleInfo) []string {
	paths := make([]string, len(subs))
	for i, sub := range subs {
		paths[i] = sub.Path
	}
	return paths
}

// resolveRef resolves a git ref to a short SHA (7 chars).
// Returns "unknown" if the ref cannot be resolved.
func resolveRef(ctx context.Context, gitSvc *git.ExecGit, dir, ref string) string {
	// Use CurrentSHA for HEAD, otherwise rev-parse the ref via the git service.
	if ref == "HEAD" {
		sha, err := gitSvc.CurrentSHA(ctx, dir)
		if err != nil {
			return "unknown"
		}
		if len(sha) > 7 {
			return sha[:7]
		}
		return sha
	}

	// For arbitrary refs, use CurrentSHA won't work; we need rev-parse <ref>.
	// The ExecGit doesn't expose a generic rev-parse, but we can use the
	// underlying ExecGit.run via a fresh git call. Since ExecGit.run is unexported,
	// we'll just call git.CurrentSHA on the dir after noting this is a remote ref.
	// Actually, the simplest approach: create a temporary helper that shells out.
	sha, err := revParseShort(ctx, dir, ref)
	if err != nil {
		return "unknown"
	}
	return sha
}

// revParseShort resolves a ref to a short SHA via git rev-parse --short.
func revParseShort(ctx context.Context, dir, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--short", ref)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// printUpdateSummary prints the final update summary and returns an error
// if any actions failed or had conflicts.
func printUpdateSummary(cmd *cobra.Command, pr *output.Printer, result *engine.UpdateResult) error {
	var updated, skipped, conflicts, errors int
	for _, action := range result.Actions {
		switch {
		case action.ConflictHint != "":
			conflicts++
		case action.Error != nil:
			errors++
		case strings.Contains(action.Action, "merged"):
			updated++
		default:
			skipped++
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	pr.Infof("Summary: %d updated, %d skipped, %d conflicts, %d errors",
		updated, skipped, conflicts, errors)

	// Print conflict hints.
	for _, action := range result.Actions {
		if action.ConflictHint != "" {
			pr.Warningf("Conflict in %s -- resolve with:\n  %s", action.Path, action.ConflictHint)
		}
	}

	// Print error details.
	for _, action := range result.Actions {
		if action.Error != nil && action.ConflictHint == "" {
			pr.Errorf("%s: %v", action.Path, action.Error)
		}
	}

	if conflicts > 0 {
		return &exitError{code: ExitConflict}
	}
	if errors > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}

// printPartialResults prints what was updated before cancellation.
func printPartialResults(cmd *cobra.Command, pr *output.Printer, result *engine.UpdateResult, totalTargets int) {
	completed := len(result.Actions)
	fmt.Fprintf(cmd.OutOrStdout(), "\nCancelled. %d/%d submodules updated before interruption:\n", completed, totalTargets)
	for _, action := range result.Actions {
		if action.Error != nil {
			pr.Errorf("%s: %s", action.Path, action.Action)
		} else {
			pr.Successf("%s: %s", action.Path, action.Action)
		}
	}
}

// createBackupIfEnabled creates a backup of the target submodules before updating.
func createBackupIfEnabled(ctx context.Context, gitSvc *git.ExecGit, rootDir string, targets []*engine.SubmoduleInfo, cfg *config.Config) error {
	// Check if backups are enabled.
	if cfg != nil && !cfg.Backup.Enabled {
		return nil
	}

	projectName := backup.ProjectName(rootDir)
	backupDir, err := backup.BackupDir(projectName)
	if err != nil {
		return err
	}

	// Build submodule state map.
	states := make(map[string]backup.SubmoduleState)
	for _, sub := range targets {
		subDir := filepath.Join(rootDir, sub.Path)
		sha, shaErr := gitSvc.CurrentSHA(ctx, subDir)
		if shaErr != nil {
			slog.Warn("Could not get SHA for backup", "path", sub.Path, "error", shaErr)
			continue
		}
		states[sub.Path] = backup.SubmoduleState{
			SHA:    sha,
			Branch: sub.CurrentBranch,
		}
	}

	if len(states) == 0 {
		return nil
	}

	filename, err := backup.Create(backupDir, states)
	if err != nil {
		return err
	}

	slog.Info("Backup created", "file", filename)
	return nil
}
