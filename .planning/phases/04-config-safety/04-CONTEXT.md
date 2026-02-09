# Phase 4: Config + Safety - Context

**Gathered:** 2026-02-09
**Status:** Ready for planning

<domain>
## Phase Boundary

Layered YAML configuration, reliable backup/rollback, and structured logging. Users configure SSU behavior through config files and env vars, get automatic backups before destructive operations, can rollback to previous states, and have persistent logs for debugging. This phase builds the safety infrastructure that the engine (Phase 3) and commands (Phase 5) depend on.

</domain>

<decisions>
## Implementation Decisions

### Config file design
- Nested group structure: `git.parallel_jobs`, `git.skip`, `branches.priority` etc.
- Config layering: defaults < ~/.ssu/config.yaml < .ssu.yaml < env vars < CLI flags
- Env vars use `SSU_` prefix as canonical form (e.g., `SSU_GIT_PARALLEL_JOBS`)
- Legacy unprefixed env vars (e.g., `PARALLEL_JOBS`) silently supported but undocumented
- Config is opt-in only — no file needed, defaults just work. No `ssu init` generates config.
- `ssu config show` command displays merged config with source annotations (which layer each value came from)

### Backup behavior
- Backups stored in `~/.ssu/<project>/backups/` (restructured from bash-era `~/.ssu/<project>/`)
- Auto-backup before update operations only (not push — push is low-risk)
- Must read bash-era `.submodule-backup-*.json` files — seamless migration from bash version
- `ssu backup clean` supports both count-based (`--keep 5`) and time-based (`--keep 7d`) — detected by suffix

### Rollback experience
- `ssu rollback` with no args shows interactive picker of recent backups with timestamps
- Rollback restores both SHA and branch checkout (not just SHA — avoids detached HEAD)
- All-or-nothing restore — no partial/selective submodule rollback
- Rollback auto-creates a backup of current state before restoring (safety net — can undo a rollback)

### Log output & rotation
- Human-readable format: `[2024-01-15 10:30:00] [INFO] message` — matches bash behavior
- Size-based rotation (e.g., 10MB per file)
- Keep current + 5 rotated log files by default
- `-v` flag prints debug-level output to stderr in real time; logs always go to file regardless

### Claude's Discretion
- Exact YAML key names and nesting depth
- Viper binding patterns for config layering
- Atomic write implementation details (temp file + rename)
- slog handler configuration
- Rotation library choice (lumberjack or custom)
- Backup JSON format evolution (as long as bash-era format is readable)

</decisions>

<specifics>
## Specific Ideas

- Config should feel opt-in and invisible until needed — zero config works out of the box
- `ssu config show` with source annotations helps users debug "why is this value set?"
- Backup clean with suffix detection (`5` = count, `7d` = time) is a nice ergonomic touch
- Rollback creating its own backup before restoring means the user can never lose state

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-config-safety*
*Context gathered: 2026-02-09*
