package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/git"
)

// NewInitCmd creates the init subcommand.
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize SSU configuration",
		Long:  "Create a .ssu.yaml configuration file with interactive prompts.",
		Example: `  ssu init`,
		RunE: runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, _ []string) error {
	if !output.IsTTY() {
		return fmt.Errorf("init requires an interactive terminal (TTY)")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Check if config already exists.
	configPath := filepath.Join(cwd, ".ssu.yaml")
	if _, statErr := os.Stat(configPath); statErr == nil {
		return fmt.Errorf("config already exists: .ssu.yaml")
	}

	pr := output.NewPrinter(cmd.OutOrStdout())

	// Detect submodules.
	ctx := context.Background()
	gitSvc := git.NewExecGit()
	paths, err := gitSvc.SubmodulePaths(ctx, cwd)
	if err != nil {
		pr.Warningf("Could not detect submodules: %v", err)
		paths = nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Found %d submodule(s) in project.\n\n", len(paths))

	scanner := bufio.NewScanner(os.Stdin)

	// Prompt for parallel jobs.
	fmt.Fprint(cmd.OutOrStdout(), "Parallel fetch jobs [8]: ")
	parallelJobs := "8"
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			parallelJobs = input
		}
	}

	// Prompt for branch priority.
	fmt.Fprint(cmd.OutOrStdout(), "Branch priority (comma-separated) [develop,master,main]: ")
	branchPriority := "develop,master,main"
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			branchPriority = input
		}
	}

	// Prompt for skip list.
	fmt.Fprint(cmd.OutOrStdout(), "Submodules to skip (comma-separated, empty for none) []: ")
	skipList := ""
	if scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input != "" {
			skipList = input
		}
	}

	// Build YAML content.
	var b strings.Builder
	b.WriteString("# SSU configuration\n")
	b.WriteString("# See: https://github.com/pxpxltd/ssu\n\n")
	b.WriteString("git:\n")
	b.WriteString(fmt.Sprintf("  parallel_jobs: %s\n", parallelJobs))
	if skipList != "" {
		b.WriteString("  skip:\n")
		for _, s := range strings.Split(skipList, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				b.WriteString(fmt.Sprintf("    - %s\n", s))
			}
		}
	} else {
		b.WriteString("  skip: []\n")
	}
	b.WriteString("\nbranches:\n")
	b.WriteString("  priority:\n")
	for _, br := range strings.Split(branchPriority, ",") {
		br = strings.TrimSpace(br)
		if br != "" {
			b.WriteString(fmt.Sprintf("    - %s\n", br))
		}
	}
	b.WriteString("\nbackup:\n")
	b.WriteString("  enabled: true\n")
	b.WriteString("  max_backups: 10\n")
	b.WriteString("\nlog:\n")
	b.WriteString("  max_size_mb: 10\n")
	b.WriteString("  max_backups: 5\n")

	// Write file.
	if err := os.WriteFile(configPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	pr.Successf("Created %s", configPath)
	return nil
}
