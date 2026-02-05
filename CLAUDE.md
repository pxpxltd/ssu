# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

SSU (Smart Submodule Updater) is a bash-based intelligent git submodule management tool. It provides interactive and automated workflows for updating and pushing submodules with smart branch detection, conflict handling, and backup/rollback capabilities.

**Key characteristics:**
- Single executable bash script (`ssu`) - no dependencies beyond git and standard Unix tools
- Bash 3.2+ compatible (macOS and Linux)
- No external dependencies (jq, etc.) - uses manual JSON parsing
- Designed for safety: backup-before-update, dry-run preview, rollback support
- Push ahead submodules with automatic tracking branch setup
- TUI-style interactive selector with arrow key navigation
- Root repository status display alongside submodules

## Architecture

### Three-Phase Workflow

1. **Scanning Phase**
   - **Phase 0**: Scan root repository (fetch, detect status, display-only)
   - Parallel `git fetch --all` for all submodules (configurable via `PARALLEL_JOBS` env var, default 8)
   - Status analysis using cached remote refs (no additional network calls)
   - Smart branch detection: tries `develop` → `master` → `main` → remote HEAD → fallback

2. **Display & Selection Phase**
   - Colorized status table showing root repository first (as "(root)"), then submodules
   - Table shows: path, current branch, commits behind, feature branch indicator, status
   - Interactive mode: TUI selector with arrow key navigation, space to toggle, visual checkboxes
   - Batch mode (`--auto`): selects all "pending" submodules automatically
   - Root repository is display-only, excluded from selection

3. **Processing Phase**
   - Creates timestamped JSON backup before any modifications (excludes root)
   - Updates selected submodules with automatic conflict handling
   - Tracks counts: updated, skipped, conflicts, errors
   - Root repository is excluded from all operations

### Bash 3.2 Compatible Array Simulation

Since macOS ships with Bash 3.2 (lacks associative arrays), the script uses **parallel indexed arrays** with a custom access pattern:

```bash
# "Key" array - stores submodule paths
SUBMODULE_PATHS=()

# "Value" arrays - indexed in parallel
SUBMODULE_STATUS=()
SUBMODULE_BRANCH=()
SUBMODULE_CURRENT_BRANCH=()
SUBMODULE_IS_FEATURE=()
SUBMODULE_BEHIND=()
SUBMODULE_CHANGELOG=()
```

Access via helper functions:
- `set_array_value "ARRAY_NAME" "path" "value"` - finds/creates index in SUBMODULE_PATHS, sets value
- `get_array_value "ARRAY_NAME" "path"` - linear search for path, returns value at that index

**When modifying:** Never use Bash 4.0+ features (associative arrays, `[[`, `&>`, etc.)

### Smart Branch Detection Algorithm (lines 533-574)

1. Check for `--branch` CLI override (highest priority)
2. **Check if on a feature branch** (not develop/master/main) - if yes and remote exists, use current branch
3. Try branches in priority order: `["develop", "master", "main"]`
4. Uses `git branch -r` (cached from fetch) - no network calls
5. Fallback to `origin/HEAD` symbolic ref
6. Final fallback: first available remote branch or "master"

**Key insights:**
- Branch detection happens AFTER parallel fetch, so all remote refs are locally cached
- Feature branches are preserved - submodules on feature branches will pull from their own remote branch, not switch to develop/master
- Only falls back to priority branches if current branch has no remote tracking branch

### Conflict Handling Strategy (lines 631-659)

Three-step automatic resolution:
1. Detect merge failure
2. Stash local changes (`git stash push`)
3. Retry merge on clean state
4. Re-apply stash if merge succeeds

If automatic resolution fails:
- Abort merge, restore stash, report error
- User can resolve manually or use `--rollback`
- With `--fail-fast`, script exits immediately

### Push Workflow (lines 661-706)

When `--push` flag is used, the script operates in push mode instead of update mode:

**Detection:**
- Identifies submodules with "ahead" status (unpushed commits)
- Uses `has_unpushed_commits()` to detect commits not on remote

**Push function (`push_submodule()`):**
1. Check for detached HEAD state (cannot push)
2. Detect tracking branch (`@{upstream}`)
3. If no tracking branch: set up with `git push -u origin <branch>`
4. If tracking branch exists: push with `git push`
5. Return codes: 0=success, 1=error, 2=detached HEAD

**Interactive mode:**
- Shows list of ahead submodules
- User selects which to push (all/none/specific)
- Confirms each push individually

**Auto mode (`--push --auto`):**
- Automatically pushes all ahead submodules
- No prompts or confirmations

**Dry-run mode (`--push --dry-run`):**
- Shows what would be pushed without pushing
- Displays branch names and status

**Special cases:**
- Submodules in detached HEAD: skipped with warning
- No tracking branch: automatically sets up `origin/<branch>` as upstream
- Push failures: logged as errors, continues unless `--fail-fast`

