# Roadmap: SSU Go Rewrite

## Overview

Rewrite SSU from a 950-line bash script into a structured Go CLI with interactive TUI, YAML configuration, and cross-platform distribution. The build follows architectural dependencies: CLI framework first, then the git abstraction layer, then the engine that orchestrates operations, then config/safety infrastructure, then user-facing commands wired to the bubbletea TUI, and finally distribution packaging. Phases 3 and 4 are independent and can be developed in parallel.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation** - Go project structure, cobra CLI skeleton, and cross-cutting concerns
- [x] **Phase 2: Git Layer** - GitService interface and os/exec implementation with smart branch detection
- [ ] **Phase 3: Engine** - Scanning, status analysis, parallel fetch, and conflict handling orchestration
- [ ] **Phase 4: Config + Safety** - YAML configuration, backup/rollback, and structured logging
- [ ] **Phase 5: Commands + TUI** - User-facing commands wired to bubbletea interactive selector
- [ ] **Phase 6: Distribution** - Cross-platform builds, package managers, and install script

## Phase Details

### Phase 1: Foundation
**Goal**: A runnable `ssu` binary with subcommand routing, version info, and correct terminal behavior
**Depends on**: Nothing (first phase)
**Requirements**: CLI-01, CLI-02, CLI-03, CLI-04, CLI-05, TUI-06
**Plans:** 2 plans
**Success Criteria** (what must be TRUE):
  1. Running `ssu` displays help with available subcommands (status, update, push, rollback)
  2. Running `ssu version` prints version, commit hash, and build date
  3. Running `ssu --status` prints a hint suggesting `ssu status` instead
  4. Tab completion works in bash, zsh, and fish after running the completion setup command
  5. Output respects NO_COLOR env var and disables colors when stdout is not a TTY

Plans:
- [x] 01-01-PLAN.md — Go module init, project layout, output utilities, and cobra root command with subcommand stubs
- [x] 01-02-PLAN.md — Version command, shell completions, backwards compat hints, exit codes, and test suite

### Phase 2: Git Layer
**Goal**: A testable git abstraction that handles all git operations SSU needs, with context timeouts and correct branch detection
**Depends on**: Phase 1
**Requirements**: CORE-02, CORE-03, CORE-06, CORE-07, PUSH-03
**Plans:** 3 plans
**Success Criteria** (what must be TRUE):
  1. GitService interface exists with mock implementation, enabling unit tests without a real git repo
  2. Smart branch detection follows the priority chain (develop > master > main > remote HEAD > fallback) and preserves feature branches
  3. Git operations use configurable context timeouts that kill hung fetches
  4. Git operations work with remotes other than "origin" when configured
  5. Push operation automatically sets up tracking branch when none exists

Plans:
- [x] 02-01-PLAN.md — GitService interface, result/error types, Status enum, and MockGitService with compile-time check
- [x] 02-02-PLAN.md — Smart branch detection algorithm (DetectBestBranch) with 14 table-driven test cases using mock
- [x] 02-03-PLAN.md — ExecGit production implementation (os/exec, timeouts, stderr capture) with integration tests

### Phase 3: Engine
**Goal**: The core orchestrator that scans submodules in parallel, detects status, handles conflicts, and coordinates update/push workflows
**Depends on**: Phase 2
**Requirements**: CORE-01, CORE-04, CORE-05, CORE-08, CONF-01, CONF-02, CONF-03, CONF-04, PUSH-01, PUSH-04
**Plans:** 3 plans
**Success Criteria** (what must be TRUE):
  1. Engine scans all submodules in parallel with bounded concurrency and returns status for each (pending, current, modified, ahead, conflict, missing, skipped)
  2. Engine detects root repository status and includes it as display-only (excluded from operations)
  3. Engine shows changelog preview of incoming commits per submodule
  4. Dirty submodules get per-submodule handling: stash+merge, skip, or force -- with actionable conflict resolution hints on failure
  5. Engine detects ahead submodules with unpushed commits and skips detached HEAD submodules with a warning

Plans:
- [ ] 03-01-PLAN.md — Engine types, progress callback, parallel scanner with errgroup, submodule enumeration, and status detection
- [ ] 03-02-PLAN.md — Update workflow with 3-step conflict resolution (stash/retry/reapply), actionable hints
- [ ] 03-03-PLAN.md — Push workflow orchestration, ahead detection, detached HEAD handling

