---
phase: 07-complete-rollback
plan: 01
subsystem: cli
tags: [rollback, git-reset, backup, lipgloss, table]

# Dependency graph
requires:
  - phase: 04-config-safety
    provides: "backup.Rollback function with callback injection pattern"
  - phase: 02-git-layer
    provides: "GitService interface and ExecGit implementation"
  - phase: 05-commands-tui
    provides: "CLI command structure, lipgloss table patterns, resolveBackupDir"
provides:
  - "Fully functional ssu rollback <backup-file> command"
  - "GitService.ResetHard method on interface, ExecGit, and MockGitService"
  - "Results table with per-submodule restore status"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Closure callback wiring: CLI wraps GitService methods as backup.Rollback function params"

key-files:
  created: []
  modified:
    - "internal/git/git.go"
    - "internal/git/exec.go"
    - "internal/git/mock.go"
    - "internal/cli/rollback.go"
    - "internal/backup/backup_test.go"

key-decisions:
  - "ResetHard returns just error (no result type), matching MergeAbort pattern"
  - "Closures join projectRoot with relative paths before calling ExecGit (backup package passes relative paths)"
  - "Interactive confirmation uses bufio.Scanner matching init.go pattern"

patterns-established:
  - "Closure callback wiring: CLI layer adapts GitService to backup package function signatures"

# Metrics
duration: 3min
completed: 2026-02-10
---

# Phase 7 Plan 01: Wire Rollback Command Summary

**Fully wired `ssu rollback <backup-file>` with ResetHard on GitService, closure callbacks to backup.Rollback, lipgloss results table, and safety backup creation**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-10T12:59:52Z
- **Completed:** 2026-02-10T13:02:18Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- Added ResetHard to GitService interface, ExecGit, and MockGitService (4-file pattern)
- Replaced stub in rollback.go with full backup.Rollback() wiring using closure-wrapped GitService methods
- Results table displays path, branch, previous SHA, restored SHA, and status per submodule
- Safety backup created before any restore operation
- Interactive confirmation prompt in TTY mode (skipped with --auto)
- Added error case and bash-era format integration tests

## Task Commits

Each task was committed atomically:

1. **Task 1: Add ResetHard to GitService interface and implementations** - `915e70b` (feat)
2. **Task 2: Wire rollback command with git callbacks and results table** - `f548521` (feat)

## Files Created/Modified
- `internal/git/git.go` - Added ResetHard to GitService interface
- `internal/git/exec.go` - Added ResetHard production implementation (git reset --hard)
- `internal/git/mock.go` - Added ResetHardFn field and method on MockGitService
- `internal/cli/rollback.go` - Complete rewrite: stub replaced with backup.Rollback() wiring, results table, confirmation
- `internal/backup/backup_test.go` - Added TestRollbackWithResetError and TestRollbackBashEra tests

## Decisions Made
- ResetHard returns just `error` (no result struct) -- matches the MergeAbort pattern since reset is pass/fail
- Closures join projectRoot with relative submodule paths before calling ExecGit -- the backup package passes relative paths
- Interactive confirmation uses bufio.Scanner on os.Stdin (matching init.go pattern, not bubbletea)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Rollback is the last integration gap -- SSU v1 codebase is now fully wired
- All commands (status, update, push, exec, init, rollback, backup, config) are functional

---
*Phase: 07-complete-rollback*
*Completed: 2026-02-10*
