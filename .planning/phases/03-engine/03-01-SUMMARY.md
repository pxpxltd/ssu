---
phase: 03-engine
plan: 01
status: complete
completed: 2026-02-09
duration: 4min
subsystem: engine
tags: [scan, parallel, errgroup, concurrency, progress-callback]
dependency-graph:
  requires: [02-git-layer]
  provides: [engine-types, scan-method, progress-events]
  affects: [03-02, 03-03, 05-tui]
tech-stack:
  added: [golang.org/x/sync v0.7.0]
  patterns: [errgroup-bounded-concurrency, continue-on-error, compound-status, mutex-protected-results]
key-files:
  created:
    - internal/engine/types.go
    - internal/engine/progress.go
    - internal/engine/engine.go
    - internal/engine/scan.go
    - internal/engine/engine_test.go
    - internal/engine/scan_test.go
  modified:
    - internal/git/git.go
    - go.mod
    - go.sum
decisions:
  - "x/sync pinned to v0.7.0 (last Go 1.21 compatible version; v0.19.0 requires Go 1.24)"
  - "Zero-value errgroup.Group (not WithContext) for continue-on-error semantics"
  - "Root scanned in same parallel batch as submodules, separated in results"
  - "Status priority map for PrimaryStatus display ordering"
metrics:
  tests: 13
  test-time: 0.004s
  files-created: 6
  files-modified: 3
  lines-added: 887
---

# Phase 3 Plan 1: Engine Types, Progress, and Scan Summary

Parallel submodule scanner with bounded concurrency via errgroup, compound status detection, and progress callbacks using MockGitService for full test coverage.

## What Was Built

### Engine Package Foundation (`internal/engine/`)

**types.go** - All shared types for the engine package:
- `ScanOpts` / `ScanResult` / `SubmoduleInfo` for scan workflow
- `UpdateOpts` / `UpdateResult` / `UpdateAction` for Plan 02
- `PushOpts` / `PushResult` / `PushAction` for Plan 03
- `HasStatus()` helper for compound status queries
- `PrimaryStatus()` with priority-ordered display logic

**progress.go** - Progress event system:
- `ProgressEvent` with Type/Path/Phase/Error/Total/Done fields
- `ProgressFunc` callback type (goroutine-safe by contract)
- Three event types: `EventStarted`, `EventCompleted`, `EventFailed`

**engine.go** - Core struct:
- `Engine` struct with `git.GitService` field
- `New(svc)` constructor for dependency injection

**scan.go** - Parallel scan implementation:
- `Scan(ctx, opts)` method with bounded concurrency via `errgroup.SetLimit`
- `scanOne()` helper for fetch + status detection per directory
- Skip list filtering (no git calls for skipped submodules)
- Uninitialized submodule detection (StatusMissing, no git calls)
- Compound statuses: a submodule can be modified AND ahead simultaneously
- Root repository included in parallel batch, separated as display-only in results
- Results sorted by path after collection

### git.go Enhancement

- Added `StatusError SubmoduleStatus = "error"` for scan failures (network timeout, permission denied)

## Decisions Made

| Decision | Rationale |
|----------|-----------|
| Pin x/sync to v0.7.0 | v0.19.0 requires Go 1.24; v0.7.0 is last version with Go 1.18 minimum (compatible with our Go 1.21) |
| Zero-value errgroup | `errgroup.WithContext` cancels sibling goroutines on first error; zero-value allows continue-on-error |
| Root in parallel batch | Simpler code than special-casing root; separated in results post-collection |
| Status priority map | Explicit numeric priority avoids fragile switch-case ordering |

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Pinned golang.org/x/sync to v0.7.0 instead of latest**

- **Found during:** Task 2 dependency setup
- **Issue:** `go get golang.org/x/sync` pulled v0.19.0 which requires Go 1.24, bumping our go.mod from Go 1.21
- **Fix:** Explicitly pinned to v0.7.0 (Go 1.18 minimum, has errgroup.SetLimit)
- **Files modified:** go.mod, go.sum

## Test Results

```
13 tests, 0 failures, 0.004s
```

| Test | What it Verifies |
|------|-----------------|
| TestNew | Engine constructor with MockGitService |
| TestScan_HappyPath_AllBehind | 3 submodules behind -> StatusPending with correct count |
| TestScan_MixedStatuses | Pending + current + compound (modified+ahead) |
| TestScan_FetchFailure | Failed fetch -> StatusError, others succeed |
| TestScan_UninitializedSubmodule | Uninitialized -> StatusMissing, no crash |
| TestScan_SkipList | Skipped submodule -> StatusSkipped, no fetch |
| TestScan_RootIncluded | Root in result.Root with IsRoot=true |
| TestScan_EmptyRepo | No submodules -> empty slice, root still scanned |
| TestScan_DetachedHead | Detached HEAD -> DetachedHead=true |
| TestScan_ProgressCallback | Events fired for all items |
| TestScan_SubmodulesSortedByPath | Results ordered alphabetically |
| TestSubmoduleInfo_HasStatus | Compound status membership check |
| TestSubmoduleInfo_PrimaryStatus | Priority ordering (5 sub-cases) |

## Next Phase Readiness

Plans 02 (update workflow) and 03 (push workflow) can now build on:
- `Engine` struct and `Scan` method for discovering submodule state
- `UpdateOpts`/`UpdateResult` and `PushOpts`/`PushResult` types are pre-defined
- `ProgressFunc` callback pattern established for consistent progress reporting
