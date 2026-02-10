# Phase 7: Complete Rollback - Research

**Researched:** 2026-02-10
**Domain:** Git operations wiring, CLI command completion, integration testing
**Confidence:** HIGH

## Summary

Phase 7 closes the only remaining integration gap in the SSU v1 codebase: the rollback command reads and validates backup files but does not execute git operations to restore submodules. The `backup.Rollback()` function already exists with a well-designed callback injection pattern, the `cli/rollback.go` command has placeholder code at lines 97-102, and the `GitService` interface already has `Checkout` but lacks `ResetHard`.

The work is entirely internal to the existing codebase. No new dependencies are needed. The pattern for adding a new method to `GitService` is well-established (19 methods already exist), and the pattern for wiring git operations in CLI commands is demonstrated in `update.go` and `push.go`. The results table pattern is demonstrated in `status.go` (lipgloss/table) and `update.go` (summary printing).

**Primary recommendation:** Add `ResetHard` to `GitService` interface + `ExecGit` + `MockGitService`, then replace the stub in `cli/rollback.go` with a call to `backup.Rollback()` using closure callbacks that wrap `gitSvc.Checkout()` and `gitSvc.ResetHard()`. Display results with a lipgloss table. Add unit tests for `ResetHard` and integration tests for the rollback flow.

## Standard Stack

This phase uses no new libraries. Everything is already in the project.

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| charmbracelet/lipgloss | v1.0.0 | Results table rendering | Already used in status.go and update.go |
| charmbracelet/lipgloss/table | v1.0.0 | Table component | Already used for dry-run diff table |
| spf13/cobra | v1.10.2 | CLI command structure | Already used for all commands |
| fatih/color | v1.18.0 | Non-TUI colored output (printer) | Already used for success/error/warning |
| standard library: testing | - | Unit tests | Project convention (no testify) |
| standard library: os/exec | - | Git command execution | Used by ExecGit |

### Supporting
No new supporting libraries needed.

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| lipgloss/table for results | fmt.Fprintf with column alignment | Table is more visually consistent with existing status/dry-run tables |
| Closure callbacks for git ops | Direct GitService injection into backup package | Decision [04-02] locks this: backup package stays independent from git package |

**Installation:** No new dependencies to install.

## Architecture Patterns

### Recommended File Changes
```
internal/
├── git/
│   ├── git.go          # ADD: ResetHard to GitService interface + ResetHardResult type
│   ├── exec.go         # ADD: ResetHard method on ExecGit
│   ├── mock.go         # ADD: ResetHardFn field + ResetHard method on MockGitService
│   └── exec_test.go    # ADD: TestResetHard (optional, requires real git repo)
├── backup/
│   ├── rollback.go     # NO CHANGES (already complete with callback injection)
│   └── backup_test.go  # ADD: TestRollbackWithResetError, TestRollbackBashEra
└── cli/
    └── rollback.go     # REWRITE: Replace stub with backup.Rollback() call + results table
```

### Pattern 1: Adding a Method to GitService
**What:** Every new git operation follows the same 4-file pattern
**When to use:** Adding any new git capability
**Example:**

Step 1 - Interface (`git.go`):
```go
// In GitService interface:
ResetHard(ctx context.Context, dir, ref string) error
```

Step 2 - Production (`exec.go`):
```go
func (g *ExecGit) ResetHard(ctx context.Context, dir, ref string) error {
    _, stderr, err := g.run(ctx, dir, g.Timeouts.Default, "reset", "--hard", ref)
    if err != nil {
        return err
    }
    _ = stderr
    return nil
}
```

Step 3 - Mock (`mock.go`):
```go
// Add field:
ResetHardFn func(ctx context.Context, dir, ref string) error

// Add method:
func (m *MockGitService) ResetHard(ctx context.Context, dir, ref string) error {
    if m.ResetHardFn != nil {
        return m.ResetHardFn(ctx, dir, ref)
    }
    return nil
}
```

