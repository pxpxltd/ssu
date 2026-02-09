---
phase: 04-config-safety
plan: 01
subsystem: config
tags: [viper, yaml, config-layering, cobra, env-vars]

# Dependency graph
requires:
  - phase: 01-foundation
    provides: cobra CLI skeleton, root command, global flags, output utilities
provides:
  - Config struct with Git, Branches, Backup, Log sections
  - 5-layer config loading (defaults < global yaml < project yaml < env < flags)
  - AnnotatedConfig source tracking for config show
  - Context helpers for passing config to subcommands
  - ssu config show command with source annotations
affects: [04-02-backup, 04-03-logging, 05-commands-tui]

# Tech tracking
tech-stack:
  added: [spf13/viper v1.20.1]
  patterns: [PersistentPreRunE config loading, cmd.Flags().Changed() pattern, context.Value config passing]

key-files:
  created:
    - internal/config/config.go
    - internal/config/source.go
    - internal/config/context.go
    - internal/config/config_test.go
    - internal/cli/config.go
  modified:
    - internal/cli/root.go
    - internal/cli/root_test.go
    - go.mod
    - go.sum

key-decisions:
  - "Viper instance per Load() call (not global) for testability"
  - "Config errors are warnings not fatal -- version/completion/help work without git repo"
  - "cmd.Flags().Changed() pattern for flag overrides (avoids Viper BindPFlags pitfall)"
  - "Legacy PARALLEL_JOBS env var supported via explicit os.Getenv check (not Viper binding)"

patterns-established:
  - "Config in context: config.WithConfig/FromContext for subcommand access"
  - "Source annotation: AnnotatedConfig tracks which layer set each value"
  - "Graceful degradation: config load failure falls back to defaults with stderr warning"

# Metrics
duration: 5min
completed: 2026-02-09
---

# Phase 4 Plan 1: Config Loading Summary

**Viper-based 5-layer YAML config with defaults/global/project/env/flag priority, source annotations, and ssu config show command**

## Performance

- **Duration:** 5 min
- **Started:** 2026-02-09T12:57:15Z
- **Completed:** 2026-02-09T13:02:29Z
- **Tasks:** 2
- **Files modified:** 9

## Accomplishments
- Config struct with 4 sections (Git, Branches, Backup, Log) and all fields from CONTEXT.md
- 5-layer priority chain: defaults < ~/.ssu/config.yaml < .ssu.yaml < SSU_ env vars < CLI flags
- Legacy PARALLEL_JOBS env var silently supported with SSU_ prefix taking priority
- AnnotatedConfig tracks source for every value, displayed via `ssu config show`
- PersistentPreRunE loads config for all subcommands without breaking version/completion/help

## Task Commits

Each task was committed atomically:

1. **Task 1: Config package with Viper layered loading** - `940da03` (feat)
2. **Task 2: Root PersistentPreRunE and config show command** - `e8cc986` (feat)

## Files Created/Modified
- `internal/config/config.go` - Config struct, Load() with 5-layer priority, Defaults()
- `internal/config/source.go` - Source type, AnnotatedValue, AnnotatedConfig for tracking
- `internal/config/context.go` - WithConfig/FromContext, WithAnnotated/AnnotatedFromContext
- `internal/config/config_test.go` - 12 tests covering defaults, project config, env vars, legacy, layering
- `internal/cli/config.go` - NewConfigCmd, NewConfigShowCmd with formatted source output
- `internal/cli/root.go` - PersistentPreRunE calling loadConfig, detectProjectRoot
- `internal/cli/root_test.go` - Updated help test to include "config" subcommand
- `go.mod` - Added spf13/viper v1.20.1 direct dependency
- `go.sum` - Updated checksums

## Decisions Made
- Used Viper instance (`viper.New()`) not global Viper for testability -- each Load() call is isolated
- Config errors produce stderr warning and fall back to defaults -- ssu never crashes on bad config
- Used `cmd.Flags().Changed("jobs")` pattern instead of BindPFlags to avoid the Viper pitfall where flag defaults override config file values
- Legacy PARALLEL_JOBS env var handled via explicit os.Getenv check rather than Viper binding, ensuring SSU_ prefix always takes priority
- Viper v1.20.1 pinned (v1.21.0+ requires Go 1.23, project uses Go 1.21)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Config package ready for backup subsystem (04-02) to read BackupConfig
- Config package ready for logging subsystem (04-03) to read LogConfig
- PersistentPreRunE pattern established for adding verbose/logging init later
- No blockers

---
*Phase: 04-config-safety*
*Completed: 2026-02-09*
