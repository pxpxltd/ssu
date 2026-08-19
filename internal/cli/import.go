package cli

import (
	"fmt"
	"os"
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/cli/tui"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/pxpxltd/ssu/internal/git"
	"github.com/pxpxltd/ssu/internal/stack"
)

func NewImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <filename>.json",
		Short: "Synchronize submodules from an exported stack",
		Args:  cobra.ExactArgs(1),
		Example: `  ssu import .ssu-stack.json
  ssu import .ssu-stack.json --auto
  ssu import .ssu-stack.json --dry-run`,
		RunE: runImport,
	}
}

type stackSelectorItem struct {
	module stack.Module
}

func (i stackSelectorItem) Path() string  { return i.module.Path }
func (i stackSelectorItem) Label() string { return i.module.Path }
func (i stackSelectorItem) Metadata() string {
	branch := i.module.Branch
	if branch == "" {
		branch = "(detached)"
	}
	return fmt.Sprintf("%s @ %s", branch, shortDisplaySHA(i.module.SHA))
}
func (i stackSelectorItem) DetailContent() string {
	return fmt.Sprintf("Branch: %s\nCommit: %s", i.module.Branch, i.module.SHA)
}

func runImport(cmd *cobra.Command, args []string) error {
	file, err := stack.Read(args[0])
	if err != nil {
		return err
	}
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	gitSvc := git.NewExecGit()
	configured, err := gitSvc.SubmodulePaths(cmd.Context(), rootDir)
	if err != nil {
		return fmt.Errorf("listing submodules: %w", err)
	}
	configuredSet := make(map[string]bool, len(configured))
	for _, path := range configured {
		configuredSet[path] = true
	}

	var available []stack.Module
	var unknown, uninitialized []string
	for _, module := range file.Modules {
		switch {
		case !configuredSet[module.Path]:
			unknown = append(unknown, module.Path)
		case !gitSvc.IsSubmoduleInitialized(rootDir, module.Path):
			uninitialized = append(uninitialized, module.Path)
		default:
			available = append(available, module)
		}
	}

	autoMode, _ := cmd.Flags().GetBool("auto")
	selected := available
	if output.IsTTY() && !autoMode && len(available) > 0 {
		selected, err = selectStackModules(available)
		if err != nil {
			return err
		}
		if selected == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "Import cancelled.")
			return nil
		}
	}

	selectedSet := make(map[string]bool, len(selected))
	for _, module := range selected {
		selectedSet[module.Path] = true
	}
	var deselected []string
	for _, module := range available {
		if !selectedSet[module.Path] {
			deselected = append(deselected, module.Path)
		}
	}

	jobs := 8
	if cfg := config.FromContext(cmd.Context()); cfg != nil {
		jobs = cfg.Git.ParallelJobs
	}
	if cmd.Flags().Changed("jobs") {
		jobs, _ = cmd.Flags().GetInt("jobs")
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	actions := stack.NewService(gitSvc).Import(cmd.Context(), selected, stack.ImportOptions{
		RootDir: rootDir, Concurrency: jobs, DryRun: dryRun,
	})
	return printImportSummary(cmd, actions, deselected, unknown, uninitialized, dryRun)
}

func selectStackModules(modules []stack.Module) ([]stack.Module, error) {
	items := make([]tui.SelectorItem, 0, len(modules))
	byPath := make(map[string]stack.Module, len(modules))
	for _, module := range modules {
		items = append(items, stackSelectorItem{module: module})
		byPath[module.Path] = module
	}
	model := tui.NewSelectorModel(items, tui.SelectorOpts{
		Title: "Select submodules to import", Subtitle: fmt.Sprintf("%d modules in stack file", len(modules)),
		ShowDetail: true, Operation: "import", SelectAll: true,
	})
	final, err := tea.NewProgram(model).Run()
	if err != nil {
		return nil, fmt.Errorf("selector: %w", err)
	}
	selectedModel, ok := final.(tui.SelectorModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type from selector")
	}
	if selectedModel.Cancelled() || !selectedModel.Confirmed() {
		return nil, nil
	}
	paths := selectedModel.SelectedPaths()
	selected := make([]stack.Module, 0, len(paths))
	for _, path := range paths {
		selected = append(selected, byPath[path])
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].Path < selected[j].Path })
	return selected, nil
}

func printImportSummary(cmd *cobra.Command, actions []stack.ImportAction, deselected, unknown, uninitialized []string, dryRun bool) error {
	counts := make(map[stack.ImportStatus]int)
	dryRunCount := 0
	synchronizedCount := 0
	for _, action := range actions {
		counts[action.Status]++
		if action.DryRun {
			dryRunCount++
		} else if action.Status == stack.StatusSynced {
			synchronizedCount++
		}
		switch action.Status {
		case stack.StatusFallback:
			verb := "synchronized"
			if action.DryRun {
				verb = "would synchronize"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Warning: %s: exported SHA is unavailable; %s to origin/%s. The exporter probably forgot to push this module.\n", action.Module.Path, verb, action.Module.Branch)
		case stack.StatusDirty:
			fmt.Fprintf(cmd.OutOrStdout(), "Skipped dirty: %s\n", action.Module.Path)
		case stack.StatusDivergent:
			fmt.Fprintf(cmd.OutOrStdout(), "Skipped divergent: %s (target %s)\n", action.Module.Path, action.Target)
		case stack.StatusFailed:
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed: %s: %v\n", action.Module.Path, action.Error)
		}
	}
	printPathGroup(cmd, "Deselected", deselected)
	printPathGroup(cmd, "Unknown", unknown)
	printPathGroup(cmd, "Uninitialized", uninitialized)

	label := "Import complete"
	if dryRun {
		label = "Import dry-run complete"
	}
	fmt.Fprintf(cmd.OutOrStdout(),
		"\n%s: %d synchronized, %d remote fallback, %d dry-run, %d dirty, %d divergent, %d deselected, %d unknown, %d uninitialized, %d failed\n",
		label, synchronizedCount, counts[stack.StatusFallback], dryRunCount,
		counts[stack.StatusDirty], counts[stack.StatusDivergent], len(deselected), len(unknown),
		len(uninitialized), counts[stack.StatusFailed])
	if counts[stack.StatusFailed] > 0 {
		return &exitError{code: ExitError}
	}
	return nil
}

func printPathGroup(cmd *cobra.Command, label string, paths []string) {
	for _, path := range paths {
		fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", label, path)
	}
}

func shortDisplaySHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
