package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"

	"github.com/pxpxltd/ssu/internal/cli/output"
	"github.com/pxpxltd/ssu/internal/cli/tui"
	"github.com/pxpxltd/ssu/internal/config"
	"github.com/pxpxltd/ssu/internal/engine"
	"github.com/pxpxltd/ssu/internal/git"
	"github.com/pxpxltd/ssu/internal/github"
)

// NewStatusCmd creates the status subcommand.
func NewStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show submodule status",
		Long: `Display the status of all submodules including branch, commits behind, and modification state.

With --pr, also list the open GitHub pull requests of the root repository and
every submodule, with links. This uses the GitHub CLI (gh), which must be
installed and logged in ('gh auth login').`,
		Example: `  ssu status
  ssu status --json
  ssu status --pr
  ssu status --pr --json`,
		RunE: runStatus,
	}

	cmd.Flags().Bool("json", false, "Output status as JSON")
	cmd.Flags().Bool("pr", false, "Also list open GitHub pull requests per module (requires gh)")

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
	prFlag, _ := cmd.Flags().GetBool("pr")

	// Check for gh before the scan so a missing CLI fails fast.
	var gh *github.ExecGH
	if prFlag {
		gh = github.NewExecGH()
		if err := gh.Available(); err != nil {
			return err
		}
	}

	// Create engine and scan.
	eng := engine.New(git.NewExecGit())

	var result *engine.ScanResult
	if !jsonFlag && output.IsTTY() {
		result, err = runScanWithSpinner(cmd.Context(), eng, scanOpts)
	} else {
		result, err = eng.Scan(cmd.Context(), scanOpts)
	}
	if err != nil {
		return fmt.Errorf("scanning submodules: %w", err)
	}

	if prFlag {
		prOpts := engine.PROpts{
			RootDir:     cwd,
			Remote:      scanOpts.BranchOpts.DefaultRemote,
			Concurrency: scanOpts.Concurrency,
		}
		if !jsonFlag && output.IsTTY() {
			if err := runPRLookupWithSpinner(cmd.Context(), eng, gh, result, prOpts); err != nil {
				return err
			}
		} else {
			eng.AttachPullRequests(cmd.Context(), gh, result, prOpts)
		}
	}

	// Mode branching: --json or table.
	if jsonFlag {
		return printStatusJSON(cmd.OutOrStdout(), result)
	}
	if err := printStatusTable(cmd.OutOrStdout(), result, prFlag); err != nil {
		return err
	}
	if prFlag {
		printPRList(cmd.OutOrStdout(), result)
	}
	return nil
}

