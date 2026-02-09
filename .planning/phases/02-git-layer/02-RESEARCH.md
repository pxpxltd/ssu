# Phase 2: Git Layer - Research

**Researched:** 2026-02-09
**Domain:** Go os/exec git abstraction, interface design, context timeouts, branch detection, push operations
**Confidence:** HIGH

## Summary

This phase builds a testable git abstraction layer (`internal/git/`) that wraps all git CLI operations SSU needs via `os/exec.CommandContext`. The bash reference implementation defines exactly which git commands are needed: `rev-parse`, `branch -r`, `show-ref`, `symbolic-ref`, `fetch`, `merge`, `push`, `diff`, `status`, `stash`, `checkout`, `log`, `rev-list`, and `config --file .gitmodules`. All operations return structured Go types with stderr always captured.

The design follows a single `GitService` interface backed by an `ExecGit` struct for production and a `MockGitService` struct for testing. Go 1.20+ `exec.CommandContext` with `WaitDelay` handles timeouts robustly -- the context cancels the process, and `WaitDelay` force-kills lingering subprocesses. No third-party libraries are needed for this phase beyond the standard library.

The bash script's branch detection algorithm (lines 535-590 of `legacy/ssu`) is the authoritative reference. It must be ported faithfully: override > feature branch preservation > priority chain > remote HEAD > fallback. The feature branch rule (if current branch is not in priority list and has a remote, stay on it) is critical for correctness.

**Primary recommendation:** Define a `GitService` interface with ~15 methods that map 1:1 to the git operations the bash script performs. Return structured result types with `Stderr string` on every result. Use `exec.CommandContext` with `WaitDelay` for timeouts. Use a hand-written stub mock (no framework) for testing.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` | stdlib | Shell out to git CLI | Decision: os/exec not go-git. CommandContext provides timeout/cancellation |
| `context` | stdlib | Timeout propagation, cancellation | Standard Go pattern. Caller creates context, GitService respects it |
| `bytes` | stdlib | Capture stdout/stderr buffers | Simple, no allocation overhead for typical git output |
| `strings` | stdlib | Parse git command output | Git output is line-oriented text |
| `fmt` | stdlib | Error formatting | Wrapping errors with context |
| `errors` | stdlib | errors.Is, errors.As, sentinel errors | Go 1.13+ error handling |
| `time` | stdlib | Timeout durations | Default timeout values per operation |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `path/filepath` | stdlib | Submodule path resolution | Joining project root with submodule paths |
| `strconv` | stdlib | Parse commit counts from rev-list | `Atoi` for "git rev-list --count" output |
| `regexp` | stdlib | Parse .gitmodules output | Extract submodule paths from git config output |
| `syscall` | stdlib | SIGTERM for graceful process shutdown | Custom Cancel function on exec.Cmd (optional, Kill is fine for git) |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| os/exec (locked decision) | go-git v5 | go-git is pure Go but has edge-case incompatibilities with real git. Decision already locked: os/exec for 100% compatibility |
| Hand-written stub mock | testify/mock, gomock | Frameworks add dependencies and boilerplate. SSU's mock needs are simple (preconfigured responses). Hand-written is more idiomatic for this scope |
| Custom exec wrapper | ldez/go-git-cmd-wrapper | Adds a dependency for something we can do in ~50 lines. Our interface is narrower than a general-purpose wrapper |
| gg-scm.io/pkg/git | Custom interface | Full-featured but heavy. We need ~15 operations, not the full git surface. Our interface is purpose-built for SSU |

**Installation:**
```bash
# No new dependencies needed -- stdlib only for this phase
```

## Architecture Patterns

### Recommended Project Structure

```
internal/
  git/
    git.go           # GitService interface + result types + error types + Status enum
    exec.go           # ExecGit struct implementing GitService via os/exec
    branch.go         # Smart branch detection algorithm (DetectBestBranch)
    mock.go           # MockGitService for testing
    mock_test.go      # Verify mock satisfies interface
    exec_test.go      # Integration tests (need real git repo, build-tagged or skipped in CI)
    branch_test.go    # Unit tests for branch detection logic (uses mock)
```

### Pattern 1: Single Interface with Structured Results

**What:** One `GitService` interface with ~15 methods. Each returns a dedicated result struct + error.
**When to use:** Always -- this is the only way the Engine (Phase 3) and Commands (Phase 5) interact with git.

```go
// git.go

