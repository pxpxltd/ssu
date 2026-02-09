---
phase: quick-003
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/cli/tui/tui.go
  - internal/cli/tui/selector.go
  - internal/cli/exec.go
autonomous: true

must_haves:
  truths:
    - "ssu exec shows a progress bar during submodule scanning (TTY mode)"
    - "ssu exec opens TUI selector with all submodules pre-selected"
    - "ssu exec 'git status' executes correctly via shell interpretation"
    - "ssu exec git status (multi-arg) still works correctly"
  artifacts:
    - path: "internal/cli/tui/tui.go"
      provides: "SelectAll option in SelectorOpts"
      contains: "SelectAll"
    - path: "internal/cli/tui/selector.go"
      provides: "Pre-selection logic in NewSelectorModel"
      contains: "SelectAll"
    - path: "internal/cli/exec.go"
      provides: "Progress bar scan + sh -c execution + SelectAll usage"
      contains: "sh"
  key_links:
    - from: "internal/cli/exec.go"
      to: "internal/cli/tui/tui.go"
      via: "SelectorOpts.SelectAll = true"
      pattern: "SelectAll.*true"
    - from: "internal/cli/exec.go"
      to: "runScanWithProgress"
      via: "shared helper in update.go"
      pattern: "runScanWithProgress"
---

<objective>
Fix three bugs/UX issues in the `ssu exec` command:

1. No progress bar during scan - exec calls `eng.Scan()` directly while status/update use `runScanWithProgress()` with the TUI progress bar
2. All submodules deselected by default - the TUI selector starts empty, but for exec the user expects all selected and deselects what they don't want
3. Shell command execution broken - `ssu exec 'git status'` passes "git status" as a single arg to `exec.CommandContext` which looks for a binary named "git status"

Purpose: Make exec command consistent with other commands (progress bar) and fix broken core functionality (shell execution, selection default).
Output: Working exec command with progress bar, pre-selected submodules, and correct shell command execution.
</objective>

<execution_context>
@/home/jin/.claude/get-shit-done/workflows/execute-plan.md
@/home/jin/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@internal/cli/exec.go
@internal/cli/tui/tui.go
@internal/cli/tui/selector.go
@internal/cli/update.go (lines 314-372 for runScanWithProgress pattern)
@internal/cli/tui/progress.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Add SelectAll option to TUI selector</name>
  <files>internal/cli/tui/tui.go, internal/cli/tui/selector.go</files>
  <action>
  1. In `internal/cli/tui/tui.go`, add a `SelectAll bool` field to the `SelectorOpts` struct (after the `Operation` field):
     ```go
     SelectAll  bool   // Pre-select all items (user deselects unwanted)
     ```

  2. In `internal/cli/tui/selector.go`, in `NewSelectorModel()`, after the `allSelected: make(map[int]bool)` line in the model construction (around line 87), add logic to pre-populate selections when `opts.SelectAll` is true:
     ```go
     m := SelectorModel{
         // ... existing fields ...
         allSelected: make(map[int]bool),
         // ... existing fields ...
     }

     // Pre-select all items if requested.
     if opts.SelectAll {
         for i := range filtered {
             m.allSelected[i] = true
         }
     }
     ```
     Place this block AFTER the model struct literal but BEFORE the detail content initialization (`if len(m.items) > 0`).
  </action>
  <verify>
  Run `go build ./...` to confirm compilation. Run `go vet ./...` for correctness. The SelectAll field should be optional (zero value is false) so existing callers are unaffected.
  </verify>
  <done>SelectorOpts has SelectAll field. When SelectAll=true, NewSelectorModel pre-populates allSelected map with all filtered items set to true. Existing callers (update, push) unaffected since they don't set SelectAll.</done>
</task>

