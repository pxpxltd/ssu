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

- [ ] Subcommand-based CLI (`ssu status`, `ssu update`, `ssu push`, `ssu rollback`)
- [ ] Backwards compatibility hints for old flag syntax (`--status` → suggests `ssu status`)
- [ ] Polished TUI using bubbletea/charm library
- [ ] YAML config file in `~/.ssu/config.yaml` (skip list, branch priority, parallel jobs, etc.)
- [ ] Per-project config override (`.ssu.yaml` in project root)
- [ ] Modular command architecture — easy to add new subcommands
- [ ] Comprehensive test suite
- [ ] Cross-platform binary distribution (goreleaser)
- [ ] Homebrew tap
- [ ] AUR package
- [ ] `go install` support

### Out of Scope

- GUI/web interface — CLI tool, terminal is the interface
- Non-git VCS support — git-only tool
- Submodule creation/removal — only manages existing submodules
- Windows native support — Linux/macOS only (WSL may work incidentally)

## Context

- The bash version is at v1.1.1 and feature-complete for its scope
- Primary pain points: hard to extend (950 lines of bash), no tests, Bash 3.2 compat hacks
- The `versions/go` branch exists for this rewrite
- Existing codebase map in `.planning/codebase/` documents the bash architecture in detail
- The `~/.ssu/` directory convention is established and should be preserved
- Users of the bash version exist and will migrate — old flag syntax should give helpful hints

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
| Go over Rust/Python | Single binary, fast compile, easy cross-platform, familiar ecosystem | — Pending |
| Subcommand CLI over flags | More intuitive, extensible, standard pattern for CLI tools | — Pending |
| bubbletea for TUI | Most popular Go TUI framework, great ecosystem (lipgloss, bubbles) | — Pending |
| YAML config | Human-friendly, widely used, good Go library support | — Pending |
| Preserve ~/.ssu/ convention | Existing users have backups there, no migration needed | — Pending |
| Backwards compat hints | Old --flag users get pointed to new subcommand, smooth migration | — Pending |

---
*Last updated: 2026-02-09 after initialization*
