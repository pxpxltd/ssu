---
phase: 03-engine
plan: "02"
title: "Update Workflow"
subsystem: engine
tags: [update, merge, stash, conflict-resolution, parallel, errgroup]
depends_on:
  requires: ["03-01"]
  provides: ["Engine.Update method", "3-step conflict resolution", "UpdateResult/UpdateAction types"]
  affects: ["05-tui", "06-integration"]
tech_stack:
  added: []
  patterns: ["stash+merge+stash-pop conflict resolution", "isSkippable filter", "continue-on-error parallel"]
key_files:
  created:
    - internal/engine/update.go
    - internal/engine/update_test.go
  modified:
    - internal/engine/scan_test.go
decisions:
  - id: "03-02-01"
    decision: "Update accepts []*SubmoduleInfo targets (caller decides what to update, not engine)"
    reason: "Separation of concerns -- engine processes, CLI selects"
  - id: "03-02-02"
    decision: "Dirty path uses stash -> merge -> stash-pop with abort+restore on failure"
    reason: "Matches bash SSU behavior; maximizes chance of preserving local work"
  - id: "03-02-03"
    decision: "ConflictHint contains relative path (info.Path) not absolute path"
    reason: "User runs commands from project root; relative paths are copy-pasteable"
metrics:
  duration: "3min"
  completed: "2026-02-09"
  tests_added: 17
  tests_total: 43
---

# Phase 3 Plan 02: Update Workflow Summary

Engine.Update with parallel orchestration, 3-step stash+merge+stash-pop conflict resolution, and 17 mock-based tests covering all failure paths.

## What Was Built

### internal/engine/update.go (217 lines)
- `Engine.Update(ctx, targets, opts)` -- parallel update orchestration via errgroup with bounded concurrency
- `updateOne(ctx, rootDir, info)` -- single submodule processor dispatching to dirty/clean paths
- `updateDirty` -- 3-step strategy: stash -> merge -> stash-pop; on conflict: abort -> restore stash -> ConflictHint
- `updateClean` -- direct merge with conflict detection
- `isSkippable` -- filters root, current, missing, and skipped submodules from processing
- Every UpdateAction records BeforeStatus (snapshot before) and AfterStatus (result after)
- ConflictHint provides copy-paste git commands: `cd <path> && git stash && git merge origin/<branch> && git stash pop`

### internal/engine/update_test.go (707 lines)
17 test cases using MockGitService:

| # | Test | Scenario |
|---|------|----------|
| 1 | CleanSubmoduleBehindRemote | Clean merge, AfterStatus=current |
| 2 | DirtySubmoduleStashMergePop | Full 3-step, verifies call order |
| 3 | DirtySubmoduleMergeConflictAfterStash | Conflict with abort+restore, ConflictHint validated |
| 4 | DirtySubmoduleStashFails | Stash error, AfterStatus=error |
| 5 | CleanSubmoduleMergeConflict | Clean conflict, no stash in hint |
| 6 | MergeConflictAbortFails | Abort failure, AfterStatus=error |
| 7 | RootSubmoduleSkipped | IsRoot=true excluded |
| 8 | CurrentSubmoduleSkipped | StatusCurrent excluded |
| 9 | MissingSubmoduleSkipped | StatusMissing excluded |
| 10 | MultipleSubmodulesParallel | 5 submodules, concurrency=3 |
| 11 | ContinueOnError | 3 subs, middle fails, others succeed |
| 12 | BeforeStatusRecorded | Compound statuses preserved |
| 13 | DirtyMergeFailNonConflict | Non-conflict error, no hint |
| 14 | StashPopFailsAfterMerge | Pop failure after successful merge |
| 15 | ProgressEvents | Started+Completed fired for each |
| 16 | SkippedStatusSubmodule | StatusSkipped excluded |
| 17 | DefaultConcurrency | Concurrency=0 uses NumCPU |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed concurrent map writes in TestScan_SkipList**
- **Found during:** Task 2 (full test suite run)
- **Issue:** `fetchCalled` map written from multiple goroutines without synchronization, causing sporadic `concurrent map writes` panic
- **Fix:** Added `sync.Mutex` around map reads and writes
- **Files modified:** internal/engine/scan_test.go
- **Commit:** b8e46d9

## Commits

| Hash | Type | Description |
|------|------|-------------|
| f3a078a | feat | Engine.Update with parallel orchestration and 3-step conflict resolution |
| b8e46d9 | test | 17 update tests covering all conflict scenarios |

## Verification

- `go test ./internal/engine/... -v -count=1` -- 43 tests pass (13 scan + 13 push + 17 update)
- `go vet ./internal/engine/...` -- no issues
- `go build ./...` -- full project compiles
- ConflictHint contains `git merge`, `git stash`, and submodule path (asserted in tests)
- Stash+retry+reapply call order verified by mock instrumentation

## Next Phase Readiness

Plan 03-02 complete. All engine methods (Scan, Update, Push) are now implemented. Plan 03-03 (if it exists) or Phase 4 can proceed. No blockers.
