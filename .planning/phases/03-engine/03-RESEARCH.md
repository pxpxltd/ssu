# Phase 3: Engine - Research

**Researched:** 2026-02-09
**Domain:** Go concurrency orchestration, parallel git scanning, conflict resolution
**Confidence:** HIGH

## Summary

Phase 3 builds the core engine that orchestrates parallel submodule scanning, status detection, conflict handling, and update/push workflows. The engine sits between the GitService layer (Phase 2, complete) and the command/TUI layer (Phase 5, future). It accepts configuration as parameters (no config loading -- that is Phase 4) and exposes progress events for TUI consumption.

The standard approach uses `golang.org/x/sync/errgroup` with `SetLimit` for bounded concurrency, a zero-value Group (not `WithContext`) to achieve continue-on-error behavior, and typed result structs collected via mutex-protected slices. The engine is a single `Engine` struct with methods for `Scan`, `Update`, and `Push`, each returning aggregate result types that Phase 5 commands consume directly.

The compound status model uses a set of flags (not a single enum) so a submodule can be simultaneously modified AND ahead. The `StatusError` value must be added to the existing `SubmoduleStatus` enum in `git.go`. Progress events flow through a callback function parameter (not a channel) to keep the API simple and testable.

**Primary recommendation:** Use errgroup with SetLimit for all parallel operations. Return nil from goroutines and collect per-submodule results in a mutex-protected slice to achieve continue-on-error semantics. Keep the engine as a pure library with no I/O -- all display is deferred to Phase 5.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/sync/errgroup` | v0.19.0+ | Bounded parallel goroutine management | Standard Go ecosystem package for concurrent task groups with SetLimit |
| `sync` (stdlib) | Go 1.21 | Mutex for result collection | Built-in, no external dependency needed |
| `context` (stdlib) | Go 1.21 | Timeout propagation to individual git ops | Already used by GitService interface |
| `runtime` (stdlib) | Go 1.21 | `runtime.NumCPU()` for default concurrency | Per CONTEXT.md decision |
| `path/filepath` (stdlib) | Go 1.21 | Submodule path resolution | Already in use |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `fmt` (stdlib) | Go 1.21 | Formatting conflict hint commands | Always, for actionable error messages |
| `strings` (stdlib) | Go 1.21 | Parsing conflict file paths from git output | During conflict detection |
| `time` (stdlib) | Go 1.21 | Operation timing for result metadata | Optional, for tracking scan/update duration |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| errgroup | Custom worker pool with channels | More code, more bugs, no benefit -- errgroup.SetLimit does exactly what we need |
| errgroup | `sourcegraph/conc` | External dependency for marginal ergonomic gain; overkill for this use case |
| Mutex-protected slice | Result channel | Channel adds complexity for collecting ordered results; mutex is simpler |
| Callback function for progress | Channel-based event bus | Channel requires goroutine management for consumer; callback is simpler and directly testable |

**Installation:**
```bash
go get golang.org/x/sync
```

**Note:** This is the only new dependency. Everything else uses stdlib.

## Architecture Patterns

### Recommended Project Structure
```
internal/
  engine/
    engine.go        # Engine struct, constructor, Scan method
    scan.go          # scanSubmodule helper, status detection logic
    update.go        # Update method, conflict resolution (stash/retry/reapply)
    push.go          # Push method, ahead detection, detached HEAD handling
    types.go         # All engine types: ScanResult, SubmoduleInfo, UpdateResult, etc.
    progress.go      # Progress event types and callback interface
    engine_test.go   # Unit tests using MockGitService
    scan_test.go     # Scan-specific test tables
    update_test.go   # Update/conflict test tables
    push_test.go     # Push workflow test tables
```

### Pattern 1: Engine Struct with Dependency Injection

**What:** The Engine struct holds a GitService and exposes Scan/Update/Push methods. All configuration (concurrency, skip list, branch opts) is passed per-call, not stored on the struct.

**When to use:** Always -- this is the core pattern for the entire phase.

**Example:**
```go
// Source: Designed for this project, following Phase 2 patterns
type Engine struct {
    git git.GitService
}

func New(svc git.GitService) *Engine {
    return &Engine{git: svc}
}

