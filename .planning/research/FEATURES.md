# Feature Landscape

**Domain:** Git submodule management CLI tool (Go rewrite of bash SSU)
**Researched:** 2026-02-09
**Overall Confidence:** HIGH (grounded in existing bash codebase + established Go CLI patterns)

## Table Stakes

### Parity Features (Must Match Bash v1.1.1)

Missing any of these = migration blocker for existing users.

| Feature | Complexity | Notes |
|---------|------------|-------|
| Status overview table | Medium | Colorized, column-aligned, status-coded. Root repo display. |
| Parallel fetch | Medium | Go goroutines + errgroup. Add per-fetch timeout (bash has none). |
| Smart branch detection | Medium | develop > master > main priority. Feature branch preservation. Unit test every path. |
| Interactive TUI selector | High | bubbletea. Arrow/vim keys, space toggle, all/none, confirm/quit. |
| Auto/batch mode | Low | `ssu update --all` or `--auto` for CI/CD. |
| Dry-run preview | Low | Show incoming commits without modifying. |
| Backup before modification | Medium | JSON backup of SHAs to ~/.ssu/<project>/. Atomic writes. |
| Rollback from backup | Medium | Parse backup JSON, checkout exact SHAs. Validate format. |
| Conflict handling (stash/retry) | Medium | Stash, retry merge, reapply. Handle stash pop failures. |
| Push mode | Medium | Detect ahead, set up tracking branch, push. |
| Skip list | Low | Configurable paths to exclude. |
| Branch override | Low | `--branch` flag. |
| Fail-fast mode | Low | `--fail-fast` flag. |
| Feature branch detection | Low | Visual indicator in status table. |
| Logging | Low | Write to ~/.ssu/<project>/logs/. Add rotation. |
| Root repository display | Low | Display-only, never modify root. |
| Changelog preview | Low | Show incoming commits before updating. |

### Go CLI Table Stakes (Expected of Modern CLI Tools)

| Feature | Complexity | Notes |
|---------|------------|-------|
| Subcommand CLI | Medium | `ssu status`, `ssu update`, `ssu push`, `ssu rollback` |
| `--help` at every level | Low | Free with cobra |
| Shell completions | Low | Cobra built-in: bash/zsh/fish |
| Version command | Low | `ssu version` with build info via ldflags |
| `NO_COLOR` support | Low | Respect NO_COLOR env var, auto-detect TTY |
| `--json` output | Medium | Machine-readable output on `ssu status` |
| Config file support | Medium | ~/.ssu/config.yaml + .ssu.yaml per-project |
| Meaningful exit codes | Low | 0=success, 1=error, 2=conflict |
| Progress during fetch | Medium | bubbletea spinner/progress bar |
| Graceful Ctrl+C | Low | Clean terminal state, show partial results |

## Differentiators

### High-Value

| Feature | Value | Complexity |
|---------|-------|------------|
| Progress bar during parallel fetch | Real-time per-submodule progress ("Fetching [12/34]") | Medium |
| Backwards compatibility hints | `ssu --status` → suggests `ssu status` | Low |
| Per-project config override | `.ssu.yaml` in project root | Medium |
| Log rotation and cleanup | Automatic rotation by size/date | Medium |
| Fetch timeout per submodule | Kill hung fetch after configurable timeout | Medium |
| Error aggregation summary | Grouped error report after processing | Low |
| Atomic backup writes | Write temp file, rename (prevents corruption) | Low |
| Backup listing/management | `ssu backup list`, `ssu backup clean --keep 5` | Low |
| Multiple remote support | Don't hardcode "origin" | Medium |

### Medium-Value (Post-MVP)

| Feature | Value | Complexity |
|---------|-------|------------|
| `ssu init` wizard | Interactive first-time setup | Medium |
| `ssu diff` command | Show changes since last update/backup | Medium |
| `ssu exec <command>` | Run command across submodules with selection UI | Medium |
| Submodule grouping | Group by category, `ssu update --group plugins` | Medium |
| Dashboard view | Rich TUI beyond table | High |

## Anti-Features (Do NOT Build)

| Anti-Feature | Why Avoid |
|--------------|-----------|
| Submodule creation/removal | Out of scope. `git submodule add/deinit` works fine. |
| GUI/web interface | CLI tool stays in terminal. Invest in TUI. |
| Non-git VCS | Git submodules are a git concept. |
| Windows native | WSL exists. Target audience has Unix-like env. |
| Root repo modification | Display-only for safety. Users manage root manually. |
| Auto-commit after update | Presumptuous. Show hint instead. |
| Plugin system | Enormous complexity. Tool does one thing well. |
| GitHub/GitLab API | Stay at git protocol level. |
| Recursive submodules | Rare, exponential complexity. Document limitation. |
| Interactive conflict resolution | Building a merge tool is massive. Use vimdiff/meld. |

## Feature Dependencies

```
Config → Skip List, Branch Priority, Per-project Override, Fetch Timeout
Parallel Fetch → Status Detection → Status Table, Selector, Dry-run, Progress
Smart Branch Detection → Feature Branch Detection, Update Processing
Backup System → Rollback, Backup Management
Subcommand CLI (cobra) → Shell Completions, Help, Version, Compat Hints
TUI (bubbletea) → Interactive Selector, Progress, Dashboard
```

## MVP Recommendation

Full parity with bash v1.1.1 + Go CLI table stakes. Priority order:

1. Core git operations (scan, fetch, detect, branch detection)
2. Subcommand CLI with cobra
3. Status table output (colorized, root repo)
4. Config file (skip list, branch priority, parallel jobs)
5. Backup/rollback (atomic writes, validation)
6. Interactive TUI selector (bubbletea + progress)
7. Conflict handling (stash/retry + better errors)
8. Push mode (detect ahead, select, push)
9. Backwards compat hints
10. `--json` output on status

## Competitive Landscape

| Tool | Type | SSU's Advantage |
|------|------|----------------|
| `git submodule foreach` | Built-in | No smart branch detection, no status overview, no backup |
| `git submodule update --remote` | Built-in | No selective update, no conflict handling, no backup |
| `myrepos (mr)` | Multi-repo | Not submodule-specific, less smart branch detection |
| `repo` (Android) | Multi-repo | Designed for Android-scale, overkill |

**Tools to learn from:** lazygit (TUI excellence), gh (subcommand structure, --json), gum (bubbletea patterns), hugo (goreleaser distribution)

---
*Features research: 2026-02-09*
