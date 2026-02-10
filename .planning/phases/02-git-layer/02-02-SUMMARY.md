---
phase: 02-git-layer
plan: 02
subsystem: git
tags: [branch-detection, algorithm, testing]
dependency-graph:
  requires: [02-01]
  provides: [DetectBestBranch, BranchDetectOpts]
  affects: [02-03, 03-xx]
tech-stack:
  added: []
  patterns: [standalone-function-on-interface, table-driven-tests]
key-files:
  created:
    - internal/git/branch.go
    - internal/git/branch_test.go
  modified: []
decisions:
  - "DetectBestBranch is a standalone function, not a method on any struct"
  - "Remote errors degrade gracefully (non-fatal) at every level"
  - "Priority branch matching checks ALL remotes, not just DefaultRemote"
metrics:
  duration: 2min
  completed: 2026-02-09
---

# Phase 02 Plan 02: Smart Branch Detection Summary

Smart branch detection algorithm ported from bash as a standalone function on the GitService interface, with 14 table-driven test cases covering every priority level and error path.

## What Was Done

### Task 1: DetectBestBranch function (2a18b53)
Created `internal/git/branch.go` implementing the exact 5-level priority chain from the bash `detect_best_branch()`:

1. **Override** -- `opts.Override` (CLI `--branch` flag) returns immediately
2. **Feature branch** -- current branch preserved if not in priority list AND has remote
3. **Priority chain** -- first match in `opts.PriorityBranches` against any remote branch
4. **Remote HEAD** -- parse `refs/remotes/<remote>/HEAD` symbolic ref
5. **Final fallback** -- first available remote branch, or `"master"`

Key design: function takes `GitService` interface, not a concrete type, so it can be fully tested with `MockGitService`.

Helper: unexported `isInList()` for simple linear search in priority list.

### Task 2: Table-driven tests (ed4a025)
Created `internal/git/branch_test.go` with 14 test cases in `TestDetectBestBranch`:

| # | Test Case | Level Tested |
|---|-----------|-------------|
| 1 | override takes priority | Level 1 |
| 2 | feature branch preserved when remote exists | Level 2 |
| 3 | feature branch without remote falls to priority chain | Level 2 -> 3 |
| 4 | develop wins over master | Level 3 |
| 5 | master when no develop | Level 3 |
| 6 | main as last priority | Level 3 |
| 7 | detached HEAD skips feature check | Level 2 skip -> 3 |
| 8 | remote HEAD fallback | Level 4 |
| 9 | absolute fallback to master | Level 5 |
| 10 | first remote branch as fallback | Level 5 |
| 11 | custom priority branches | Level 3 (config) |
| 12 | feature branch check uses default remote | Level 2 (config) |
| 13 | HasRemoteBranch error is non-fatal | Error handling |
| 14 | RemoteBranches error is non-fatal | Error handling |

## Decisions Made

1. **Standalone function, not method**: `DetectBestBranch(ctx, svc, dir, opts)` operates on the interface. This means any `GitService` implementation works -- no coupling to `ExecGit`.

2. **Non-fatal remote errors**: Both `HasRemoteBranch` and `RemoteBranches` errors cause graceful fallthrough to the next priority level, matching the bash behavior of `|| true` patterns.

3. **Match against any remote**: Priority chain checks `rb.Branch == prio` regardless of `rb.Remote`. If `upstream/develop` exists, `develop` matches. This matches the bash behavior which uses `git branch -r` (lists all remotes).

## Deviations from Plan

None -- plan executed exactly as written.

## Verification

- `go test ./internal/git/ -v -run TestDetectBestBranch` -- 14/14 pass
- `go vet ./internal/git/` -- clean
- `go test ./... -count=1` -- all packages pass, no regressions

## Commits

| Hash | Message |
|------|---------|
| 2a18b53 | feat(02-02): implement DetectBestBranch with 5-level priority chain |
| ed4a025 | test(02-02): add 14 table-driven tests for DetectBestBranch |

## Next Phase Readiness

Plan 02-03 (ExecGit) can proceed independently. DetectBestBranch is ready for integration once ExecGit implements the GitService interface methods it calls (CurrentBranch, HasRemoteBranch, RemoteBranches, SymbolicRef).
