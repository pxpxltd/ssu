# Architecture

**Analysis Date:** 2026-02-09

## Pattern Overview

**Overall:** Three-Phase Scan-Display-Process Pipeline

**Key Characteristics:**
- Single-file executable bash script with no external dependencies
- Parallel-first scanning approach with sequential processing fallback
- State management via parallel indexed arrays (Bash 3.2 compatible)
- TUI (Terminal User Interface) mode with fallback to non-interactive
- Backup-before-modify safety pattern with JSON-based rollback

## Layers

**Phase 0: Root Scanning Layer**
- Purpose: Scan and display root repository status (display-only, excluded from modifications)
- Location: `ssu` lines 1005-1019 (root repository scanning within `scan_submodules()`)
- Contains: Git status checking, branch detection, commit counting
- Depends on: Git commands, environment state
- Used by: Status display layer, summary generation

**Phase 1: Quick Local Checks Layer**
- Purpose: Identify skipped/missing submodules without network calls
- Location: `ssu` lines 1025-1038 (`scan_submodules()` phase 1)
- Contains: Skip list validation, submodule initialization checks
- Depends on: Filesystem state (`.git` file/directory presence)
- Used by: Parallel fetch layer to build work list

**Phase 2: Parallel Fetch Layer**
- Purpose: Fetch all remote refs concurrently to populate local cache
- Location: `ssu` lines 1040-1061 (`scan_submodules()` phase 2)
- Contains: Background job spawning, semaphore-based concurrency control
- Depends on: Network access, `git fetch --all`
- Used by: Status detection layer (all subsequent analysis uses cached refs)
- Configuration: `PARALLEL_JOBS` env var (default 8, max 16 recommended)

**Phase 3: Status Analysis Layer**
- Purpose: Analyze branch state, detect status, determine updates needed
- Location: `ssu` lines 1063-1102 (`scan_submodules()` phase 3)
- Contains: Branch detection, status classification, changelog generation
- Depends on: Cached remote refs (no network), helper functions
- Used by: Display layer, selection layer

**Display & Interaction Layer**
- Purpose: Show status table and collect user selections
- Location: `ssu` lines 1166-1343 (`display_status_table()`, `tui_select()`, `prompt_selection()`, `prompt_push_selection()`)
- Contains: Colorized table rendering, TUI selector with arrow key navigation, fallback prompts
- Depends on: Terminal control functions, TTY detection
- Used by: Main workflow control

**Processing Layer**
- Purpose: Execute updates or pushes based on selections
- Location: `ssu` lines 1369-1527 (`process_updates()`, `process_pushes()`)
- Contains: Conflict handling, push logic, backup creation
- Depends on: Update/push functions, error handling
- Used by: Main workflow

## Data Flow

**Update Workflow:**

1. **Parse CLI Args** → Global state variables (`MODE`, `DRY_RUN`, `PUSH_MODE`, `OVERRIDE_BRANCH`)
2. **Scan Phase 0 (Root)** → Populate indexed arrays with root data (key: ".")
3. **Scan Phase 1 (Quick Local)** → Filter to initialized paths only
4. **Scan Phase 2 (Parallel Fetch)** → Background jobs fetch all remote refs
5. **Scan Phase 3 (Analysis)** → Populate indexed arrays: `SUBMODULE_PATHS`, `SUBMODULE_STATUS`, `SUBMODULE_BRANCH`, `SUBMODULE_CURRENT_BRANCH`, `SUBMODULE_BEHIND`, `SUBMODULE_CHANGELOG`
6. **Display Table** → Show status using indexed array lookups
7. **Prompt Selection** (interactive) → User selects via TUI, marks as "selected" in `SUBMODULE_STATUS`
8. **Auto Selection** (auto mode) → All "pending" statuses converted to "selected"
9. **Create Backup** → JSON serialization of all submodule SHAs (excludes root)
10. **Process Updates** → Iterate submodules, update if "selected", handle conflicts with stash/retry
11. **Print Summary** → Display counts and instructions

**Push Workflow:**

1. Same scan phases as update workflow
2. Status analysis identifies "ahead" submodules (unpushed commits)
3. **Prompt Push Selection** → User selects which "ahead" modules to push
4. **Process Pushes** → For each selected, establish tracking branch if needed, push
5. Print summary with pushed count

**State Management:**

```
SUBMODULE_PATHS[i]          # String key: path to submodule
SUBMODULE_STATUS[i]         # Status string: "pending", "current", "modified", "ahead", "conflict", "selected", "push-selected"
SUBMODULE_BRANCH[i]         # Target branch name (detected via smart detection)
SUBMODULE_CURRENT_BRANCH[i] # Current branch in working tree
SUBMODULE_IS_FEATURE[i]     # "Yes" or "No" - is on non-standard branch
SUBMODULE_BEHIND[i]         # Integer: number of commits behind remote
SUBMODULE_CHANGELOG[i]      # Multiline git log of incoming commits
```

Access is via `set_array_value()` and `get_array_value()` helper functions which perform linear search in `SUBMODULE_PATHS[]` to find index, then read/write corresponding parallel array.

## Key Abstractions

**Smart Branch Detection:**
- Purpose: Select the best target branch for a submodule
- Examples: `detect_best_branch()` function (lines 533-590)
- Pattern: Priority order (develop → master → main) with fallback to remote HEAD, then first available branch
- Special behavior: If on a feature branch (not in priority list) with remote tracking, preserves current branch

