# Phase 3: Engine - Context

**Gathered:** 2026-02-09
**Status:** Ready for planning

<domain>
## Phase Boundary

The core orchestrator that scans submodules in parallel, detects status, handles conflicts, and coordinates update/push workflows. This phase builds on the GitService from Phase 2 and produces the engine that Phase 5 commands will consume. It does NOT include TUI/display (Phase 5) or config loading (Phase 4) — the engine accepts config values as parameters.

</domain>

<decisions>
## Implementation Decisions

### Concurrency & scanning
- Default concurrency: `runtime.NumCPU()` (scale with machine, not hardcoded 8)
- Concurrency is configurable (engine accepts it as a parameter; Phase 4 wires config)
- On fetch failure: continue scanning all remaining submodules, collect failures, report at end (no fail-fast during scan)
- Engine exposes progress via callback/channel pattern — sends events per submodule (started, completed, failed) so Phase 5 TUI can subscribe for progress bars
- Submodule enumeration: auto-discover from git, then filter against a skip list (passed in as parameter)

### Conflict resolution behavior
- Keep bash version's 3-step strategy: detect merge failure → stash local changes → retry merge on clean state → reapply stash
- On failure after stash+retry: abort merge, restore stash, provide actionable git commands the user can copy-paste to resolve manually
- Track conflicting files — capture which specific files conflicted so hints can reference them (e.g., "Conflict in path/to/sub: file1.go, file2.go. Run: cd path/to/sub && git merge --abort && git stash pop")
- Changelog preview: one-line summaries per incoming commit (short hash + subject line, like `git log --oneline`)

### Status model & transitions
- 8 statuses: pending, current, modified, ahead, conflict, missing, skipped, error
- 'error' is new — covers submodules where scanning itself failed (network timeout, permission denied, etc.)
- Compound statuses allowed — a submodule can be both modified AND ahead (has uncommitted changes + unpushed commits). Not mutually exclusive
- Result struct per submodule includes: before status, after status, action taken (e.g., "was pending, merged 3 commits, now current"). Enables rich summary display in Phase 5

### Update/push orchestration
- Updates processed in parallel with bounded concurrency (same model as scanning)
- Default behavior: continue on error, report all results at end (no fail-fast for updates)
- Push and update are separate concerns but share the scan result

### Claude's Discretion
- Root repository handling approach (display-only, matching or improving on bash behavior)
- Dry-run implementation: whether engine handles it internally or caller handles it
- API shape for push vs update (separate methods vs unified entry point)
- errgroup vs custom worker pool for concurrency implementation
- Context timeout values for git operations

</decisions>

<specifics>
## Specific Ideas

- Progress events should be typed (scan progress vs update progress vs push progress) so the TUI can render different views
- Conflict hints should include exact `cd` + `git` commands — the user should be able to copy-paste directly from the error output
- The compound status model means the display layer needs to handle showing multiple indicators (Phase 5 concern, but engine must provide the data)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 03-engine*
*Context gathered: 2026-02-09*