Step 4 - Compile check (`mock_test.go`): Already exists via `var _ git.GitService = (*git.MockGitService)(nil)`.

### Pattern 2: Wiring Callbacks in CLI Commands
**What:** The rollback command must create closure functions that capture a `*git.ExecGit` instance and adapt its methods to the `backup.GitCheckoutFunc` / `backup.GitResetHardFunc` signatures
**When to use:** When connecting the backup package to git operations
**Example:**

```go
gitSvc := git.NewExecGit()
ctx := cmd.Context()

// Adapt GitService.Checkout to backup.GitCheckoutFunc
gitCheckout := func(dir, branch string) error {
    fullDir := filepath.Join(projectRoot, dir)
    _, err := gitSvc.Checkout(ctx, fullDir, branch)
    return err
}

// Adapt GitService.ResetHard to backup.GitResetHardFunc
gitResetHard := func(dir, sha string) error {
    fullDir := filepath.Join(projectRoot, dir)
    return gitSvc.ResetHard(ctx, fullDir, sha)
}

// Adapt for GetCurrentStatesFunc
getCurrentStates := func(projectRoot string, paths []string) (map[string]backup.SubmoduleState, error) {
    states := make(map[string]backup.SubmoduleState)
    for _, p := range paths {
        subDir := filepath.Join(projectRoot, p)
        sha, err := gitSvc.CurrentSHA(ctx, subDir)
        if err != nil {
            return nil, fmt.Errorf("getting SHA for %s: %w", p, err)
        }
        br, _ := gitSvc.CurrentBranch(ctx, subDir)
        states[p] = backup.SubmoduleState{SHA: sha, Branch: br.Name}
    }
    return states, nil
}
```

**Critical detail:** The backup package stores submodule paths as relative paths (e.g., `plugins/auth`). The `GitCheckoutFunc` and `GitResetHardFunc` receive these relative paths. The CLI wrappers must join them with `projectRoot` to get absolute paths for `ExecGit`.

### Pattern 3: Results Table Display
**What:** Lipgloss table showing rollback results per submodule
**When to use:** After rollback completes
**Example:**

```go
t := table.New().
    Headers("Path", "Branch", "SHA", "Status").
    Border(lipgloss.NormalBorder()).
    BorderHeader(true).
    BorderColumn(true).
    Width(100)

for _, sub := range result.Submodules {
    status := "restored"
    if sub.Error != nil {
        status = "error: " + sub.Error.Error()
    }
    sha := sub.SHA
    if len(sha) > 7 {
        sha = sha[:7]
    }
    t.Row(sub.Path, sub.Branch, sha, status)
}
```

### Anti-Patterns to Avoid
- **Importing git package from backup package:** Decision [04-02] explicitly forbids this. The callback injection pattern MUST be used.
- **Using `git.ResetHard` with no context parameter:** All `ExecGit` methods use `context.Context` for timeout handling. The `ResetHard` method must follow this pattern.
- **Skipping safety backup:** Decision [04-02] requires a safety backup before any rollback restore. The `backup.Rollback()` already does this when `getCurrentStates != nil && opts.BackupDir != ""`.
- **Returning a complex result type from ResetHard:** `git reset --hard` is a simple operation. A plain `error` return is sufficient (matches the `MergeAbort` pattern which also returns just `error`).

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Path resolution for submodules | Manual string concat | `filepath.Join(projectRoot, subPath)` | Cross-platform path handling |
| Backup parsing | Custom JSON parser | `backup.Read()` | Already handles both go-era and bash-era formats |
| Safety backup creation | Manual file ops | `backup.Create()` | Already has atomic writes |
| User confirmation | Custom bufio prompt | Existing pattern: `bufio.Scanner` on stdin | Match `init.go` confirmation pattern |
| SHA truncation | Manual string slicing | `sha[:minInt(7, len(sha))]` | `minInt` already exists in `rollback.go` |