// runPRLookupWithSpinner attaches pull requests to result while showing a
// spinner, mirroring runScanWithSpinner.
func runPRLookupWithSpinner(ctx context.Context, eng *engine.Engine, lister engine.PRLister, result *engine.ScanResult, opts engine.PROpts) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	p := tea.NewProgram(tui.NewSpinnerModelWithLabel("Checking PRs"))

	opts.OnProgress = func(evt engine.ProgressEvent) {
		p.Send(tui.FetchProgressMsg{
			Path:  evt.Path,
			Done:  evt.Done,
			Total: evt.Total,
			Err:   evt.Error,
		})
	}

	go func() {
		eng.AttachPullRequests(ctx, lister, result, opts)
		p.Send(tui.FetchCompleteMsg{})
	}()

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("spinner: %w", err)
	}
	if sm, ok := finalModel.(tui.SpinnerModel); ok && sm.Err() != nil {
		return sm.Err()
	}
	return nil
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
// When showPRs is set, a "PRs" column with the open pull request count is
// added before Status.
func printStatusTable(w io.Writer, result *engine.ScanResult, showPRs bool) error {
	headers := []string{"Path", "Branch", "Behind", "Feature"}
	prCol := -1
	if showPRs {
		prCol = len(headers)
		headers = append(headers, "PRs")
	}
	statusCol := len(headers)
	headers = append(headers, "Status")

	t := table.New().
		Headers(headers...).
		Border(lipgloss.NormalBorder()).
		BorderHeader(true).
		BorderColumn(true).
		Width(100)

	// Rows in display order: root first, then submodules (already sorted by
	// path from engine).
	var rows []*engine.SubmoduleInfo
	if result.Root != nil {
		rows = append(rows, result.Root)
	}
	rows = append(rows, result.Submodules...)

	for _, info := range rows {
		path := info.Path
		feature := "No"
		if isFeatureBranch(info) {
			feature = "Yes"
		}
		if info.IsRoot {
			path = "(root)"
			feature = ""
		}
		cells := []string{path, info.CurrentBranch, strconv.Itoa(info.CommitsBehind), feature}
		if showPRs {
			cells = append(cells, prCell(info))
		}
		cells = append(cells, string(info.PrimaryStatus()))
		t.Row(cells...)
	}

	// Apply styles per cell.
	t.StyleFunc(func(row, col int) lipgloss.Style {
		// Header row.
		if row == table.HeaderRow {
			return tui.HeaderStyle
		}
		if row < 0 || row >= len(rows) {
			return lipgloss.NewStyle()
		}
		info := rows[row]

		style := lipgloss.NewStyle()
		switch col {
		case statusCol:
			style = tui.StyleForStatus(info.PrimaryStatus())
		case prCol:
			style = prCellStyle(info)
		}

		// Root row: bold for all columns.
		if info.IsRoot {
			style = style.Bold(true)
		}
		return style
	})

	fmt.Fprintln(w, t.Render())
	return nil
}

// prCell returns the PRs column value: the open PR count, "-" for none,
// "?" if the lookup failed, and empty if the module was not looked up.
func prCell(info *engine.SubmoduleInfo) string {
	switch {
	case !info.PRChecked:
		return ""
	case info.PRError != nil:
		return "?"
	case len(info.PullRequests) == 0:
		return "-"
	default:
		return strconv.Itoa(len(info.PullRequests))
	}
}

// prCellStyle colors the PRs column: red for a failed lookup, gray for none.
func prCellStyle(info *engine.SubmoduleInfo) lipgloss.Style {
	switch {
	case !info.PRChecked:
		return lipgloss.NewStyle()
	case info.PRError != nil:
		return tui.StatusConflictStyle
	case len(info.PullRequests) == 0:
		return tui.MutedStyle
	default:
		return tui.StatusAheadStyle
	}
}

// --------------------------------------------------------------------------
// Pull request list
// --------------------------------------------------------------------------

// prTitleMaxRunes caps PR title length in the list so rows stay on one line.
const prTitleMaxRunes = 60