// GitService abstracts all git operations SSU needs.
// Implementations: ExecGit (production), MockGitService (testing).
type GitService interface {
    // Repository discovery
    SubmodulePaths(ctx context.Context, rootDir string) ([]string, error)
    IsSubmoduleInitialized(rootDir, subPath string) bool

    // Branch & revision queries
    CurrentBranch(ctx context.Context, dir string) (BranchResult, error)
    CurrentSHA(ctx context.Context, dir string) (string, error)
    IsDetachedHead(ctx context.Context, dir string) (bool, error)
    RemoteBranches(ctx context.Context, dir string) ([]RemoteBranch, error)
    CommitsBehind(ctx context.Context, dir, localRef, remoteRef string) (int, error)
    CommitsAhead(ctx context.Context, dir, remoteRef string) (int, error)
    HasRemoteBranch(ctx context.Context, dir, remote, branch string) (bool, error)
    SymbolicRef(ctx context.Context, dir, ref string) (string, error)
    TrackingBranch(ctx context.Context, dir string) (TrackingInfo, error)

    // Status queries
    HasLocalChanges(ctx context.Context, dir string) (bool, error)
    IncomingChangelog(ctx context.Context, dir, remoteRef string, limit int) ([]string, error)

    // Mutating operations
    Fetch(ctx context.Context, dir string, opts FetchOpts) (FetchResult, error)
    Checkout(ctx context.Context, dir, branch string) (CheckoutResult, error)
    CheckoutNewBranch(ctx context.Context, dir, branch, startPoint string) (CheckoutResult, error)
    Merge(ctx context.Context, dir, ref string) (MergeResult, error)
    Push(ctx context.Context, dir string, opts PushOpts) (PushResult, error)
    Stash(ctx context.Context, dir string) (StashResult, error)
    StashPop(ctx context.Context, dir string) (StashResult, error)
    MergeAbort(ctx context.Context, dir string) error
}
```

### Pattern 2: Result Types with Always-Present Stderr

**What:** Every result struct includes `Stderr string` for verbose/debug output even on success.
**When to use:** All git operations. Decision: stderr always captured, not just on error.

```go
// Result types -- every one carries Stderr

type FetchResult struct {
    Stderr string
}

type MergeResult struct {
    Success  bool
    Conflict bool  // true if merge failed due to conflict
    Stderr   string
}

type PushResult struct {
    Remote          string // which remote was pushed to
    Branch          string // which branch was pushed
    NewTracking     bool   // true if -u was used to set up tracking
    Stderr          string
}

type CheckoutResult struct {
    Branch string
    Stderr string
}

type StashResult struct {
    Created bool   // true if stash was created (changes existed)
    Stderr  string
}

type BranchResult struct {
    Name     string
    Detached bool   // true if HEAD is detached
    Stderr   string
}

type TrackingInfo struct {
    Remote string // e.g. "origin"
    Branch string // e.g. "develop"
    Set    bool   // false if no tracking branch configured
    Stderr string
}

type RemoteBranch struct {
    Remote string // e.g. "origin"
    Branch string // e.g. "develop"
}
```

### Pattern 3: ExecGit with CommandContext + WaitDelay

**What:** Production implementation runs git via `exec.CommandContext` with configurable per-operation timeouts.
**When to use:** All real git operations.

```go
// exec.go

type ExecGit struct {
    GitBin    string        // path to git binary, default "git"
    Timeouts  TimeoutConfig // per-operation timeout overrides
}

type TimeoutConfig struct {
    Fetch    time.Duration // default 120s
    Push     time.Duration // default 60s
    Merge    time.Duration // default 30s
    Default  time.Duration // default 30s -- for queries like rev-parse
}

// run executes a git command with context timeout and stderr capture.
func (g *ExecGit) run(ctx context.Context, dir string, timeout time.Duration, args ...string) (stdout, stderr string, err error) {
    if timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, timeout)
        defer cancel()
    }

    cmd := exec.CommandContext(ctx, g.gitBin(), args...)
    cmd.Dir = dir
    cmd.WaitDelay = 5 * time.Second // force-kill lingering processes

    var outBuf, errBuf bytes.Buffer
    cmd.Stdout = &outBuf
    cmd.Stderr = &errBuf

    err = cmd.Run()
    return strings.TrimSpace(outBuf.String()), errBuf.String(), err
}
```

### Pattern 4: Branch Detection as Pure Function on Interface

**What:** `DetectBestBranch` is a standalone function that calls GitService methods, not a method on ExecGit. This makes it testable with MockGitService.
**When to use:** The smart branch detection algorithm.

```go
// branch.go

