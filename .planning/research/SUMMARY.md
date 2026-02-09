# Project Research Summary

**Project:** SSU (Smart Submodule Updater) - Go Rewrite
**Domain:** CLI tool for git submodule management
**Researched:** 2026-02-09
**Confidence:** HIGH

## Executive Summary

SSU is rewriting a battle-tested 950-line bash tool into a modern Go CLI with interactive TUI. The bash version has clear workflows and known pain points that the Go rewrite must preserve and improve. The recommended approach is to use standard Go CLI patterns (cobra + bubbletea + os/exec to git) rather than pure Go git implementations. This gives 100% compatibility with git behavior, leverages established Go TUI patterns from tools like lazygit and gh, and keeps the codebase maintainable at 1500-2500 lines.

The core architectural insight is simple: shell out to git via os/exec behind an interface. This is the opposite of what most Go developers expect (go-git is tempting), but it's the correct choice. Git submodule operations are complex, poorly documented, and behavior-sensitive. Reimplementing them in pure Go guarantees incompatibilities. The bash version works precisely because it delegates to git CLI. The Go version should do the same, just with better error handling, concurrency, and UX.

Key risks are well-understood: goroutine leaks in parallel fetch (solved with context timeouts + errgroup), smart branch detection regressions (solved with test-first approach porting exact bash algorithm), and TUI hanging in CI/CD (solved with TTY detection). All are preventable with established Go patterns.

## Key Findings

### Recommended Stack

Use the standard Go CLI stack: cobra for subcommands, bubbletea/bubbles/lipgloss for TUI, viper for config, slog for logging, testify for testing, goreleaser for distribution. All are de facto standards with mature ecosystems.

**Core technologies:**
- **cobra (v1.8+)**: Subcommand routing — used by kubectl, gh, hugo, docker CLI. Handles flags, help, completions.
- **bubbletea (v1.2+)**: Interactive TUI with Elm architecture — the standard Go TUI framework.
- **os/exec (stdlib)**: Shell out to git — guarantees 100% git compatibility, no abstraction leaks.
- **viper (v1.19+)**: Config file management — YAML/JSON/TOML, multi-source merging, cobra integration.
- **errgroup (golang.org/x/sync)**: Parallel fetch with bounded concurrency — SetLimit() prevents goroutine explosion.
- **slog (stdlib)**: Structured logging — Go 1.21+, leveled, composable handlers.
- **goreleaser (v2+)**: Cross-platform builds, GitHub releases, Homebrew formula automation.

**Critical rejection:** Do NOT use go-git (incomplete submodule support, 10MB binary size, wrong abstraction) or git2go (CGo breaks cross-compilation). Shell out to git CLI.

### Expected Features

Full parity with bash v1.1.1 plus standard Go CLI expectations. The bash version has 32 table stakes features that must be matched exactly for migration. Beyond parity, users expect modern CLI conveniences: subcommands instead of flags, --json output, shell completions, and progress feedback during long operations.

**Must have (table stakes):**
- Status overview table with root repository display (colorized, column-aligned, shows behind count)
- Parallel fetch with smart branch detection (develop > master > main priority, feature branch preservation)
- Interactive TUI selector (arrow/vim keys, space toggle, all/none, confirm/quit)
- Auto/batch mode for CI/CD (--auto flag bypasses all prompts)
- Backup/rollback system (JSON backup to ~/.ssu/\<project\>/, atomic writes, exact SHA restoration)
- Conflict handling with automatic stash/retry
- Push mode for ahead submodules (detect unpushed, set tracking branch, push)
- Config file support (skip list, branch priority, parallel jobs)
- Logging to ~/.ssu/\<project\>/logs/ with rotation

**Should have (competitive):**
- Subcommand structure (ssu status, ssu update, ssu push, ssu rollback)
- --json output on status command
- Shell completions (bash/zsh/fish, free with cobra)
- Progress bar during parallel fetch
- Per-submodule fetch timeout (bash version hangs on network issues)
- Backwards compatibility hints (ssu --status → suggests ssu status)
- Graceful Ctrl+C handling (clean terminal, show partial results)

**Defer (v2+):**
- ssu init wizard, ssu diff command, ssu exec \<command\>, dashboard view
- Submodule grouping (--group plugins)
- Backup management commands (ssu backup list, ssu backup clean)

### Architecture Approach

Use a layered CLI architecture with separated TUI: cobra commands wire dependencies and invoke an Engine orchestrator, which calls a GitService interface (real impl uses os/exec, mock for tests). TUI is a side channel that receives data and returns selections but never calls git directly. This separation makes the codebase testable, maintains clean boundaries, and allows the same engine to power both interactive and batch modes.

