package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// NewVersionCmd creates the version subcommand that prints build information.
func NewVersionCmd(version, commit, date string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "ssu %s\n", version)
			fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", commit)
			fmt.Fprintf(cmd.OutOrStdout(), "  built:  %s\n", date)
			fmt.Fprintf(cmd.OutOrStdout(), "  go:     %s\n", runtime.Version())
		},
	}
}
