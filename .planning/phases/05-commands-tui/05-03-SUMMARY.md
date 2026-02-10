---
phase: 05-commands-tui
plan: 03
subsystem: cli-commands
tags: [bubbletea, lipgloss, update-command, tui, dry-run, auto-mode, backup]
dependency-graph:
  requires:
    - "03-engine: Engine.Scan(), Engine.Update() for scan+merge workflows"
    - "04-config-safety: backup.Create() for pre-update backups"
    - "05-01: shared TUI styles, lipgloss/table, charmbracelet deps"
    - "05-02: SelectorModel, ProgressModel, shared tea.Msg types"
  provides:
    - "ssu update command: interactive TUI, auto, dry-run modes"
    - "runScanWithProgress(): reusable TUI scan+progress helper"
    - "filterPending(): reusable submodule filter for pending status"
    - "resolveRef()/revParseShort(): git ref to short SHA helpers"
    - "createBackupIfEnabled(): backup integration for any command"
    - "printUpdateSummary(): reusable update result summary renderer"
    - "exitError type centralized in exitcodes.go"
  affects:
    - "05-04: push command can reuse runScanWithProgress, createBackupIfEnabled patterns"
tech-stack:
  added: []
  patterns:
    - "Three-branch mode dispatch in RunE: dry-run / auto+nonTTY / interactive"
    - "Three-phase TUI composition: progress bar -> selector -> result streaming"
    - "External goroutine -> p.Send() -> tea.Msg for async scan-to-TUI communication"
    - "Signal handler (os/signal) for Ctrl+C partial results during Phase C"
    - "os/exec direct usage for rev-parse (ExecGit.run is unexported)"
key-files:
  created: []
  modified:
    - internal/cli/update.go
    - internal/cli/exitcodes.go
    - internal/cli/push.go
    - internal/cli/root_test.go
key-decisions:
  - "exitError type moved from push.go to exitcodes.go (shared between update+push commands)"
  - "Backup created before every update (auto and interactive), non-fatal on failure"
  - "resolveRef uses os/exec directly for rev-parse --short (ExecGit.run is unexported)"
  - "Non-TTY falls back to auto mode (same code path as --auto flag)"
  - "Progress bar uses dynamic total (starts at 0, updated via FetchProgressMsg.Total)"
patterns-established:
  - "Mode dispatch pattern: if dryRun -> else if auto||!TTY -> else interactive"
  - "Scan-with-progress: goroutine runs eng.Scan(), sends events via p.Send(), TUI blocks on p.Run()"
  - "Partial results on cancellation: Ctrl+C -> context.Cancel -> print completed actions with count"
duration: 6min
completed: 2026-02-09
---

# Phase 5 Plan 3: Update Command Summary

**Full ssu update command with three modes: interactive TUI (progress bar -> selector -> streaming results), --auto for CI/CD, --dry-run with diff table showing Path/Current SHA/Target SHA/Behind columns, plus backup integration and Ctrl+C partial results.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-09T13:41:01Z
- **Completed:** 2026-02-09T13:47:00Z
- **Tasks:** 1/1
- **Files modified:** 4

## Accomplishments

- Replaced update command stub with 522-line implementation covering all three modes
- Interactive TUI mode chains three phases: scan with animated progress bar, multi-select selector for pending submodules, then streams per-submodule update results
- Auto mode (--auto or non-TTY pipe) scans and updates all pending submodules without prompts, prints summary banner
- Dry-run mode (--dry-run) shows lipgloss diff table with exact headers: Path, Current SHA, Target SHA, Behind
- Backup integration: creates timestamped backup before any update operation (both auto and interactive)
- Ctrl+C handling: during scan (bubbletea handles it), during selector (bubbletea handles it), during update phase (signal handler cancels context, prints partial results with "Cancelled. N/M submodules updated before interruption:" message)
- Non-zero exit codes: ExitConflict (2) for merge conflicts, ExitError (1) for other failures
- Conflict hints printed as actionable messages (e.g., "cd path && git merge origin/branch")
- Centralized exitError type in exitcodes.go for shared use between update.go and push.go

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Update command - interactive TUI mode with progress and selector | `4067e71` | update.go, exitcodes.go, push.go, root_test.go |

## Files Changed

### Modified
- `internal/cli/update.go` -- replaced stub with full implementation (522 lines): runUpdate, runUpdateDryRun, runUpdateAuto, runUpdateInteractive, runScanWithProgress, filterPending, resolveRef, revParseShort, printUpdateSummary, printPartialResults, createBackupIfEnabled
- `internal/cli/exitcodes.go` -- added exitError type (previously defined in push.go, now centralized)
- `internal/cli/push.go` -- removed exitError definition (moved to exitcodes.go)
- `internal/cli/root_test.go` -- updated TestRootCmdSubcommandStubs: removed stale "not implemented yet" assertions for status/update/push (now real implementations)

## Decisions Made

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | exitError centralized in exitcodes.go | Both update.go and push.go need it; exitcodes.go is the natural home alongside exit code constants |
| 2 | Backup runs before every update (auto + interactive) | Matches bash-era safety guarantee; non-fatal on failure so updates still proceed |
| 3 | resolveRef uses os/exec directly for rev-parse | ExecGit.run is unexported; adding a public method to GitService would be scope creep for a display-only helper |
| 4 | Non-TTY and auto share same code path | Simplifies implementation; both need non-interactive behavior |
| 5 | Dynamic total in ProgressModel | Scan total isn't known until SubmodulePaths returns; ProgressModel handles this via FetchProgressMsg.Total |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] exitError redeclared across files**
- **Found during:** Task 1
- **Issue:** push.go already defined exitError type. Adding it to update.go caused "redeclared in this block" compile error.
- **Fix:** Moved exitError to exitcodes.go as the canonical location, removed from both push.go and update.go.
- **Files modified:** exitcodes.go, push.go, update.go
- **Commit:** `4067e71`

**2. [Rule 1 - Bug] Stale test assertions for "not implemented yet" stubs**
- **Found during:** Task 1
- **Issue:** TestRootCmdSubcommandStubs in root_test.go checked for "not implemented yet" output from status/update/push commands. These are now real implementations.
- **Fix:** Merged the stub test into TestRootCmdImplementedSubcommands which just verifies commands run without error.
- **Files modified:** root_test.go
- **Commit:** `4067e71`

---

**Total deviations:** 2 auto-fixed (2 bugs)
**Impact on plan:** Both fixes necessary for compilation and test passage. No scope creep.

## Issues & Risks

None. All success criteria met.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 05-03 deliverables are ready for consumption:
- `ssu update` is fully functional across all three modes
- `runScanWithProgress()` pattern can be reused by push command (currently push does inline scan)
- `createBackupIfEnabled()` can be called from any command that modifies submodules
- `filterPending()` is available for any command needing pending-only submodules
- exitError type in exitcodes.go is shared across all command files
- Root test no longer has stale stub assertions -- safe for future command implementations

---
*Phase: 05-commands-tui*
*Completed: 2026-02-09*
