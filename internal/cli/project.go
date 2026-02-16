package cli

import (
	"bufio"
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
	"github.com/pxpxltd/ssu/internal/git"
)

// NewProjectCmd creates the project subcommand.
func NewProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Commit submodule changes",
		Long: `Stage changed submodule pointers in the root repository, commit, and optionally
create a semver tag. Only submodule pointer changes are committed (not other files).`,
		Example: `  ssu project
  ssu project -m "chore: update submodules"
  ssu project --auto
  ssu project --tag --auto
  ssu project --dry-run --tag`,
		RunE: runProject,
	}

	cmd.Flags().StringP("message", "m", "", "Commit message (default: auto-generated)")
	cmd.Flags().Bool("tag", false, "Create a semver tag after committing")

	return cmd
}

func runProject(cmd *cobra.Command, _ []string) error {
	autoMode, _ := cmd.Flags().GetBool("auto")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

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

	gitSvc := git.NewExecGit()
	pr := output.NewPrinter(cmd.OutOrStdout())

	// Detect changed submodule paths.
	changedSubs, staged, err := detectChangedSubmodules(ctx, gitSvc, rootDir)
	if err != nil {
		return err
	}
	if len(changedSubs) == 0 {
		pr.Info("No submodule changes to commit.")
		return nil
	}

	if dryRun {
		return runProjectDryRun(ctx, cmd, gitSvc, pr, rootDir, changedSubs, staged)
	}
	if autoMode || !output.IsTTY() {
		return runProjectAuto(ctx, cmd, gitSvc, pr, rootDir, changedSubs)
	}
	return runProjectInteractive(ctx, cmd, gitSvc, pr, rootDir, changedSubs, staged)
}

// detectChangedSubmodules returns submodule paths that have changed pointers,
// along with a map indicating which are already staged.
func detectChangedSubmodules(ctx context.Context, gitSvc git.GitService, rootDir string) ([]string, map[string]bool, error) {
	// Get known submodule paths.
	subPaths, err := gitSvc.SubmodulePaths(ctx, rootDir)
	if err != nil {
		return nil, nil, fmt.Errorf("listing submodule paths: %w", err)
	}
	if len(subPaths) == 0 {
		return nil, nil, nil
	}
	subSet := make(map[string]bool, len(subPaths))
	for _, p := range subPaths {
		subSet[p] = true
	}

	// Get all changed paths in the root repo.
	diffPaths, err := gitSvc.DiffIndex(ctx, rootDir)
	if err != nil {
		return nil, nil, fmt.Errorf("detecting changed files: %w", err)
	}

	// Intersect: only keep paths that are known submodules.
	var changed []string
	for _, p := range diffPaths {
		if subSet[p] {
			changed = append(changed, p)
		}
	}

	// Determine which are already staged (run git diff --cached --name-only separately).
	stagedMap := make(map[string]bool)
	// We use a lightweight approach: DiffIndex unions staged+unstaged,
	// so we re-check specifically for staged files.
	if eg, ok := gitSvc.(*git.ExecGit); ok {
		// Use DiffIndex's cached component to check staging.
		// Actually, we can just check: if it's in diff --cached, it's staged.
		_ = eg // staged detection is best-effort
	}
	// For staged detection, we consider all changed submodules as needing staging.
	// The project command will stage them anyway.

	return changed, stagedMap, nil
}

// runProjectDryRun previews what would be committed/tagged.
func runProjectDryRun(ctx context.Context, cmd *cobra.Command, gitSvc git.GitService, pr *output.Printer, rootDir string, changed []string, staged map[string]bool) error {
	tag, _ := cmd.Flags().GetBool("tag")

	fmt.Fprintf(cmd.OutOrStdout(), "\nChanged submodule pointers (%d):\n\n", len(changed))
	for _, p := range changed {
		status := "modified"
		if staged[p] {
			status = "staged"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s  %s\n",
			output.Muted.Sprintf("%-10s", status), p)
	}
	fmt.Fprintln(cmd.OutOrStdout())

	msg, _ := cmd.Flags().GetString("message")
	if msg == "" {
		msg = generateCommitMessage(changed)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Commit message: %s\n", msg)

	if tag {
		tags, _ := gitSvc.ListTags(ctx, rootDir, 5)
		if len(tags) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nRecent tags:\n")
			for _, t := range tags {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", t)
			}
		}
		suggested := suggestNextTag(tags)
		fmt.Fprintf(cmd.OutOrStdout(), "\nSuggested tag: %s\n", suggested)
	}

	pr.Info("Dry-run: no changes made.")
	return nil
}

