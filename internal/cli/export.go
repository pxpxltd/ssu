package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/git"
	"github.com/pxpxltd/ssu/internal/stack"
)

func NewExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "export <filename>.json",
		Short:   "Export the current submodule stack",
		Args:    cobra.ExactArgs(1),
		Example: "  ssu export .ssu-stack.json",
		RunE:    runExport,
	}
}

func runExport(cmd *cobra.Command, args []string) error {
	if !strings.EqualFold(filepath.Ext(args[0]), ".json") {
		return fmt.Errorf("export filename must end in .json")
	}
	rootDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	service := stack.NewService(git.NewExecGit())
	file, err := service.Export(cmd.Context(), rootDir)
	if err != nil {
		return err
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "Would export %d submodule(s) to %s (dry-run)\n", len(file.Modules), args[0])
		return nil
	}
	if err := stack.Write(args[0], file); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Exported %d submodule(s) to %s\n", len(file.Modules), args[0])
	return nil
}
