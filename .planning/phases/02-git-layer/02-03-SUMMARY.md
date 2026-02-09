---
phase: 02-git-layer
plan: 03
subsystem: git
tags: [os-exec, context-timeout, integration-test, git-cli]

# Dependency graph
requires:
  - phase: 02-git-layer/02-01
    provides: "GitService interface, result types, error types, option types"
provides:
  - "ExecGit struct implementing all 19 GitService methods via os/exec"
  - "TimeoutConfig with per-operation configurable timeouts"
  - "Integration test suite proving real git operations work"
affects: [03-scanner, 04-engine, 05-tui]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "os/exec.CommandContext with context.WithTimeout for all git operations"
    - "cmd.WaitDelay=5s to force-kill lingering git processes"
    - "GIT_TERMINAL_PROMPT=0 environment variable for headless operation"
    - "GitError wrapping with Op, Stderr, Err for debuggable errors"
    - "Integration tests using t.TempDir() + git init for isolated real repos"

key-files:
  created:
    - internal/git/exec.go
    - internal/git/exec_test.go
  modified: []

key-decisions:
  - "Merge conflict detection checks both stdout and stderr (git writes CONFLICT to stdout)"
  - "CommitsBehind/CommitsAhead return 0 on error to match bash behavior"
  - "Push auto-detects missing tracking branch and adds -u flag"
  - "isGitExitError helper distinguishes normal exit codes from real failures"
  - "Test remote repos configured with receive.denyCurrentBranch=ignore for push tests"

patterns-established:
  - "Integration test pattern: setupTestRepo/setupClonedRepo helpers with t.TempDir()"
  - "testing.Short() guard on all integration tests for -short flag skip"
  - "Test fixtures use raw exec.Command (not ExecGit) to avoid testing the system under test"

# Metrics
duration: 4min
completed: 2026-02-09
---

# Phase 2 Plan 3: ExecGit Implementation Summary

**Production ExecGit with 19 GitService methods via os/exec.CommandContext, configurable timeouts, and 23 integration tests against real git repos**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-09T11:42:13Z
- **Completed:** 2026-02-09T11:46:35Z
- **Tasks:** 2
- **Files created:** 2

## Accomplishments
- ExecGit implements all 19 GitService methods with proper timeout handling via context.WithTimeout
- Every command hardened with GIT_TERMINAL_PROMPT=0 and cmd.WaitDelay=5s
- 23 integration tests proving real git operations: branch queries, fetch, checkout, merge, push, stash, conflict handling, timeout behavior
- Push auto-sets tracking branch when none exists (git push -u)
- TrackingBranch returns the actual configured remote, not hardcoded "origin"

## Task Commits

Each task was committed atomically:

1. **Task 1: ExecGit struct with run helper and all query methods** - `12581f9` (feat)
2. **Task 2: Integration tests with real git repos in t.TempDir()** - `abdddfa` (test)

## Files Created/Modified
- `internal/git/exec.go` - ExecGit struct implementing all GitService methods via os/exec, TimeoutConfig, run helper
- `internal/git/exec_test.go` - 23 integration tests using t.TempDir() + git init to validate real git operations

## Decisions Made
- Merge conflict detection checks both stdout and stderr because git writes "CONFLICT" to stdout, not stderr
- CommitsBehind and CommitsAhead return 0 on error (matching original bash behavior of `|| echo "0"`)
- Push auto-detects missing tracking branch via TrackingBranch() and falls back to -u flag
- Used isGitExitError helper (errors.As with exec.ExitError) to distinguish normal git exit codes from real failures
- Test remote repos configured with `receive.denyCurrentBranch=ignore` to allow push tests against non-bare repos

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Merge conflict detection checks stdout not just stderr**
- **Found during:** Task 2 (integration test for MergeAbort)
- **Issue:** Plan said check stderr for "CONFLICT", but git writes conflict markers to stdout
- **Fix:** Updated Merge() to check both stdout and stderr for "CONFLICT"
- **Files modified:** internal/git/exec.go
- **Verification:** TestExecGitMergeAbort passes with correct Conflict=true
- **Committed in:** abdddfa (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** Essential for correct conflict detection. No scope creep.

## Issues Encountered
- Push to non-bare remote rejected by default git config -- resolved by setting `receive.denyCurrentBranch=ignore` on test remote repos

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- ExecGit is the production git backend, ready for use by scanner (Phase 3) and engine (Phase 4)
- All GitService methods have both mock (02-01) and real (02-03) implementations
- No blockers or concerns

---
*Phase: 02-git-layer*
*Completed: 2026-02-09*
