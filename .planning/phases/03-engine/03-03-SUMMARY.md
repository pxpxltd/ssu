---
phase: 03-engine
plan: 03
status: complete
completed: 2026-02-09
duration: 2min
subsystem: engine
tags: [push, parallel, errgroup, detached-head, tracking-branch, continue-on-error]
dependency-graph:
  requires: [03-01]
  provides: [engine-push-workflow]
  affects: [05-tui]
tech-stack:
  added: []
  patterns: [parallel-push-orchestration, pushOne-helper, root-filtering]
key-files:
  created:
    - internal/engine/push.go
    - internal/engine/push_test.go
  modified: []
decisions:
  - id: "03-03-01"
    decision: "Push delegates tracking branch detection to GitService.Push"
    rationale: "ExecGit.Push already handles -u flag logic; no need to duplicate in engine"
  - id: "03-03-02"
    decision: "Detached HEAD returns PushAction with no error (skip, not failure)"
    rationale: "Detached HEAD is an expected state, not an error condition"
metrics:
  tasks: 2
  tests-added: 12
  tests-total: 40
---

# Phase 3 Plan 03: Push Workflow Summary

Engine.Push method with parallel bounded-concurrency push, detached HEAD skip, root filtering, and auto-tracking-branch setup via GitService delegation.

## What Was Built

### internal/engine/push.go (110 lines)

**Engine.Push method:**
- Takes `[]*SubmoduleInfo` targets and `PushOpts` (engine-level, not git-level)
- Filters out root submodules (IsRoot=true) before processing
- Defaults concurrency to `runtime.NumCPU()` when opts.Concurrency <= 0
- Uses zero-value `errgroup.Group` with `SetLimit()` for bounded parallel push
- Continue-on-error: every goroutine returns nil to errgroup, errors captured in PushAction
- Fires ProgressEvents (EventStarted/EventCompleted/EventFailed) per submodule
- Mutex-protected slice accumulation of PushActions

**pushOne helper:**
- Checks `info.DetachedHead` first -- returns "skipped (detached HEAD)" PushAction immediately
- Calls `e.git.Push(ctx, dir, git.PushOpts{})` -- delegates tracking branch logic to GitService
- Maps `git.PushResult.NewTracking` to descriptive action string: "set up tracking + pushed" vs "pushed"
- On failure: captures error in PushAction with Action="push failed"

### internal/engine/push_test.go (535 lines)

12 test cases using MockGitService:

| # | Test | Validates |
|---|------|-----------|
| 1 | SimplePush | Basic push returns Action="pushed", correct Branch, no error |
| 2 | NewTracking | NewTracking=true maps to "set up tracking + pushed" |
| 3 | DetachedHeadSkipped | DetachedHead=true skipped without error, Push mock NOT called |
| 4 | PushFailure | Push error captured in PushAction.Error, Action="push failed" |
| 5 | RootSkipped | IsRoot=true filtered out, 0 actions, Push mock NOT called |
| 6 | MultipleSubmodulesParallel | 4 targets all pushed, atomic counter verifies 4 calls |
| 7 | ContinueOnError | 1 fail + 2 succeed, all 3 have PushActions |
| 8 | ProgressCallback | EventStarted + EventCompleted fire for each target |
| 9 | ProgressCallbackFiresFailedEvent | EventFailed fires with non-nil Error |
| 10 | EmptyTargets | nil targets returns 0 actions |
| 11 | DefaultConcurrency | Concurrency=0 uses runtime.NumCPU, no panic |
| 12 | MixedScenarios | Root + detached + tracking + normal in one call |

## Decisions Made

1. **Push delegates tracking to GitService.Push** -- The ExecGit.Push implementation already detects missing tracking branch and uses `-u` flag. The engine passes empty `git.PushOpts{}` and lets the git layer handle it, avoiding logic duplication.

2. **Detached HEAD is a skip, not an error** -- PushAction for detached HEAD has Action="skipped (detached HEAD)" with nil Error. This matches the bash script behavior where detached HEAD submodules are warned about but don't count as errors.

## Deviations from Plan

None -- plan executed exactly as written.

## Verification Results

- `go test ./internal/engine/... -v -count=1` -- all 40 tests pass (13 scan + 12 push + 15 update)
- `go vet ./internal/engine/...` -- no issues
- `go build ./...` -- entire project compiles

## Commits

| Hash | Type | Description |
|------|------|-------------|
| a3488eb | feat | Push method with parallel orchestration and detached HEAD handling |
| cb8f30a | test | Push tests covering all scenarios with MockGitService |