**Major components:**
1. **Commands (internal/cmd/)**: Parse flags, load config, wire dependencies, invoke engine. Thin layer.
2. **Engine (internal/engine/)**: Orchestrates scan/update/push workflows. Calls git service, handles business logic (branch detection, conflict resolution).
3. **Git Service (internal/git/)**: Interface abstracting all git operations. Real impl shells to git via os/exec with context timeouts. Mock impl for unit tests.
4. **TUI (internal/tui/)**: Bubbletea models for selector, table, progress. Receives []SubmoduleInfo, returns selected paths. Never touches git.
5. **Config (internal/config/)**: Viper-based multi-layer config (defaults < ~/.ssu/config.yaml < .ssu.yaml < env < flags).
6. **Backup (internal/backup/)**: JSON backup/restore with atomic writes. Format compatible with bash version.

**Data flow:** Commands → Engine → Git Service (down). Results flow up. TUI is orthogonal: receives data, returns selections.

### Critical Pitfalls

Research identified 18 pitfalls across 8 categories. The top 5 are architectural decisions that must be made correctly in Phase 1 or will require major refactoring later.

1. **Using go-git instead of os/exec** — go-git has incomplete submodule support and behavior differences. Shell out to git CLI behind a GitService interface. This guarantees compatibility and respects user git config.

2. **Swallowing git's stderr** — cmd.Output() captures stdout only. Always capture both stdout and stderr, include stderr in error messages for debuggability.

3. **Goroutine leaks in parallel fetch** — Hung git fetch (network timeout, SSH prompt) blocks goroutine forever. Use context.WithTimeout per fetch + errgroup.SetLimit() for bounded concurrency.

4. **Bubbletea TUI in non-interactive environments** — Starting bubbletea in CI/CD or piped contexts hangs. Check term.IsTerminal for both stdin and stdout before launching TUI. --auto flag must bypass all TTY checks.

5. **Smart branch detection regressions** — The bash v1.1.1 fix was specifically for branch detection bugs. Write test matrix BEFORE implementing (feature branch with remote, feature branch without remote, detached HEAD, etc.). Port exact bash algorithm, then refactor.

## Implications for Roadmap

Based on research, suggested 6-phase structure following architectural dependencies:

### Phase 1: Foundation
**Rationale:** No dependencies. Must establish project structure, CLI framework, and critical architectural decisions (os/exec over go-git) before any feature work.
**Delivers:** go mod init, cobra root command with subcommand structure, project layout (internal/cmd, internal/engine, internal/git, etc.), CI pipeline, TTY detection, backwards compat hints.
**Addresses:** Subcommand CLI, version command, shell completions, NO_COLOR support (from FEATURES.md)
**Avoids:** Pitfall #1 (go-git), #4 (TUI in CI), #10 (cobra boilerplate), #12 (breaking compat)

### Phase 2: Git Layer
**Rationale:** Everything depends on git operations. Interface-based abstraction enables testing and development of higher layers in parallel.
**Delivers:** GitService interface, exec-based implementation with stderr capture and context timeouts, mock implementation for tests.
**Addresses:** Git operations abstraction (from ARCHITECTURE.md)
**Avoids:** Pitfall #1 (go-git), #2 (stderr), #5 (os.Chdir), #13 (locale), #17 (credentials)

### Phase 3: Engine Core
**Rationale:** Builds on git layer. Implements business logic for scanning, branch detection, and status analysis. Required before any user-facing features work.
**Delivers:** Scanner (list submodules, parallel fetch with errgroup, analyze status), smart branch detection algorithm (ported from bash with test matrix), Engine struct with dependency injection.
**Addresses:** Status overview table, parallel fetch, smart branch detection, skip list (from FEATURES.md)
**Avoids:** Pitfall #3 (goroutine leaks), #6 (branch detection regressions), #16 (race conditions)

### Phase 4: Config + Backup
**Rationale:** Parallel track to Phase 3. Config needed before commands have useful behavior. Backup required before any modification operations.
**Delivers:** Viper-based config loading (defaults < ~/.ssu/config.yaml < .ssu.yaml < env < flags), slog + lumberjack logging to ~/.ssu/\<project\>/logs/, JSON backup creation/restore with atomic writes.
**Addresses:** Config file support, logging, backup/rollback (from FEATURES.md)
**Avoids:** Pitfall #9 (XDG over-compliance), #15 (backup format compatibility)

