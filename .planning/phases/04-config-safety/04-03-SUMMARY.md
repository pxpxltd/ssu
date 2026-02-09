---
phase: 04-config-safety
plan: 03
subsystem: logging
tags: [slog, lumberjack, log-rotation, bracket-format]

requires:
  - phase: 04-01
    provides: Config struct with LogConfig (MaxSizeMB, MaxBackups), PersistentPreRunE in root.go
provides:
  - BracketHandler slog.Handler with bash-compatible log format
  - MultiHandler for fan-out to file + stderr
  - InitLogger with lumberjack rotation
  - LogDir helper for standard log path
  - Logger wired into root PersistentPreRunE
affects: [05-commands-tui, 06-polish]

tech-stack:
  added: [gopkg.in/natefinch/lumberjack.v2]
  patterns: [slog.Handler interface, MultiHandler fan-out, non-fatal logger init]

key-files:
  created:
    - internal/logging/handler.go
    - internal/logging/logging.go
    - internal/logging/logging_test.go
  modified:
    - internal/cli/root.go
    - go.mod
    - go.sum

key-decisions:
  - "BracketHandler uses slog level strings as-is (WARN not WARNING) -- slog standard wins over bash compat"
  - "Logger failure is non-fatal -- stderr warning, command continues without file logging"
  - "version/completion commands skip logger init (lightweight utility commands)"
  - "slog.SetDefault() makes logger available to all packages without context threading"

patterns-established:
  - "Non-fatal infrastructure: logging/backup failures warn but don't block operations"
  - "Command exclusion via switch on cmd.Name() for lightweight utility commands"

duration: 3min
completed: 2026-02-09
---

# Phase 4 Plan 3: Logging Summary

**BracketHandler slog.Handler with bash-era format, lumberjack rotation, and verbose stderr via MultiHandler fan-out**

## Performance

- **Duration:** 3 min
- **Started:** 2026-02-09T13:07:03Z
- **Completed:** 2026-02-09T13:09:37Z
- **Tasks:** 2
- **Files modified:** 6

## Accomplishments
- BracketHandler produces exact `[YYYY-MM-DD HH:MM:SS] [LEVEL] message` format matching bash SSU
- MultiHandler fans out to file (INFO+) and stderr (DEBUG+) in verbose mode
- InitLogger sets up lumberjack rotation (10MB default, 5 backups default)
- Logger wired into root.go PersistentPreRunE; version/completion skip logging
- 14 tests covering format, levels, routing, file creation, and handler types

## Task Commits

Each task was committed atomically:

1. **Task 1: BracketHandler and InitLogger with lumberjack rotation** - `b2e1ced` (feat)
2. **Task 2: Wire logger initialization into root PersistentPreRunE** - `be35700` (feat)

## Files Created/Modified
- `internal/logging/handler.go` - BracketHandler implementing slog.Handler with bracket format
- `internal/logging/logging.go` - MultiHandler, InitLogger with lumberjack, LogDir helper
- `internal/logging/logging_test.go` - 14 tests for handler format, levels, routing, file creation
- `internal/cli/root.go` - Added initLogger call in loadConfig, skip for version/completion
- `go.mod` - Added gopkg.in/natefinch/lumberjack.v2 dependency
- `go.sum` - Updated checksums

## Decisions Made
- BracketHandler uses slog level strings as-is (WARN not WARNING) -- slog standard is close enough to bash era and avoids special-case mapping
- Logger failure is non-fatal -- a warning is printed to stderr and the command continues. SSU must work even if ~/.ssu directory cannot be created
- version and completion commands skip logger initialization entirely (no disk I/O for utility commands)
- slog.SetDefault() is used rather than context-based logger threading -- simpler, and all packages can use slog.Info() etc. directly

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- All Phase 4 infrastructure complete (config, backup, logging)
- Phase 5 (Commands + TUI) can wire real workflows using config, backup, and logging
- slog.Info/Debug/Warn/Error available throughout codebase for operational visibility

---
*Phase: 04-config-safety*
*Completed: 2026-02-09*
