package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/spf13/cobra"
)

// NewAllowCmd creates the allow subcommand that adds safe.directory entries
// for the root repository and all submodules that have dubious ownership errors.
func NewAllowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "allow",
		Short: "Fix dubious ownership errors for all submodules",
		Long: `Detect and fix git "dubious ownership" errors by adding safe.directory
entries to the global git config.

This is needed when the repository owner differs from the current user
(common on servers or shared filesystems). The command scans the root
repository and all submodules, and runs:

  git config --global --add safe.directory <path>

for each directory that has the error.`,
		Example: `  ssu allow
  ssu allow --dry-run`,
		RunE: runAllow,
	}

	return cmd
}

func runAllow(cmd *cobra.Command, _ []string) error {
	pr := output.NewPrinter(cmd.OutOrStdout())
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	// Find project root by walking up for .git (can't use git rev-parse since
	// it may itself fail with dubious ownership).
	rootDir, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Parse .gitmodules directly (file read, not git command).
	subPaths, err := parseGitmodules(rootDir)
	if err != nil && !os.IsNotExist(err) {
		pr.Warningf("Could not read .gitmodules: %v", err)
	}

	// Build list of all directories to check: root + submodules.
	allPaths := []string{rootDir}
	for _, sub := range subPaths {
		allPaths = append(allPaths, filepath.Join(rootDir, sub))
	}

	var fixed []string
	var alreadyOK []string

	for _, dir := range allPaths {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		dubious, err := hasDubiousOwnership(dir)
		if err != nil {
			pr.Warningf("Could not check %s: %v", relOrAbs(rootDir, dir), err)
			continue
		}

		label := relOrAbs(rootDir, dir)
		if dir == rootDir {
			label = "(root)"
		}

		if !dubious {
			alreadyOK = append(alreadyOK, label)
			continue
		}

		if dryRun {
			pr.Infof("Would allow: %s", dir)
			fixed = append(fixed, label)
			continue
		}

		if err := addSafeDirectory(dir); err != nil {
			pr.Errorf("Failed to allow %s: %v", label, err)
			continue
		}

		pr.Successf("Allowed: %s", label)
		fixed = append(fixed, label)
	}

	// Summary.
	fmt.Fprintln(cmd.OutOrStdout())
	if len(fixed) == 0 {
		pr.Info("All directories are already allowed.")
	} else if dryRun {
		pr.Infof("Would fix %d director%s (use without --dry-run to apply).", len(fixed), plural(len(fixed)))
	} else {
		pr.Successf("Fixed %d director%s.", len(fixed), plural(len(fixed)))
	}

	return nil
}

// findGitRoot walks up from cwd looking for a .git file or directory.
func findGitRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no .git found")
		}
		dir = parent
	}
}

// parseGitmodules reads .gitmodules and extracts submodule paths.
var submodulePathRe = regexp.MustCompile(`^\s*path\s*=\s*(.+?)\s*$`)

func parseGitmodules(rootDir string) ([]string, error) {
	f, err := os.Open(filepath.Join(rootDir, ".gitmodules"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var paths []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := submodulePathRe.FindStringSubmatch(scanner.Text()); m != nil {
			paths = append(paths, m[1])
		}
	}
	return paths, scanner.Err()
}

// hasDubiousOwnership runs a trivial git command in dir and checks stderr
// for the "dubious ownership" error message.
func hasDubiousOwnership(dir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && strings.Contains(stderr.String(), "dubious ownership") {
		return true, nil
	}
	// Any other error is not a dubious ownership issue.
	return false, nil
}

// addSafeDirectory runs git config --global --add safe.directory <dir>.
func addSafeDirectory(dir string) error {
	cmd := exec.Command("git", "config", "--global", "--add", "safe.directory", dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// relOrAbs returns a relative path from root, or the absolute path on failure.
func relOrAbs(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return dir
	}
	return rel
}

// plural returns "y" for 1, "ies" for other counts.
func plural(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}