**Key insight:** The `backup.Rollback()` function is already fully implemented. The CLI just needs to provide the three callback functions and display the result.

## Common Pitfalls

### Pitfall 1: Relative vs Absolute Paths
**What goes wrong:** `backup.Rollback()` passes relative submodule paths (from the backup file) to the callback functions. `ExecGit` methods expect directory paths where commands will be executed.
**Why it happens:** Backup files store paths like `plugins/auth`, but git commands need to run in the actual directory.
**How to avoid:** The closure wrappers must `filepath.Join(projectRoot, subPath)` before calling `ExecGit` methods.
**Warning signs:** "No such file or directory" errors from git commands during rollback.

### Pitfall 2: ResetHard Return Type Inconsistency
**What goes wrong:** Other mutating operations (Checkout, Merge, Push) return result types with Stderr. `ResetHard` could follow this pattern or use the simpler `MergeAbort` pattern (just error return).
**Why it happens:** Inconsistency in the interface design.
**How to avoid:** Use the simpler `error`-only return. `git reset --hard` either works or fails; there's no useful intermediate state. The `MergeAbort` method already sets the precedent for `error`-only returns on simple operations.
**Warning signs:** If you need the Stderr for logging, you can capture it in the method but not expose it in the return type.

### Pitfall 3: Missing User Confirmation
**What goes wrong:** Rollback modifies submodule state destructively. Running without confirmation could surprise users.
**Why it happens:** The current stub doesn't have a confirmation step.
**How to avoid:** In interactive mode (TTY), prompt for confirmation before executing. In auto mode, proceed without confirmation. Match the update command's pattern.
**Warning signs:** Users accidentally restoring wrong backup.

### Pitfall 4: Context Not Threaded Through
**What goes wrong:** The `backup.GitCheckoutFunc` and `backup.GitResetHardFunc` signatures don't include `context.Context`, but `ExecGit.Checkout` and `ExecGit.ResetHard` require it.
**Why it happens:** The backup package was designed to be git-independent. Its callback signatures are simple.
**How to avoid:** Capture `ctx` in the closure. The CLI command creates `ctx` and the closure closes over it. This is the intended pattern.
**Warning signs:** None -- this is already the pattern used in `TestRollbackWithCallbacks`.

### Pitfall 5: Forgetting to Resolve Backup Path
**What goes wrong:** User might provide a relative path or just a filename. The command needs to resolve it to an absolute path.
**Why it happens:** Users might be in a different directory than the backup directory.
**How to avoid:** If the provided path doesn't exist as-is, try resolving it relative to the backup directory (`~/.ssu/<project>/backups/`). The current code in `rollback.go` already passes `args[0]` directly to `backup.Read()`.
**Warning signs:** "No such file or directory" when the backup exists but the path is relative.

## Code Examples

Verified patterns from the existing codebase:

### ExecGit Method Pattern (from exec.go)
```go
// Source: internal/git/exec.go - MergeAbort (simplest mutating operation, error-only return)
func (g *ExecGit) MergeAbort(ctx context.Context, dir string) error {
    _, _, err := g.run(ctx, dir, g.Timeouts.Default, "merge", "--abort")
    return err
}
```

### Rollback Callback Test Pattern (from backup_test.go)
```go
// Source: internal/backup/backup_test.go - TestRollbackWithCallbacks
checkout := func(dir, branch string) error {
    checkedOutBranch = branch
    return nil
}
resetHard := func(dir, sha string) error {
    resetSHA = sha
    return nil
}
```

### Update Command Backup Pattern (from update.go)
```go
// Source: internal/cli/update.go - createBackupIfEnabled
states := make(map[string]backup.SubmoduleState)
for _, sub := range targets {
    subDir := filepath.Join(rootDir, sub.Path)
    sha, shaErr := gitSvc.CurrentSHA(ctx, subDir)
    if shaErr != nil {
        continue
    }
    states[sub.Path] = backup.SubmoduleState{
        SHA:    sha,
        Branch: sub.CurrentBranch,
    }
}
```

