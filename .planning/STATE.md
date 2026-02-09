# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-09)

**Core value:** Safely update and push git submodules with zero data loss -- smart branch detection, automatic backups, conflict resolution.
**Current focus:** Phase 5.1 - Claude Code Integration -- COMPLETE. Next: Phase 6 (Distribution)

## Current Position

Phase: 5.1 of 6 (Claude Code Integration)
Plan: 2 of 2 in current phase -- COMPLETE
Status: Phase complete
Last activity: 2026-02-09 -- Completed 05.1-02-PLAN.md

Progress: [##################] 94% (17/18 plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 17
- Average duration: 3.5min
- Total execution time: 65min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation | 2/2 | 7min | 3.5min |
| 02-git-layer | 3/3 | 9min | 3min |
| 03-engine | 3/3 | 9min | 3min |
| 04-config-safety | 3/3 | 12min | 4min |
| 05-commands-tui | 4/4 | 26min | 6.5min |
| 05.1-claude-code | 2/2 | 4min | 2min |

**Recent Trend:**
- Last 5 plans: 6min, 6min, 6min, 2min, 2min
- Trend: fast (lightweight CLI wiring, no new deps)

*Updated after each plan completion*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Use os/exec to shell out to git (not go-git) for 100% compatibility
- [Roadmap]: Phases 3 and 4 can run in parallel (no dependency between them)
- [Roadmap]: 6 phases, 18 plans total at standard depth
- [01-01]: Used mattn/go-isatty for IsTTY() instead of just color.NoColor
- [01-01]: Interactive root menu uses simple numbered list -- Bubble Tea replaces in Phase 5
- [01-01]: Added .gitignore for build artifact (deviation from plan)
- [01-02]: Compat check runs before cobra dispatch -- intercepts old flags before cobra parses them
- [01-02]: Only args[1] checked for old flags -- avoids false positives for valid subcommand flags
- [01-02]: Completion uses os.Stdout directly per cobra API
- [02-01]: 19 methods on GitService -- one per semantic operation, not per raw command
- [02-01]: RemoteBranch has no Stderr (data type, not operation result)
- [02-01]: IsSubmoduleInitialized has no context param (local filesystem check)
- [02-02]: DetectBestBranch is a standalone function, not a method on any struct
- [02-02]: Remote errors degrade gracefully (non-fatal) at every priority level
- [02-02]: Priority branch matching checks ALL remotes, not just DefaultRemote
- [02-03]: Merge conflict detection checks both stdout and stderr (git writes CONFLICT to stdout)
- [02-03]: CommitsBehind/CommitsAhead return 0 on error (matching bash behavior)
- [02-03]: Push auto-detects missing tracking branch and uses -u flag
- [03-01]: x/sync pinned to v0.7.0 (last Go 1.21 compatible; v0.19.0 requires Go 1.24)
- [03-01]: Zero-value errgroup (not WithContext) for continue-on-error scan semantics
- [03-01]: Root scanned in same parallel batch as submodules, separated in results
- [03-01]: Status priority map for PrimaryStatus display ordering
- [03-02]: Update accepts []*SubmoduleInfo targets (caller decides what to update, not engine)
- [03-02]: Dirty path uses stash -> merge -> stash-pop with abort+restore on failure
- [03-02]: ConflictHint contains relative path (info.Path) not absolute path
- [03-03]: Push delegates tracking branch detection to GitService.Push (no logic duplication)
- [03-03]: Detached HEAD returns PushAction with no error (skip, not failure)
- [04-01]: Viper instance per Load() call (not global) for testability
- [04-01]: Config errors are warnings not fatal -- version/completion/help work without git repo
- [04-01]: cmd.Flags().Changed() pattern for flag overrides (avoids Viper BindPFlags pitfall)
- [04-01]: Legacy PARALLEL_JOBS env var via explicit os.Getenv (SSU_ prefix always wins)
- [04-02]: Rollback uses injected function callbacks instead of importing git package directly
- [04-02]: Bash-era backups are discovered but never auto-deleted by clean command
- [04-02]: Go-era backup filenames have no dot prefix (backup-*.json vs .submodule-backup-*.json)
- [04-02]: Safety backup created automatically before any rollback restore operation
- [04-03]: BracketHandler uses slog level strings as-is (WARN not WARNING)
- [04-03]: Logger failure is non-fatal -- stderr warning, command continues
- [04-03]: version/completion skip logger init (lightweight utility commands)
- [04-03]: slog.SetDefault() for global access (no context threading needed)
- [05-01]: Pin lipgloss v1.0.0 (not v1.1.0) to avoid Go version risk
- [05-01]: Feature branch detection inline in status command (SubmoduleInfo has no IsFeature field)
- [05-01]: Table width hardcoded to 120 columns (avoids x/term dep)
- [05-01]: Mode branching pattern: --json flag checked in RunE, shared engine call
- [05-02]: Confirmation is a state within SelectorModel, not a separate bubbletea model
- [05-02]: Selections tracked by original index to persist across filter changes
- [05-02]: ProgressModel.Result() returns interface{} to avoid circular tui/engine import
- [05-03]: exitError type centralized in exitcodes.go (shared between update+push commands)
- [05-03]: Backup created before every update (auto and interactive), non-fatal on failure
- [05-03]: resolveRef uses os/exec directly for rev-parse (ExecGit.run is unexported)
- [05-03]: Non-TTY falls back to auto mode (same code path as --auto flag)
- [05-03]: Dynamic total in ProgressModel (starts at 0, updated via FetchProgressMsg.Total)
- [05-04]: exec runs sequentially (not parallel) for readable output ordering
- [05-04]: init uses bufio.Scanner prompts, not bubbletea (simple sequential flow)
- [05-04]: Interactive menu now has 7 items (exec added between push and rollback)
- [05.1-01]: Standard Go testing (not testify) to match project conventions
- [05.1-02]: Import alias claudepkg to avoid naming collision with cobra command variable

### Roadmap Evolution

- Phase 5.1 inserted after Phase 5: Claude Code Integration (URGENT) -- slash commands + CLAUDE.md snippet for AI-assisted submodule management

### Pending Todos

None.

### Blockers/Concerns

None.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 001 | Remove Target column from status table + add progress bar | 2026-02-09 | 18150b4 | [001-remove-target-column](./quick/001-remove-target-column-from-status-table/) |
| 002 | Green progress bar + informative selector header | 2026-02-09 | 324e153 | [002-green-progress-bar](./quick/002-green-progress-bar-and-selector-header/) |

## Session Continuity

Last session: 2026-02-09
Stopped at: Completed quick tasks 001-002 (status table + TUI polish)
Resume file: None
