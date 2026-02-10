# Coding Conventions

**Analysis Date:** 2026-02-09

## Naming Patterns

**Files:**
- Single executable script: `ssu` (no extension, marked executable)
- Installer script: `install.sh` (shell script with .sh extension)
- Uppercase constants with underscores: `SCRIPT_DIR`, `PROJECT_ROOT`, `SKIP_LIST`, `BRANCH_PRIORITY`
- File names follow lowercase with hyphens for multi-word documentation: `CLAUDE.md`, `README.md`

**Functions:**
- Lowercase with underscores for public functions: `set_array_value()`, `detect_best_branch()`, `scan_submodules()`
- Private/helper functions prefixed with underscore: `_get_path_index()` (lines 103-114 in `ssu`)
- Descriptive verb-noun pairs: `get_*`, `set_*`, `is_*`, `has_*`, `detect_*`, `print_*`, `show_*`, `create_*`, `init_*`
- Short function names for frequently used utilities: `log()`, `display_status_table()`, `process_updates()`

**Variables:**
- Global state variables in UPPERCASE: `MODE`, `DRY_RUN`, `FAIL_FAST`, `STATUS_ONLY`, `PUSH_MODE`, `UPDATED_COUNT`
- Local variables in lowercase with underscores: `local idx`, `local message`, `local timestamp`
- Array variables suffixed with array type: `SUBMODULE_PATHS=()`, `TUI_ITEMS=()`, `TUI_SELECTED=()`, `BRANCH_PRIORITY=()`
- Configuration variables prefixed with domain: `SSU_HOME`, `BACKUP_PREFIX`, `LOG_DIR`
- Flags stored as boolean strings: `BACKUPS_ENABLED=true`, `DRY_RUN=false`

**Constants:**
- Color codes in UPPERCASE: `RED`, `GREEN`, `YELLOW`, `BLUE`, `MAGENTA`, `CYAN`, `BOLD`, `NC` (lines 51-62 in `ssu`)
- Exit codes with semantic meaning: `return 0` (success), `return 1` (error), `return 2` (special state like detached HEAD)
- Settings near top of script: `DEFAULT_PARALLEL_JOBS=8` (line 44), `BRANCH_PRIORITY=("develop" "master" "main")` (line 41)

## Code Style

**Formatting:**
- No external formatter (relies on consistent manual formatting)
- 4-space indentation throughout (not tabs)
- Lines generally under 120 characters
- Colors detected via TTY: `if [[ -t 1 ]]; then` disables colors for pipes (lines 50-63 in `ssu`)

**Linting:**
- Uses `set -euo pipefail` at top of every script (line 18 in `ssu`, line 13 in `install.sh`)
  - `set -e`: Exit on any error
  - `set -u`: Error on undefined variable reference
  - `set -o pipefail`: Pipeline fails if any command fails
- Shellcheck annotations used for suppressions: `# shellcheck disable=SC1091` (install.sh line 94)
- No Bash 4.0+ features used (Bash 3.2 compatibility required):
  - No `[[` (use `[` or `test`)
  - No associative arrays (use parallel indexed arrays)
  - No `&>` redirection (use `2>&1`)
  - No command substitution with `$()` (use backticks where needed for compatibility)

## Import Organization

**Not applicable** - This is a single-file bash script with no module system. All functions are defined within one file.

**Code organization by section:**
1. Shebang and header comments
2. Configuration section (variables, arrays, settings)
3. Colors and formatting setup
4. Global state declarations
5. Array helpers (Bash 3.2 compatibility layer)
6. TUI functions (terminal control)
7. Logging functions
8. Usage and help
9. Argument parsing
10. Helper functions (git operations)
11. Root repository functions
12. Backup/rollback functions
13. Update functions
14. Status and display functions
15. Interactive mode functions
16. Main processing functions
17. Summary and exit handling
18. Main entry point

**Section markers:**
- Delimited by `# =============================================================================` comment blocks
- Each section has clear purpose comment: `# CONFIGURATION`, `# TUI FUNCTIONS`, `# LOGGING`, etc. (lines 20-1557)

## Error Handling

**Strategy:** Defensive programming with explicit exit codes and graceful degradation

**Patterns:**

1. **Command substitution with error handling:**
   ```bash
   local timestamp
   timestamp="$(date '+%Y-%m-%d %H:%M:%S')"
   # Uses simple assignment, date assumed to succeed
   ```

2. **Conditional git operations (allow failures, handle gracefully):**
   ```bash
   cd "$PROJECT_ROOT/$path"
   git fetch origin "$branch" --quiet 2>/dev/null || true
   # Redirect stderr to /dev/null, allow failure without aborting
   ```

3. **Function return codes with semantic meaning:**
   ```bash
   push_submodule() {
       if [[ "$current_branch" == "detached" ]]; then
           return 2  # Special return code for detached HEAD (line 976)
       fi
   }
   # Callers check: if [[ $push_result -eq 2 ]]; then (line 1510)
   ```

4. **Try-catch-like pattern for conflict handling:**
   ```bash
   handle_conflict() {
       git stash --quiet 2>/dev/null || true
       if git merge "origin/$branch" --quiet 2>/dev/null; then
           # Try to restore stash
           git stash pop --quiet 2>/dev/null || {
               print_status WARNING "Stash could not be applied"
           }
           return 0
       else
           git merge --abort 2>/dev/null || true
           return 1
       fi
   }
   # Pattern from lines 935-963
   ```

