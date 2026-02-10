---
phase: 05-commands-tui
plan: 04
subsystem: cli-commands
tags: [cobra, bubbletea, push, exec, init, tui-selector, interactive-menu]
dependency-graph:
  requires:
    - "02-01: GitService interface (Push, SubmodulePaths)"
    - "03-03: engine.Push() with parallel push and detached HEAD skip"
    - "05-02: SelectorModel, ProgressModel, FilterAhead, SubmoduleItems"
  provides:
    - "NewPushCmd: push command with TUI selector filtered to ahead submodules"
    - "NewExecCmd: exec command running arbitrary commands across submodules"
    - "NewInitCmd: init wizard for .ssu.yaml creation"
    - "Root command updated with all subcommands and 7-item interactive menu"
  affects:
    - "05-05+: Any future command additions follow same registration pattern"
    - "06-distribution: All commands complete, ready for packaging"
tech-stack:
  added: []
  patterns:
    - "3-phase TUI flow reused: scan -> selector -> operation (push mirrors update)"
    - "Auto mode / non-TTY branch: same guard pattern across push, exec, init"
    - "exitError in exitcodes.go shared by all commands needing non-zero exit"
    - "Sequential exec with continue-on-error and summary count"
key-files:
  created:
    - internal/cli/push.go
    - internal/cli/exec.go
    - internal/cli/init.go
  modified:
    - internal/cli/root.go
key-decisions:
  - "exitError type lives in exitcodes.go, shared across push.go and update.go"
  - "exec runs commands sequentially (not parallel) for readable output ordering"
  - "init uses bufio.Scanner prompts (not bubbletea) for simplicity"
  - "Interactive menu now has 7 items (exec added between push and rollback)"
duration: 6min
completed: 2026-02-09
---

# Phase 5 Plan 4: Push, Exec, Init Commands Summary

**Push command with TUI selector for ahead submodules, exec command for arbitrary commands across submodules, init wizard for .ssu.yaml creation, and root command updated with all subcommands.**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-09T13:41:55Z
- **Completed:** 2026-02-09T13:47:46Z
- **Tasks:** 3
- **Files modified:** 4

## Accomplishments

1. **Push command (push.go):** Full 3-phase interactive flow (scan -> TUI selector filtered to ahead submodules -> push with result streaming). Auto mode pushes all ahead without prompts. Detached HEAD submodules skipped with warning. Ctrl+C cancellation with context propagation. Summary banner with pushed/failed counts.

2. **Exec command (exec.go):** Runs arbitrary commands (`ssu exec git status`, `ssu exec npm install`) across selected submodules. Interactive mode shows TUI selector with all non-skipped submodules. Auto mode runs in all non-skipped submodules. Sequential execution with bold `==> path` separator headers. Continues on error, tracks failures, summary at end.

3. **Init wizard (init.go):** Creates .ssu.yaml with interactive prompts for parallel jobs, branch priority, and skip list. Detects submodule count. Validates no existing config. TTY-only (rejects non-interactive).

4. **Root command updates (root.go):** Registers NewExecCmd() and NewInitCmd(). Interactive menu expanded to 7 items with exec between push and rollback.

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | Push command with TUI selector for ahead submodules | ad8b475 | push.go |
| 2 | Exec command with TUI selector and --auto mode | 61314f3 | exec.go |
| 3 | Init wizard and root command updates | 13f8353 | init.go, root.go |

## Files Changed

### Created
- `internal/cli/push.go` - Push command: scan, TUI selector (ahead filter), push with streaming results
- `internal/cli/exec.go` - Exec command: scan, TUI selector, sequential command execution
- `internal/cli/init.go` - Init wizard: interactive prompts, .ssu.yaml generation

### Modified
- `internal/cli/root.go` - Added NewExecCmd(), NewInitCmd() registration; exec in interactive menu

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| exitError shared in exitcodes.go | Both push.go and update.go need it; central location avoids duplication |
| exec runs sequentially, not in parallel | Output from parallel commands would be interleaved and unreadable |
| init uses bufio.Scanner, not bubbletea | Simple sequential prompts don't need Elm Architecture complexity |
| Interactive menu has 7 items | exec added as option 4; rollback/backup/help shifted to 5/6/7 |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] exitError type conflict with parallel plan 05-03**
- **Found during:** Task 1 (push.go implementation)
- **Issue:** Plan 05-03 (running in parallel) moved exitError to exitcodes.go; push.go initially declared its own copy causing compilation error
- **Fix:** Removed duplicate exitError from push.go, used the one from exitcodes.go
- **Files modified:** internal/cli/push.go
- **Verification:** `go build ./...` succeeds
- **Committed in:** ad8b475 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Coordination with parallel plan 05-03; no scope creep.

## Issues Encountered

None beyond the parallel plan coordination noted above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

**All user-facing commands are now implemented:**
- `ssu status` (05-01)
- `ssu update` (05-03)
- `ssu push` (05-04)
- `ssu exec` (05-04)
- `ssu init` (05-04)
- `ssu rollback`, `ssu backup`, `ssu config` (Phase 4)
- `ssu version`, `ssu completion` (Phase 1)

Ready for Phase 6 (Distribution) or Phase 5.1 (Claude Code Integration).

---
*Phase: 05-commands-tui*
*Completed: 2026-02-09*