### TUI Interactive Selector (v1.1.0+)

**Terminal Control Functions:**
- `hide_cursor()` / `show_cursor()` - cursor visibility using tput with ANSI fallbacks
- `move_cursor_to_line(line)` - cursor positioning
- `clear_screen()` - full screen clear
- `get_terminal_height()` - terminal size detection
- `restore_terminal()` - cleanup on exit (trap handler)

**TUI State Management:**
- `TUI_ITEMS[]` - items to display
- `TUI_SELECTED[]` - selection state: "yes"/"no"
- `TUI_CURSOR` - current cursor position

**Core Function: `tui_select(title, items_array_name, metadata_array_name)`**
1. Initialize TUI state arrays by copying from source array
2. Set up terminal (hide cursor, trap for cleanup)
3. Main loop:
   - Redraw screen with `draw_tui_screen()`
   - Read single character with `read -rsn1`
   - Detect arrow keys (escape sequences `\x1b[A`, `\x1b[B`)
   - Handle input: arrows/j/k (move), space (toggle), a/A (all/none), enter (confirm), q (quit)
4. Restore terminal (show cursor, move to bottom)
5. Return 0 if selections made, 1 if cancelled

**TTY Detection:**
- Both `prompt_selection()` and `prompt_push_selection()` check for TTY: `[ -t 0 ] && [ -t 1 ]`
- If TTY: Use TUI selector with visual feedback
- If not TTY: Show warning message to use `--auto` mode

**Visual Feedback:**
- Cursor indicator: `>` on current line (cyan)
- Checkboxes: `[✓]` for selected (green), `[ ]` for unselected
- Metadata displayed next to each item (gray)
- Selection count shown in footer

### Root Repository Display (v1.1.0+)

**Root Status Functions:**
- `get_root_current_branch()` - returns current branch or "detached"
- `get_root_commits_behind()` - count of commits behind remote
- `get_root_status()` - returns "modified", "ahead", "pending", or "current" (priority order)

**Integration with Scanning:**
- Root is scanned in Phase 0 of `scan_submodules()`
- Stored in parallel arrays using "." as the key
- Root data includes: branch, behind count, status, but no changelog

**Display Logic:**
- `display_status_line()` helper function renders a single row
- Special handling: path "." displayed as "(root)" in bold
- Root shown first in status table, then submodules sorted

**Exclusion from Operations:**
- Root is skipped in `prompt_selection()` and `prompt_push_selection()`
- Root is excluded from `process_updates()` and `process_pushes()`
- Root is not included in backup creation (`create_backup()`)
- Users manage root repository manually

### Backup/Rollback Mechanism

**Backup and log location:**
- Backups are stored in `~/.ssu/<project-name>/` (determined by project root directory name)
- Logs are stored in `~/.ssu/<project-name>/logs/` (keeps project directory clean)
- On first run, the script checks if `~/.ssu` and the project-specific directory exist
- In interactive mode: prompts user to create directory if it doesn't exist
- In auto mode (`--auto`): automatically creates the directory
- If user declines directory creation: backups are disabled for that session

**Backup format:**
```json
{
  "timestamp": "2024-01-15T10:30:00+00:00",
  "submodules": {
    "path/to/submodule": {"sha": "abc123", "branch": "develop"}
  }
}
```

- Created before any updates: `~/.ssu/<project-name>/.submodule-backup-YYYYMMDD-HHMMSS.json`
- Manual JSON parsing (regex-based, no jq dependency)
- Rollback via `ssu --rollback <backup-file>`
- Restores exact SHAs (may leave submodules in detached HEAD)

**Implementation details:**
- `get_project_name()` function extracts project directory name
- `init_backup_directory()` handles directory initialization and user prompt
- Global variables: `BACKUPS_ENABLED`, `BACKUP_DIR`, `SSU_HOME`
- Backup creation skipped if `BACKUPS_ENABLED=false`

## Development Commands

### Testing the Script

```bash
# Status check only (no modifications)
./ssu --status

# Dry-run to preview updates
./ssu --dry-run

# Test interactive mode in a git repo with submodules
cd /path/to/project/with/submodules
/path/to/ssu/ssu

# Test batch mode
./ssu --auto --dry-run
```

### Installation Testing

```bash
# Test installer
./install.sh

# Test uninstaller
./install.sh --uninstall

# Manual verification
which ssu
ssu --help
```

### Code Validation

**No formal test suite exists.** Testing requires:
1. A git repository with multiple submodules
2. Submodules with varying states (ahead, behind, modified, feature branches)
3. Manual verification of behavior

**To create test scenarios:**
```bash
# In a submodule:
cd path/to/submodule

# Create "behind" state
git reset --hard HEAD~3

# Create "modified" state
echo "test" >> somefile.txt

# Create "feature branch" state
git checkout -b feature/test

# Create "ahead" state
git commit --allow-empty -m "local commit"
```

