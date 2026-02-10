---
phase: 02-git-layer
plan: 01
subsystem: git-abstraction
tags: [interface, types, mock, error-handling]
requires:
  - "01-foundation (Go module, project layout)"
provides:
  - "GitService interface (19 methods) -- contract for all git operations"
  - "MockGitService -- zero-dependency test double with sensible defaults"
  - "Result types with Stderr fields -- structured output for every operation"
  - "GitError with Unwrap -- Go-idiomatic error wrapping"
  - "SubmoduleStatus enum -- 7 states for submodule lifecycle"
  - "BranchDetectOpts -- input type for branch detection (02-02)"
affects:
  - "02-02 (branch detection uses BranchDetectOpts, GitService, MockGitService)"
  - "02-03 (ExecGit must implement GitService interface)"
  - "03-engine (Engine calls GitService methods, uses result types)"
  - "05-commands (Commands interact with git through GitService)"
tech-stack:
  added: []
  patterns:
    - "Interface + mock pattern: single interface, function-field mock for per-test override"
    - "Structured results with always-present Stderr for verbose/debug output"
    - "GitError wraps underlying errors for errors.Is/errors.As chain"
key-files:
  created:
    - "internal/git/git.go"
    - "internal/git/mock.go"
    - "internal/git/mock_test.go"
  modified: []
key-decisions:
  - id: "02-01-01"
    decision: "19 methods on GitService -- one per semantic git operation, not one per raw command"
    rationale: "HasLocalChanges internally may run multiple git commands, but callers see one semantic operation"
  - id: "02-01-02"
    decision: "RemoteBranch has no Stderr field (data type, not operation result)"
    rationale: "RemoteBranch is parsed from output, not a direct git call result"
  - id: "02-01-03"
    decision: "IsSubmoduleInitialized has no context parameter (local filesystem check only)"
    rationale: "Checking if a directory exists does not require context timeout or cancellation"
duration: "2min 42s"
completed: "2026-02-09"
---

# Phase 2 Plan 1: GitService Interface, Types, and Mock Summary

**GitService interface with 19 methods, structured result/error types, and MockGitService with function-field overrides and sensible defaults.**

## Performance

| Metric | Value |
|--------|-------|
| Start | 2026-02-09T11:35:31Z |
| End | 2026-02-09T11:38:13Z |
| Duration | 2min 42s |
| Tasks | 2/2 |

## Accomplishments

1. **GitService interface** with 19 methods organized into 4 groups: repository discovery (2), branch/revision queries (9), status queries (2), mutating operations (6). This is the single contract through which Engine and Commands interact with git.

2. **10 structured types**: BranchResult, TrackingInfo, RemoteBranch, FetchResult, FetchOpts, CheckoutResult, MergeResult, PushResult, PushOpts, StashResult. Every operation result carries `Stderr string` for verbose/debug logging.

3. **BranchDetectOpts** ready for Plan 02-02: Override, PriorityBranches, DefaultRemote fields. DefaultBranchPriority package var provides the default chain.

4. **SubmoduleStatus enum** with 7 string constants: pending, current, modified, ahead, conflict, missing, skipped.

5. **GitError** with Op, Stderr, Err fields. Supports `errors.Is`/`errors.As` via `Unwrap()`. Helper functions `IsTimeout()` and `IsConflict()` for common checks.

6. **MockGitService** with one `Fn` function field per interface method. Zero-value mock returns sensible defaults (no panics). Tests can override individual methods by setting the corresponding Fn field.

7. **Compile-time interface check** in test file: `var _ GitService = (*MockGitService)(nil)`.

## Task Commits

| Task | Name | Commit | Key Files |
|------|------|--------|-----------|
| 1 | GitService interface, result types, error types, and option types | aedc53c | internal/git/git.go |
| 2 | MockGitService implementation and interface satisfaction test | 7f93c58 | internal/git/mock.go, internal/git/mock_test.go |

## Files Created

| File | Purpose | Lines |
|------|---------|-------|
| internal/git/git.go | GitService interface, all types, error types, Status enum | 186 |
| internal/git/mock.go | MockGitService with Fn fields and sensible defaults | 164 |
| internal/git/mock_test.go | Compile-time check, default tests, override test | 132 |

## Decisions Made

1. **19 methods on GitService** -- one per semantic git operation rather than one per raw git command. `HasLocalChanges` may internally run multiple git commands but presents as one semantic check.

2. **RemoteBranch has no Stderr field** -- it is a parsed data type (from `git branch -r` output), not a direct operation result.

3. **IsSubmoduleInitialized has no context parameter** -- it is a local filesystem check (`os.Stat` on the submodule directory) that does not need timeout/cancellation.

## Deviations from Plan

None -- plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

Plan 02-02 (smart branch detection) can proceed immediately:
- `BranchDetectOpts` is defined and ready
- `GitService` interface provides all methods `DetectBestBranch` needs
- `MockGitService` is ready for table-driven branch detection tests
- `DefaultBranchPriority` provides the default priority chain

Plan 02-03 (ExecGit production implementation) can proceed immediately:
- `GitService` interface defines the exact contract to implement
- All result types are defined
- `GitError` is ready for wrapping exec failures