// DetectBestBranch determines the target branch for a submodule.
// Priority: override > feature branch with remote > priority chain > remote HEAD > fallback.
// This matches the bash implementation's detect_best_branch() exactly.
func DetectBestBranch(ctx context.Context, svc GitService, dir string, opts BranchDetectOpts) (string, error) {
    // 1. Override
    if opts.Override != "" {
        return opts.Override, nil
    }

    // 2. Feature branch: if current branch is not in priority list and has remote, stay
    current, err := svc.CurrentBranch(ctx, dir)
    if err != nil {
        return "", fmt.Errorf("detect branch: %w", err)
    }
    if !current.Detached && !isInList(current.Name, opts.PriorityBranches) {
        hasRemote, err := svc.HasRemoteBranch(ctx, dir, "origin", current.Name)
        if err == nil && hasRemote {
            return current.Name, nil
        }
    }

    // 3. Priority chain
    remotes, err := svc.RemoteBranches(ctx, dir)
    // ... check priority branches against remote list
    // 4. Remote HEAD fallback
    // 5. First available or "master"
}

type BranchDetectOpts struct {
    Override         string   // --branch CLI flag
    PriorityBranches []string // default: ["develop", "master", "main"]
}
```

### Pattern 5: Simple Stub Mock

**What:** MockGitService with preconfigured return values -- no framework, no recording.
**When to use:** Unit testing Engine (Phase 3) and branch detection logic.

```go
// mock.go

type MockGitService struct {
    CurrentBranchFn     func(ctx context.Context, dir string) (BranchResult, error)
    RemoteBranchesFn    func(ctx context.Context, dir string) ([]RemoteBranch, error)
    HasRemoteBranchFn   func(ctx context.Context, dir, remote, branch string) (bool, error)
    FetchFn             func(ctx context.Context, dir string, opts FetchOpts) (FetchResult, error)
    MergeFn             func(ctx context.Context, dir, ref string) (MergeResult, error)
    PushFn              func(ctx context.Context, dir string, opts PushOpts) (PushResult, error)
    // ... one field per interface method
}