### ShellCheck

Run shellcheck for static analysis:
```bash
shellcheck ssu
shellcheck install.sh
```

**Common issues to avoid:**
- No `[[` (use `[` or `test`)
- No `&>` redirection (use `2>&1`)
- No associative arrays
- Quote all variable expansions

## Configuration

### Customizing Skip List

Edit the `SKIP_LIST` array in `ssu` (lines 29-32):
```bash
SKIP_LIST=(
    "plugins/deprecated-module"
    "vendor/legacy-lib"
)
```

### Customizing Branch Priority

Edit `BRANCH_PRIORITY` (line 35):
```bash
BRANCH_PRIORITY=("staging" "develop" "master" "main")
```

### Parallel Jobs

Set via environment variable:
```bash
PARALLEL_JOBS=16 ./ssu --status
```

## File Structure

```
ssu/
├── ssu              # Main executable (single-file bash script, ~950 lines)
├── install.sh       # Cross-platform installer with shell detection
├── README.md        # User-facing documentation
└── LICENSE          # MIT license
```

**No build system, no dependencies, no package manager.**

## Important Implementation Notes

### Performance Considerations

- Parallel fetch is CPU/network bound - limit via `PARALLEL_JOBS`
- Status scanning is O(n) for n submodules (linear lookups in arrays)
- No optimization needed for typical use cases (<50 submodules)

### Error Handling

- `set -euo pipefail` for safety, but git commands explicitly handle errors
- Most functions return exit codes rather than crashing
- Logs all operations to `logs/submodule-update.log` with timestamps

### Color Output

- Detects TTY via `[[ -t 1 ]]` and disables colors for pipes/redirects
- Status indicators: green (pending), cyan (current), yellow (modified), magenta (ahead), red (conflict)

### Portability

**Tested on:**
- macOS (Bash 3.2+)
- Linux (Bash 3.2+, 4.x+, 5.x+)

**Known limitations:**
- Requires GNU coreutils (awk, sed, grep)
- Git 2.0+ required
- No Windows support (WSL/Git Bash may work but untested)

## Making Changes

### Adding New Features

**Before adding features:**
1. Check Bash 3.2 compatibility (no `declare -A`, `[[`, `&>`)
2. Avoid external dependencies (jq, python, etc.)
3. Update README.md examples if adding CLI flags
4. Test on both macOS and Linux if possible

**Common patterns in codebase:**
- Use `print_status` for user-facing messages
- Use `log` for persistent logging
- Use `set_array_value`/`get_array_value` for state tracking
- Colors defined at top (lines 44-56), use `${GREEN}`, `${NC}`, etc.

### Modifying Smart Branch Detection

Location: `detect_best_branch()` function (lines 301-340)

**Example: Add "staging" branch to priority:**
```bash
# Line 35
BRANCH_PRIORITY=("staging" "develop" "master" "main")
```

No other changes needed - function iterates `BRANCH_PRIORITY` array.

### Modifying Status Table

Location: `display_status_table()` function (lines 666-743)

**Table format:**
```
| Path              | Current  | Target   | Behind | Feature |
|-------------------|----------|----------|--------|---------|
| plugins/module1   | develop  | develop  | 3      | No      |
```

Status column colors determined by `SUBMODULE_STATUS` value:
- "pending" → green
- "current" → cyan
- "modified" → yellow
- "ahead" → magenta
- "conflict" → red

### Adding New CLI Flags

1. Add to `usage()` function (lines 186-220)
2. Add to `parse_args()` case statement (lines 223-271)
3. Declare global variable near top (lines 63-68)
4. Implement feature logic in appropriate phase function

## Logging

All operations logged to `~/.ssu/<project-name>/logs/submodule-update.log`:
```
[2024-01-15 10:30:00] [INFO] Starting submodule scan...
[2024-01-15 10:30:05] [SUCCESS] Updated plugins/module1
[2024-01-15 10:30:10] [ERROR] Conflict in plugins/module2
```

This keeps the project directory clean - no log files clutter the repository.

Use `log` function:
```bash
log "INFO" "Your message here"
log "ERROR" "Something failed"
```

## Version History

Version information in script header (line 4). Current: 1.1.1

**Recent changes (from git log):**
- v1.0.3: Better detection of non-committed changes
- v1.0.4: Local .ssu folder with backups
- v1.0.5: Store logs in home folder
- v1.0.6: Add push functionality for ahead submodules
- v1.1.0: TUI interactive selector + root repository display
- v1.1.1: Fix feature branch switching bug - respects current branch (Current version)

When incrementing version:
1. Update line 4 in `ssu`
2. Update line 5 in `README.md` badge
3. Tag the commit: `git tag -a v1.0.x -m "Release 1.0.x"`
