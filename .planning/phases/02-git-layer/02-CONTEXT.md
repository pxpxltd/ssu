# Phase 2: Git Layer - Context

**Gathered:** 2026-02-09
**Status:** Ready for planning

<domain>
## Phase Boundary

Testable git abstraction that handles all git operations SSU needs. Includes: GitService interface with mock for testing, os/exec real implementation with stderr capture, smart branch detection with configurable priority chain, context timeouts, multi-remote support, and push with automatic tracking branch setup. Does NOT include engine orchestration, config file loading, or TUI — those are later phases.

</domain>

<decisions>
## Implementation Decisions

### Interface design
- Method granularity: Claude's discretion — pick the right balance based on how Engine and Commands will call GitService
- Mock implementation: simple stub mock — preconfigured success/error responses, no scenario simulation
- Return types: structured result types (e.g., FetchResult, MergeResult with typed fields), not raw strings
- Stderr capture: always captured in result types (every result includes Stderr field), not just on error — enables verbose mode and logging

### Branch detection behavior
- Priority order: develop > master > main as default, but configurable via config file parameter (Phase 4 wires config, but detection code accepts priority list as input now)
- Detached HEAD: try to resolve to a branch by checking if SHA matches any branch tip; if no match, fall back to detached status
- Unreachable remote: fall back to local-only branch refs with a warning flag on the result — no hard failure
- Feature branches: match bash behavior exactly — if current branch is not in priority list and has a remote tracking branch, stay on it

### Timeout & error policy
- Per-operation timeouts: fetch, push, merge, etc. each have their own configurable timeout value
- Retry: configurable retry count (default 0), settable via config for flaky networks
- Error typing: Claude's discretion — pick the error design that works best with Go conventions for distinguishing conflict vs timeout vs auth errors
- Context propagation: caller passes context (standard Go pattern) — Engine creates context.WithTimeout and GitService respects it

### Push & tracking setup
- No tracking branch: auto `push -u origin <branch>` — same as bash version, no confirmation needed
- Remote selection: follow tracking branch remote (if branch tracks `upstream/develop`, push to `upstream`; fall back to `origin` if no tracking info)
- Force-push policy: GitService never force-pushes — if push rejected due to divergence, return error. Caller decides. Zero data loss principle.
- Push results: rich PushResult including remote, branch, commit count/range, and whether a new tracking branch was created

### Claude's Discretion
- GitService method granularity (one-per-command vs higher-level operations)
- Error type design (typed constants vs error struct with code)
- Internal implementation details of os/exec wrapping
- Default timeout values per operation

</decisions>

<specifics>
## Specific Ideas

- Branch priority must be accepted as a parameter by detection code now (Phase 4 wires the config file, but the interface should be ready)
- Bash version behavior is the reference implementation — feature branch detection, push -u, detached HEAD handling should match proven behavior
- Stderr always available enables verbose/debug logging in later phases without changing the interface

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-git-layer*
*Context gathered: 2026-02-09*