### Phase 5: Commands + TUI
**Rationale:** Depends on engine, config, backup. Combines all infrastructure into user-facing commands. TUI is most complex UX but isolated from business logic.
**Delivers:** status, update, push, rollback commands wired to engine. Bubbletea selector with arrow keys, checkboxes, all/none. Progress bar during fetch. Colorized table output. --json flag on status.
**Addresses:** Interactive TUI selector, auto/batch mode, dry-run, conflict handling, push mode, feature branch detection, changelog preview (from FEATURES.md)
**Avoids:** Pitfall #7 (blocking Update), #8 (TTY detection), #14 (colors in tests)

### Phase 6: Distribution
**Rationale:** Final phase. No point automating releases until the tool works end-to-end.
**Delivers:** .goreleaser.yaml with cross-platform builds (linux/darwin, amd64/arm64), GitHub releases, Homebrew formula, version injection via ldflags, golangci-lint integration.
**Addresses:** Release automation (from STACK.md)
**Avoids:** Pitfall #11 (goreleaser config mistakes)

### Phase Ordering Rationale

- **Foundation first** because architectural decisions (os/exec, cobra structure, project layout) are expensive to change later.
- **Git layer second** because it's the critical path — everything depends on git operations working correctly.
- **Engine and Config in parallel** (Phases 3-4) because they don't depend on each other. Config can be developed/tested independently while engine implements core logic.
- **TUI last** (Phase 5) because it's purely presentation. The engine can be tested with mock git service and simple CLI output before adding bubbletea complexity.
- **Distribution final** (Phase 6) because goreleaser needs a working binary to package.

This ordering avoids rework: make the hard architectural decisions early (git abstraction, concurrency model), build testable core logic, add UX polish, automate distribution.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 3 (Engine Core):** Smart branch detection algorithm — needs careful porting from bash with comprehensive test cases. Existing bash code is reference but subtle bugs likely.
- **Phase 5 (TUI):** Bubbletea selector UX — needs review of bubbletea examples (gum, soft-serve, lazygit) for arrow key handling, checkbox rendering, progress updates.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Foundation):** Cobra CLI structure is well-documented, many examples (gh, hugo, kubectl).
- **Phase 2 (Git Layer):** os/exec is stdlib, pattern is straightforward.
- **Phase 4 (Config + Backup):** Viper and slog have clear documentation, JSON handling is stdlib.
- **Phase 6 (Distribution):** goreleaser has extensive examples and docs.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All recommendations are de facto standards with mature ecosystems. Cobra, bubbletea, viper, goreleaser are used by hundreds of popular Go CLIs. |
| Features | HIGH | Grounded in existing bash v1.1.1 codebase. Parity features are known, Go CLI expectations are established patterns from gh/kubectl/hugo. |
| Architecture | HIGH | Patterns from cobra, bubbletea, gh, lazygit are well-proven. Interface-based git abstraction is standard Go practice. |
| Pitfalls | HIGH | Based on established Go patterns and analysis of bash codebase pain points. All pitfalls have clear prevention strategies. |

**Overall confidence:** HIGH

### Gaps to Address

The research is highly confident because it builds on a working bash implementation and uses established Go patterns. However, two areas need validation during implementation:

- **Exact bash parity verification:** Need to test Go version against bash version on same repository states to catch behavioral differences. Create test matrix of submodule states (ahead, behind, modified, feature branch, detached, conflict) and verify identical behavior.
- **Bubbletea selector UX details:** Arrow key handling, checkbox rendering, progress updates during fetch. Need hands-on prototyping with bubbletea to confirm approach matches bash TUI fidelity. Review gum and lazygit source for patterns.

Both gaps are implementation details, not architectural uncertainties. The recommended stack and structure are solid.

## Sources

### Primary (HIGH confidence)
- Existing SSU bash codebase (v1.1.1) — working implementation with known workflows
- Cobra documentation (cobra.dev) — official CLI framework docs
- Bubbletea documentation (github.com/charmbracelet/bubbletea) — official TUI framework docs
- Go standard library documentation (go.dev/pkg) — os/exec, context, testing, slog
- goreleaser documentation (goreleaser.com) — official release automation docs

### Secondary (MEDIUM confidence)
- gh (GitHub CLI) source code — reference for cobra + bubbletea patterns
- lazygit source code — reference for git CLI TUI patterns
- gum source code — reference for bubbletea component patterns
- go-git issues (github.com/go-git/go-git/issues) — documented submodule limitations

### Tertiary (LOW confidence)
- Community blog posts on Go CLI patterns — general best practices, not project-specific

---
*Research completed: 2026-02-09*
*Ready for roadmap: yes*