**Conflict Handling:**
- Purpose: Automatically resolve merge conflicts without data loss
- Examples: `handle_conflict()` function (lines 935-963)
- Pattern: Stash local changes → retry merge → optionally reapply stash
- Fallback: If retry fails, abort merge and restore stash, report error

**Backup/Rollback:**
- Purpose: Enable point-in-time recovery of submodule SHAs
- Examples: `create_backup()` (lines 799-846), `rollback_all()` (lines 861-890)
- Pattern: JSON serialization stored in `~/.ssu/<project-name>/`
- Format: `{"timestamp": "ISO-8601", "submodules": {"path": {"sha": "hash", "branch": "name"}}}`

**TUI Selector:**
- Purpose: Provide interactive selection with visual feedback
- Examples: `tui_select()` function (lines 248-369), `draw_tui_screen()` (lines 195-244)
- Pattern: Terminal control via tput with ANSI fallbacks, single-character input handling
- Input: Arrow keys/vim keys (hjkl) for navigation, space to toggle, a/A for all/none, Enter to confirm, q to quit
- State: `TUI_ITEMS[]`, `TUI_SELECTED[]`, `TUI_CURSOR` position

**Root Repository Display:**
- Purpose: Show root repository status in status table alongside submodules
- Examples: `get_root_status()` (lines 683-714), `get_root_current_branch()` (lines 661-666)
- Pattern: Root stored in indexed arrays with path key "."
- Special handling: Display path as "(root)" in bold, excluded from selection and modification

## Entry Points

**Main Entry Point:**
- Location: `ssu` line 1559-1654 (`main()` function)
- Triggers: Script execution with optional CLI flags
- Responsibilities:
  - Argument parsing via `parse_args()` (lines 452-502)
  - Verify git repo with `.gitmodules` file
  - Initialize backup directory
  - Orchestrate three-phase scan via `scan_submodules()`
  - Route to rollback, status-only, update, or push mode
  - Return appropriate exit code

**CLI Flags (via parse_args):**
- `-h, --help`: Show usage and exit 0
- `-a, --auto`: Set `MODE="auto"` (batch processing, no prompts)
- `-d, --dry-run`: Set `DRY_RUN=true` (preview without modifying)
- `-b, --branch`: Set `OVERRIDE_BRANCH` (force all submodules to specific branch)
- `-f, --fail-fast`: Set `FAIL_FAST=true` (exit immediately on error)
- `-s, --status`: Set `STATUS_ONLY=true` (scan and display only)
- `-p, --push`: Set `PUSH_MODE=true` (operate in push mode, not update mode)
- `-r, --rollback FILE`: Set `ROLLBACK_FILE` (restore from backup, then exit)

**Exit Codes:**
- 0: Success
- 1: Error or push failures
- 2: Conflicts detected (merge conflicts) or fail-fast triggered

## Error Handling

**Strategy:** Fail-safe approach: never lose data, always provide recovery path

**Patterns:**

**Merge Conflicts (lines 935-963):**
- Detect: `git merge` exit code non-zero
- Stash local changes to preserve them
- Retry merge on clean state
- On success, attempt to reapply stash
- On failure, abort merge and restore stash, report error

**Push Failures (lines 965-999):**
- Detached HEAD: Return code 2, skip with warning
- No tracking branch: Automatically set up `git push -u origin <branch>`
- Push failure: Log error, continue unless `--fail-fast`

**Submodule Not Initialized (lines 522-527, 1032-1034):**
- Check for `.git` file/directory presence
- Skip with "missing" status if absent
- Log and continue (does not block processing)

**Network Failures (lines 1048-1051, 896-903):**
- `git fetch` failures silently ignored (use cached refs)
- `git` commands always redirect errors to `/dev/null`
- No network failure stops the script

**Logging (lines 375-384):**
- All operations logged to `~/.ssu/<project-name>/logs/submodule-update.log`
- Timestamp, level (INFO/SUCCESS/ERROR/WARNING), and message
- Creates log directory if needed

## Cross-Cutting Concerns

**Logging:**
- Function: `log()` (lines 375-384)
- Usage: Called after major operations (update, conflict, push, rollback)
- Location: `~/.ssu/<project-name>/logs/submodule-update.log` with timestamps

**Validation:**
- Submodule initialization: `is_submodule_initialized()` (lines 522-527)
- Skip list membership: `is_skipped()` (lines 508-520)
- Detached HEAD detection: `is_detached_head()` (lines 636-643)
- Local changes detection: `has_local_changes()` (lines 606-614)

**Authentication:**
- Git credential helpers used for SSH/HTTPS
- No explicit authentication code (delegates to git)
- Fetch failures silently ignored (uses cached refs if available)

**Color Output:**
- Detects TTY: `[[ -t 1 ]]` (line 50)
- Disables colors for pipes/redirects
- Color constants defined lines 50-63
- Applied in: `print_status()`, `display_status_line()`, `draw_tui_screen()`, installation script

**Terminal Control:**
- Cursor hiding: `hide_cursor()` / `show_cursor()` (lines 168-174)
- Screen clear: `clear_screen()` (lines 181-183)
- Cursor positioning: `move_cursor_to_line()` (lines 176-179)
- Terminal dimensions: `get_terminal_height()` (lines 185-187)
- Fallbacks: All use `tput` with ANSI escape code fallbacks

---

*Architecture analysis: 2026-02-09*
