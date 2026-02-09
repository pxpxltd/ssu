# Codebase Structure

**Analysis Date:** 2026-02-09

## Directory Layout

```
ssu/
├── ssu                 # Main executable (950 lines, Bash 3.2+)
├── install.sh          # Cross-platform installer (560 lines, handles Linux/macOS/BSD)
├── README.md           # User-facing documentation with examples
├── LICENSE             # MIT license
└── CLAUDE.md           # Project guidelines for Claude AI (this repo only)
```

**No other directories.** This is intentionally minimal:
- No `src/` or `lib/` subdirectories
- No build artifacts
- No configuration directories
- All code in single executable file

## Directory Purposes

**Project Root (`ssu/`):**
- Purpose: Complete, self-contained distribution of SSU tool
- Contains: Single executable, installer, documentation
- Key files: `ssu` (main), `install.sh` (setup), `README.md` (user guide)

## Key File Locations

**Entry Points:**
- `ssu`: Main executable entry point (shebang: `#!/usr/bin/env bash`, line 1)
  - Main function at line 1559
  - Invokes `parse_args()` → orchestrates scanning and processing

- `install.sh`: Installation entry point (shebang: `#!/usr/bin/env bash`, line 1)
  - Main function at line 411
  - Handles OS detection, path selection, symlink creation

**Configuration (Hardcoded in `ssu`):**
- Line 24: `SCRIPT_DIR` - Current script directory
- Line 25: `PROJECT_ROOT` - Current working directory when script runs
- Lines 29-32: `SSU_HOME`, `BACKUP_PREFIX` - Backup directory configuration
- Lines 35-38: `SKIP_LIST` - Array of submodules to skip (empty by default)
- Lines 40-41: `BRANCH_PRIORITY` - Branch detection priority order
- Line 44: `DEFAULT_PARALLEL_JOBS` - Parallel fetch concurrency (default 8)

**Core Logic Sections (in `ssu`):**

| Section | Lines | Purpose |
|---------|-------|---------|
| Configuration | 18-44 | Global vars, colors, skip list, branch priority |
| Colors & Formatting | 50-63 | TTY detection and ANSI color codes |
| Global State | 69-96 | Mode flags, indexed arrays, counters, TUI state |
| Array Helpers | 103-161 | `set_array_value()`, `get_array_value()` for Bash 3.2 |
| TUI Functions | 168-369 | Terminal control, cursor, screen, main `tui_select()` |
| Logging | 375-401 | `log()` and `print_status()` functions |
| Usage & Help | 407-446 | `usage()` function for `--help` |
| Argument Parsing | 452-502 | `parse_args()` to set global flags from CLI |
| Helper Functions | 508-643 | Status checking: skip, init, branch, local changes, detached HEAD |
| Root Functions | 661-732 | Root repo status, branch, behind count |
| Backup Functions | 799-890 | `create_backup()`, `rollback_all()`, directory init |
| Update Functions | 896-999 | `fetch_submodule()`, `update_submodule()`, `push_submodule()`, `handle_conflict()` |
| Display Functions | 1005-1222 | `scan_submodules()`, `display_status_table()`, `show_preview()` |
| Interactive Mode | 1228-1363 | `prompt_selection()`, `confirm_update()`, push prompts |
| Main Processing | 1369-1553 | `process_updates()`, `process_pushes()`, summary |
| Main Entry | 1559-1654 | `main()` function with orchestration logic |

**Testing/Validation:**
- No formal test suite
- Manual validation via git repos with submodules
- ShellCheck compatible (no `[[`, `&>`, associative arrays)

## Naming Conventions

**Files:**
- Executable: lowercase with dashes (`install.sh`, `ssu`)
- Documentation: uppercase with extensions (`README.md`, `LICENSE`, `CLAUDE.md`)
- No version numbers in filenames (use git tags)

**Functions (in `ssu`):**
- Verb prefixes: `get_*`, `is_*`, `detect_*`, `process_*`, `create_*`, `rollback_*`, `handle_*`, `prompt_*`, `display_*`, `scan_*`, `push_*`, `update_*`
- Examples:
  - `get_submodule_paths()` - Retrieve submodule paths from git config
  - `is_submodule_initialized()` - Check if `.git` exists
  - `detect_best_branch()` - Find target branch via priority list
  - `has_local_changes()` - Check working tree status
  - `is_detached_head()` - Verify branch state
  - `handle_conflict()` - Auto-resolve merge conflicts

