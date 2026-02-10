# SSU — Smart Submodule Updater (Go Rewrite)

## What This Is

A Go rewrite of SSU, an intelligent git submodule management tool. SSU provides interactive and automated workflows for updating and pushing submodules with smart branch detection, conflict handling, and backup/rollback. The Go version replaces the existing 950-line bash script with a properly structured, testable, cross-platform binary.

## Core Value

Safely update and push git submodules with zero data loss — smart branch detection picks the right branch, backups happen before any modification, and conflicts resolve automatically.

## Requirements

### Validated

These capabilities exist in the bash version and are proven valuable:

- ✓ Parallel fetch of all submodule remote refs — existing
- ✓ Smart branch detection (develop → master → main → remote HEAD → fallback) — existing
- ✓ Feature branch preservation (submodules on feature branches stay on them) — existing
- ✓ Status table showing path, branch, behind count, feature flag, status — existing
- ✓ Interactive selection of submodules to update — existing
- ✓ Auto mode for batch/CI usage (no prompts) — existing
- ✓ Dry-run preview mode — existing
- ✓ Automatic conflict handling (stash → retry merge → reapply) — existing
- ✓ JSON backup before modifications with rollback support — existing
- ✓ Push mode for ahead submodules with tracking branch setup — existing
- ✓ Skip list for excluding specific submodules — existing
- ✓ Branch override via CLI flag — existing
- ✓ Fail-fast mode — existing
- ✓ Root repository status display (display-only, not modifiable) — existing
- ✓ TUI interactive selector with arrow key navigation — existing
- ✓ Logging to ~/.ssu/<project>/logs/ — existing

### Active

(Next milestone requirements will be defined here)

### Validated v1.0

All v1.0 requirements shipped:

- ✓ Subcommand-based CLI (`ssu status`, `ssu update`, `ssu push`, `ssu rollback`) — v1.0
- ✓ Backwards compatibility hints for old flag syntax (`--status` → suggests `ssu status`) — v1.0
- ✓ Polished TUI using bubbletea/charm library — v1.0
- ✓ YAML config file in `~/.ssu/config.yaml` (skip list, branch priority, parallel jobs, etc.) — v1.0
- ✓ Per-project config override (`.ssu.yaml` in project root) — v1.0
- ✓ Modular command architecture — easy to add new subcommands — v1.0
- ✓ Comprehensive test suite — v1.0 (100+ tests)
- ✓ Cross-platform binary distribution (goreleaser) — v1.0 (8 platforms)
- ✓ Homebrew tap — v1.0
- ✓ AUR package — v1.0 (config ready, deferred pending AUR registration availability)
- ✓ `go install` support — v1.0

### Out of Scope

- GUI/web interface — CLI tool, terminal is the interface
- Non-git VCS support — git-only tool
- Submodule creation/removal — only manages existing submodules
- Windows native support — Go binaries are provided but no Windows-specific testing or features

## Context

**Current State (v1.0 shipped 2026-02-10):**
- Complete Go rewrite: 10,909 lines of Go code across 140 files
- Tech stack: Go 1.21, cobra CLI framework, bubbletea TUI, viper config, lumberjack logging
- GitService abstraction enables 100+ unit tests without real git operations
- Cross-platform binaries via goreleaser (linux/darwin/freebsd/windows on amd64/arm64)
- GitHub Actions release workflow + Homebrew tap configured
- 8 phases completed over 7 days (2026-02-03 → 2026-02-10)

**Migration from bash version:**
- Bash version at v1.1.1 (950 lines) remains in `legacy/ssu` for reference
- Backwards compatibility hints guide users from old `--flags` to new `ssu subcommands`
- `~/.ssu/` directory convention preserved (backups and logs)
- JSON backup format compatible with bash-era backups for rollback

## Constraints

- **Language**: Go — chosen for cross-compilation, single binary distribution, and maintainability
- **TUI**: bubbletea/charm ecosystem — polished terminal UI library
- **Config**: YAML format in `~/.ssu/config.yaml` — human-readable, widely understood
- **Compatibility**: Must handle same git scenarios as bash version (detached HEAD, uninitialized submodules, merge conflicts, etc.)
- **Data dir**: `~/.ssu/` — preserve existing convention for backups and logs
- **Distribution**: go install, GitHub releases (goreleaser), Homebrew, AUR

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Go over Rust/Python | Single binary, fast compile, easy cross-platform, familiar ecosystem | ✓ Good — 10,909 LOC, compiles in <2s, 8 platform targets |
| Subcommand CLI over flags | More intuitive, extensible, standard pattern for CLI tools | ✓ Good — 11 subcommands, easy to extend, clean `ssu help` |
| bubbletea for TUI | Most popular Go TUI framework, great ecosystem (lipgloss, bubbles) | ✓ Good — Polished progress bars, multi-select, 11 keybindings |
| YAML config | Human-friendly, widely used, good Go library support | ✓ Good — 5-layer config with viper works perfectly |
| Preserve ~/.ssu/ convention | Existing users have backups there, no migration needed | ✓ Good — Bash-era backup compatibility confirmed |
| Backwards compat hints | Old --flag users get pointed to new subcommand, smooth migration | ✓ Good — compat.CheckOldFlags guides users to new syntax |
| GitService interface | Enable unit testing without real git | ✓ Good — 100+ tests via MockGitService, zero git calls in tests |
| errgroup for concurrency | Bounded parallelism with continue-on-error | ✓ Good — Clean concurrent scan, respects --jobs flag |

---
*Last updated: 2026-02-10 after v1.0 milestone*
