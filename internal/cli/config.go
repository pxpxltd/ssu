package cli

import (
	"fmt"
	"strings"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/spf13/cobra"
)

// NewConfigCmd creates the config command with subcommands.
func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "View configuration",
		Long:  "Display the current SSU configuration and which layer set each value.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(NewConfigShowCmd())
	return cmd
}

// configSection groups config keys under a heading for display.
type configSection struct {
	name string
	keys []configEntry
}

// configEntry represents a single config key for display.
type configEntry struct {
	key   string
	value any
}

// NewConfigShowCmd creates the "config show" subcommand that prints
// the merged configuration with source annotations.
func NewConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show merged configuration with sources",
		Long: `Display all configuration values and which layer set each one.

Sources (in priority order):
  flag      CLI flag (--jobs, etc.)
  env       Environment variable (SSU_GIT_PARALLEL_JOBS, etc.)
  project   Project config (.ssu.yaml)
  global    Global config (~/.ssu/config.yaml)
  default   Built-in default`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromContext(cmd.Context())
			ac := config.AnnotatedFromContext(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("config not loaded")
			}

			p := output.NewPrinter(cmd.OutOrStdout())
			_ = p

			sections := buildSections(cfg)

			for i, section := range sections {
				if i > 0 {
					fmt.Fprintln(cmd.OutOrStdout())
				}
				output.Bold.Fprintf(cmd.OutOrStdout(), "[%s]\n", section.name)

				for _, entry := range section.keys {
					source := config.SourceDefault
					if ac != nil {
						source = ac.Get(entry.key).Source
					}

					valStr := formatValue(entry.value)
					sourceStr := formatSource(source)

					fmt.Fprintf(cmd.OutOrStdout(), "  %-24s = %-20s %s\n",
						entry.key, valStr, sourceStr)
				}
			}

			return nil
		},
	}
}

// buildSections organizes config values into display sections.
func buildSections(cfg *config.Config) []configSection {
	return []configSection{
		{
			name: "Git",
			keys: []configEntry{
				{"git.parallel_jobs", cfg.Git.ParallelJobs},
				{"git.skip", cfg.Git.Skip},
				{"git.fail_fast", cfg.Git.FailFast},
			},
		},
		{
			name: "Branches",
			keys: []configEntry{
				{"branches.priority", cfg.Branches.Priority},
				{"branches.override", cfg.Branches.Override},
			},
		},
		{
			name: "Backup",
			keys: []configEntry{
				{"backup.enabled", cfg.Backup.Enabled},
				{"backup.max_backups", cfg.Backup.MaxBackups},
			},
		},
		{
			name: "Log",
			keys: []configEntry{
				{"log.max_size_mb", cfg.Log.MaxSizeMB},
				{"log.max_backups", cfg.Log.MaxBackups},
			},
		},
	}
}

// formatValue formats a config value for display.
func formatValue(v any) string {
	switch val := v.(type) {
	case []string:
		if len(val) == 0 {
			return "[]"
		}
		return "[" + strings.Join(val, ", ") + "]"
	case string:
		if val == "" {
			return "(unset)"
		}
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

// formatSource formats a source annotation for display, with color when available.
func formatSource(s config.Source) string {
	return output.Muted.Sprintf("(%s)", s)
}
