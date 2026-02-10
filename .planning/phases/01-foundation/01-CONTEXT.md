# Phase 1: Foundation - Context

**Gathered:** 2026-02-09
**Status:** Ready for planning

<domain>
## Phase Boundary

A runnable `ssu` Go binary with cobra CLI skeleton, subcommand routing (status, update, push, rollback), version injection, shell completions, and correct terminal behavior (NO_COLOR, TTY detection). This phase establishes the project structure and CLI surface — no git operations or business logic yet.

</domain>

<decisions>
## Implementation Decisions

### Project layout
- Standard Go layout: `cmd/ssu/main.go` + `internal/` packages
- Module path: `github.com/pxpxltd/ssu`
- Move bash script to `legacy/ssu` — Go code owns the root
- Internal packages split by domain: `internal/cli`, `internal/git`, `internal/engine`, `internal/config`, `internal/backup`, `internal/tui`
- Minimum Go version: 1.21+ (enables slog, improved generics)
- Tests alongside source files: `service.go` + `service_test.go` in same package (standard Go convention)

### Command surface
- Root command (`ssu` with no subcommand): show interactive menu to pick an action
- Subcommands: `status`, `update`, `push`, `rollback`, `backup`, `version`
- Flags: both short and long forms for common flags (`-v/--verbose`, `-n/--dry-run`, `-a/--auto`, `-j/--jobs`)
- Context check: validate `.git` and `.gitmodules` on startup, show clear error if missing (e.g., "Not a git repository with submodules")
- Help text: terse descriptions in subcommand list, detailed examples in per-command `--help`

### Terminal output style
- Color scheme: evolve from bash version — Claude picks improved colors for better contrast/accessibility
- Verbosity: minimal by default (results and errors only), `-v` for progress details
- Error messages: actionable — always suggest what to do next (e.g., "Merge conflict in plugins/foo. Run `ssu rollback` to restore, or resolve manually.")
- Symbols: Unicode (`✓`, `✗`, `→`, `●`, `│`) — modern look

### Backwards compatibility
- Old bash-era flags (`--status`, `--push`, etc.): hint and redirect — print "Did you mean `ssu status`? Run `ssu help` for the new syntax." then exit
- Binary name: `ssu` (same as bash version, direct replacement)
- Data directory: `~/.ssu/<project>/` (same location as bash version for continuity)
- Backup compatibility: full — Go version reads bash-era JSON backups and can rollback from them

### Claude's Discretion
- Exact color palette choices (improved from bash version)
- Interactive menu implementation for root command
- Shell completion generation approach
- Exit code numbering scheme
- Internal package API design

</decisions>

<specifics>
## Specific Ideas

- Root command shows an interactive menu (not just help text) — should feel like a quick launcher for the most common actions
- Old flag handling should be friendly, not punishing — hint the new syntax, don't just error

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 01-foundation*
*Context gathered: 2026-02-09*