### Phase 4: Config + Safety
**Goal**: Layered YAML configuration and reliable backup/rollback with structured logging
**Depends on**: Phase 1 (can run parallel with Phase 3)
**Requirements**: CFG-01, CFG-02, CFG-03, CFG-04, SAFE-01, SAFE-02, SAFE-03, SAFE-04, SAFE-05, SAFE-06
**Success Criteria** (what must be TRUE):
  1. Config loads from defaults < ~/.ssu/config.yaml < .ssu.yaml < env vars < CLI flags, with each layer overriding the previous
  2. Skip list, branch priority, and parallel jobs are configurable and respected by engine
  3. JSON backup is created atomically (write temp, rename) before any submodule modification, and rollback restores exact SHAs (compatible with bash-era backup format)
  4. `ssu backup list` shows available backups and `ssu backup clean --keep N` removes old ones
  5. Logs are written to ~/.ssu/<project>/logs/ with size/date-based rotation
**Plans**: TBD

Plans:
- [ ] 04-01: Viper config loading with layering, config types, and defaults
- [ ] 04-02: JSON backup with atomic writes, rollback restore, bash-format compatibility, backup management commands
- [ ] 04-03: slog-based structured logging to ~/.ssu/<project>/logs/ with rotation, fail-fast mode

### Phase 5: Commands + TUI
**Goal**: Fully functional interactive CLI where users can scan, select, update, push, and rollback submodules through a polished TUI
**Depends on**: Phase 3, Phase 4
**Requirements**: CLI-06, CLI-07, CLI-08, TUI-01, TUI-02, TUI-03, TUI-04, TUI-05, TUI-07, PUSH-02
**Success Criteria** (what must be TRUE):
  1. `ssu status` displays a colorized table with root repo and all submodules; `ssu status --json` outputs machine-readable JSON
  2. `ssu update` launches a bubbletea multi-select TUI (arrow/vim keys, space toggle, all/none, confirm/quit) for choosing submodules to update, with `--auto` bypassing the TUI for CI/CD
  3. `ssu push` shows ahead submodules in the TUI selector for interactive push selection
  4. `ssu update --dry-run` previews what would change without modifying anything
  5. Parallel fetch shows a progress indicator per submodule, and Ctrl+C cleanly restores terminal state and shows partial results
**Plans**: TBD

Plans:
- [ ] 05-01: Status command with colorized table output and --json flag
- [ ] 05-02: Bubbletea multi-select TUI model (selector, checkboxes, progress bar)
- [ ] 05-03: Update and push commands wired to engine + TUI, dry-run mode, auto mode
- [ ] 05-04: Exec command, init wizard, Ctrl+C handling

### Phase 6: Distribution
**Goal**: Users can install SSU via their preferred method on any supported platform
**Depends on**: Phase 5
**Requirements**: DIST-01, DIST-02, DIST-03, DIST-04, DIST-05, DIST-06
**Success Criteria** (what must be TRUE):
  1. `go install github.com/.../cmd/ssu@latest` installs a working binary
  2. GitHub releases contain static binaries for linux/darwin on amd64/arm64
  3. `brew install <tap>/ssu` installs the latest release on macOS and Linux
  4. AUR package is available for Arch Linux users
**Plans**: TBD

Plans:
- [ ] 06-01: cmd/ssu/main.go entry point, go install support, CGO_ENABLED=0 static builds
- [ ] 06-02: goreleaser config, GitHub Actions release workflow, Homebrew tap
- [ ] 06-03: AUR package, install script (curl-pipe-bash)

## Progress

**Execution Order:**
Phases execute in numeric order: 1 > 2 > 3 (parallel with 4) > 5 > 6

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation | 2/2 | Complete | 2026-02-09 |
| 2. Git Layer | 3/3 | Complete | 2026-02-09 |
| 3. Engine | 0/3 | Not started | - |
| 4. Config + Safety | 0/3 | Not started | - |
| 5. Commands + TUI | 0/4 | Not started | - |
| 6. Distribution | 0/3 | Not started | - |