**Variables:**
- Global flags: UPPERCASE (`MODE`, `DRY_RUN`, `PUSH_MODE`, `STATUS_ONLY`)
- Global counters: UPPERCASE (`UPDATED_COUNT`, `SKIPPED_COUNT`, `ERROR_COUNT`)
- Global arrays: UPPERCASE prefixed with context (`SUBMODULE_*`, `TUI_*`, `INSTALL_OPTIONS`)
- Local variables: lowercase with underscores (`local_changes`, `current_branch`, `backup_file`)
- Constants: UPPERCASE (`SCRIPT_DIR`, `PROJECT_ROOT`, `SSU_HOME`)

**Color Variables:**
- Prefixed `BOLD`, `RED`, `GREEN`, `YELLOW`, `BLUE`, `CYAN`, `GRAY`, `WHITE`, `MAGENTA`, `NC` (no color)
- Example: `echo -e "${GREEN}Success${NC}"`

## Where to Add New Code

**New Feature (e.g., new status type):**
- Status classification logic: Add case statement in `scan_submodules()` phase 3 (around line 1093-1101)
- Status display color: Add case statement in `display_status_line()` (around line 1143-1153)
- Status handling in updates: Add to `process_updates()` status check (around line 1396-1405)
- Status handling in pushes: Add to `process_pushes()` status check (around line 1472-1481)
- Documentation: Update README.md status legend section

**New CLI Flag:**
- Flag parsing: Add to `parse_args()` case statement (lines 454-500)
- Global variable: Declare near top with other flags (lines 69-74)
- Logic implementation: Add to appropriate processing function
- Help text: Add to `usage()` function (lines 407-446)
- Example: Add to examples section in README.md

**New Conflict Handling Strategy:**
- Primary implementation: `handle_conflict()` function (lines 935-963)
- Alternative: Create new function and call from `process_updates()` (line 1436)
- Test: Create test scenario with local changes + incoming update

**New Backup/Rollback Feature:**
- Backup creation: Modify `create_backup()` function (lines 799-846)
- Rollback parsing: Modify `rollback_all()` regex (line 875)
- Storage location: Use `BACKUP_DIR` variable (initialized in `init_backup_directory()`)

**Modifying TUI Display:**
- Screen rendering: Update `draw_tui_screen()` function (lines 195-244)
- Input handling: Add cases to main loop (lines 276-368)
- State arrays: May need to add to `TUI_ITEMS`, `TUI_SELECTED` initialization
- Example: To add a new selection key, add case in lines 329-366

**Adding New Helper Function:**
- Location: Add after existing helpers in same category
  - Status helpers (lines 508-643)
  - Root helpers (lines 661-732)
  - Backup helpers (lines 799-890)
  - Update helpers (lines 896-999)
- Follow naming convention: verb prefix + descriptive name
- Use `cd "$PROJECT_ROOT"` for submodule operations, restore with `cd "$PROJECT_ROOT"` at end
- Quote all variable expansions to prevent globbing

## Special Directories

**Backup Storage:**
- Location: `~/.ssu/<project-name>/` (user's home directory)
- Committed: No (user-local, not in git)
- Generated: Yes, on first update or auto mode
- Purpose: Store JSON backup files and logs
- Structure:
  ```
  ~/.ssu/
  └── my-project/
      ├── .submodule-backup-20240101-120000.json
      ├── .submodule-backup-20240102-143000.json
      └── logs/
          └── submodule-update.log
  ```

**Logs:**
- Location: `~/.ssu/<project-name>/logs/` (kept out of project)
- Committed: No
- Generated: Yes, automatically created on first log entry
- Format: One log per project, appended to with timestamps
- Initialization: Done in `main()` at line 1567

## Bash Compatibility Layer

**Bash 3.2 Support (macOS):**
- No associative arrays (`declare -A`) - use parallel indexed arrays instead
- No `[[` - use `[` or `test` instead
- No `&>` redirection - use `2>&1` instead
- No extended features - use portable alternatives
- ShellCheck enforces this via linting

**Array Simulation Pattern (lines 77-84 and 103-161):**

```bash
# Instead of: SUBMODULE_STATUS["path/to/module"] = "pending"
# Use parallel arrays:
SUBMODULE_PATHS=()          # Primary index: path
SUBMODULE_STATUS=()         # Value array: status
SUBMODULE_BRANCH=()         # Value array: branch

# Helper functions handle the mapping:
set_array_value "SUBMODULE_STATUS" "path/to/module" "pending"
# → finds index of "path/to/module" in SUBMODULE_PATHS
# → sets SUBMODULE_STATUS[$index] = "pending"

get_array_value "SUBMODULE_STATUS" "path/to/module"
# → linear search for "path/to/module" in SUBMODULE_PATHS
# → returns SUBMODULE_STATUS[$index]
```

**Performance:** O(n) lookups are acceptable for typical submodule counts (<50). For larger repos, consider pre-caching index lookups.

---

*Structure analysis: 2026-02-09*