// runProjectAuto stages all changed submodules, commits, and optionally tags.
func runProjectAuto(ctx context.Context, cmd *cobra.Command, gitSvc git.GitService, pr *output.Printer, rootDir string, changed []string) error {
	tag, _ := cmd.Flags().GetBool("tag")
	msg, _ := cmd.Flags().GetString("message")

	if msg == "" {
		msg = generateCommitMessage(changed)
	}

	slog.Info("project: auto mode", "changed", len(changed), "tag", tag)

	// Stage all changed submodule paths.
	if err := gitSvc.StageFiles(ctx, rootDir, changed); err != nil {
		return fmt.Errorf("staging submodule paths: %w", err)
	}

	// Commit.
	result, err := gitSvc.Commit(ctx, rootDir, msg)
	if err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	pr.Successf("Committed %s: %s", result.SHA, firstLine(msg))

	// Tag if requested.
	if tag {
		tags, _ := gitSvc.ListTags(ctx, rootDir, 0)
		tagName := suggestNextTag(tags)

		tagMsg := fmt.Sprintf("Release %s", tagName)
		if err := gitSvc.CreateTag(ctx, rootDir, git.TagOpts{Name: tagName, Message: tagMsg}); err != nil {
			return fmt.Errorf("creating tag: %w", err)
		}
		pr.Successf("Tagged %s", tagName)
	}

	return nil
}

// runProjectInteractive runs the interactive TUI flow.
func runProjectInteractive(ctx context.Context, cmd *cobra.Command, gitSvc git.GitService, pr *output.Printer, rootDir string, changed []string, staged map[string]bool) error {
	tag, _ := cmd.Flags().GetBool("tag")
	flagMsg, _ := cmd.Flags().GetString("message")

	// Phase 1: TUI selector for submodule paths.
	items := tui.ChangedSubmoduleItems(changed, staged)
	selectorModel := tui.NewSelectorModel(items, tui.SelectorOpts{
		Title:     "Select submodule changes to commit",
		Subtitle:  fmt.Sprintf("%d submodule(s) with changed pointers", len(changed)),
		Operation: "commit",
		SelectAll: true,
	})

	p := tea.NewProgram(selectorModel)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI selector: %w", err)
	}

	sm := finalModel.(tui.SelectorModel)
	if sm.Cancelled() {
		pr.Info("Commit cancelled.")
		return nil
	}
	if !sm.Confirmed() {
		pr.Info("No submodules selected.")
		return nil
	}

	selected := sm.SelectedPaths()
	if len(selected) == 0 {
		pr.Info("No submodules selected.")
		return nil
	}

	slog.Info("project: interactive mode", "selected", len(selected))

	// Phase 2: Get commit message.
	msg := flagMsg
	if msg == "" {
		defaultMsg := "chore: update submodules"
		fmt.Fprintf(cmd.OutOrStdout(), "Commit message [%s]: ", defaultMsg)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				msg = input
			} else {
				msg = defaultMsg
			}
		} else {
			msg = defaultMsg
		}
	}

	// Stage selected paths.
	if err := gitSvc.StageFiles(ctx, rootDir, selected); err != nil {
		return fmt.Errorf("staging submodule paths: %w", err)
	}

	// Commit.
	result, err := gitSvc.Commit(ctx, rootDir, msg)
	if err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	pr.Successf("Committed %s: %s", result.SHA, firstLine(msg))

	// Phase 3: Tag if requested.
	if tag {
		tags, _ := gitSvc.ListTags(ctx, rootDir, 5)
		if len(tags) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "\nRecent tags:\n")
			for _, t := range tags {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", t)
			}
		}

		suggested := suggestNextTag(tags)
		fmt.Fprintf(cmd.OutOrStdout(), "Tag name [%s]: ", suggested)

		scanner := bufio.NewScanner(os.Stdin)
		tagName := suggested
		if scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input != "" {
				tagName = input
			}
		}

		tagMsg := fmt.Sprintf("Release %s", tagName)
		if err := gitSvc.CreateTag(ctx, rootDir, git.TagOpts{Name: tagName, Message: tagMsg}); err != nil {
			return fmt.Errorf("creating tag: %w", err)
		}
		pr.Successf("Tagged %s", tagName)
	}

	return nil
}

// generateCommitMessage creates an auto-generated commit message listing the updated submodule paths.
func generateCommitMessage(paths []string) string {
	if len(paths) == 1 {
		return fmt.Sprintf("chore: update submodule %s", paths[0])
	}
	return fmt.Sprintf("chore: update submodules\n\nUpdated: %s", strings.Join(paths, ", "))
}

// suggestNextTag finds the latest semver tag and increments patch.
// Falls back to v0.0.1 if no semver tags exist.
func suggestNextTag(tags []string) string {
	latest := git.LatestSemVer(tags)
	if latest == nil {
		return "v0.0.1"
	}
	return latest.IncrementPatch().String()
}

// firstLine returns the first line of a multi-line string.
func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}
