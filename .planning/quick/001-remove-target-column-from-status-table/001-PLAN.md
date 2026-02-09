---
type: quick
plan: 001
files_modified:
  - internal/cli/status.go
autonomous: true

must_haves:
  truths:
    - "ssu status table no longer shows a Target column"
    - "ssu status shows a progress bar during scanning (like ssu update does)"
    - "Status column colors still work correctly after column index shift"
    - "JSON output is unaffected (target_branch still present in JSON)"
  artifacts:
    - path: "internal/cli/status.go"
      provides: "Status command with narrower table and progress bar"
  key_links:
    - from: "internal/cli/status.go"
      to: "internal/cli/tui/progress.go"
      via: "tui.NewProgressModel + tea.NewProgram"
      pattern: "tui\\.NewProgressModel|runScanWithProgress"
---

<objective>
Remove the Target column from `ssu status` table output and add a scanning progress bar.

Purpose: The status table is too wide and the Target column adds little value since it duplicates info users rarely need. Additionally, `ssu update` shows a progress bar during scanning but `ssu status` does not, making it feel unresponsive on large repos.

Output: A slimmer 5-column status table (Path, Branch, Behind, Feature, Status) and a bubbletea progress bar shown during the scan phase when running in a TTY.
</objective>

<execution_context>
@/home/jin/.claude/get-shit-done/workflows/execute-plan.md
@/home/jin/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/cli/status.go
@internal/cli/update.go (reference for runScanWithProgress pattern)
@internal/cli/tui/progress.go (ProgressModel, FetchProgressMsg, FetchCompleteMsg)
@internal/cli/output/printer.go
@internal/cli/tui/styles.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Remove Target column and add progress bar to ssu status</name>
  <files>internal/cli/status.go</files>
  <action>
Two changes in `internal/cli/status.go`:

**1. Remove the Target column from `printStatusTable`:**

- Line 107: Change headers from `"Path", "Branch", "Target", "Behind", "Feature", "Status"` to `"Path", "Branch", "Behind", "Feature", "Status"` (5 columns).
- Lines 121-128 (root row): Remove the `result.Root.TargetBranch` argument from `t.Row(...)`. The root row becomes: `"(root)", result.Root.CurrentBranch, behind, "", string(status)`.
- Lines 139-146 (submodule rows): Remove the `sm.TargetBranch` argument from `t.Row(...)`. Each row becomes: `sm.Path, sm.CurrentBranch, behind, feature, string(status)`.
- Line 111: Reduce `.Width(120)` to `.Width(100)` since the table is narrower.
- Line 159: Update the status column index from `col == 5` to `col == 4` (it shifted left by one).
- Line 167: Update the status column index from `col == 5` to `col == 4` for the submodule status styling.

Do NOT touch `printStatusJSON` or `toSubmoduleJSON` -- JSON output keeps `target_branch` as-is.

**2. Add scanning progress bar to `runStatus`:**

Add a TTY check. When running in a TTY, use the same `runScanWithProgress` function that `update.go` already defines. When not in a TTY (piped, --json), scan directly without progress.

In `runStatus`, after building `scanOpts` and before `eng.Scan(...)`:
- Add import for `"github.com/pxpxltd/ssu/internal/cli/output"` (for `output.IsTTY()`).
- Check: if NOT `jsonFlag` AND `output.IsTTY()`, use `runScanWithProgress(cmd.Context(), eng, scanOpts)` instead of `eng.Scan(cmd.Context(), scanOpts)`.
- The `runScanWithProgress` function is already defined in `update.go` in the same `cli` package, so it is directly callable -- no need to duplicate it.

The resulting `runStatus` flow becomes:
```go
jsonFlag, _ := cmd.Flags().GetBool("json")

var result *engine.ScanResult
var err error

if !jsonFlag && output.IsTTY() {
    result, err = runScanWithProgress(cmd.Context(), eng, scanOpts)
} else {
    result, err = eng.Scan(cmd.Context(), scanOpts)
}
if err != nil {
    return fmt.Errorf("scanning submodules: %w", err)
}

if jsonFlag {
    return printStatusJSON(cmd.OutOrStdout(), result)
}
return printStatusTable(cmd.OutOrStdout(), result)
```

Move the `jsonFlag` parsing up before the scan call (currently it is parsed at line 80, after the scan). The `output` import is needed; `context` import may also be needed for `context.Canceled` check but `runScanWithProgress` handles that internally.

Remove unused imports if any result from these changes (the `"os"` import is used for `os.Getwd` so keep it).
  </action>
  <verify>
Run `go build ./...` to confirm compilation succeeds. Run `go vet ./...` for correctness. If the project has tests: `go test ./internal/cli/... -count=1`.
  </verify>
  <done>
- `ssu status` table shows 5 columns: Path, Branch, Behind, Feature, Status (no Target)
- Status column styling still applies correctly (colors per status)
- Root row still renders bold
- `ssu status --json` still includes target_branch in output
- Progress bar appears when running `ssu status` in a TTY
- No progress bar when piped or using `--json`
  </done>
</task>

</tasks>

<verification>
1. `go build ./...` -- compiles without errors
2. `go vet ./...` -- no issues
3. `go test ./internal/cli/... -count=1` -- tests pass (if any exist)
4. Manual: `ssu status` in a git repo with submodules shows 5-column table with progress bar
5. Manual: `ssu status --json` still includes target_branch field, no progress bar
6. Manual: `ssu status | cat` shows no progress bar (non-TTY)
</verification>

<success_criteria>
- Status table is 5 columns wide (Target column removed)
- Progress bar displays during scan phase in TTY mode
- JSON output unchanged
- All existing tests pass
- Code compiles cleanly
</success_criteria>

<output>
After completion, create `.planning/quick/001-remove-target-column-from-status-table/001-SUMMARY.md`
</output>
