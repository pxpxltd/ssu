# Common Pitfalls

**Domain:** Go CLI git submodule manager (rewrite from bash)
**Researched:** 2026-02-09
**Confidence:** HIGH (based on established Go patterns + analysis of existing bash codebase)

## Critical Pitfalls

### 1. Using go-git Instead of os/exec

**Risk:** go-git is a pure Go git reimplementation. Incomplete submodule support, complex transport config, behavior differences from real git.

**Prevention:** Shell out to `git` via `os/exec` behind a `GitService` interface. This guarantees identical behavior to bash version and respects user's git config (credentials, hooks).

**Phase:** Phase 1 (Git Layer) — this is the most important architectural decision.

### 2. Swallowing Git's stderr

**Risk:** `cmd.Output()` captures stdout only. Errors from git go to stderr and are lost, making debugging impossible.

**Prevention:**
```go
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr
if err := cmd.Run(); err != nil {
    return fmt.Errorf("git %s: %w\nstderr: %s", args, err, stderr.String())
}
```

**Phase:** Phase 1 (Git Layer)

### 3. Goroutine Leaks in Parallel Fetch

**Risk:** Hung git fetch (network timeout, SSH prompt) blocks goroutine forever. With 20+ submodules, leaked goroutines accumulate.

**Prevention:** Use `context.WithTimeout` per fetch + `errgroup.SetLimit()`:
```go
fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
cmd := exec.CommandContext(fetchCtx, "git", "fetch", "--all")
```

**Phase:** Phase 2 (Parallel Operations)

### 4. Bubbletea TUI in Non-Interactive Environments

**Risk:** Starting bubbletea in CI/CD or piped contexts. TUI reads from stdin, causing hangs.

**Prevention:** Check TTY before launching TUI:
```go
canUseTUI := term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
```
`--auto` flag bypasses ALL TTY checks.

**Phase:** Phase 1 (CLI Framework)

### 5. Working Directory Mutation with os.Chdir

**Risk:** `os.Chdir()` is process-global. In concurrent goroutines, one goroutine's chdir affects all others.

**Prevention:** Never use `os.Chdir()`. Use `cmd.Dir` for exec.Command:
```go
cmd := exec.Command("git", "status")
cmd.Dir = submodulePath
```

**Phase:** Phase 1 (Git Layer)

### 6. Smart Branch Detection Regressions

**Risk:** The bash v1.1.1 fix was specifically for branch detection bugs. This is where bugs concentrate.

**Prevention:** Write test cases BEFORE implementing:
| Current Branch | Has Remote? | Priority Available | Expected |
|---|---|---|---|
| feature/auth | yes | develop, master | feature/auth |
| feature/auth | no | develop, master | develop |
| detached | n/a | develop, master | develop |
| develop | yes | develop, master | develop |
| main | yes | main only | main |

Port exact algorithm from bash, then refactor.

**Phase:** Phase 1 (Core Logic)

## Moderate Pitfalls

### 7. Bubbletea State Management

**Risk:** Blocking I/O in Update() freezes UI. Mutating model wrong.

**Prevention:** Use `tea.Cmd` for async operations. Never call git in Update().
```go
case tea.KeyMsg:
    if msg.String() == "enter" {
        return m, startFetchCmd // Returns Cmd, doesn't block
    }
case fetchCompleteMsg:
    m.results = msg.results
```

**Phase:** Phase 2 (TUI)

### 8. TTY Detection Breaking Piped Usage

**Risk:** `ssu status | grep pending` fails because non-TTY disables everything.

**Prevention:** Check stdin and stdout independently:
```go
canUseTUI := stdinIsTerminal && stdoutIsTerminal  // TUI needs both
useColors := stdoutIsTerminal || forceColor         // Colors only need stdout
```
Support `NO_COLOR` and `FORCE_COLOR` env vars.

**Phase:** Phase 1 (CLI Framework)

### 9. Config File XDG Over-Compliance

**Risk:** Splitting config across `~/.config/ssu/`, `~/.local/share/ssu/`, `~/.cache/ssu/` — three directories for one CLI tool. Orphans existing `~/.ssu/` backups.

**Prevention:** Keep `~/.ssu/` as primary directory. It's established, simple, users have it. Support `$SSU_HOME` override.

**Phase:** Phase 1 (Config)

### 10. Cobra Boilerplate Explosion

**Risk:** One file per command with `init()` functions, flag duplication, hard-to-test wiring.

**Prevention:** Builder pattern with dependency injection:
```go
func newRootCmd(app *App) *cobra.Command {
    root := &cobra.Command{Use: "ssu"}
    root.AddCommand(newStatusCmd(app), newUpdateCmd(app), ...)
    return root
}
```
No `init()` functions.

**Phase:** Phase 1 (CLI Framework)

### 11. goreleaser Config Mistakes

**Risk:** Dynamic linking (GLIBC issues), missing version injection, Homebrew SHA mismatch.

**Prevention:**
```yaml
builds:
  - env: [CGO_ENABLED=0]
    goos: [linux, darwin]
    goarch: [amd64, arm64]
    ldflags: [-s -w, -X main.version={{.Version}}]
```
Test locally: `goreleaser release --snapshot --clean`

**Phase:** Final phase (Distribution)

### 12. Breaking Backward Compatibility

**Risk:** `ssu --status` fails with unhelpful error. CI pipelines break.

**Prevention:** Detect old flags, print migration hints:
```
$ ssu --status
Hint: The --status flag is now a subcommand. Use: ssu status
```

**Phase:** Phase 1 (CLI Framework)

## Minor Pitfalls

### 13. Git Output Locale Sensitivity

Set `LC_ALL=C` on all git commands. Use porcelain/machine-readable flags.

### 14. Color Output in Tests

Set `NO_COLOR=1` in tests. Test logic separately from rendering.

### 15. JSON Backup Format Incompatibility

Match bash format exactly. Use `json.MarshalIndent`. Test with real bash-generated backup files. Keep `.submodule-backup-YYYYMMDD-HHMMSS.json` filename pattern.

### 16. Race Conditions in Progress Reporting

Use channels for progress, not shared counters. Or use `tea.Msg` for bubbletea progress.

### 17. Git Credential Prompts Hanging

Set `GIT_TERMINAL_PROMPT=0`. Provide clear auth error messages. Never connect stdin for parallel operations.

### 18. Over-Engineering

**Target:** ~1500-2500 lines of Go (including tests). If significantly larger, question why. Start with concrete types, extract interfaces only when needed for testing. 5-7 packages, not 15.

## Phase-Specific Warnings

| Phase | Top Pitfalls |
|-------|-------------|
| Git Layer | #1 (go-git), #2 (stderr), #5 (os.Chdir), #13 (locale), #17 (credentials) |
| Branch Detection | #6 (regression — test matrix first) |
| CLI Framework | #4 (TUI in CI), #8 (TTY), #10 (cobra boilerplate), #12 (compat) |
| Parallel Ops | #3 (goroutine leaks), #16 (race conditions) |
| TUI | #7 (blocking Update), #14 (colors in tests) |
| Config | #9 (XDG over-compliance) |
| Backup | #15 (format compatibility) |
| Distribution | #11 (goreleaser) |
| Overall | #18 (over-engineering) |

---
*Pitfalls research: 2026-02-09*