func (e *Engine) Scan(ctx context.Context, opts ScanOpts) (*ScanResult, error) {
    // 1. Enumerate submodules
    // 2. Filter against skip list
    // 3. Parallel fetch with bounded concurrency
    // 4. Status detection per submodule
    // 5. Root repo status (display-only)
    // 6. Return aggregate result
}
```

### Pattern 2: Zero-Value errgroup for Continue-on-Error

**What:** Use `var g errgroup.Group` (zero value, NOT `errgroup.WithContext`) so that errors do not cancel other goroutines. Each goroutine catches its own error and stores it in the per-submodule result. The goroutine returns nil to errgroup so processing continues.

**When to use:** For all parallel operations (scan, update, push) since the CONTEXT.md mandates continue-on-error.

**Why not WithContext:** `errgroup.WithContext` cancels the derived context on the first error, which would abort remaining submodule operations. We want to collect ALL results, including failures.

**Example:**
```go
var g errgroup.Group
g.SetLimit(opts.Concurrency)

var mu sync.Mutex
results := make([]SubmoduleInfo, 0, len(paths))

for _, path := range paths {
    g.Go(func() error {
        info := e.scanSubmodule(ctx, rootDir, path, opts)
        // info.Error is set if scan failed -- we still collect the result

        mu.Lock()
        results = append(results, info)
        mu.Unlock()

        // Always return nil so other goroutines continue
        return nil
    })
}
_ = g.Wait() // error is always nil since goroutines return nil
```

**IMPORTANT:** Go 1.22 changed loop variable semantics (no more loop variable capture bug). Since this project uses Go 1.21, each goroutine closure MUST capture the loop variable explicitly:
```go
for _, path := range paths {
    path := path // capture for goroutine
    g.Go(func() error {
        info := e.scanSubmodule(ctx, rootDir, path, opts)
        // ...
    })
}
```

### Pattern 3: Compound Status with Flag Set

**What:** Instead of a single SubmoduleStatus enum, the engine result uses a set of statuses. A submodule can have multiple simultaneous states (e.g., modified + ahead).

**When to use:** For the SubmoduleInfo result type returned from scanning.

**Example:**
```go
// SubmoduleInfo holds the scan result for one submodule.
type SubmoduleInfo struct {
    Path          string
    CurrentBranch string
    TargetBranch  string
    CommitsBehind int
    CommitsAhead  int
    IsFeature     bool
    Statuses      []git.SubmoduleStatus // Can hold multiple: [modified, ahead]
    Changelog     []string              // One-line summaries of incoming commits
    Error         error                 // Non-nil if scan itself failed
}

// HasStatus checks if the submodule has a specific status.
func (s *SubmoduleInfo) HasStatus(status git.SubmoduleStatus) bool {
    for _, st := range s.Statuses {
        if st == status {
            return true
        }
    }
    return false
}

// PrimaryStatus returns the most important status for display ordering.
func (s *SubmoduleInfo) PrimaryStatus() git.SubmoduleStatus {
    // Priority: error > conflict > modified > ahead > pending > current > missing > skipped
}
```

### Pattern 4: Progress Callback

**What:** Engine methods accept an optional callback function that receives typed progress events. The callback is called synchronously from the goroutine processing each submodule.

**When to use:** For Scan, Update, and Push methods so Phase 5 TUI can display progress.

**Example:**
```go
// ProgressEvent represents a progress notification from the engine.
type ProgressEvent struct {
    Type  ProgressType
    Path  string   // Submodule path
    Total int      // Total submodules being processed
    Done  int      // Submodules completed so far
    Err   error    // Non-nil if this submodule failed
}

type ProgressType int
const (
    ProgressScanStart ProgressType = iota
    ProgressScanDone
    ProgressFetchStart
    ProgressFetchDone
    ProgressUpdateStart
    ProgressUpdateDone
    ProgressPushStart
    ProgressPushDone
)

// ProgressFunc is called by the engine to report progress.
// If nil, progress reporting is skipped.
type ProgressFunc func(ProgressEvent)

type ScanOpts struct {
    RootDir          string
    Concurrency      int
    SkipList         []string
    BranchOpts       git.BranchDetectOpts
    OnProgress       ProgressFunc
}
```

### Pattern 5: Before/After Result for Rich Summary

**What:** Update and push results capture the before-state and after-state so Phase 5 can display "was pending, merged 3 commits, now current".

**Example:**
```go
type UpdateResult struct {
    Submodules []SubmoduleUpdateResult
    Summary    UpdateSummary
}

type SubmoduleUpdateResult struct {
    Path         string
    BeforeStatus []git.SubmoduleStatus
    AfterStatus  []git.SubmoduleStatus
    Action       string   // "merged 3 commits", "skipped (modified)", "conflict resolved via stash"
    Error        error
    ConflictHint string   // Actionable git commands if conflict
}