<task type="auto">
  <name>Task 2: Fix exec command - progress bar, select-all, shell execution</name>
  <files>internal/cli/exec.go</files>
  <action>
  Apply three changes to `runExec()` in `internal/cli/exec.go`:

  **Fix 1: Progress bar during scan (lines 87-91)**
  Replace the direct `eng.Scan()` call with the same pattern used by status.go. Add `strings` to imports.

  Replace:
  ```go
  result, err := eng.Scan(ctx, scanOpts)
  if err != nil {
      return fmt.Errorf("scanning submodules: %w", err)
  }
  ```

  With:
  ```go
  autoMode, _ := cmd.Flags().GetBool("auto")
  isTTY := output.IsTTY()

  var result *engine.ScanResult
  if isTTY && !autoMode {
      result, err = runScanWithProgress(ctx, eng, scanOpts)
  } else {
      result, err = eng.Scan(ctx, scanOpts)
  }
  if err != nil {
      return fmt.Errorf("scanning submodules: %w", err)
  }
  ```

  Note: Move the `autoMode` and `isTTY` variable declarations from their current location (lines 108-109) to BEFORE the scan call, since they are now needed earlier. Remove the duplicate declarations that were previously on lines 108-109.

  **Fix 2: Select all submodules by default (line 127)**
  In the `SelectorOpts` passed to `NewSelectorModel`, add `SelectAll: true`:
  ```go
  selModel := tui.NewSelectorModel(items, tui.SelectorOpts{
      Title:     fmt.Sprintf("Select submodules for: %s", cmdLabel),
      Subtitle:  fmt.Sprintf("%d submodules available", len(nonSkipped)),
      Operation: "exec",
      SelectAll: true,
  })
  ```

  **Fix 3: Shell command execution (line 171)**
  Replace the direct `exec.CommandContext` call that breaks on quoted args:
  ```go
  // OLD: c := exec.CommandContext(ctx, args[0], args[1:]...)
  // NEW: Use sh -c to handle shell interpretation correctly.
  // This handles both 'git status' (single arg) and git status (multi-arg).
  c := exec.CommandContext(ctx, "sh", "-c", strings.Join(args, " "))
  ```

  Make sure `"strings"` is in the imports (it is not currently imported in exec.go).

  Also update the Examples in `NewExecCmd()` to reflect that quoted commands work:
  ```go
  Example: `  ssu exec git status
    ssu exec --auto npm install
    ssu exec 'git log --oneline -5'
    ssu exec ls -la`,
  ```
  </action>
  <verify>
  1. `go build ./...` compiles cleanly
  2. `go vet ./...` passes
  3. Manual test in a repo with submodules:
     - `ssu exec git status` -- should show progress bar, then selector with all checked, then run git status in each
     - `ssu exec 'git status'` -- single quoted arg should work identically
     - `ssu exec --auto ls` -- should skip progress bar and selector, run in all submodules
  </verify>
  <done>
  - Exec command shows TUI progress bar during scan (TTY mode), falls back to direct scan (non-TTY/auto mode)
  - TUI selector opens with all submodules pre-selected (user deselects unwanted ones)
  - Shell command execution uses `sh -c` so both `ssu exec 'git status'` and `ssu exec git status` work correctly
  </done>
</task>

</tasks>

<verification>
1. `go build ./...` -- project compiles
2. `go vet ./...` -- no issues
3. `go test ./internal/cli/tui/...` -- existing TUI tests pass (SelectAll=false default preserves behavior)
4. In a real repo with submodules:
   - `ssu exec git status` shows progress bar, selector with all items checked, executes git status
   - `ssu exec 'git log --oneline -5'` works (quoted compound command)
   - `ssu exec --auto echo hello` runs without progress bar or selector
</verification>

<success_criteria>
- Progress bar appears during exec scan in TTY mode (matching status/update behavior)
- TUI selector opens with all submodules pre-checked
- `ssu exec 'git status'` and `ssu exec git status` both execute correctly
- No regressions in existing update/push/status commands
</success_criteria>

<output>
After completion, create `.planning/quick/003-exec-command-fixes/003-SUMMARY.md`
</output>
