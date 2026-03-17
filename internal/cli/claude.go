package cli

import (
	"fmt"

	claudepkg "github.com/pxpxltd/ssu/internal/claude"
	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/spf13/cobra"
)

// NewClaudeCmd creates the claude command with install and snippet subcommands.
func NewClaudeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "claude",
		Short: "Claude Code integration",
		Long: `Manage Claude Code integration for SSU.

Install slash commands for Claude Code and generate CLAUDE.md snippets
that teach Claude about SSU's submodule management capabilities.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(NewClaudeInstallCmd())
	cmd.AddCommand(NewClaudeSnippetCmd())
	return cmd
}

// NewClaudeInstallCmd creates the "claude install" subcommand that copies
// slash command files to ~/.claude/commands/ssu/.
func NewClaudeInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install slash commands for Claude Code",
		Long: `Install SSU slash commands into ~/.claude/commands/ssu/.

This copies command files that enable Claude Code to use SSU directly:
  /ssu:status    - Check submodule status
  /ssu:update    - Update submodules
  /ssu:push      - Push ahead submodules
  /ssu:checkout  - Resolve detached HEAD
  /ssu:project   - Commit submodule pointer changes
  /ssu:exec      - Run command across submodules
  /ssu:rollback  - Restore from backup
  /ssu:rm        - Remove a submodule cleanly
  /ssu:allow     - Fix git dubious ownership errors

Use --force to overwrite existing files that differ from the current version.
Identical files are always skipped.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")

			result, err := claudepkg.InstallCommands(force)
			if err != nil {
				return fmt.Errorf("install failed: %w", err)
			}

			p := output.NewPrinter(cmd.OutOrStdout())

			for _, name := range result.Installed {
				p.Successf("Installed %s", name)
			}
			for _, name := range result.Skipped {
				p.Infof("Skipped %s (identical)", name)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nInstalled %d command(s) to %s\n", len(result.Installed), result.Dir)
			fmt.Fprintln(cmd.OutOrStdout(), "Commands available: /ssu:status, /ssu:update, /ssu:push, /ssu:checkout, /ssu:project, /ssu:exec, /ssu:rollback, /ssu:rm, /ssu:allow")

			return nil
		},
	}

	cmd.Flags().Bool("force", false, "Overwrite existing files that differ")
	return cmd
}

// NewClaudeSnippetCmd creates the "claude snippet" subcommand that prints
// the CLAUDE.md snippet for SSU.
func NewClaudeSnippetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "snippet",
		Short: "Print CLAUDE.md snippet for SSU",
		Long: `Print a markdown block to add to your project's CLAUDE.md file.

This snippet teaches Claude Code about SSU's capabilities so it can
use SSU commands instead of raw git submodule commands.

Usage:
  ssu claude snippet >> CLAUDE.md`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStdout(), claudepkg.SnippetContent)
			return nil
		},
	}
}
