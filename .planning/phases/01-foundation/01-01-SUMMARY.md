---
phase: 01-foundation
plan: 01
subsystem: cli
tags: [cobra, fatih-color, go-module, ldflags, cli-skeleton]

# Dependency graph
requires: []
provides:
  - Go module at github.com/pxpxltd/ssu with cobra and fatih/color
  - Cobra root command with global flags (verbose, dry-run, auto, jobs)
  - 5 subcommand stubs (status, update, push, rollback, backup)
  - Output package with color definitions, symbols, and printer
  - Makefile with ldflags version injection
  - Legacy bash script preserved at legacy/ssu
affects: [01-02, 02-scanning, 03-update-engine, 04-backup, 05-tui, 06-polish]

# Tech tracking
tech-stack:
  added: [cobra v1.10.2, fatih/color v1.18.0, go-isatty v0.0.20 (transitive), pflag v1.0.9 (transitive)]
  patterns: [one-file-per-command, NewXxxCmd() factory pattern, ldflags + debug.ReadBuildInfo version fallback, centralized color definitions]

key-files:
  created:
    - cmd/ssu/main.go
    - internal/cli/root.go
    - internal/cli/status.go
    - internal/cli/update.go
    - internal/cli/push.go
    - internal/cli/rollback.go
    - internal/cli/backup.go
    - internal/cli/output/color.go
    - internal/cli/output/symbols.go
    - internal/cli/output/printer.go
    - Makefile
    - .gitignore
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "Used mattn/go-isatty for IsTTY() instead of just color.NoColor -- pipe detection works independently of NO_COLOR"
  - "Interactive root menu uses simple numbered list with bufio.Scanner -- Bubble Tea replaces in Phase 5"
  - "Added .gitignore for build artifact (ssu binary) -- deviation from plan, necessary for clean repo"

patterns-established:
  - "One file per cobra command: NewXxxCmd() returns *cobra.Command"
  - "Output package centralizes all color/symbol definitions"
  - "Printer struct wraps io.Writer for testable formatted output"
  - "SilenceUsage: true + SilenceErrors: true on root command"
  - "Version vars in main with ldflags injection + debug.ReadBuildInfo fallback"

# Metrics
duration: 4min
completed: 2026-02-09
---

# Phase 1 Plan 1: Go Project Init and CLI Skeleton Summary

**Cobra CLI skeleton with 5 subcommand stubs, color-aware output package, and Makefile with ldflags version injection**

## Performance

- **Duration:** 4 min
- **Started:** 2026-02-09T10:54:24Z
- **Completed:** 2026-02-09T10:58:15Z
- **Tasks:** 2
- **Files modified:** 17

## Accomplishments
- Initialized Go module with cobra v1.10.2 and fatih/color v1.18.0 dependencies
- Built complete CLI skeleton with root command, global flags, and 5 subcommand stubs
- Created output utilities package (color definitions, Unicode symbols, Printer)
- Moved bash script to legacy/ssu preserving git history
- Built Makefile with ldflags version injection (build, test, lint, clean targets)
- Added interactive numbered menu for TTY root invocation

## Task Commits

Each task was committed atomically:

1. **Task 1: Go module init, dependencies, project structure, and Makefile** - `b325cc1` (chore)
2. **Task 2: Output utilities (color, symbols, printer) and cobra CLI skeleton** - `f96cca9` (feat)

## Files Created/Modified
- `go.mod` - Go module definition at github.com/pxpxltd/ssu with Go 1.21+
- `go.sum` - Dependency checksums
- `Makefile` - Build targets with ldflags version injection
- `legacy/ssu` - Original bash script (moved from root)
- `cmd/ssu/main.go` - Entry point with version vars and debug.ReadBuildInfo fallback
- `internal/cli/root.go` - Root cobra command with global flags and interactive menu
- `internal/cli/status.go` - Status subcommand stub with --json flag
- `internal/cli/update.go` - Update subcommand stub
- `internal/cli/push.go` - Push subcommand stub
- `internal/cli/rollback.go` - Rollback subcommand stub with optional positional arg
- `internal/cli/backup.go` - Backup subcommand stub
- `internal/cli/output/color.go` - Color definitions and NO_COLOR/TTY detection
- `internal/cli/output/symbols.go` - Unicode symbol constants (Check, Cross, Arrow, etc.)
- `internal/cli/output/printer.go` - Formatted output helpers (Printer struct)
- `.gitignore` - Excludes build artifact and IDE files

## Decisions Made
- Used `mattn/go-isatty` directly for `IsTTY()` rather than relying on `color.NoColor` -- a piped process with NO_COLOR unset should still detect non-TTY correctly
- Interactive root menu implemented as simple numbered list with `bufio.Scanner` -- keeps Phase 1 lightweight, Bubble Tea replaces it in Phase 5
- Root command dispatches menu choice by calling `cmd.SetArgs()` + `cmd.Execute()` for clean subcommand routing

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added .gitignore for build artifact**
- **Found during:** Task 2 (after building binary)
- **Issue:** `make build` produces `ssu` binary at project root, which would show in git status and could be accidentally committed
- **Fix:** Created `.gitignore` with `/ssu` entry plus IDE and OS file exclusions
- **Files modified:** `.gitignore` (new)
- **Verification:** `git status` no longer shows `ssu` as untracked
- **Committed in:** `f96cca9` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary for clean repository state. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CLI skeleton complete and compiling -- Plan 02 (version, completion, compat) can proceed immediately
- All 5 subcommand stubs ready for implementation in later phases
- Output package ready for use by all future command implementations
- Makefile provides build/test/lint workflow for CI

---
*Phase: 01-foundation*
*Completed: 2026-02-09*
