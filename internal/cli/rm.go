package cli

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/git"
)

// NewRmCmd creates the rm subcommand for removing submodules.
func NewRmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm <path>",
		Short: "Remove a submodule",
		Long: `Fully remove a git submodule in three steps:

  1. Deinit submodule (git submodule deinit -f)
  2. Remove from index (git rm -f)
  3. Clean up .git/modules/<path>

Use --dry-run to preview without modifying anything.`,
		Example: `  ssu rm plugins/old-module
  ssu rm --dry-run plugins/old-module
  ssu rm --auto plugins/old-module`,
		Args: cobra.ExactArgs(1),
		RunE: runRm,
	}

	return cmd
}

func runRm(cmd *cobra.Command, args []string) error {
	subPath := args[0]
	ctx := cmd.Context()
	p := output.NewPrinter(cmd.OutOrStdout())

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	autoMode, _ := cmd.Flags().GetBool("auto")

	projectRoot, err := detectProjectRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository")
	}

	gitSvc := git.NewExecGit()

	// Validate path is a known submodule.
	paths, err := gitSvc.SubmodulePaths(ctx, projectRoot)
	if err != nil {
		return fmt.Errorf("failed to list submodules: %w", err)
	}

	found := false
	for _, sp := range paths {
		if sp == subPath {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("'%s' is not a submodule", subPath)
	}

	// Gather submodule details.
	subDir := filepath.Join(projectRoot, subPath)
	details := gatherSubmoduleDetails(ctx, gitSvc, subDir, subPath)

	// Display details.
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Submodule: %s\n", subPath)
	fmt.Fprintf(cmd.OutOrStdout(), "  Branch:  %s\n", details.branch)
	fmt.Fprintf(cmd.OutOrStdout(), "  SHA:     %s\n", details.sha)
	fmt.Fprintf(cmd.OutOrStdout(), "  Status:  %s\n", details.status)
	fmt.Fprintln(cmd.OutOrStdout())

	// Dry-run: show what would happen and return.
	if dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "Dry-run: the following steps would be performed:")
		fmt.Fprintf(cmd.OutOrStdout(), "  1. git submodule deinit -f %s\n", subPath)
		fmt.Fprintf(cmd.OutOrStdout(), "  2. git rm -f %s\n", subPath)
		fmt.Fprintf(cmd.OutOrStdout(), "  3. rm -rf .git/modules/%s\n", subPath)
		return nil
	}

	// Confirm (TTY + not auto).
	if output.IsTTY() && !autoMode {
		fmt.Fprintln(cmd.OutOrStdout(), "Remove this submodule? This will:")
		fmt.Fprintf(cmd.OutOrStdout(), "  1. Deinit submodule (git submodule deinit -f)\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  2. Remove from index (git rm -f)\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  3. Clean up .git/modules/%s\n", subPath)
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprint(cmd.OutOrStdout(), "[y/N]: ")

		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return nil
		}
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			p.Info("Cancelled.")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	slog.Info("Removing submodule", "path", subPath)

	// Step 1: Deinit.
	if err := gitSvc.SubmoduleDeinit(ctx, projectRoot, subPath); err != nil {
		p.Errorf("Failed to deinit %s: %v", subPath, err)
		return err
	}
	p.Successf("Deinitialized %s", subPath)

	// Step 2: Remove from index.
	if err := gitSvc.RemovePath(ctx, projectRoot, subPath); err != nil {
		p.Errorf("Failed to remove %s from index: %v", subPath, err)
		return err
	}
	p.Successf("Removed from index")

	// Step 3: Clean up .git/modules/<path>.
	modulesDir := filepath.Join(projectRoot, ".git", "modules", subPath)
	if err := os.RemoveAll(modulesDir); err != nil {
		p.Errorf("Failed to clean .git/modules/%s: %v", subPath, err)
		return err
	}
	p.Successf("Cleaned .git/modules/%s", subPath)

	fmt.Fprintln(cmd.OutOrStdout())
	p.Info("Run 'ssu project' to commit this removal.")

	slog.Info("Submodule removed", "path", subPath)
	return nil
}

// submoduleDetails holds gathered info about a submodule for display.
type submoduleDetails struct {
	branch string
	sha    string
	status string
}

// gatherSubmoduleDetails collects branch, SHA, and status info for display.
func gatherSubmoduleDetails(ctx context.Context, gitSvc git.GitService, subDir, subPath string) submoduleDetails {
	d := submoduleDetails{
		branch: "unknown",
		sha:    "unknown",
		status: "unknown",
	}

	initialized := gitSvc.IsSubmoduleInitialized(filepath.Dir(subDir), subPath)

	if !initialized {
		d.status = "not initialized"
		return d
	}

	br, err := gitSvc.CurrentBranch(ctx, subDir)
	if err == nil {
		if br.Detached {
			d.branch = "detached"
		} else {
			d.branch = br.Name
		}
	}

	sha, err := gitSvc.CurrentSHA(ctx, subDir)
	if err == nil {
		if len(sha) > 7 {
			sha = sha[:7]
		}
		d.sha = sha
	}

	// Build status string.
	var parts []string
	parts = append(parts, "initialized")

	hasChanges, err := gitSvc.HasLocalChanges(ctx, subDir)
	if err == nil {
		if hasChanges {
			parts = append(parts, "local changes")
		} else {
			parts = append(parts, "no local changes")
		}
	}

	d.status = strings.Join(parts, ", ")
	return d
}