func (m *MockGitService) CurrentBranch(ctx context.Context, dir string) (BranchResult, error) {
    if m.CurrentBranchFn != nil {
        return m.CurrentBranchFn(ctx, dir)
    }
    return BranchResult{Name: "develop"}, nil // sensible default
}
// ... repeat for each method
```

### Anti-Patterns to Avoid

- **Embedding git command strings in Engine/Commands:** All git interaction MUST go through GitService. Never call `exec.Command("git", ...)` outside `internal/git/`.
- **Parsing stderr for control flow:** Use exit codes and structured results, not stderr string matching. Stderr is for logging/debug.
- **Using Output() instead of Run() with buffers:** `exec.CommandContext.Output()` has known issues with context timeout (Go issue #57129). Use `Run()` with `bytes.Buffer` assigned to Stdout/Stderr.
- **Hardcoding "origin":** Use TrackingInfo.Remote or accept remote as parameter. The bash version hardcodes origin in many places -- the Go version should be remote-aware.
- **Giant interface:** Don't add methods speculatively. Every method must correspond to a real git operation the bash script performs. ~15-20 methods is the right size.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Git output parsing | Custom parser for each command | `strings.TrimSpace`, `strings.Split`, `strconv.Atoi` | Git output is simple line-oriented text. No complex parser needed, but don't ad-hoc it either -- centralize in the `run` helper |
| Process timeout | Manual timer + kill | `exec.CommandContext` + `WaitDelay` (Go 1.20+) | Built into stdlib since Go 1.20. Handles orphaned subprocesses. Our go.mod has go 1.21 |
| Context cancellation | Channel-based signaling | `context.WithTimeout` / `context.WithCancel` | Standard Go pattern. Engine creates context, passes to GitService |
| Mock generation | testify/mock, gomock | Hand-written stub struct | SSU mock needs are simple. Function fields on a struct give full control with no dependency |
| Path joining | String concatenation | `filepath.Join` | Handles OS-specific separators correctly |
| Error wrapping | Custom error strings | `fmt.Errorf("operation: %w", err)` | Standard Go 1.13+ pattern, works with errors.Is/As |

**Key insight:** This phase is stdlib-only. The only non-stdlib dependency in the entire Go project is cobra/color from Phase 1, and the git layer doesn't need those.

## Common Pitfalls

### Pitfall 1: exec.CommandContext Output() Timeout Bug

**What goes wrong:** Using `cmd.Output()` with `CommandContext` may not respect the context deadline. The process continues running.
**Why it happens:** Go issue #57129. `Output()` creates internal pipes that may block even after context cancellation.
**How to avoid:** Use `cmd.Run()` with `bytes.Buffer` assigned to `cmd.Stdout` and `cmd.Stderr`. Always set `cmd.WaitDelay` to force-kill lingering processes.
**Warning signs:** Git operations that should timeout but hang instead.

### Pitfall 2: Feature Branch Detection Order

**What goes wrong:** If branch detection checks the priority chain before checking feature branches, submodules on feature branches get switched to develop/master.
**Why it happens:** The bash algorithm specifically checks feature branches BEFORE the priority chain. The order matters.
**How to avoid:** Port the bash algorithm faithfully. The order is: (1) override, (2) feature branch with remote, (3) priority chain, (4) remote HEAD, (5) fallback.
**Warning signs:** Submodules on `feature/xyz` branches switching to `develop` after update.

### Pitfall 3: Detached HEAD from rev-parse --abbrev-ref

**What goes wrong:** `git rev-parse --abbrev-ref HEAD` returns the literal string `"HEAD"` when in detached HEAD state, not an error.
**Why it happens:** Git's behavior. This isn't an error condition -- it's how git signals detached HEAD.
**How to avoid:** In `CurrentBranch`, check if the output is `"HEAD"` and set `BranchResult.Detached = true`. Don't treat it as a branch named "HEAD".
**Warning signs:** Trying to push/pull branch named "HEAD".

### Pitfall 4: Remote Branch Listing Includes HEAD Alias

**What goes wrong:** `git branch -r` output includes `origin/HEAD -> origin/develop`. Naively parsing this produces a branch named "HEAD -> origin/develop".
**Why it happens:** Git includes the symbolic ref alias in branch -r output.
**How to avoid:** Filter lines containing `->` when parsing `git branch -r` output. The bash version does `sed 's|origin/||'` which accidentally handles this because the arrow line doesn't match a simple branch name.
**Warning signs:** Branch detection finding phantom branches.

### Pitfall 5: Hardcoded "origin" in Remote Operations

**What goes wrong:** Push goes to wrong remote. Feature branch check fails because it only looks at `origin`.
**Why it happens:** Bash version hardcodes `origin` everywhere (e.g., `git show-ref --verify "refs/remotes/origin/$current_branch"`).
**How to avoid:** Use `TrackingBranch()` to discover the actual remote. Fall back to "origin" only as last resort. Accept remote as parameter where possible.
**Warning signs:** Operations failing in repos with non-origin remotes (upstream, fork, etc.).

### Pitfall 6: Stash Pop After Failed Merge

**What goes wrong:** `git stash pop` re-applies changes that conflict with the failed merge state, leaving the repo in a mess.
**Why it happens:** The conflict handling sequence (stash > merge > pop) is fragile. If merge succeeds but stash pop conflicts, you get a new conflict.
**How to avoid:** Return a result that indicates stash-pop status separately from merge status. Let the Engine (Phase 3) decide how to report this to the user. The bash version handles this in `handle_conflict()` lines 935-963.
**Warning signs:** User sees "Stash could not be applied cleanly" message and doesn't know what to do.

### Pitfall 7: Context Timeout vs Git Credential Prompts

**What goes wrong:** Git hangs waiting for a credential prompt (username/password) instead of timing out cleanly.
**Why it happens:** If git is configured to ask for credentials interactively, the process blocks on stdin. `WaitDelay` eventually kills it, but the error message is unhelpful.
**How to avoid:** Set `GIT_TERMINAL_PROMPT=0` in the command environment to disable interactive credential prompts. Also set `GIT_SSH_COMMAND` to use `BatchMode=yes` for SSH.
**Warning signs:** Fetch operations hitting timeout with no meaningful error.

## Code Examples

### Running a git command with timeout and stderr capture

```go
// Source: Go stdlib os/exec, verified against pkg.go.dev
func (g *ExecGit) run(ctx context.Context, dir string, timeout time.Duration, args ...string) (string, string, error) {
    if timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, timeout)
        defer cancel()
    }

    bin := g.GitBin
    if bin == "" {
        bin = "git"
    }

    cmd := exec.CommandContext(ctx, bin, args...)
    cmd.Dir = dir
    cmd.WaitDelay = 5 * time.Second

    // Disable interactive prompts that could hang
    cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
    return strings.TrimSpace(stdout.String()), stderr.String(), err
}
```

### Parsing submodule paths from .gitmodules

```go
// Equivalent to bash: git config --file .gitmodules --get-regexp '^submodule\..*\.path$' | awk '{print $2}'
func (g *ExecGit) SubmodulePaths(ctx context.Context, rootDir string) ([]string, error) {
    out, stderr, err := g.run(ctx, rootDir, g.Timeouts.Default,
        "config", "--file", ".gitmodules", "--get-regexp", `^submodule\..*\.path$`)
    if err != nil {
        // No .gitmodules or no submodules configured
        if strings.Contains(stderr, "No such file") || out == "" {
            return nil, nil
        }
        return nil, fmt.Errorf("list submodules: %w", err)
    }

    var paths []string
    for _, line := range strings.Split(out, "\n") {
        parts := strings.Fields(line)
        if len(parts) >= 2 {
            paths = append(paths, parts[1])
        }
    }
    sort.Strings(paths)
    return paths, nil
}
```

### Checking for unpushed commits (ahead detection)

```go
// Equivalent to bash: git rev-list --count "origin/$branch..HEAD"
func (g *ExecGit) CommitsAhead(ctx context.Context, dir, remoteRef string) (int, error) {
    out, _, err := g.run(ctx, dir, g.Timeouts.Default,
        "rev-list", "--count", remoteRef+"..HEAD")
    if err != nil {
        return 0, nil // treat errors as 0 ahead (same as bash: || echo "0")
    }
    n, err := strconv.Atoi(out)
    if err != nil {
        return 0, nil
    }
    return n, nil
}
```

### Push with automatic tracking branch setup

```go
// Matches bash push_submodule() logic exactly
func (g *ExecGit) Push(ctx context.Context, dir string, opts PushOpts) (PushResult, error) {
    result := PushResult{
        Remote: opts.Remote,
        Branch: opts.Branch,
    }

    // Check tracking branch
    tracking, _ := g.TrackingBranch(ctx, dir)

    if !tracking.Set {
        // No tracking branch -- set up with -u
        remote := opts.Remote
        if remote == "" {
            remote = "origin"
        }
        _, stderr, err := g.run(ctx, dir, g.Timeouts.Push,
            "push", "-u", remote, opts.Branch)
        result.Stderr = stderr
        result.NewTracking = true
        result.Remote = remote
        if err != nil {
            return result, &GitError{Op: "push", Err: err, Stderr: stderr}
        }
        return result, nil
    }

    // Has tracking -- push to tracked remote
    result.Remote = tracking.Remote
    _, stderr, err := g.run(ctx, dir, g.Timeouts.Push, "push")
    result.Stderr = stderr
    if err != nil {
        return result, &GitError{Op: "push", Err: err, Stderr: stderr}
    }
    return result, nil
}
```

### Error type design

```go
// GitError wraps a git operation failure with context.
type GitError struct {
    Op     string // "fetch", "push", "merge", etc.
    Stderr string // raw stderr output from git
    Err    error  // underlying error (exec.ExitError, context.DeadlineExceeded, etc.)
}