// printPRList prints the open pull requests of every module that has any,
// root first, each with its URL on its own line. Modules whose lookup failed
// follow in a separate "Skipped" section with the reason.
func printPRList(w io.Writer, result *engine.ScanResult) {
	var infos []*engine.SubmoduleInfo
	if result.Root != nil {
		infos = append(infos, result.Root)
	}
	infos = append(infos, result.Submodules...)

	var withPRs, failed []*engine.SubmoduleInfo
	for _, info := range infos {
		switch {
		case !info.PRChecked:
		case info.PRError != nil:
			failed = append(failed, info)
		case len(info.PullRequests) > 0:
			withPRs = append(withPRs, info)
		}
	}

	fmt.Fprintln(w)
	output.Bold.Fprintln(w, "Open pull requests")
	fmt.Fprintln(w)

	for _, info := range withPRs {
		output.Bold.Fprintln(w, prModuleName(info))

		// Pad number, title and branch columns to align within the module.
		numW, titleW, branchW := 0, 0, 0
		for _, pr := range info.PullRequests {
			numW = max(numW, lipgloss.Width(prNumber(pr)))
			titleW = max(titleW, lipgloss.Width(prTitle(pr)))
			branchW = max(branchW, lipgloss.Width(prBranches(pr)))
		}

		for _, pr := range info.PullRequests {
			title := truncateRunes(pr.Title, prTitleMaxRunes)
			fmt.Fprintf(w, "  %s  %s", padRight(prNumber(pr), numW), title)
			if pr.Draft {
				output.Muted.Fprint(w, " (draft)")
			}
			fmt.Fprint(w, strings.Repeat(" ", titleW-lipgloss.Width(prTitle(pr))))
			branches := prBranches(pr)
			if pr.Author != "" {
				fmt.Fprintf(w, "  %s", padRight(branches, branchW))
				output.Muted.Fprintf(w, "  @%s", pr.Author)
			} else {
				fmt.Fprintf(w, "  %s", branches)
			}
			fmt.Fprintln(w)

			fmt.Fprintf(w, "  %s  ", strings.Repeat(" ", numW))
			output.Info.Fprintln(w, pr.URL)
		}
	}

	if len(withPRs) == 0 {
		output.Muted.Fprintln(w, "No open pull requests.")
	}

	if len(failed) == 0 {
		return
	}
	fmt.Fprintln(w)
	output.Bold.Fprintln(w, "Skipped")
	fmt.Fprintln(w)
	for _, info := range failed {
		output.Bold.Fprintln(w, prModuleName(info))
		output.Error.Fprintf(w, "  ! %s\n", firstErrorLine(info.PRError))
	}
}

// prModuleName returns the module label used in the PR list.
func prModuleName(info *engine.SubmoduleInfo) string {
	if info.IsRoot {
		return "(root)"
	}
	return info.Path
}

func prNumber(pr github.PullRequest) string {
	return "#" + strconv.Itoa(pr.Number)
}

// prTitle returns the title as displayed, including the draft marker.
func prTitle(pr github.PullRequest) string {
	title := truncateRunes(pr.Title, prTitleMaxRunes)
	if pr.Draft {
		title += " (draft)"
	}
	return title
}

func prBranches(pr github.PullRequest) string {
	return pr.BaseBranch + " <- " + pr.HeadBranch
}

// truncateRunes shortens s to at most n runes, ending with an ellipsis.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + output.Ellipsis
}

func padRight(s string, width int) string {
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// firstErrorLine returns the first line of err's message; git errors carry
// stderr on following lines.
func firstErrorLine(err error) string {
	msg := err.Error()
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		return msg[:i]
	}
	return msg
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

	// Present only with --pr. A pointer so a module with no open PRs emits
	// [] while modules that were not looked up omit the field.
	PullRequests *[]pullRequestJSON `json:"pull_requests,omitempty"`
	PRError      string             `json:"pr_error,omitempty"`
}

// pullRequestJSON is the per-PR JSON output structure.
type pullRequestJSON struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	HeadBranch string `json:"head_branch"`
	BaseBranch string `json:"base_branch"`
	Author     string `json:"author"`
	Draft      bool   `json:"draft"`
}

// toSubmoduleJSON converts a SubmoduleInfo to the JSON output type.
func toSubmoduleJSON(info *engine.SubmoduleInfo) *submoduleJSON {
	statuses := make([]string, len(info.Statuses))
	for i, s := range info.Statuses {
		statuses[i] = string(s)
	}
	out := &submoduleJSON{
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
	if info.PRChecked {
		if info.PRError != nil {
			out.PRError = firstErrorLine(info.PRError)
		} else {
			prs := make([]pullRequestJSON, 0, len(info.PullRequests))
			for _, pr := range info.PullRequests {
				prs = append(prs, pullRequestJSON{
					Number:     pr.Number,
					Title:      pr.Title,
					URL:        pr.URL,
					HeadBranch: pr.HeadBranch,
					BaseBranch: pr.BaseBranch,
					Author:     pr.Author,
					Draft:      pr.Draft,
				})
			}
			out.PullRequests = &prs
		}
	}
	return out
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