type UpdateSummary struct {
    Updated    int
    Skipped    int
    Conflicts  int
    Errors     int
    Duration   time.Duration
}
```

### Anti-Patterns to Avoid

- **Global state for scan results:** All state must be in the result types, not on the Engine struct. Engine must be safe for concurrent use across multiple calls.
- **errgroup.WithContext for continue-on-error:** This cancels on first error. Use zero-value Group instead.
- **Returning error from goroutines in continue-on-error mode:** Return nil and store errors in the per-item result struct.
- **Channel for result collection:** A mutex-protected slice is simpler and preserves the ability to sort results after collection.
- **Storing config on Engine struct:** The CONTEXT.md says engine accepts config as parameters per-call. Phase 4 wires config; engine is agnostic.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Bounded concurrency | Custom semaphore channel + WaitGroup | `errgroup.Group` with `SetLimit` | Battle-tested, handles edge cases (panics, cleanup) correctly |
| Context propagation | Manual timeout tracking | `context.Context` passed through to GitService | Already wired in Phase 2 |
| Branch detection | Custom logic in engine | `git.DetectBestBranch()` from Phase 2 | Already implemented and tested with 14 test cases |
| Conflict detection | Custom stderr parsing | `git.IsConflict(err)` from Phase 2 | Already implemented in git.go |
| Timeout detection | Manual deadline checks | `git.IsTimeout(err)` from Phase 2 | Already implemented in git.go |

**Key insight:** Phase 2 already built all the git operation primitives and detection helpers. The engine should compose them, not reimplement them.

## Common Pitfalls

### Pitfall 1: Loop Variable Capture in Go 1.21
**What goes wrong:** Goroutines all reference the same loop variable, processing only the last submodule N times.
**Why it happens:** Go 1.21 does not have per-iteration loop variable scoping (that was added in Go 1.22).
**How to avoid:** Always shadow the loop variable: `path := path` before `g.Go(func() error { ... })`.
**Warning signs:** Tests show all results for the same submodule path.

### Pitfall 2: errgroup.WithContext Cancels on First Error
**What goes wrong:** First fetch failure cancels the context, aborting all remaining submodule scans.
**Why it happens:** `errgroup.WithContext` is designed for fail-fast semantics.
**How to avoid:** Use zero-value `var g errgroup.Group` with `g.SetLimit(n)`. Pass the original context (not a derived one) to each goroutine.
**Warning signs:** Scan returns partial results when any single submodule has a network issue.

### Pitfall 3: Race Condition on Result Collection
**What goes wrong:** Data race when multiple goroutines append to the results slice simultaneously.
**Why it happens:** Slice append is not thread-safe.
**How to avoid:** Use `sync.Mutex` to protect the append, OR pre-allocate a results array indexed by position.
**Warning signs:** `-race` flag detects data race, or results are randomly missing.

### Pitfall 4: Stash Pop After Failed Merge Abort
**What goes wrong:** Stash pop fails or produces unexpected state when merge abort did not fully clean up.
**Why it happens:** If merge abort itself fails, the working tree is in an inconsistent state.
**How to avoid:** Check MergeAbort return value before attempting StashPop. If abort fails, report both the conflict AND the abort failure in the hint, and do NOT attempt stash pop.
**Warning signs:** StashPop returns error after conflict resolution flow.

### Pitfall 5: Root Repository Path Handling
**What goes wrong:** Root repo (path ".") gets included in update/push operations.
**Why it happens:** Root is included in the scan results for display purposes.
**How to avoid:** Mark root as display-only in the result type (e.g., `IsRoot bool` field). Filter it out explicitly in Update and Push methods.
**Warning signs:** Engine attempts to merge or push in the root repository.

### Pitfall 6: Submodule Not Initialized
**What goes wrong:** Git operations fail on submodules that exist in .gitmodules but haven't been `git submodule init`'d.
**Why it happens:** Submodule path exists in config but the directory is empty or missing.
**How to avoid:** Check `IsSubmoduleInitialized` before attempting any git operations. Set status to `StatusMissing` for uninitialized submodules.
**Warning signs:** Git errors about missing .git directory inside submodule path.

### Pitfall 7: Detached HEAD in Push Workflow
**What goes wrong:** Push fails because git cannot push from a detached HEAD.
**Why it happens:** Some submodules are checked out at a specific commit (detached HEAD), common after `git submodule update --init`.
**How to avoid:** Check `IsDetachedHead` before attempting push. Set result to skipped with a warning message.
**Warning signs:** Push returns error about "detached HEAD".

## Code Examples

### Scan One Submodule (Status Detection Logic)

```go
// Source: Composed from Phase 2 GitService methods
func (e *Engine) scanSubmodule(ctx context.Context, rootDir, subPath string, opts ScanOpts) SubmoduleInfo {
    info := SubmoduleInfo{Path: subPath}
    fullPath := filepath.Join(rootDir, subPath)

    // Check if initialized
    if !e.git.IsSubmoduleInitialized(rootDir, subPath) {
        info.Statuses = []git.SubmoduleStatus{git.StatusMissing}
        return info
    }

    // Get current branch
    br, err := e.git.CurrentBranch(ctx, fullPath)
    if err != nil {
        info.Statuses = []git.SubmoduleStatus{git.StatusError}
        info.Error = fmt.Errorf("current branch: %w", err)
        return info
    }
    info.CurrentBranch = br.Name

    // Detect best branch (target)
    target, err := git.DetectBestBranch(ctx, e.git, fullPath, opts.BranchOpts)
    if err != nil {
        info.Statuses = []git.SubmoduleStatus{git.StatusError}
        info.Error = fmt.Errorf("detect branch: %w", err)
        return info
    }
    info.TargetBranch = target
    info.IsFeature = br.Name != target && !br.Detached

    // Check for local changes
    hasChanges, _ := e.git.HasLocalChanges(ctx, fullPath)

    // Check commits behind
    remoteRef := "origin/" + target
    behind, _ := e.git.CommitsBehind(ctx, fullPath, "HEAD", remoteRef)
    info.CommitsBehind = behind

    // Check commits ahead
    ahead, _ := e.git.CommitsAhead(ctx, fullPath, remoteRef)
    info.CommitsAhead = ahead

    // Build status set
    var statuses []git.SubmoduleStatus
    if hasChanges {
        statuses = append(statuses, git.StatusModified)
    }
    if ahead > 0 {
        statuses = append(statuses, git.StatusAhead)
    }
    if behind > 0 {
        statuses = append(statuses, git.StatusPending)
    }
    if len(statuses) == 0 {
        statuses = []git.SubmoduleStatus{git.StatusCurrent}
    }
    info.Statuses = statuses

    // Changelog preview for pending submodules
    if behind > 0 {
        changelog, _ := e.git.IncomingChangelog(ctx, fullPath, remoteRef, 20)
        info.Changelog = changelog
    }

    return info
}
```

### Conflict Resolution (3-Step Strategy)

```go
// Source: Matching bash version's stash-retry-reapply pattern
func (e *Engine) resolveConflict(ctx context.Context, dir, ref string) ConflictResult {
    result := ConflictResult{}

    // Step 1: Stash local changes
    stashRes, err := e.git.Stash(ctx, dir)
    if err != nil {
        result.Error = fmt.Errorf("stash before retry: %w", err)
        return result
    }
    result.Stashed = stashRes.Created

    // Step 2: Retry merge on clean state
    mergeRes, err := e.git.Merge(ctx, dir, ref)
    if err != nil {
        // Merge still failed -- abort and restore
        _ = e.git.MergeAbort(ctx, dir)
        if stashRes.Created {
            _, popErr := e.git.StashPop(ctx, dir)
            if popErr != nil {
                result.Error = fmt.Errorf("merge retry failed AND stash pop failed: %w", err)
                return result
            }
        }
        result.Error = fmt.Errorf("merge retry after stash: %w", err)
        result.ConflictFiles = parseConflictFiles(mergeRes.Stderr)
        return result
    }

    // Step 3: Re-apply stash
    if stashRes.Created {
        _, popErr := e.git.StashPop(ctx, dir)
        if popErr != nil {
            result.Error = fmt.Errorf("merge succeeded but stash pop failed: %w", popErr)
            result.MergeSucceeded = true
            return result
        }
    }

    result.MergeSucceeded = true
    result.Resolved = true
    return result
}
```

### Actionable Conflict Hint Generation

```go
// Source: Per CONTEXT.md requirement for copy-paste git commands
func buildConflictHint(subPath string, conflictFiles []string) string {
    var b strings.Builder
    b.WriteString(fmt.Sprintf("Conflict in %s", subPath))
    if len(conflictFiles) > 0 {
        b.WriteString(": " + strings.Join(conflictFiles, ", "))
    }
    b.WriteString("\n  To resolve manually:\n")
    b.WriteString(fmt.Sprintf("    cd %s\n", subPath))
    b.WriteString("    git merge --abort\n")
    b.WriteString("    git stash pop\n")
    b.WriteString("    # Fix conflicts, then: git add . && git commit\n")
    return b.String()
}
```

### Parallel Fetch with Progress

```go
// Source: errgroup.SetLimit pattern for bounded concurrency
func (e *Engine) parallelFetch(ctx context.Context, rootDir string, paths []string, concurrency int, onProgress ProgressFunc) {
    var g errgroup.Group
    g.SetLimit(concurrency)

    var done int64

    for _, path := range paths {
        path := path // Go 1.21 capture
        g.Go(func() error {
            fullPath := filepath.Join(rootDir, path)
            if onProgress != nil {
                onProgress(ProgressEvent{Type: ProgressFetchStart, Path: path})
            }

            _, err := e.git.Fetch(ctx, fullPath, git.FetchOpts{Prune: true})

            current := atomic.AddInt64(&done, 1)
            if onProgress != nil {
                onProgress(ProgressEvent{
                    Type:  ProgressFetchDone,
                    Path:  path,
                    Total: len(paths),
                    Done:  int(current),
                    Err:   err,
                })
            }
            return nil // Always continue
        })
    }
    _ = g.Wait()
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Custom semaphore + WaitGroup | `errgroup.SetLimit()` | errgroup v0.1.0 (2022) | Eliminates custom concurrency boilerplate |
| Loop variable bug workaround | Automatic per-iteration scoping | Go 1.22 (Feb 2024) | Project uses Go 1.21, so workaround still needed |
| Single enum status | Compound status set | This project | Enables "modified AND ahead" display |

**Deprecated/outdated:**
- `go-git` library: Not relevant -- project decision is os/exec for 100% git compatibility
- `sync.WaitGroup` + channel: Still works but errgroup.SetLimit is strictly better for this use case

## Open Questions

1. **Conflict file parsing from git output**
   - What we know: Git merge output includes `CONFLICT (content): Merge conflict in <file>` lines
   - What's unclear: The exact format varies by conflict type (content, rename, delete). Need to handle all variants or just content conflicts.
   - Recommendation: Start with regex parsing of `CONFLICT.*in (.+)$` pattern. Improve based on real-world testing. If parsing fails, fall back to empty file list in the hint.

2. **Root repository fetch**
   - What we know: Root repo status should be display-only, matching bash behavior
   - What's unclear: Should root repo be fetched in parallel with submodules, or separately before/after?
   - Recommendation: Fetch root repo in the same parallel batch as submodules but mark its result as `IsRoot: true`. This is simpler and avoids a separate fetch step.

3. **Dry-run scope**
   - What we know: CONTEXT.md lists dry-run as Claude's discretion
   - What's unclear: Whether engine should have a dry-run flag or if the caller (Phase 5 command) should simply call Scan but not call Update
   - Recommendation: Engine does NOT handle dry-run internally. Dry-run means the command calls `Scan()` to show what would change but does not call `Update()`. This keeps the engine simple and avoids conditional logic throughout. The scan result contains all information needed for a dry-run display.

4. **Push vs Update API shape**
   - What we know: CONTEXT.md lists API shape as Claude's discretion. Push and update share the scan result.
   - Recommendation: Separate methods: `Scan()`, `Update(scanResult, opts)`, `Push(scanResult, opts)`. The scan result is the shared input. This matches the bash workflow where scan happens first, then the user chooses update or push.

## Sources

### Primary (HIGH confidence)
- `golang.org/x/sync/errgroup` official docs -- API surface, SetLimit behavior, zero-value semantics
- Phase 2 source code (`internal/git/git.go`, `exec.go`, `branch.go`, `mock.go`) -- 19-method GitService interface, result types, error helpers
- Phase 2 test code (`exec_test.go`) -- Integration test patterns, test repo setup helpers

### Secondary (MEDIUM confidence)
- [errgroup with SetLimit pattern](https://pkg.go.dev/golang.org/x/sync/errgroup) -- Official Go package documentation
- [GitHub issue #23595: get all errors from errgroup](https://github.com/golang/go/issues/23595) -- Confirmed that zero-value Group does not cancel on error
- [GitHub errgroup source](https://github.com/golang/sync/blob/master/errgroup/errgroup.go) -- Verified cancel behavior: only WithContext sets cancel func

### Tertiary (LOW confidence)
- Web search results on worker pool and progress patterns -- Community consensus, not official guidance

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- errgroup is the standard Go solution for bounded concurrency; confirmed API with official docs
- Architecture: HIGH -- Patterns derived directly from Phase 2 interface and CONTEXT.md decisions
- Pitfalls: HIGH -- Go 1.21 loop capture bug and errgroup cancel semantics are well-documented known issues
- Conflict resolution: MEDIUM -- The 3-step strategy is locked by CONTEXT.md; conflict file parsing regex is unverified against all git output variants

**Research date:** 2026-02-09
**Valid until:** 2026-03-09 (stable domain -- Go stdlib and errgroup API are stable)