func (e *GitError) Error() string {
    if e.Stderr != "" {
        return fmt.Sprintf("git %s: %v\nstderr: %s", e.Op, e.Err, strings.TrimSpace(e.Stderr))
    }
    return fmt.Sprintf("git %s: %v", e.Op, e.Err)
}

func (e *GitError) Unwrap() error { return e.Err }

// Sentinel checks via errors.Is on the underlying error
func IsTimeout(err error) bool {
    return errors.Is(err, context.DeadlineExceeded)
}

func IsConflict(err error) bool {
    var ge *GitError
    if errors.As(err, &ge) {
        return ge.Op == "merge" && strings.Contains(ge.Stderr, "CONFLICT")
    }
    return false
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `cmd.Output()` for capturing stdout | `cmd.Run()` with `bytes.Buffer` | Go 1.20+ awareness | Avoids timeout bugs (issue #57129) |
| Manual process kill on timeout | `exec.CommandContext` + `cmd.WaitDelay` | Go 1.20 | Robust subprocess cleanup, handles orphaned children |
| `cmd.Cancel` default (Kill) | Can override with SIGTERM + WaitDelay | Go 1.20 | Graceful shutdown option (not needed for git, Kill is fine) |
| `errors.New` / string matching | `errors.Is` / `errors.As` with wrapped errors | Go 1.13+ | Type-safe error checking without string comparison |
| Manual error type switch | `GitError` struct with Unwrap | Go 1.13+ | Callers use `errors.As` to extract GitError details |

**Deprecated/outdated:**
- `go-git/go-git` v4: superseded by v5, but we don't use it anyway (locked decision: os/exec)
- `cmd.Process.Kill()` manual calls: replaced by CommandContext Cancel mechanism
- `cmd.Output()` for timeout scenarios: unreliable, use Run() with buffers

## Open Questions

Things that couldn't be fully resolved:

1. **Exact method count on the interface**
   - What we know: The bash script uses ~15 distinct git commands. Each maps to roughly one interface method.
   - What's unclear: Some operations like `has_local_changes` call multiple git commands (`diff --quiet`, `diff --cached --quiet`, `status --porcelain`). Should this be one method or three?
   - Recommendation: One method `HasLocalChanges()` that internally runs the three checks. The interface represents semantic operations, not raw git commands.

2. **Multi-remote branch detection**
   - What we know: CORE-07 requires multi-remote support. Bash hardcodes "origin" in `detect_best_branch`.
   - What's unclear: Should `DetectBestBranch` accept a remote parameter, or should it check all remotes?
   - Recommendation: Accept a `defaultRemote` parameter (usually "origin"), but `HasRemoteBranch` should accept any remote string. Phase 4 config wires the actual remote preference.

3. **Integration test strategy**
   - What we know: ExecGit needs real git repos for integration tests. Phase 1 has no integration tests.
   - What's unclear: How to set up test repos -- `t.TempDir()` with `git init`? Or skip integration tests and rely on mock tests + manual testing?
   - Recommendation: Use `t.TempDir()` + `git init` for a small set of integration tests in `exec_test.go`. Guard with `testing.Short()` so `go test -short ./...` skips them. This validates the real exec path works.

4. **Retry mechanism implementation**
   - What we know: Decision says "configurable retry count (default 0)". This is just for network operations (fetch, push).
   - What's unclear: Should retry logic live in GitService methods or in the Engine caller?
   - Recommendation: GitService does NOT retry. It's a single-shot executor. The Engine (Phase 3) implements retry logic by calling GitService methods in a loop. This keeps GitService simple and testable.

## Sources

### Primary (HIGH confidence)
- Go stdlib `os/exec` package: [pkg.go.dev/os/exec](https://pkg.go.dev/os/exec) -- CommandContext, WaitDelay, Cancel fields verified
- Go issue #57129: [github.com/golang/go/issues/57129](https://github.com/golang/go/issues/57129) -- Output() timeout bug confirmed
- Bash reference implementation: `/media/nvme/dev/pxpx/ssu/legacy/ssu` lines 535-999 -- all git commands catalogued
- Phase 1 codebase: existing patterns in `internal/cli/`, test conventions, module structure

### Secondary (MEDIUM confidence)
- DoltHub os/exec patterns: [dolthub.com/blog/2022-11-28-go-os-exec-patterns/](https://www.dolthub.com/blog/2022-11-28-go-os-exec-patterns/) -- errgroup + CommandContext patterns
- gg-scm.io/pkg/git: [pkg.go.dev/gg-scm.io/pkg/git](https://pkg.go.dev/gg-scm.io/pkg/git) -- reference for interface design with Runner abstraction and structured results
- Learn Go with Tests (os/exec): [quii.gitbook.io](https://quii.gitbook.io/learn-go-with-tests/questions-and-answers/os-exec) -- interface + dependency injection pattern for testable exec

### Tertiary (LOW confidence)
- go-git-cmd-wrapper: [pkg.go.dev/github.com/ldez/go-git-cmd-wrapper](https://pkg.go.dev/github.com/ldez/go-git-cmd-wrapper) -- functional options pattern (interesting but overkill for our scope)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- stdlib only, no library choices to validate
- Architecture: HIGH -- interface design informed by gg-scm reference, bash behavior fully catalogued
- Pitfalls: HIGH -- exec.CommandContext issues verified against Go issue tracker, branch detection order verified against bash source
- Error design: MEDIUM -- GitError struct pattern is idiomatic Go, but exact conflict detection heuristics may need tuning during implementation

**Research date:** 2026-02-09
**Valid until:** 2026-03-09 (stable -- stdlib doesn't change, bash reference is fixed)