### Results Table Pattern (from update.go dry-run)
```go
// Source: internal/cli/update.go - runUpdateDryRun
t := table.New().
    Headers("Path", "Current SHA", "Target SHA", "Behind").
    Border(lipgloss.NormalBorder()).
    BorderHeader(true).
    BorderColumn(true).
    Width(120)

// ... add rows ...

t.StyleFunc(func(row, col int) lipgloss.Style {
    if row == table.HeaderRow {
        return tui.HeaderStyle
    }
    return lipgloss.NewStyle()
})
fmt.Fprintln(cmd.OutOrStdout(), t.Render())
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Stub with warning message | Fully wired rollback | Phase 7 | Completes the only broken E2E flow |

**Deprecated/outdated:**
- The `p.Warning("Git operations not yet wired (Phase 5)")` stub in `rollback.go:97-102` must be completely replaced.

## Open Questions

Things that couldn't be fully resolved:

1. **Should rollback include a user confirmation prompt?**
   - What we know: Update and push commands use TUI selectors. Rollback is a destructive operation.
   - What's unclear: The success criteria don't mention confirmation, but the command description in `rollback.go` says "A safety backup of the current state is automatically created before restoring."
   - Recommendation: Add a simple Y/N confirmation in interactive mode (matching init.go's bufio pattern). Skip in auto mode. The safety backup provides the undo mechanism.

2. **Should the results table show the "previous SHA" column?**
   - What we know: Success criteria say "path, previous SHA, restored SHA, status". The `backup.RollbackResult` only contains target SHA and error per submodule, not previous SHA.
   - What's unclear: To show previous SHA, we'd need to capture it before rollback or from the safety backup.
   - Recommendation: Use the safety backup filename as context. For previous SHA, the `getCurrentStates` callback already collects this -- store it in a local map before calling `Rollback()` and display alongside results.

3. **Should `ResetHard` return a result type with Stderr, or just error?**
   - What we know: Most mutating operations return result types with Stderr. But `MergeAbort` returns just error.
   - What's unclear: Whether the caller ever needs Stderr from reset --hard.
   - Recommendation: Return just `error`. `git reset --hard` is a simple pass/fail operation. If logging is needed, the `GitError` type already captures stderr in the error itself. This matches the `MergeAbort` pattern exactly.

## Sources

### Primary (HIGH confidence)
- Codebase analysis: `internal/backup/rollback.go` -- Rollback function fully implemented with callback injection
- Codebase analysis: `internal/cli/rollback.go` -- Stub at lines 97-102 identified
- Codebase analysis: `internal/git/git.go` -- GitService interface (19 methods, no ResetHard)
- Codebase analysis: `internal/git/exec.go` -- ExecGit production implementation pattern
- Codebase analysis: `internal/git/mock.go` -- MockGitService pattern (Fn fields + defaults)
- Codebase analysis: `internal/cli/update.go` -- Backup creation and results display patterns
- Codebase analysis: `internal/cli/status.go` -- Lipgloss table rendering pattern
- Codebase analysis: `internal/backup/backup_test.go` -- TestRollbackWithCallbacks and TestRollbackDryRun

### Secondary (MEDIUM confidence)
- `.planning/v1-MILESTONE-AUDIT.md` -- Integration gap analysis, fix recommendations
- `.planning/ROADMAP.md` -- Phase 7 success criteria and plan structure
- `.planning/STATE.md` -- Decision [04-02] on callback injection pattern

### Tertiary (LOW confidence)
- None. All findings are from direct codebase inspection.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - No new dependencies; all patterns already exist in codebase
- Architecture: HIGH - Direct inspection of all involved files; clear 4-file pattern for GitService extension
- Pitfalls: HIGH - Based on concrete code analysis (path handling, context threading, callback signatures)

**Research date:** 2026-02-10
**Valid until:** Indefinite (internal codebase patterns, not external dependencies)
