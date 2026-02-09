# Phase 5: Commands + TUI - Context

**Gathered:** 2026-02-09
**Status:** Ready for planning

<domain>
## Phase Boundary

Fully functional interactive CLI where users scan, select, update, push, and rollback submodules through a polished bubbletea TUI. Commands: `ssu status`, `ssu update`, `ssu push`, `ssu exec`, with `--auto`, `--dry-run`, `--json`, and `--verbose` modes. Init wizard for first-run setup. Ctrl+C handling with clean terminal restoration.

</domain>

<decisions>
## Implementation Decisions

### TUI Selector Design
- Keep bash-era keybindings: arrow keys + j/k for navigation, space to toggle, a/A for all/none, enter to confirm, q to quit
- Add / key for filtering/searching submodules by name (narrows visible list)
- Add ? key to show/hide keybinding help line (hidden by default)
- Alt-screen vs inline rendering: Claude's discretion
- Rich metadata per item: each line shows path, current branch, behind count, and status badge
- Confirmation step before action: after pressing enter, show summary of selected items with y/N prompt
- Changelog detail pane: vertical split (list on left, changelog on right)
- Detail pane visible by default, toggled with a key to hide/show

### Status Table Presentation
- Keep all columns from bash version: Path, Current Branch, Target Branch, Behind, Feature, Status
- Root repository row: bold text + separator line before submodules begin
- Default sort: alphabetical by path; sortable by key press or config option
- `--json` output: structured object with root, submodules array, and scanned_at timestamp

### Command Behavior & Modes
- `--auto` mode: summary output by default, `--verbose` flag enables streaming per-submodule results (CI-friendly)
- `--dry-run`: full diff table showing path, current SHA, target SHA, and commits behind
- Partial failure: non-zero exit code if ANY submodule fails
- `ssu exec`: runs arbitrary command in submodules with TUI selector to choose which submodules (foreach + selection)

### Progress & Feedback
- Parallel fetch: single progress bar with count + currently-fetching submodule name (`[========>    ] 12/25 fetching plugins/auth`)
- Update/push processing: stream results as each submodule completes (checkmark/cross + path + outcome)
- Ctrl+C: show partial results ("Cancelled. 8/15 submodules updated before interruption:" + result list)
- Final summary: banner with counts ("12 updated, 2 skipped, 1 conflict") -- no next-step hints
- Conflict messages: include actionable hints with specific git commands to resolve
- Non-TTY: downgrade progress to simple log lines ("Fetching 1/25..."), no bars or spinners

### Claude's Discretion
- Alt-screen vs inline rendering for TUI selector
- Exact key for toggling detail pane (tab, p, or similar)
- Exact key for changing sort order in status table
- Progress bar visual style (characters, colors)
- Init wizard flow and prompts

</decisions>

<specifics>
## Specific Ideas

- Detail pane layout inspired by split-pane terminal tools -- list on left, changelog for highlighted submodule on right
- Filtering with / should feel like vim's search -- type to narrow, escape to clear
- Help line on ? should show all available keys in a compact format at the bottom
- Progress bar should show the name of what's being fetched, not just a count -- gives the user a sense of activity

</specifics>

<deferred>
## Deferred Ideas

None -- discussion stayed within phase scope

</deferred>

---

*Phase: 05-commands-tui*
*Context gathered: 2026-02-09*