5. **Validation with early return:**
   ```bash
   if [[ ! -f ".gitmodules" ]]; then
       print_status ERROR "No .gitmodules found"
       exit 1
   fi
   # From lines 1582-1585
   ```

6. **Directory creation with fallback:**
   ```bash
   mkdir -p "$LOG_DIR"
   echo "[$timestamp] [$level] $message" >> "$LOG_DIR/submodule-update.log"
   # From lines 382-383: assumes mkdir succeeds, logs to file
   ```

**Error handling philosophy:**
- Pre-flight checks at start of main functions (check git repo, .gitmodules exists)
- Most git operations redirect stderr to /dev/null to avoid noise
- User-facing errors use `print_status ERROR` with clear messages
- Internal logging uses `log ERROR` function for persistent record (line 375-384)
- `set -euo pipefail` catches unexpected errors early

## Logging

**Framework:** Custom `log()` function (lines 375-384)

**Patterns:**
- Function signature: `log "LEVEL" "message"`
- Log levels: `INFO`, `SUCCESS`, `WARNING`, `ERROR`
- Output format: `[YYYY-MM-DD HH:MM:SS] [LEVEL] message`
- Logs appended to: `~/.ssu/<project-name>/logs/submodule-update.log`
- Prints to user via `print_status()` function (lines 386-401) with color coding

**Usage examples:**
```bash
log INFO "Starting submodule scan..." # line 1569
log "INFO" "Updated $path to latest $branch" # line 1433
log ERROR "Conflict in $path could not be resolved" # line 1443
```

**Print status function:**
- Color-coded console output: `print_status STATUS "message"`
- Status types: `INFO`, `SUCCESS`, `WARNING`, `ERROR`, `UPDATED`, `SKIPPED`, `CONFLICT`, `UPTODATE`
- Falls back to plain text if not connected to TTY

## Comments

**When to Comment:**
- Before major sections, not before every line
- Explain *why*, not what code does
- Document gotchas and non-obvious behavior

**Patterns:**
```bash
# Section headers (self-documenting)
# =============================================================================
# ARRAY HELPERS (Bash 3.2 compatible associative array simulation)
# =============================================================================

# Short explanation before complex logic
# Note: Call this AFTER fetch so remote refs are available locally
detect_best_branch() {

# Inline comments for non-obvious code
if [[ "$current_branch" != "develop" ]] && [[ "$current_branch" != "master" ]] && ...
    # Check if remote branch exists for current feature branch
    if git show-ref --verify --quiet "refs/remotes/origin/$current_branch"; then
```

**JSDoc/TSDoc:** Not used (bash convention favors minimal comments)

## Function Design

**Size:** Generally 10-50 lines
- Utility functions under 15 lines: `get_current_branch()` (lines 599-604), `is_detached_head()` (lines 636-643)
- Complex functions 20-40 lines: `scan_submodules()` (lines 1005-1103), `process_updates()` (lines 1369-1454)
- Longest function: `tui_select()` at ~120 lines (lines 248-369) - complex TUI handler justified

**Parameters:**
- Usually 1-3 parameters
- First parameter is often the subject: `path`, `array_name`
- Use `"$@"` to pass all arguments: `main "$@"` (line 1656)
- Parameters used directly, not reassigned: `local path="$1"` pattern throughout

**Return Values:**
- Implicit return code (last command's exit status)
- Explicit `return N` with semantic codes: 0=success, 1=error, 2=special state
- Output via `echo` for values meant to be captured: `echo "$value"` (lines 159, 580)
- Side effects (setting globals) without echo for state changes

**Early return pattern:**
```bash
if [[ ! -f ".gitmodules" ]]; then
    print_status ERROR "..."
    exit 1
fi
# Skip checking nested conditions below
```

## Module Design

**Exports:** Not applicable (single bash script, no module system)

**Global variables:** Used as implicit "exports"
- Configuration globals at top: `SCRIPT_DIR`, `PROJECT_ROOT`, `SKIP_LIST`
- State globals initialized: `MODE="interactive"`, `DRY_RUN=false`
- Arrays for Bash 3.2 compatibility: `SUBMODULE_PATHS=()`, `SUBMODULE_STATUS=()`

**Barrel Files:** Not applicable

## Bash 3.2 Compatibility

**Array Simulation (lines 77-95):**
Parallel indexed arrays simulate associative arrays:
```bash
SUBMODULE_PATHS=()          # "keys" array
SUBMODULE_STATUS=()         # "values" array 1
SUBMODULE_BRANCH=()         # "values" array 2
SUBMODULE_CURRENT_BRANCH=() # "values" array 3
```

Access via helper functions only:
- `set_array_value "ARRAY_NAME" "path" "value"` (lines 117-140)
- `get_array_value "ARRAY_NAME" "path"` (lines 143-161)

**String interpolation:**
- Uses `"$variable"` with double quotes (safe with spaces)
- Uses `${variable:-default}` for defaults (lines 1112-1121)
- Uses `${#array[@]}` for array length (compatible with Bash 3.2)

**No unsupported features:**
- No `[[ ]]` conditional (use `[ ]` or `test`)
- No `&>` redirection (use `2>&1`)
- No `declare -A` associative arrays
- No `mapfile` or `readarray`
- No bash arithmetic `((...))`
- Arithmetic via `expr` or `$((...))`  when needed

---

*Convention analysis: 2026-02-09*
