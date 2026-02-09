package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/cli/tui"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
)

// NewStatusCmd creates the status subcommand.
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show submodule status",
		Long:  "Display the status of all submodules including branch, commits behind, and modification state.",
		Example: `  ssu status
  ssu status --json`,
		RunE: runStatus,
	}

	cmd.Flags().Bool("json", false, "Output status as JSON")

	return cmd
}

func runStatus(cmd *cobra.Command, _ []string) error {
	// Get config from context (set by PersistentPreRunE in root.go).
	cfg := config.FromContext(cmd.Context())

	// Build scan options from config or defaults.
	var scanOpts engine.ScanOpts
	if cfg != nil {
		scanOpts = engine.ScanOpts{
			SkipList:    cfg.Git.Skip,
			Concurrency: cfg.Git.ParallelJobs,
			BranchOpts: git.BranchDetectOpts{
				PriorityBranches: cfg.Branches.Priority,
				Override:         cfg.Branches.Override,
			},
		}
	} else {
		scanOpts = engine.ScanOpts{
			Concurrency: 8,
		}
	}

	// Determine project root. For now use cwd (Phase 4 config stores this,
	// but the config struct does not expose a ProjectRoot field yet).
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	scanOpts.RootDir = cwd

	// Respect --jobs flag override.
	if cmd.Flags().Changed("jobs") {
		jobs, _ := cmd.Flags().GetInt("jobs")
		scanOpts.Concurrency = jobs
	}

	// Parse --json flag early (needed before scan to decide progress bar).
	jsonFlag, _ := cmd.Flags().GetBool("json")

	// Create engine and scan.
	eng := engine.New(git.NewExecGit())

	var result *engine.ScanResult
	if !jsonFlag && output.IsTTY() {
		result, err = runScanWithProgress(cmd.Context(), eng, scanOpts)
	} else {
		result, err = eng.Scan(cmd.Context(), scanOpts)
	}
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	// Mode branching: --json or table.
	if jsonFlag {
		return printStatusJSON(cmd.OutOrStdout(), result)
	}
	return printStatusTable(cmd.OutOrStdout(), result)
}

// defaultBranches is the set of standard branches that are NOT feature branches.
var defaultBranches = map[string]bool{
	"develop": true,
	"master":  true,
	"main":    true,
}

// isFeatureBranch returns true if the submodule is on a feature branch.
// A feature branch is any branch that is not empty, not detached, and not
// in the standard priority list (develop, master, main).
func isFeatureBranch(info *engine.SubmoduleInfo) bool {
	if info.CurrentBranch == "" || info.DetachedHead {
		return false
	}
	return !defaultBranches[info.CurrentBranch]
}

// printStatusTable renders the scan result as a colorized lipgloss table.
func printStatusTable(w io.Writer, result *engine.ScanResult) error {
	t := table.New().
		Headers("Path", "Branch", "Behind", "Feature", "Status").
		Border(lipgloss.NormalBorder()).
		BorderHeader(true).
		BorderColumn(true).
		Width(100)

	// Determine the row offset for root vs submodules.
	hasRoot := result.Root != nil
	rootRowIdx := 0

	// Add root row first.
	if hasRoot {
		status := result.Root.PrimaryStatus()
		behind := strconv.Itoa(result.Root.CommitsBehind)
		t.Row(
			"(root)",
			result.Root.CurrentBranch,
			behind,
			"",
			string(status),
		)
	}

	// Add submodule rows (already sorted by path from engine).
	for _, sm := range result.Submodules {
		status := sm.PrimaryStatus()
		behind := strconv.Itoa(sm.CommitsBehind)
		feature := "No"
		if isFeatureBranch(sm) {
			feature = "Yes"
		}
		t.Row(
			sm.Path,
			sm.CurrentBranch,
			behind,
			feature,
			string(status),
		)
	}

	// Apply styles per cell.
	t.StyleFunc(func(row, col int) lipgloss.Style {
		// Header row.
		if row == table.HeaderRow {
			return tui.HeaderStyle
		}

		// Root row: bold for all columns.
		if hasRoot && row == rootRowIdx {
			// Status column gets status-specific color + bold.
			if col == 4 {
				status := result.Root.PrimaryStatus()
				return tui.StyleForStatus(status).Bold(true)
			}
			return tui.RootPathStyle
		}

		// Status column: color per status.
		if col == 4 {
			// Calculate which submodule this row maps to.
			smIdx := row
			if hasRoot {
				smIdx = row - 1
			}
			if smIdx >= 0 && smIdx < len(result.Submodules) {
				status := result.Submodules[smIdx].PrimaryStatus()
				return tui.StyleForStatus(status)
			}
		}

		return lipgloss.NewStyle()
	})

	fmt.Fprintln(w, t.Render())
	return nil
}

// --------------------------------------------------------------------------
// JSON output
// --------------------------------------------------------------------------

// statusJSON is the top-level JSON output structure.
type statusJSON struct {
	Root       *submoduleJSON  `json:"root"`
	Submodules []submoduleJSON `json:"submodules"`
	ScannedAt  string          `json:"scanned_at"`
}

// submoduleJSON is the per-submodule JSON output structure.
type submoduleJSON struct {
	Path          string   `json:"path"`
	CurrentBranch string   `json:"current_branch"`
	TargetBranch  string   `json:"target_branch"`
	CommitsBehind int      `json:"commits_behind"`
	CommitsAhead  int      `json:"commits_ahead"`
	IsFeature     bool     `json:"is_feature"`
	Statuses      []string `json:"statuses"`
	HasChanges    bool     `json:"has_changes"`
	Changelog     []string `json:"changelog,omitempty"`
}

// toSubmoduleJSON converts a SubmoduleInfo to the JSON output type.
func toSubmoduleJSON(info *engine.SubmoduleInfo) *submoduleJSON {
	statuses := make([]string, len(info.Statuses))
	for i, s := range info.Statuses {
		statuses[i] = string(s)
	}
	return &submoduleJSON{
		Path:          info.Path,
		CurrentBranch: info.CurrentBranch,
		TargetBranch:  info.TargetBranch,
		CommitsBehind: info.CommitsBehind,
		CommitsAhead:  info.CommitsAhead,
		IsFeature:     isFeatureBranch(info),
		Statuses:      statuses,
		HasChanges:    info.HasChanges,
		Changelog:     info.Changelog,
	}
}

// printStatusJSON outputs the scan result as indented JSON.
func printStatusJSON(w io.Writer, result *engine.ScanResult) error {
	out := statusJSON{
		ScannedAt:  time.Now().Format(time.RFC3339),
		Submodules: make([]submoduleJSON, 0, len(result.Submodules)),
	}

	if result.Root != nil {
		out.Root = toSubmoduleJSON(result.Root)
	}
	for _, sm := range result.Submodules {
		out.Submodules = append(out.Submodules, *toSubmoduleJSON(sm))
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
